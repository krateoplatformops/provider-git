package repo

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/krateoplatformops/provider-git/pkg/clients/git"
	"github.com/krateoplatformops/provider-git/pkg/clients/github"
)

type ProviderOpts struct {
	HttpClient *http.Client
	Token      string
	Debug      bool
}

type existsFunc func(o *git.RepoOpts) (bool, error)

type createFunc func(o *git.RepoOpts) error

// Create the specified repository if does not exists.
func Create(cfg *ProviderOpts, opts *git.RepoOpts) error {
	host := opts.Provider
	if len(host) == 0 {
		var err error
		host, err = opts.Host()
		if err != nil {
			return err
		}
	}

	var fn createFunc
	switch h := host; {
	case strings.Contains(h, "github"):
		fn = createOnGitHub(cfg.HttpClient, cfg.Token)
	default:
		return fmt.Errorf("provider: %s not implemented yet", host)
	}

	return fn(opts)
}

// createOnGitHub creates a repository using GitHub ReST API.
func createOnGitHub(httpClient *http.Client, token string) createFunc {
	return func(opts *git.RepoOpts) error {
		return github.NewClient(token,
			github.HttpClient(httpClient),
			github.ApiUrl(opts.ApiUrl)).
			Repos().
			Create(opts)
	}
}
