package backup

import "io/fs"

// Size sums the sizes of the regular files in the snapshot; links and directories count nothing.
func Size(fsys FS, e Entry) int64 {
	var total int64
	if e.IsLink {
		return total
	}
	walk(fsys, e.Path, "", func(_ string, info fs.FileInfo) {
		if info.Mode().IsRegular() {
			total += info.Size()
		}
	})
	return total
}
