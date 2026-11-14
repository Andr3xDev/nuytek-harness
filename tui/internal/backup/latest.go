package backup

import (
	"io/fs"
	"time"
)

// FS is the only filesystem access the backup package needs.
type FS interface {
	Lstat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
}

// Dir is a directory of timestamped backups; Label names it in the dashboard.
type Dir struct {
	Path, Label string
	Kind        StoreKind
	Base        string // mirror root of the snapshot paths
}

type Info struct {
	Time  time.Time
	Label string
}

// Latest returns the newest backup across dirs, or nil when there is none.
func Latest(fsys FS, dirs []Dir) *Info {
	entries := List(fsys, dirs)
	if len(entries) == 0 {
		return nil
	}
	return &Info{Time: entries[0].Time, Label: entries[0].Label}
}
