package clients

import (
	"context"
	"fmt"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/krateoplatformops/provider-git/apis/v1alpha1"
	"github.com/krateoplatformops/provider-git/pkg/helpers"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	Insecure             bool
	DeploymentServiceUrl string
	FromRepoCreds        transport.AuthMethod
	ToRepoCreds          transport.AuthMethod
}

// GetConfig constructs a RepoCreds pair that can be used to authenticate to the git provider.
func GetConfig(ctx context.Context, c client.Client, mg resource.Managed) (*Config, error) {
	switch {
	case mg.GetProviderConfigReference() != nil:
		return useProviderConfig(ctx, c, mg)
	default:
		return nil, errors.New("providerConfigRef is not given")
	}
}

// useProviderConfig to produce a config that can be used to copy a repo content.
func useProviderConfig(ctx context.Context, k client.Client, mg resource.Managed) (*Config, error) {
	pc := &v1alpha1.ProviderConfig{}
	err := k.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get referenced Provider")
	}

	t := resource.NewProviderConfigUsageTracker(k, &v1alpha1.ProviderConfigUsage{})
	err = t.Track(ctx, mg)
	if err != nil {
		return nil, errors.Wrap(err, "cannot track ProviderConfig usage")
	}

	if len(pc.Spec.DeploymentServiceUrl) == 0 {
		return nil, errors.Wrapf(err, "deplyment service url must be specified")
	}

	ret := &Config{
		Insecure:             helpers.BoolValue(pc.Spec.Insecure),
		DeploymentServiceUrl: pc.Spec.DeploymentServiceUrl,
	}

	ret.FromRepoCreds, err = getFromRepoCredentials(ctx, k, pc)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving from repo credentials")
	}

	ret.ToRepoCreds, err = getToRepoCredentials(ctx, k, pc)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving to repo credentials")
	}

	return ret, nil
}

// getFromRepoCredentials returns the from repo credentials stored in a secret.
func getFromRepoCredentials(ctx context.Context, k client.Client, pc *v1alpha1.ProviderConfig) (transport.AuthMethod, error) {
	if pc.Spec.FromRepoCredentials == nil {
		return nil, nil
	}

	if s := pc.Spec.FromRepoCredentials.Source; s != xpv1.CredentialsSourceSecret {
		return nil, fmt.Errorf("credentials source %s is not currently supported", s)
	}

	csr := pc.Spec.FromRepoCredentials.SecretRef
	if csr == nil {
		return nil, fmt.Errorf("no credentials secret referenced")
	}

	authMethod := helpers.StringValue(pc.Spec.FromRepoCredentials.AuthMethod)
	token, err := helpers.GetSecret(ctx, k, csr.DeepCopy())
	if err != nil {
		return nil, err
	}

	fmt.Printf("authMethod: %s\n", authMethod)
	if strings.EqualFold(authMethod, "bearer") {
		return &http.TokenAuth{
			Token: token,
		}, nil
	}

	return &http.BasicAuth{
		Username: "krateoctl",
		Password: token,
	}, nil
}

// getToRepoCredentials returns the to repo credentials stored in a secret.
func getToRepoCredentials(ctx context.Context, k client.Client, pc *v1alpha1.ProviderConfig) (transport.AuthMethod, error) {
	if pc.Spec.ToRepoCredentials == nil {
		return nil, nil
	}

	if s := pc.Spec.ToRepoCredentials.Source; s != xpv1.CredentialsSourceSecret {
		return nil, fmt.Errorf("credentials source %s is not currently supported", s)
	}

	csr := pc.Spec.ToRepoCredentials.SecretRef
	if csr == nil {
		return nil, fmt.Errorf("no credentials secret referenced")
	}

	authMethod := helpers.StringValue(pc.Spec.ToRepoCredentials.AuthMethod)
	token, err := helpers.GetSecret(ctx, k, csr.DeepCopy())
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(authMethod, "bearer") {
		return &http.TokenAuth{
			Token: token,
		}, nil
	}

	return &http.BasicAuth{
		Username: "abc123",
		Password: token,
	}, nil
}
