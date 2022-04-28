package repo

import (
	"context"
	"errors"

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
	"github.com/krateoplatformops/provider-git/pkg/clients/git"
	"github.com/krateoplatformops/provider-git/pkg/clients/repo"
)

const (
	errNotRepo = "managed resource is not a repo custom resource"
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
			recorder: mgr.GetEventRecorderFor("Repo"),
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

	return &external{kube: c.kube, log: c.log, cfg: cfg, rec: c.recorder}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube client.Client
	log  logging.Logger
	cfg  *repo.ProviderOpts
	rec  record.EventRecorder
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*repov1alpha1.Repo)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepo)
	}

	spec := cr.Spec.ForProvider.DeepCopy()

	toRepo, err := newRepoOpts(ctx, e.kube, &spec.ToRepo)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	ok, err = repo.Exists(e.cfg, toRepo)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if ok {
		e.log.Debug("Target repo already exists", "url", toRepo.Url)
		e.rec.Event(cr, corev1.EventTypeNormal, "AlredyExists", "Target repo already exists")

		cr.SetConditions(xpv1.Available())
		return managed.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: true,
		}, nil
	}

	e.log.Debug("Target repo does not exists", "url", toRepo.Url)

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

	cr.SetConditions(xpv1.Creating())

	spec := cr.Spec.ForProvider.DeepCopy()

	fromRepoOpts, err := newRepoOpts(ctx, e.kube, &spec.FromRepo)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	toRepoOpts, err := newRepoOpts(ctx, e.kube, &spec.ToRepo)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	err = repo.Create(e.cfg, toRepoOpts)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo created", "url", toRepoOpts.Url)
	e.rec.Event(cr, corev1.EventTypeNormal, "RepoCreated", "Target repo created")

	toRepo, err := git.Clone(toRepoOpts.Url, git.RepoCreds{Password: toRepoOpts.ApiToken})
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo cloned", "url", toRepoOpts.Url)
	e.rec.Event(cr, corev1.EventTypeNormal, "RepoCloned", "Target repo cloned")

	fromRepo, err := git.Clone(fromRepoOpts.Url, git.RepoCreds{Password: fromRepoOpts.ApiToken})
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Origin repo cloned", "url", fromRepoOpts.Url)
	e.rec.Event(cr, corev1.EventTypeNormal, "RepoCloned", "Origin repo cloned")

	err = toRepo.Branch("main")
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo on branch main")

	err = repo.Copy(repo.CopyOpts{
		FromRepo: fromRepo,
		ToRepo:   toRepo,
		FromPath: fromRepoOpts.Path,
		ToPath:   toRepoOpts.Path,
	})
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Origin and target repo synchronized",
		"fromUrl", fromRepoOpts.Url,
		"toUrl", toRepoOpts.Url,
		"fromPath", fromRepoOpts.Path,
		"toPath", toRepoOpts.Path)
	e.rec.Event(cr, corev1.EventTypeNormal, "RepoSync", "Origin and target repo synchronized")

	err = toRepo.Commit(".", ":rocket: first commit")
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo committed branch main")
	e.rec.Event(cr, corev1.EventTypeNormal, "RepoCommit", "Target repo committed branch main")

	err = toRepo.Push("origin", "main")
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	e.log.Debug("Target repo pushed branch main")
	e.rec.Event(cr, corev1.EventTypeNormal, "RepoPush", "Target repo pushed branch main")

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil // noop
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	return nil // noop
}
