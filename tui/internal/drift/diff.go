package drift

import (
	"bytes"
	"errors"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

// Blocked decides what Diff does with an entry below the root whose type blocks the copy.
type Blocked int

const (
	BlockedFail Blocked = iota
	BlockedReplace
)

const (
	rootFile     = "."
	noteLink     = "link"
	noteReplaces = "replaces "

	describeDirectory = "directory"
	describeSymlink   = "symlink"
)

// differ accumulates the rows of one Diff run.
type differ struct {
	fsys    check.FS
	filter  Filter
	blocked Blocked
	rows    []plan.Change
}

// Diff walks src after filter and returns Create and Overwrite rows against dst; it never returns Delete rows.
// A src file gives one row with Path ".". A symlink or non-directory dst root is always an ErrPathConflict.
func Diff(fsys check.FS, src, dst string, filter Filter, blocked Blocked) ([]plan.Change, error) {
	srcInfo, err := fsys.Lstat(src)
	if err != nil {
		return nil, err
	}
	dstInfo, err := fsys.Lstat(dst)
	absent := errors.Is(err, fs.ErrNotExist)
	if err != nil && !absent {
		return nil, err
	}
	d := &differ{fsys: fsys, filter: filter, blocked: blocked}
	if srcInfo.IsDir() {
		if !absent && !dstInfo.IsDir() {
			return nil, ErrPathConflict{Path: dst, Found: describe(dstInfo)}
		}
		err = d.dir(src, dst, "", absent)
		return d.rows, err
	}
	if !absent && dstInfo.IsDir() {
		return nil, ErrPathConflict{Path: dst, Found: describe(dstInfo)}
	}
	err = d.file(src, dst, rootFile, absent)
	return d.rows, err
}

func (d *differ) dir(src, dst, rel string, fresh bool) error {
	entries, err := sourceEntries(d.fsys, src, d.filter)
	if err != nil {
		return err
	}
	for _, e := range entries {
		entryRel := path.Join(rel, e.Name())
		entrySrc, entryDst := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		if !e.IsDir() {
			if err := d.file(entrySrc, entryDst, entryRel, fresh); err != nil {
				return err
			}
			continue
		}
		childFresh := fresh
		if !fresh {
			info, err := d.fsys.Lstat(entryDst)
			switch {
			case errors.Is(err, fs.ErrNotExist):
				childFresh = true
			case err != nil:
				return err
			case !info.IsDir():
				if err := d.replace(entryDst, entryRel, info); err != nil {
					return err
				}
				childFresh = true
			}
		}
		if err := d.dir(entrySrc, entryDst, entryRel, childFresh); err != nil {
			return err
		}
	}
	return nil
}

// file adds a row for src when dst is missing or differs; fresh means dst is known to be missing.
func (d *differ) file(src, dst, rel string, fresh bool) error {
	srcInfo, err := d.fsys.Lstat(src)
	if err != nil {
		return err
	}
	srcLink := srcInfo.Mode()&fs.ModeSymlink != 0
	note := ""
	if srcLink {
		note = noteLink
	}
	if fresh {
		d.add(plan.EffectCreate, rel, note)
		return nil
	}
	dstInfo, err := d.fsys.Lstat(dst)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		d.add(plan.EffectCreate, rel, note)
		return nil
	case err != nil:
		return err
	case dstInfo.IsDir():
		return d.replace(dst, rel, dstInfo)
	case srcLink:
		d.add(plan.EffectOverwrite, rel, noteLink)
		return nil
	case !dstInfo.Mode().IsRegular():
		d.add(plan.EffectOverwrite, rel, noteReplaces+describe(dstInfo))
		return nil
	}
	differs, err := d.contentDiffers(src, dst, srcInfo, dstInfo)
	if differs {
		d.add(plan.EffectOverwrite, rel, "")
	}
	return err
}

// replace handles an entry whose type blocks the copy: an error, or an Overwrite row under BlockedReplace.
func (d *differ) replace(dst, rel string, info fs.FileInfo) error {
	if d.blocked == BlockedFail {
		return ErrPathConflict{Path: dst, Found: describe(info)}
	}
	d.add(plan.EffectOverwrite, rel, noteReplaces+describe(info))
	return nil
}

func (d *differ) add(effect plan.Effect, rel, note string) {
	d.rows = append(d.rows, plan.Change{Effect: effect, Path: rel, Note: note})
}

func (d *differ) contentDiffers(src, dst string, srcInfo, dstInfo fs.FileInfo) (bool, error) {
	if srcInfo.Size() != dstInfo.Size() {
		return true, nil
	}
	srcData, err := d.fsys.ReadFile(src)
	if err != nil {
		return false, err
	}
	dstData, err := d.fsys.ReadFile(dst)
	if err != nil {
		return false, err
	}
	return !bytes.Equal(srcData, dstData), nil
}

// describe names the type of an entry for notes and conflicts.
func describe(info fs.FileInfo) string {
	switch mode := info.Mode(); {
	case mode.IsDir():
		return describeDirectory
	case mode&fs.ModeSymlink != 0:
		return describeSymlink
	case mode.IsRegular():
		return "file"
	}
	return "special file"
}

// sourceEntries lists dir without the entries the filter excludes.
func sourceEntries(fsys check.FS, dir string, filter Filter) ([]fs.DirEntry, error) {
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	kept := entries[:0:0]
	for _, e := range entries {
		if !filter.Excludes(e.Name()) {
			kept = append(kept, e)
		}
	}
	return kept, nil
}
