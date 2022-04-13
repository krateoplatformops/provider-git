package repo

import (
	"context"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/krateoplatformops/provider-git/apis/v1alpha1"
	"github.com/krateoplatformops/provider-git/pkg/helpers"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetGitToken lookup for Git PAT.
func GetGitToken(ctx context.Context, k client.Client, pc *v1alpha1.ProviderConfig) (string, error) {
	if s := pc.Spec.Credentials.Source; s != xpv1.CredentialsSourceSecret {
		return "", errors.Errorf("credentials source %s is not currently supported", s)
	}

	csr := pc.Spec.Credentials.SecretRef
	if csr == nil {
		return "", errors.New("no credentials secret referenced")
	}

	return helpers.GetSecret(ctx, k, csr.DeepCopy())
}
