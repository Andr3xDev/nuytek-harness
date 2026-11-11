package writer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const newFileMode os.FileMode = 0o600

// edit rewrites the file the path resolves to, so a symlinked config keeps its link.
// It fails instead of overwriting when another process changed the file after it was read.
func edit(e plan.Edit) error {
	path := e.Path
	resolved, err := filepath.EvalSymlinks(path)
	switch {
	case err == nil:
		path = resolved
	case errors.Is(err, os.ErrNotExist):
		if info, lerr := os.Lstat(e.Path); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a broken symlink, fix or remove it first", e.Path)
		}
	default:
		return fmt.Errorf("resolve %s: %w", e.Path, err)
	}
	before, err := os.Stat(path)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var old []byte
	mode := newFileMode
	if exists {
		if old, err = os.ReadFile(path); err != nil {
			return err
		}
		mode = before.Mode().Perm()
	}
	out, err := e.Edit(old, exists)
	if err != nil {
		return fmt.Errorf("edit %s: %w", e.Path, err)
	}
	if exists && bytes.Equal(old, out) {
		return nil
	}
	return writeAtomic(path, mode, func(w io.Writer) error {
		if err := requireUnchanged(path, before); err != nil {
			return err
		}
		_, err := w.Write(out)
		return err
	})
}

// requireUnchanged compares the file with the state recorded before it was read; before is nil for a file that was absent.
func requireUnchanged(path string, before fs.FileInfo) error {
	now, err := os.Stat(path)
	switch {
	case before == nil && errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return err
	case before == nil || err != nil || now.Size() != before.Size() || !now.ModTime().Equal(before.ModTime()):
		return fmt.Errorf("%s changed while editing, retry", path)
	}
	return nil
}
