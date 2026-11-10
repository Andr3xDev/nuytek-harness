package backup

import (
	"io/fs"
	"time"
)

// FS is the only filesystem access Latest needs.
type FS interface {
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
	var newest *Info
	for _, dir := range dirs {
		entries, err := fsys.ReadDir(dir.Path)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			when, ok := ParseStamp(e.Name())
			if !ok {
				info, err := e.Info()
				if err != nil {
					continue
				}
				when = info.ModTime()
			}
			if newest == nil || when.After(newest.Time) {
				newest = &Info{Time: when, Label: dir.Label}
			}
		}
	}
	return newest
}
