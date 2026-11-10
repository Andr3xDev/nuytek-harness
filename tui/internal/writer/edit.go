package writer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const newFileMode os.FileMode = 0o600

// edit rewrites the file the path resolves to, so a symlinked config keeps its link.
func edit(e plan.Edit) error {
	path := e.Path
	resolved, err := filepath.EvalSymlinks(path)
	switch {
	case err == nil:
		path = resolved
	case !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("resolve %s: %w", e.Path, err)
	}
	old, err := os.ReadFile(path)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	mode := newFileMode
	if exists {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		mode = info.Mode().Perm()
	}
	out, err := e.Edit(old, exists)
	if err != nil {
		return fmt.Errorf("edit %s: %w", e.Path, err)
	}
	if exists && bytes.Equal(old, out) {
		return nil
	}
	return writeAtomic(path, mode, func(w io.Writer) error {
		_, err := w.Write(out)
		return err
	})
}
