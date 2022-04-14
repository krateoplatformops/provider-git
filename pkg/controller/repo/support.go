package repo

import (
	"github.com/krateoplatformops/provider-git/apis/repo/v1alpha1"
	"github.com/krateoplatformops/provider-git/pkg/clients/git"
	"github.com/krateoplatformops/provider-git/pkg/helpers"
)

// newRepoOpts fills the git.RepoOpts
func newRepoOpts(in *v1alpha1.RepoOpts) *git.RepoOpts {
	return &git.RepoOpts{
		Url:      in.Url,
		ApiUrl:   helpers.StringValue(in.ApiUrl),
		Provider: helpers.StringValue(in.Provider),
		Path:     helpers.StringValue(in.Path),
		Private:  helpers.BoolValue(in.Private),
	}
}
