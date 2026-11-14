package backup

import (
	"io/fs"
	"path/filepath"
)

// walk visits every entry below root, not root itself; links are not followed.
func walk(fsys FS, root, rel string, visit func(rel string, info fs.FileInfo)) {
	entries, err := fsys.ReadDir(filepath.Join(root, rel))
	if err != nil {
		return
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		entryRel := filepath.Join(rel, e.Name())
		visit(entryRel, info)
		if e.IsDir() {
			walk(fsys, root, entryRel, visit)
		}
	}
}
