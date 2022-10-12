package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/cbroglie/mustache"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	repov1alpha1 "github.com/krateoplatformops/provider-git/apis/repo/v1alpha1"
	"github.com/krateoplatformops/provider-git/pkg/clients"
	"github.com/krateoplatformops/provider-git/pkg/clients/deployment"
	"github.com/krateoplatformops/provider-git/pkg/clients/git"
	"github.com/krateoplatformops/provider-git/pkg/clients/repo"
	"github.com/krateoplatformops/provider-git/pkg/helpers"

	"github.com/crossplane/crossplane-runtime/pkg/controller"

	gi "github.com/sabhiram/go-gitignore"

	corev1 "k8s.io/api/core/v1"
)

const (
	labDeploymentId = "deploymentId"

	errNotRepo                         = "managed resource is not a repo custom resource"
	errMissingDeploymentIdLabel        = "managed resource is missing 'deploymentId' label"
	errUnableToLoadConfigMapWithValues = "unable to load configmap with template values"
	errConfigMapValuesNotReadyYet      = "configmap values not ready yet"
)

// Setup adds a controller that reconciles Token managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(repov1alpha1.RepoGroupKind)

	log := o.Logger.WithValues("controller", name)

	recorder := mgr.GetEventRecorderFor(name)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(repov1alpha1.RepoGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:     mgr.GetClient(),
			log:      log,
			recorder: recorder,
		}),
		managed.WithLogger(log),
		managed.WithRecorder(event.NewAPIRecorder(recorder)))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&repov1alpha1.Repo{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube     client.Client
	log      logging.Logger
	recorder record.EventRecorder
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*repov1alpha1.Repo)
	if !ok {
		return nil, errors.New(errNotRepo)
	}

	cfg, err := clients.GetConfig(ctx, c.kube, cr)
	if err != nil {
		return nil, err
	}

	return &external{
		kube: c.kube,
		log:  c.log,
		cfg:  cfg,
		rec:  c.recorder,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube client.Client
	log  logging.Logger
	cfg  *clients.Config
	rec  record.EventRecorder
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*repov1alpha1.Repo)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepo)
	}

	deploymentID := getDeploymentId(mg)
	if len(deploymentID) == 0 {
		return managed.ExternalObservation{}, errors.New(errMissingDeploymentIdLabel)
	}

	spec := cr.Spec.ForProvider.DeepCopy()

	fromPath := helpers.StringValue(spec.FromRepo.Path)
	if len(fromPath) > 0 {
		js, err := helpers.GetConfigMapValue(ctx, e.kube, spec.ConfigMapKeyRef)
		if err != nil {
			e.log.Debug("Unable to load configmap",
				"name", spec.ConfigMapKeyRef.Name,
				"key", spec.ConfigMapKeyRef.Key,
				"namespace", spec.ConfigMapKeyRef.Namespace)
			return managed.ExternalObservation{}, errors.New(errUnableToLoadConfigMapWithValues)
		}

		if strings.TrimSpace(js) == "" {
			return managed.ExternalObservation{}, errors.New(errConfigMapValuesNotReadyYet)
		}
	}

	toRepo, err := git.Clone(spec.ToRepo.Url, e.cfg.ToRepoCreds, e.cfg.Insecure)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	e.log.Debug("Target repo cloned", "url", spec.ToRepo.Url)

	clmOk, err := toRepo.Exists("claim.yaml")
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if clmOk {
		e.log.Debug("Claim found", "url", spec.ToRepo.Url)

		cr.Status.AtProvider.DeploymentId = helpers.StringPtr(getDeploymentId(mg))
		cr.SetConditions(xpv1.Available())

		return managed.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: true,
		}, nil
	}

	e.log.Debug("Target repo is empty", "url", spec.ToRepo.Url)

	return managed.ExternalObservation{
		ResourceExists:   false,
		ResourceUpToDate: true,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*repov1alpha1.Repo)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepo)
	}

	cr.Status.SetConditions(xpv1.Creating())

	spec := cr.Spec.ForProvider.DeepCopy()

	deploymentId := getDeploymentId(mg)

	claim, err := deployment.Get(e.cfg.DeploymentServiceUrl, deploymentId)
	if err != nil {
		return managed.ExternalCreation{},
			fmt.Errorf("fetching deployment (deploymentId: %s): %w", deploymentId, err)
	}

	e.log.Debug("Claim fetched", "deploymentId", deploymentId)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "ClaimFetched", "Successfully fetched claim for deployment: %s", deploymentId)

	toRepo, err := git.Clone(spec.ToRepo.Url, e.cfg.ToRepoCreds, e.cfg.Insecure)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo cloned", "url", spec.ToRepo.Url)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "TargetRepoCloned", "Successfully cloned target repo: %s", spec.ToRepo.Url)

	fromRepo, err := git.Clone(spec.FromRepo.Url, e.cfg.FromRepoCreds, e.cfg.Insecure)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Origin repo cloned", "url", spec.FromRepo.Url)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "OriginRepoCloned", "Successfully cloned origin repo: %s", spec.FromRepo.Url)

	err = toRepo.Branch("main")
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo on branch main")

	co := &repo.CopyOpts{
		FromRepo: fromRepo,
		ToRepo:   toRepo,
	}

	// If fromPath is not specified DON'T COPY!
	fromPath := helpers.StringValue(spec.FromRepo.Path)
	if len(fromPath) > 0 {
		values, err := e.loadValuesFromConfigMap(ctx, spec.ConfigMapKeyRef)
		if err != nil {
			e.log.Debug("Unable to load configmap with template data", "msg", err.Error())
			e.rec.Eventf(cr, corev1.EventTypeWarning, "CannotLoadConfigMap", "Unable to load configmap with template data: %s", err.Error())
		}

		e.log.Debug("Loaded values from config map",
			"name", spec.ConfigMapKeyRef.Name,
			"key", spec.ConfigMapKeyRef.Key,
			"namespace", spec.ConfigMapKeyRef.Namespace,
			"values", values,
		)

		if err := loadIgnoreFileEventually(co); err != nil {
			e.log.Info("Unable to load '.krateoignore'", "msg", err.Error())
			e.rec.Eventf(cr, corev1.EventTypeWarning, "CannotLoadIgnoreFile", "Unable to load '.krateoignore' file: %s", err.Error())
		}

		createRenderFunc(co, values)

		toPath := helpers.StringValue(spec.ToRepo.Path)
		if len(toPath) == 0 {
			toPath = "/"
		}

		err = co.CopyDir(fromPath, toPath)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	// write claim data
	err = co.WriteBytes(claim, "claim.yaml")
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	e.log.Debug("Origin and target repo synchronized",
		"deploymentId", deploymentId,
		"fromUrl", spec.FromRepo.Url,
		"toUrl", spec.ToRepo.Url,
		"fromPath", helpers.StringValue(spec.FromRepo.Path),
		"toPath", helpers.StringValue(spec.ToRepo.Path))
	e.rec.Eventf(cr, corev1.EventTypeNormal, "RepoSyncSuccess", "Origin and target repo synchronized")

	commitId, err := toRepo.Commit(".", ":rocket: first commit")
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo committed branch main", "commitId", commitId)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "RepoCommitSuccess", "Target repo committed branch main")

	err = toRepo.Push("origin", "main", e.cfg.Insecure)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo pushed branch main", "deploymentId", deploymentId)
	e.rec.Eventf(cr, corev1.EventTypeNormal, "RepoPushSuccess", "Target repo pushed branch main")

	cr.Status.SetConditions(xpv1.Available())
	cr.Status.AtProvider.DeploymentId = helpers.StringPtr(deploymentId)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil // noop
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*repov1alpha1.Repo)
	if !ok {
		return errors.New(errNotRepo)
	}

	cr.Status.SetConditions(xpv1.Creating())

	return nil // noop
}

