package gitrepo

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/krateoplatformops/provider-git/pkg/clients/gitrepo/github"
)

// CreateOptions contains basic info to create a repository
// using specified rest API.
type CreateOptions struct {
	Client  *http.Client
	Token   string
	Private bool
}

// CreateEventually creates a repository (if does not exists).
// Supported Git providers: GitHub.
func CreateEventually(rawURL string, opts *CreateOptions) error {
	ri, err := GetRepoInfo(rawURL)
	if err != nil {
		return err
	}

	var fn createFunc
	switch host := ri.Host(); {
	case strings.HasPrefix(host, "github"):
		fn = createOnGitHub(opts)
	default:
		return fmt.Errorf("Host: %s not implemented yet", host)
	}

	return fn(ri)
}

type createFunc func(ri Info) error

// createOnGitHub creates a repository if does not exists
// using GitHub ReST API.
func createOnGitHub(opts *CreateOptions) createFunc {
	return func(ri Info) error {
		ghc := github.NewClient(opts.Client, opts.Token)

		ok, err := ghc.Repos.Exists(ri.Owner(), ri.RepoName())
		if err != nil {
			return err
		}

		if !ok {
			return ghc.Repos.Create(ri.Owner(), ri.RepoName(), github.CreateOptions{
				Private: opts.Private,
			})
		}

		return nil
	}
}
