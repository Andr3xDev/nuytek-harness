package drift

import (
	"path"
	"path/filepath"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

// OwnedFiles lists every file, link and directory of the filtered source.
func OwnedFiles(fsys check.FS, src string, filter Filter) ([]string, error) {
	var owned []string
	var walk func(dir, rel string) error
	walk = func(dir, rel string) error {
		entries, err := sourceEntries(fsys, dir, filter)
		if err != nil {
			return err
		}
		for _, e := range entries {
			entryRel := path.Join(rel, e.Name())
			owned = append(owned, entryRel)
			if e.IsDir() {
				if err := walk(filepath.Join(dir, e.Name()), entryRel); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return owned, walk(src, "")
}

// Extras lists entries under dst absent from OwnedFiles; links are not descended.
func Extras(fsys check.FS, src, dst string, filter Filter) ([]string, error) {
	ownedFiles, err := OwnedFiles(fsys, src, filter)
	if err != nil {
		return nil, err
	}
	owned := make(map[string]bool, len(ownedFiles))
	for _, rel := range ownedFiles {
		owned[rel] = true
	}
	var extras []string
	var walk func(dir, rel string) error
	walk = func(dir, rel string) error {
		entries, err := fsys.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			entryRel := path.Join(rel, e.Name())
			if !owned[entryRel] {
				extras = append(extras, entryRel)
			}
			if e.IsDir() {
				if err := walk(filepath.Join(dir, e.Name()), entryRel); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return extras, walk(dst, "")
}
