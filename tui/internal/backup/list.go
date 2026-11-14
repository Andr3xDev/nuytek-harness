package backup

import (
	"io/fs"
	"path/filepath"
	"slices"
	"time"
)

// Entry is one snapshot found by List; Path is the snapshot directory.
type Entry struct {
	Stamp  string
	Time   time.Time
	Label  string
	Kind   StoreKind
	Base   string
	Path   string
	IsLink bool
}

// List returns the snapshots of every store, newest first.
func List(fsys FS, dirs []Dir) []Entry {
	var entries []Entry
	for _, dir := range dirs {
		entries = append(entries, listStore(fsys, dir)...)
	}
	slices.SortStableFunc(entries, func(a, b Entry) int { return b.Time.Compare(a.Time) })
	return entries
}

// listStore skips a symlinked root; a symlinked entry is listed with IsLink.
func listStore(fsys FS, dir Dir) []Entry {
	root, err := fsys.Lstat(dir.Path)
	if err != nil || root.Mode()&fs.ModeSymlink != 0 {
		return nil
	}
	items, err := fsys.ReadDir(dir.Path)
	if err != nil {
		return nil
	}
	var entries []Entry
	for _, item := range items {
		isLink := item.Type()&fs.ModeSymlink != 0
		if !item.IsDir() && !isLink {
			continue
		}
		when, ok := ParseStamp(item.Name())
		if !ok {
			info, err := item.Info()
			if err != nil {
				continue
			}
			when = info.ModTime()
		}
		entries = append(entries, Entry{
			Stamp: item.Name(), Time: when, Label: dir.Label, Kind: dir.Kind, Base: dir.Base,
			Path: filepath.Join(dir.Path, item.Name()), IsLink: isLink,
		})
	}
	return entries
}
