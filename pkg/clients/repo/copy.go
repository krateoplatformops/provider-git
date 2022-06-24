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

func (cfg *CopyOpts) CopyFile(src, dst string, doNotRender bool) (err error) {
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

	if doNotRender || cfg.RenderFunc == nil {
		_, err = io.Copy(out, in)
		return err
	}

	return cfg.RenderFunc(in, out)
}

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must *not* exist.
// Symlinks are ignored and skipped.
func (cfg *CopyOpts) CopyDir(src, dst string) (err error) {
	if len(src) == 0 {
		src = "/"
	}

	if len(dst) == 0 {
		dst = "/"
	}

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
			var doNotRender bool
			if cfg.Ignore != nil {
				if cfg.Ignore.MatchesPath(srcPath) {
					doNotRender = true
				}
			}

			// do the copy
			err = cfg.CopyFile(srcPath, dstPath, doNotRender)
			if err != nil {
				return
			}
		}
	}

	return
}
