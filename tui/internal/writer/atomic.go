package writer

import (
	"io"
	"os"
	"path/filepath"
)

const defaultDirMode os.FileMode = 0o755

// writeAtomic writes through a temp file in the same directory and renames it over dst,
// so a reader never sees a partial file and a link at dst is replaced, not followed.
func writeAtomic(dst string, mode os.FileMode, write func(io.Writer) error) error {
	tmp, err := createTemp(dst)
	if err != nil {
		return err
	}
	if err := writeTemp(tmp, mode, write); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// createTemp makes a temp file next to dst, creating the directory first.
func createTemp(dst string) (*os.File, error) {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, defaultDirMode); err != nil {
		return nil, err
	}
	return os.CreateTemp(dir, "."+filepath.Base(dst)+".tmp-*")
}

func writeTemp(tmp *os.File, mode os.FileMode, write func(io.Writer) error) error {
	err := write(tmp)
	if err == nil {
		err = tmp.Chmod(mode)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	return err
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	return writeAtomic(dst, mode, func(w io.Writer) error {
		_, err := io.Copy(w, in)
		return err
	})
}

// replaceWithLink swaps dst for a link to target atomically.
func replaceWithLink(target, dst string) error {
	tmp, err := createTemp(dst)
	if err != nil {
		return err
	}
	tmp.Close()
	if err := os.Remove(tmp.Name()); err != nil {
		return err
	}
	if err := os.Symlink(target, tmp.Name()); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}
