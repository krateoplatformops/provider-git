package git

import "github.com/go-git/go-git/v5/plumbing/transport/http"

type RepoCreds struct {
	Username string
	Password string
}

/*
func (rc *RepoCreds) Credentials() *http.BasicAuth {
	if len(rc.Password) == 0 {
		return nil
	}
	usr := rc.Username
	if len(usr) == 0 {
		usr = "abc123" // yes, this can be anything except an empty string
	}

	return &http.BasicAuth{
		Username: usr,
		Password: rc.Password,
	}
}
*/

func (rc *RepoCreds) Credentials() *http.TokenAuth {
	if len(rc.Password) == 0 {
		return nil
	}
	return &http.TokenAuth{
		Token: rc.Password,
	}
}
