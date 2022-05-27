package repo

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/krateoplatformops/provider-git/pkg/clients/git"
)

type CopyOpts struct {
	FromRepo *git.Repo
	ToRepo   *git.Repo
	FromPath string
	ToPath   string
}

// Copy files from one in memory filesystem to another in memory filesystem
func Copy(cfg CopyOpts) (err error) {
	fromPath := cfg.FromPath
	if len(fromPath) == 0 {
		fromPath = "/"
	}

	toPath := cfg.ToPath
	if len(toPath) == 0 {
		toPath = "/"
	}

	return CopyDir(cfg.FromRepo.FS(), cfg.ToRepo.FS(), fromPath, toPath)
}

func CopyBytes(toFS billy.Filesystem, src []byte, dstfn string) (err error) {
	out, err := toFS.Create(dstfn)
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

func CopyFile(fromFS, toFS billy.Filesystem, src, dst string) (err error) {
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

	_, err = io.Copy(out, in)
	return
}

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must *not* exist.
// Symlinks are ignored and skipped.
func CopyDir(fromFS, toFS billy.Filesystem, src, dst string) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := fromFS.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = toFS.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	/*
		if err == nil {
			//err = toFS.Remove(dst)
			//if err != nil {
			//	return
			//}
			err = toFS.MkdirAll(dst, si.Mode())
			if err != nil {
				return
			}
			//return fmt.Errorf("destination already exists")
		}*/

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
			err = CopyDir(fromFS, toFS, srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = CopyFile(fromFS, toFS, srcPath, dstPath)
			if err != nil {
				return
			}

			//fmt.Fprintf(os.Stderr, " copied: %s\n", entry)
		}
	}

	return
}
