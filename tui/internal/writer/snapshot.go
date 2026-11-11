package writer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const (
	repoDirMode  os.FileMode = 0o700
	repoFileMask os.FileMode = 0o077

	homeDirOwnerRWX os.FileMode = 0o700
)

// snapshot mirrors the touched paths into a new stamp directory and returns the stamp.
// Everything that can fail before the first write is checked first, so a failure leaves no stamp.
func (w *Writer) snapshot(s plan.Snapshot) plan.StepResult {
	paths, err := snapshotPaths(s)
	if err != nil {
		return fromError(err)
	}
	if len(paths) == 0 {
		return plan.StepResult{Status: plan.StepDone}
	}
	dirMode := defaultDirMode
	if s.Store.Kind == backup.StoreRepo {
		dirMode = repoDirMode
	}
	if err := os.MkdirAll(s.Store.Path, dirMode); err != nil {
		return fromError(fmt.Errorf("create backup store %s: %w", s.Store.Path, err))
	}
	stamp := backup.NewStamp(s.Store.Kind, w.now(), func(name string) bool {
		_, err := os.Lstat(filepath.Join(s.Store.Path, name))
		return err == nil
	})
	stampDir := filepath.Join(s.Store.Path, stamp)
	if err := os.Mkdir(stampDir, dirMode); err != nil {
		return fromError(fmt.Errorf("create backup directory %s: %w", stampDir, err))
	}
	m := mirror{base: s.Store.Base, stampDir: stampDir, repo: s.Store.Kind == backup.StoreRepo, dirMode: dirMode}
	for _, p := range paths {
		if err := m.copyTree(p); err != nil {
			err = fmt.Errorf("snapshot %s: %w", p, err)
			// The stamp dir was created by this run; a partial one must not list as a valid backup.
			if rmErr := os.RemoveAll(stampDir); rmErr != nil {
				err = errors.Join(err, fmt.Errorf("remove partial backup %s: %w", stampDir, rmErr))
			}
			return fromError(err)
		}
	}
	return plan.StepResult{Status: plan.StepDone, Detail: stamp}
}

// snapshotPaths expands globs and rejects any path outside Base; absent paths are dropped.
func snapshotPaths(s plan.Snapshot) ([]string, error) {
	candidates := append([]string(nil), s.Paths...)
	for _, pattern := range s.Globs {
		if err := checkInside(s.Store.Base, pattern); err != nil {
			return nil, err
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, matches...)
	}
	var existing []string
	for _, p := range candidates {
		if err := checkInside(s.Store.Base, p); err != nil {
			return nil, err
		}
		if _, err := os.Lstat(p); err == nil {
			existing = append(existing, p)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	return existing, nil
}

func checkInside(base, p string) error {
	rel, err := filepath.Rel(base, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s is outside the backup base %s", p, base)
	}
	return nil
}

type mirror struct {
	base, stampDir string
	repo           bool
	dirMode        os.FileMode
}

// copyTree copies one path and everything below it; an entry already present keeps its first copy.
func (m mirror) copyTree(root string) error {
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(m.base, p)
		dst := filepath.Join(m.stampDir, rel)
		if d.IsDir() {
			mode := m.dirMode
			if !m.repo {
				info, err := d.Info()
				if err != nil {
					return err
				}
				// Keep the source mode, but never lose owner write or the writer cannot fill the directory.
				mode = info.Mode().Perm() | homeDirOwnerRWX
			}
			return os.MkdirAll(dst, mode)
		}
		if _, err := os.Lstat(dst); err == nil {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dst), m.dirMode); err != nil {
			return err
		}
		return m.copyEntry(p, dst, d)
	})
}

func (m mirror) copyEntry(src, dst string, d fs.DirEntry) error {
	switch {
	case d.Type()&fs.ModeSymlink != 0:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case d.Type().IsRegular():
		info, err := d.Info()
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if m.repo {
			mode &^= repoFileMask
		}
		return copyFile(src, dst, mode)
	}
	return nil
}
