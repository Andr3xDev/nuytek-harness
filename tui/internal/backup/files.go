package backup

import "io/fs"

// Files lists the relative paths the snapshot holds.
func Files(fsys FS, e Entry) []string {
	if e.IsLink {
		return nil
	}
	var files []string
	walk(fsys, e.Path, "", func(rel string, info fs.FileInfo) {
		if !info.IsDir() {
			files = append(files, rel)
		}
	})
	return files
}