func (e *external) loadValuesFromConfigMap(ctx context.Context, ref *helpers.ConfigMapKeySelector) (map[string]interface{}, error) {
	var res map[string]interface{}

	js, err := helpers.GetConfigMapValue(ctx, e.kube, ref)
	if err != nil {
		e.log.Debug(err.Error(), "name", ref.Name, "key", ref.Key, "namespace", ref.Namespace)
		return nil, err
	}

	err = json.Unmarshal([]byte(js), &res)
	if err != nil {
		e.log.Debug(err.Error(), "json", js)
		return nil, err
	}

	return res, nil
}

func createRenderFunc(cfg *repo.CopyOpts, values interface{}) {
	cfg.RenderFunc = func(in io.Reader, out io.Writer) error {
		bin, err := ioutil.ReadAll(in)
		if err != nil {
			return err
		}
		tmpl, err := mustache.ParseString(string(bin))
		if err != nil {
			return err
		}

		return tmpl.FRender(out, values)
	}
}

func loadIgnoreFileEventually(cfg *repo.CopyOpts) error {
	fp, err := cfg.FromRepo.FS().Open(".krateoignore")
	if err != nil {
		return err
	}
	defer fp.Close()

	bs, err := ioutil.ReadAll(fp)
	if err != nil {
		return err
	}

	lines := strings.Split(string(bs), "\n")

	cfg.Ignore = gi.CompileIgnoreLines(lines...)

	return nil
}

func getDeploymentId(mg resource.Managed) string {
	for k, v := range mg.GetLabels() {
		if k == labDeploymentId {
			return v
		}
	}

	return ""
}
