package drift

import (
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

const scriptSuffix = ".sh"

// Scripts lists the top-level .sh names of src after filter: the files the dirx contract makes executable.
func Scripts(fsys check.FS, src string, filter Filter) ([]string, error) {
	entries, err := sourceEntries(fsys, src, filter)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), scriptSuffix) {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// LacksExec reports whether path is missing or has no owner execute bit.
func LacksExec(fsys check.FS, path string) bool {
	info, err := fsys.Lstat(path)
	return err != nil || info.Mode()&check.OwnerExecBit == 0
}
