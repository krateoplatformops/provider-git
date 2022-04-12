package gitrepo

import (
	"net/url"
	"strings"
)

type Info struct {
	host     string
	owner    string
	repoName string
}

func (ri *Info) Owner() string {
	return ri.owner
}

func (ri *Info) RepoName() string {
	return ri.repoName
}

func (ri *Info) Host() string {
	return ri.host
}

func GetRepoInfo(rawURL string) (Info, error) {
	res := Info{}

	u, err := url.Parse(rawURL)
	if err != nil {
		return res, err
	}

	res.host = u.Host
	res.owner = parseRepoOwner(u.Path)
	res.repoName = parseRepoName(u.Path)

	return res, nil
}

func parseRepoName(pt string) string {
	pt = strings.TrimSuffix(pt, "/")
	s := len(pt) - 1
	for (s >= 0) && (pt[s] != '/') {
		s--
	}
	return pt[s+1:]
}

func parseRepoOwner(pt string) string {
	pt = strings.TrimPrefix(pt, "/")
	e := 0
	for (e < len(pt)-1) && (pt[e] != '/') {
		e++
	}

	return pt[:e]
}
