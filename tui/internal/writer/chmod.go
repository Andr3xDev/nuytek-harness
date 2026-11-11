package writer

import (
	"os"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const readBits os.FileMode = 0o444

// addExecute grants x to each class that has r; a file nobody can read and a link are left alone.
func addExecute(c plan.Chmod) error {
	for _, p := range c.Paths {
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		mode := info.Mode().Perm()
		if err := os.Chmod(p, mode|(mode&readBits)>>2); err != nil {
			return err
		}
	}
	return nil
}
