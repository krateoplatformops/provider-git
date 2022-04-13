package repo

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/krateoplatformops/provider-git/pkg/clients/git"
	"github.com/krateoplatformops/provider-git/pkg/clients/github"
)

func CreateEventually(httpClient *http.Client, token string, opts git.RepoOpts) error {
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
		fn = createOnGitHub(httpClient, token)
	default:
		return fmt.Errorf("provider: %s not implemented yet", host)
	}

	return fn(&opts)
}

type createFunc func(o *git.RepoOpts) error

// createOnGitHub creates a repository if does not exists
// using GitHub ReST API.
func createOnGitHub(httpClient *http.Client, token string) createFunc {
	return func(opts *git.RepoOpts) error {
		ghc := github.NewClient(token, github.HttpClient(httpClient), github.ApiUrl(opts.ApiUrl))

		ok, err := ghc.Repos().Exists(opts)
		if err != nil {
			return err
		}

		if !ok {
			return ghc.Repos().Create(opts)
		}

		return nil
	}
}
