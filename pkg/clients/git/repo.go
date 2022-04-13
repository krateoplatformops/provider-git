package git

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/storage/memory"
)

const (
	commitAuthorEmail = "krateoctl@krateoplatformops.io"
	commitAuthorName  = "krateoctl"
)

var (
	ErrRepositoryNotFound     = errors.New("repository not found")
	ErrEmptyRemoteRepository  = errors.New("remote repository is empty")
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrAuthorizationFailed    = errors.New("authorization failed")
)

// Repo is an in-memory git repository
type Repo struct {
	rawURL string
	creds  RepoCreds

	storer *memory.Storage
	fs     billy.Filesystem
	repo   *git.Repository
}

func Clone(repoUrl string, creds RepoCreds) (*Repo, error) {
	res := &Repo{
		rawURL: repoUrl,
		creds:  creds,
		storer: memory.NewStorage(),
		fs:     memfs.New(),
	}

	// Clone the given repository to the given directory
	var err error
	res.repo, err = git.Clone(res.storer, res.fs, &git.CloneOptions{
		RemoteName: "origin",
		URL:        repoUrl,
		//Depth: 1,
		Auth: creds.Credentials(),
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

func (s *Repo) FS() billy.Filesystem {
	return s.fs
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
			Name:  commitAuthorName,
			Email: commitAuthorEmail,
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
			Auth:       s.creds.Credentials(),
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
		Auth:       s.creds.Credentials(),
		RefSpecs: []config.RefSpec{
			config.RefSpec(refName + ":" + refName),
		},
	})
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
		Auth: s.creds.Credentials(),
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
