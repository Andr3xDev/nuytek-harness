package writer

import (
	"os"
	"path/filepath"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

// singleFile in Copy.Files means Src and Dst are the file itself.
const singleFile = "."

// copyFiles overwrites exactly the listed files and deletes nothing.
func copyFiles(c plan.Copy) error {
	for _, rel := range c.Files {
		if err := copyOne(filepath.Join(c.Src, rel), filepath.Join(c.Dst, rel), rel == singleFile); err != nil {
			return err
		}
	}
	return nil
}

func copyOne(src, dst string, followLink bool) error {
	stat := os.Lstat
	if followLink {
		stat = os.Stat
	}
	info, err := stat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return replaceWithLink(target, dst)
	}
	return copyFile(src, dst, info.Mode().Perm())
}
