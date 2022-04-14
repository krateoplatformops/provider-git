package repo

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/krateoplatformops/provider-git/pkg/clients/git"
	"github.com/krateoplatformops/provider-git/pkg/clients/github"
)

// Exists check if the specified repository exists.
func Exists(cfg *ProviderOpts, opts *git.RepoOpts) (bool, error) {
	host := opts.Provider
	if len(host) == 0 {
		var err error
		host, err = opts.Host()
		if err != nil {
			return false, err
		}
	}

	var fn existsFunc
	switch h := host; {
	case strings.Contains(h, "github"):
		fn = existsOnGitHub(cfg.HttpClient, cfg.Token)
	default:
		return false, fmt.Errorf("provider: %s not implemented yet", host)
	}

	return fn(opts)
}

// existsOnGitHub check if a repository exists using GitHub ReST API.
func existsOnGitHub(httpClient *http.Client, token string) existsFunc {
	return func(opts *git.RepoOpts) (bool, error) {
		return github.NewClient(token,
			github.HttpClient(httpClient),
			github.ApiUrl(opts.ApiUrl)).
			Repos().
			Exists(opts)
	}
}
