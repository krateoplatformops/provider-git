package git

import (
	"net/url"
	"strings"
)

type RepoOpts struct {
	// Url: the repository URL.
	Url string

	// ApiUrl: the baseUrl for the REST API provider.
	ApiUrl string

	// ApiToken: the access token for invoking REST API provider.
	ApiToken string

	// Provider: the REST API provider.
	// Actually only 'github' is supported.
	Provider string

	// Path: name of the folder in the git repository
	// to copy from (or to).
	Path string

	// Private: used only for target repository.
	Private bool
}

func (ro *RepoOpts) Host() (string, error) {
	u, err := url.Parse(ro.Url)
	if err != nil {
		return "", err
	}

	return u.Host, nil
}

func (ro *RepoOpts) RepoName() (string, error) {
	u, err := url.Parse(ro.Url)
	if err != nil {
		return "", err
	}

	pt := u.Path
	pt = strings.TrimSuffix(pt, "/")
	s := len(pt) - 1
	for (s >= 0) && (pt[s] != '/') {
		s--
	}
	return pt[s+1:], nil
}

func (ro *RepoOpts) OrgName() (string, error) {
	u, err := url.Parse(ro.Url)
	if err != nil {
		return "", err
	}

	pt := u.Path
	pt = strings.TrimPrefix(pt, "/")
	e := 0
	for (e < len(pt)-1) && (pt[e] != '/') {
		e++
	}

	return pt[:e], nil
}
