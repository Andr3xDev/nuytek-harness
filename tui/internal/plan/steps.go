package plan

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
)

type Effect string

const (
	EffectCreate    Effect = "create"
	EffectOverwrite Effect = "overwrite"
	EffectDelete    Effect = "delete"
	EffectRun       Effect = "run"
	EffectSkip      Effect = "skip"
)

// Change is a preview row; Unrestorable means the snapshot cannot bring it back.
type Change struct {
	Effect       Effect
	Path, Note   string
	Unrestorable bool
}

type Touch struct{ Paths, Globs []string }

type Step interface {
	Describe() string
	Changes() []Change
	Touches() Touch
}

type Snapshot struct {
	Store backup.Dir
	Paths []string
	Globs []string
}

// Copy rows are named Rows because a Changes field would clash with the Changes method.
type Copy struct {
	Src, Dst string
	Files    []string
	Rows     []Change
}

type Remove struct {
	Files []string
	Dirs  []string
}

type RemoveTree struct {
	Path string
	Rows []Change
}

type Chmod struct{ Paths []string }

type Edit struct {
	Path string
	Edit func(old []byte, exists bool) ([]byte, error)
	Rows []Change
}

type Exec struct {
	Argv    []string
	Timeout time.Duration
	Warn    bool
	Rows    []Change
}

type RemoveMatching struct{ Globs []string }

func (s Snapshot) Describe() string { return "snapshot into " + s.Store.Label }
func (Snapshot) Changes() []Change  { return nil }
func (s Snapshot) Touches() Touch   { return Touch{Paths: s.Paths, Globs: s.Globs} }
func (c Copy) Describe() string     { return fmt.Sprintf("copy %s to %s", c.Src, c.Dst) }
func (c Copy) Changes() []Change    { return c.Rows }
func (c Copy) Touches() Touch {
	var paths []string
	for _, row := range c.Rows {
		if row.Effect == EffectOverwrite {
			paths = append(paths, filepath.Join(c.Dst, row.Path))
		}
	}
	return Touch{Paths: paths}
}

func (Remove) Describe() string { return "remove files" }
func (r Remove) Changes() []Change {
	rows := make([]Change, 0, len(r.Files)+len(r.Dirs))
	for _, p := range r.Files {
		rows = append(rows, Change{Effect: EffectDelete, Path: p})
	}
	// The runner's rmdir ignores "not empty", so a directory may survive.
	for _, p := range r.Dirs {
		rows = append(rows, Change{Effect: EffectDelete, Path: p, Note: "if empty"})
	}
	return rows
}
func (r Remove) Touches() Touch { return Touch{Paths: r.Files} }

func (t RemoveTree) Describe() string  { return "remove " + t.Path }
func (t RemoveTree) Changes() []Change { return t.Rows }
func (t RemoveTree) Touches() Touch    { return Touch{Paths: []string{t.Path}} }

func (Chmod) Describe() string { return "make files executable" }
func (c Chmod) Changes() []Change {
	rows := make([]Change, 0, len(c.Paths))
	for _, p := range c.Paths {
		rows = append(rows, Change{Effect: EffectOverwrite, Path: p, Note: "chmod +x"})
	}
	return rows
}
func (Chmod) Touches() Touch { return Touch{} }

func (e Edit) Describe() string  { return "edit " + e.Path }
func (e Edit) Changes() []Change { return e.Rows }
func (e Edit) Touches() Touch    { return Touch{Paths: []string{e.Path}} }

func (e Exec) Describe() string { return strings.Join(e.Argv, " ") }
func (e Exec) Changes() []Change {
	if e.Rows == nil {
		return []Change{{Effect: EffectRun, Path: e.Describe()}}
	}
	return e.Rows
}
func (Exec) Touches() Touch { return Touch{} }

func (RemoveMatching) Describe() string { return "remove matching files" }
func (r RemoveMatching) Changes() []Change {
	rows := make([]Change, 0, len(r.Globs))
	for _, g := range r.Globs {
		rows = append(rows, Change{Effect: EffectDelete, Path: g, Note: "matching files, if any"})
	}
	return rows
}
func (r RemoveMatching) Touches() Touch { return Touch{Globs: r.Globs} }

const globMetaChars = `\*?[`

// EscapeGlob makes a literal path usable as the prefix of a glob pattern.
func EscapeGlob(literal string) string {
	var b strings.Builder
	for _, r := range literal {
		if strings.ContainsRune(globMetaChars, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
