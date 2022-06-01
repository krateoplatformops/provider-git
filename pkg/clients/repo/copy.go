package repo

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/krateoplatformops/provider-git/pkg/clients/git"
	gi "github.com/sabhiram/go-gitignore"
)

type CopyOpts struct {
	FromRepo   *git.Repo
	ToRepo     *git.Repo
	RenderFunc func(in io.Reader, out io.Writer) error
	Ignore     *gi.GitIgnore
}

// Copy files from one in memory filesystem to another in memory filesystem
func Copy(cfg *CopyOpts, fromPath, toPath string) (err error) {
	if len(fromPath) == 0 {
		fromPath = "/"
	}

	if len(toPath) == 0 {
		toPath = "/"
	}

	return cfg.CopyDir(fromPath, toPath)
}

func (cfg *CopyOpts) WriteBytes(src []byte, dstfn string) (err error) {
	out, err := cfg.ToRepo.FS().Create(dstfn)
	if err != nil {
		return err
	}

	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, bytes.NewReader(src))
	return
}

func (cfg *CopyOpts) CopyFile(src, dst string, render bool) (err error) {
	fromFS, toFS := cfg.FromRepo.FS(), cfg.ToRepo.FS()

	in, err := fromFS.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := toFS.Create(dst)
	if err != nil {
		return err
	}

	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	if !render || cfg.RenderFunc == nil {
		_, err = io.Copy(out, in)
		return
	}

	return cfg.RenderFunc(in, out)
}

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must *not* exist.
// Symlinks are ignored and skipped.
func (cfg *CopyOpts) CopyDir(src, dst string) (err error) {
	fromFS, toFS := cfg.FromRepo.FS(), cfg.ToRepo.FS()

	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := fromFS.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	err = toFS.MkdirAll(dst, si.Mode())
	if err != nil {
		return
	}

	entries, err := fromFS.ReadDir(src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = cfg.CopyDir(srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			// ignore file eventually
			var ignore bool
			if cfg.Ignore != nil {
				if cfg.Ignore.MatchesPath(srcPath) {
					ignore = true
				}
			}

			// do the copy
			err = cfg.CopyFile(srcPath, dstPath, ignore)
			if err != nil {
				return
			}
		}
	}

	return
}
