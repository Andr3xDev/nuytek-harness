package writer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

// removeFiles deletes the listed files, then the listed directories that ended up empty.
func removeFiles(r plan.Remove) error {
	for _, p := range r.Files {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	for _, p := range r.Dirs {
		err := os.Remove(p)
		if err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOTEMPTY) {
			return err
		}
	}
	return nil
}

// removeTree refuses a link at Path; RemoveAll never follows links inside the tree.
func removeTree(t plan.RemoveTree) error {
	info, err := os.Lstat(t.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to remove %s: it is a symlink", t.Path)
	}
	return os.RemoveAll(t.Path)
}

// removeMatching deletes regular files and links only; a directory match is left alone.
func removeMatching(r plan.RemoveMatching) error {
	for _, pattern := range r.Globs {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return err
		}
		for _, m := range matches {
			info, err := os.Lstat(m)
			if err != nil {
				return err
			}
			if info.IsDir() {
				continue
			}
			if err := os.Remove(m); err != nil {
				return err
			}
		}
	}
	return nil
}
