package repo

import (
	"context"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/krateoplatformops/provider-git/apis/repo/v1alpha1"
	"github.com/krateoplatformops/provider-git/pkg/clients/git"
	"github.com/krateoplatformops/provider-git/pkg/helpers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// newRepoOpts fills the git.RepoOpts
func newRepoOpts(ctx context.Context, kc client.Client, in *v1alpha1.RepoOpts) (*git.RepoOpts, error) {
	token, err := getGitToken(ctx, kc, in)
	if err != nil {
		return nil, err
	}

	return &git.RepoOpts{
		Url:      in.Url,
		ApiUrl:   helpers.StringValue(in.ApiUrl),
		ApiToken: token,
		Provider: helpers.StringValue(in.Provider),
		Path:     helpers.StringValue(in.Path),
		Private:  helpers.BoolValue(in.Private),
	}, nil
}

// getGitToken lookup for Git API Token.
func getGitToken(ctx context.Context, k client.Client, in *v1alpha1.RepoOpts) (string, error) {
	if s := in.ApiCredentials.Source; s != xpv1.CredentialsSourceSecret {
		return "", fmt.Errorf("credentials source %s is not currently supported", s)
	}

	csr := in.ApiCredentials.SecretRef
	if csr == nil {
		return "", fmt.Errorf("no credentials secret referenced")
	}

	return helpers.GetSecret(ctx, k, csr.DeepCopy())
}
