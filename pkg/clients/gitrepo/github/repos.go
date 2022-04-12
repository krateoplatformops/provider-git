package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/krateoplatformops/provider-git/pkg/clients/req"
)

// RepoService provides methods for creating and reading repositories.
type RepoService struct {
	client *http.Client
	token  string
}

// NewRepoService returns a new RepoService.
func NewRepoService(httpClient *http.Client, token string) *RepoService {
	return &RepoService{
		client: httpClient,
		token:  token,
	}
}

type CreateOptions struct {
	Private bool
}

func (s *RepoService) Create(owner, name string, opts CreateOptions) error {
	ok, err := s.isOrg(owner)
	if err != nil {
		return err
	}

	pt := "/user/repos"
	if ok {
		pt = fmt.Sprintf("orgs/%s/repos", owner)
	}

	githubError := &GithubError{}

	err = req.Post().Url(apiURL).Path(pt).
		Client(s.client).
		Header("Authorization", fmt.Sprintf("token %s", s.token)).
		BodyJSON(map[string]interface{}{
			"name":      name,
			"private":   opts.Private,
			"auto_init": true,
		}).
		AddValidator(req.ErrorJSON(githubError, 201)).
		Do(context.Background())
	if err != nil {
		var gerr *GithubError
		if errors.As(err, &gerr) {
			return fmt.Errorf(gerr.Error())
		}
		return err
	}

	return nil
}

// Get fetches a repository.
//
// GitHub API docs: https://docs.github.com/en/free-pro-team@latest/rest/reference/repos/#get-a-repository
func (s *RepoService) Exists(owner, name string) (bool, error) {
	pt := fmt.Sprintf("repos/%s/%s", owner, name)

	err := req.Get().Url(apiURL).Path(pt).
		Client(s.client).
		Header("Authorization", fmt.Sprintf("token %s", s.token)).
		CheckStatus(200).
		Do(context.Background())
	if err != nil {
		if req.HasStatusErr(err, 404) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (s *RepoService) isOrg(owner string) (bool, error) {
	err := req.Get().Url(apiURL).Pathf("/orgs/%s", owner).
		Header("Authorization", fmt.Sprintf("token %s", s.token)).
		Client(s.client).
		CheckStatus(200).
		Do(context.Background())
	if err != nil {
		if req.HasStatusErr(err, 404) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
