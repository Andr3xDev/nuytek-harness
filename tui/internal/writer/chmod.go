package writer

import (
	"os"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const readBits os.FileMode = 0o444

// addExecute grants x to each class that has r; a file nobody can read is left alone.
func addExecute(c plan.Chmod) error {
	for _, p := range c.Paths {
		info, err := os.Stat(p)
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if err := os.Chmod(p, mode|(mode&readBits)>>2); err != nil {
			return err
		}
	}
	return nil
}
