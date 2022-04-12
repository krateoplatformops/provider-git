package github

import (
	"fmt"
	"net/http"
)

const (
	apiURL = "https://api.github.com/"
)

// GithubError represents a Github API error response
// https://developer.github.com/v3/#client-errors
type GithubError struct {
	Message string `json:"message"`
	Errors  []struct {
		Resource string `json:"resource"`
		Field    string `json:"field"`
		Code     string `json:"code"`
	} `json:"errors,omitempty"`
	DocumentationURL string `json:"documentation_url"`
}

func (e GithubError) Error() string {
	return fmt.Sprintf("github: %v %+v %v", e.Message, e.Errors, e.DocumentationURL)
}

// Client is a tiny Github client
type GithubClient struct {
	Token string
	Repos *RepoService
}

// NewClient returns a new Github Client
func NewClient(httpClient *http.Client, token string) *GithubClient {
	return &GithubClient{
		Repos: NewRepoService(httpClient, token),
	}
}
