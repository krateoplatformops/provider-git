package gitrepo

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

var (
	ErrRepositoryNotFound     = errors.New("repository not found")
	ErrEmptyRemoteRepository  = errors.New("remote repository is empty")
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrAuthorizationFailed    = errors.New("authorization failed")
)

// Repo is an in-memory git repository
type Repo struct {
	rawURL   string
	username string
	password string

	storer *memory.Storage
	fs     billy.Filesystem
	repo   *git.Repository
}

type GitOption func(*Repo)

func GitUser(usr string) GitOption {
	return func(r *Repo) {
		r.username = usr
	}
}

func GitPassword(pwd string) GitOption {
	return func(r *Repo) {
		r.password = pwd
	}
}

func GitToken(tkn string) GitOption {
	return func(r *Repo) {
		r.username = ""
		r.password = tkn
	}
}

func Clone(repoUrl string, opts ...GitOption) (*Repo, error) {
	u, err := url.Parse(repoUrl)
	if err != nil {
		return nil, err
	}

	res := &Repo{
		rawURL: u.String(),
		storer: memory.NewStorage(),
		fs:     memfs.New(),
	}

	for _, o := range opts {
		o(res)
	}

	// Clone the given repository to the given directory
	res.repo, err = git.Clone(res.storer, res.fs, &git.CloneOptions{
		RemoteName: "origin",
		URL:        u.String(),
		//Depth: 1,
		Auth: res.credentials(),
	})
	if err != nil {
		if errors.Is(err, transport.ErrRepositoryNotFound) {
			return nil, ErrRepositoryNotFound
		}

		if errors.Is(err, transport.ErrEmptyRemoteRepository) {
			return nil, ErrEmptyRemoteRepository
		}

		if errors.Is(err, transport.ErrAuthenticationRequired) {
			return nil, ErrAuthenticationRequired
		}

		if errors.Is(err, transport.ErrAuthorizationFailed) {
			return nil, ErrAuthorizationFailed
		}

		return nil, err
		/*
			h := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/main"))
			err := res.repo.Storer.SetReference(h)
			if err != nil {
				return nil, err
			}
		*/
	}

	return res, nil
}

func (s *Repo) Branch(name string) error {
	branch := fmt.Sprintf("refs/heads/%s", name)
	ref := plumbing.ReferenceName(branch)

	h := plumbing.NewSymbolicReference(plumbing.HEAD, ref)
	err := s.repo.Storer.SetReference(h)
	if err != nil {
		return err
	}

	wt, err := s.repo.Worktree()
	if err != nil {
		return err
	}

	return wt.Checkout(&git.CheckoutOptions{
		Create: false,
		Branch: ref,
	})
}

func (s *Repo) Commit(path, msg string) error {
	wt, err := s.repo.Worktree()
	if err != nil {
		return err
	}
	// git add $path
	if _, err := wt.Add(path); err != nil {
		return err
	}

	// git commit -m $message
	_, err = wt.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Krateo",
			Email: "krateo@kiratech.it",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Repo) Push(downstream, branch string) error {
	//Push the code to the remote
	if len(branch) == 0 {
		return s.repo.Push(&git.PushOptions{
			RemoteName: downstream,
			Auth:       s.credentials(),
		})
	}

	headRef, err := s.repo.Head()
	if err != nil {
		return err
	}

	refName := plumbing.NewBranchReferenceName(branch)

	refs, err := s.repo.References()
	if err != nil {
		return err
	}

	var foundLocal bool
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name() == refName {
			//fmt.Printf("reference exists locally:\n%s\n", ref)
			foundLocal = true
		}
		return nil
	})
	if !foundLocal {
		ref := plumbing.NewHashReference(refName, headRef.Hash())
		err = s.repo.Storer.SetReference(ref)
		if err != nil {
			return err
		}
	}

	return s.repo.Push(&git.PushOptions{
		RemoteName: downstream,
		Force:      true,
		Auth:       s.credentials(),
		RefSpecs: []config.RefSpec{
			config.RefSpec(refName + ":" + refName),
		},
	})
}

func (s *Repo) credentials() *http.BasicAuth {
	if len(s.password) == 0 {
		return nil
	}
	usr := s.username
	if len(usr) == 0 {
		usr = "abc123" // yes, this can be anything except an empty string
	}

	return &http.BasicAuth{
		Username: usr,
		Password: s.password,
	}
}

func Pull(s *Repo) error {
	// Get the working directory for the repository
	wt, err := s.repo.Worktree()
	if err != nil {
		return err
	}

	err = wt.Pull(&git.PullOptions{
		RemoteName: "origin",
		//Depth:      1,
		Auth: s.credentials(),
	})

	if err != nil {
		if errors.Is(err, git.NoErrAlreadyUpToDate) {
			err = nil
		}
	}

	return err
}

func getHeadCommit(s *Repo) (*object.Commit, error) {
	// retrieve the branch being pointed by HEAD
	ref, err := s.repo.Head()
	if err != nil {
		return nil, err
	}

	// retrieve the commit object
	return s.repo.CommitObject(ref.Hash())
}
