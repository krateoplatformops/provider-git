package repo

import (
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
	return copyDir(cfg.FromRepo.FS(), cfg.ToRepo.FS(), cfg.FromPath, cfg.ToPath)
}

func copyFile(fromFS, toFS billy.Filesystem, src, dst string) (err error) {
	in, err := fromFS.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := toFS.Create(dst)
	if err != nil {
		return
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
func copyDir(fromFS, toFS billy.Filesystem, src, dst string) (err error) {
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
			err = copyDir(fromFS, toFS, srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = copyFile(fromFS, toFS, srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}

	return
}
