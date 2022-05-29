package repo

import (
	"context"
	"errors"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

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
)

const (
	errNotRepo = "managed resource is not a repo custom resource"

	reasonCannotCreate = "CannotCreateExternalResource"
	reasonCreated      = "CreatedExternalResource"
)

// Setup adds a controller that reconciles Token managed resources.
func Setup(mgr ctrl.Manager, l logging.Logger, rl workqueue.RateLimiter) error {
	name := managed.ControllerName(repov1alpha1.RepoGroupKind)

	opts := controller.Options{
		RateLimiter: ratelimiter.NewDefaultManagedRateLimiter(rl),
	}

	log := l.WithValues("controller", name)

	rec := managed.NewReconciler(mgr,
		resource.ManagedKind(repov1alpha1.RepoGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:     mgr.GetClient(),
			log:      log,
			recorder: mgr.GetEventRecorderFor(name),
		}),
		managed.WithLogger(log),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(opts).
		For(&repov1alpha1.Repo{}).
		Complete(rec)
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

	spec := cr.Spec.ForProvider.DeepCopy()

	toRepoUrl := spec.ToRepo.Url

	// Check if target repo exists
	tags, err := git.Tags(toRepoUrl, e.cfg.ToRepoCreds)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("%s :%w", toRepoUrl, err)
	}

	if len(tags) > 0 {
		e.log.Debug("Target repo is not empty", "url", toRepoUrl)
		e.rec.Event(cr, corev1.EventTypeWarning, reasonCannotCreate, "Target repo is not empty")

		cr.SetConditions(xpv1.Available())
		return managed.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: true,
		}, nil
	}

	e.log.Debug("Target repo is empty", "url", toRepoUrl)

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
	deploymentId := helpers.StringValue(spec.DeploymentId)
	deployment, err := deployment.Get(e.cfg.DeploymentServiceUrl, deploymentId)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Claim and Package info fetched", "deploymentId", deploymentId)
	e.rec.Event(cr, corev1.EventTypeNormal, reasonCreated,
		fmt.Sprintf("Claim and Package info fetched (deploymentId: %s)", deploymentId))

	toRepo, err := git.Clone(spec.ToRepo.Url, e.cfg.ToRepoCreds)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo cloned", "url", spec.ToRepo.Url)
	e.rec.Event(cr, corev1.EventTypeNormal, reasonCreated, "Target repo cloned")

	fromRepo, err := git.Clone(spec.FromRepo.Url, e.cfg.FromRepoCreds)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Origin repo cloned", "url", spec.FromRepo.Url)
	e.rec.Event(cr, corev1.EventTypeNormal, reasonCreated, "Origin repo cloned")

	err = toRepo.Branch("main")
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo on branch main")

	// If fromPath is not specified DON'T COPY!
	fromPath := helpers.StringValue(spec.FromRepo.Path)
	if len(fromPath) > 0 {
		err = repo.Copy(repo.CopyOpts{
			FromRepo: fromRepo,
			ToRepo:   toRepo,
			FromPath: helpers.StringValue(spec.FromRepo.Path),
			ToPath:   helpers.StringValue(spec.ToRepo.Path),
		})
		if err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	// write claim data
	err = repo.CopyBytes(toRepo.FS(), deployment.Claim, "claim.yaml")
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	// write package data
	err = repo.CopyBytes(toRepo.FS(), deployment.Package, "package.yaml")
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	e.log.Debug("Origin and target repo synchronized",
		"deploymentId", spec.DeploymentId,
		"fromUrl", spec.FromRepo.Url,
		"toUrl", spec.ToRepo.Url,
		"fromPath", helpers.StringValue(spec.FromRepo.Path),
		"toPath", helpers.StringValue(spec.ToRepo.Path))
	e.rec.Event(cr, corev1.EventTypeNormal, reasonCreated,
		fmt.Sprintf("Origin and target repo synchronized (deploymentId: %s)", deploymentId))

	commitId, err := toRepo.Commit(".", ":rocket: first commit")
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo committed branch main", "commitId", commitId)
	e.rec.Event(cr, corev1.EventTypeNormal, reasonCreated,
		fmt.Sprintf("Target repo committed (deploymentId:%s, commitId:%s)", deploymentId, commitId))

	err = toRepo.Push("origin", "main")
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo pushed branch main", "deploymentId", deploymentId)
	e.rec.Event(cr, corev1.EventTypeNormal, reasonCreated,
		fmt.Sprintf("Target repo pushed branch main (deploymentId:%s)", deploymentId))

	tagged, err := toRepo.CreateTag("0.1.0")
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("create tag error: %w", err)
	}

	if tagged {
		err = toRepo.PushTags(e.cfg.ToRepoCreds)
		if err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("push tag error: %w", err)
		}
	}

	cr.Status.AtProvider.DeploymentId = helpers.StringPtr(deploymentId)
	cr.Status.AtProvider.CommitId = helpers.StringPtr(commitId)

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil // noop
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	return nil // noop
}
