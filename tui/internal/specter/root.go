// Package specter models the Specter store root and its openspec registration.
// It is target-independent and reads only through check.FS and check.Exec.
package specter

import (
	"path/filepath"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/render"
)

type RootSource string

const (
	RootSourceSaved   RootSource = "saved"
	RootSourceDefault RootSource = "default"
)

type Root struct {
	Path        string
	Source      RootSource
	OutsideHome bool
	Relative    bool // a relative path is a missing root, never "outside home"
}

// ResolveRoot never touches the filesystem; an empty saved value counts as unset.
func ResolveRoot(saved, home string) Root {
	root := Root{Path: DefaultRoot(home), Source: RootSourceDefault}
	raw := root.Path
	if saved != "" {
		root = Root{Path: filepath.Clean(saved), Source: RootSourceSaved}
		raw = saved
	}
	root.Relative = !filepath.IsAbs(root.Path)
	// Validity uses the raw value: Clean would fold a ".." segment away and hide it.
	root.OutsideHome = !root.Relative && !render.ValidRoot(home, raw)
	return root
}

func DefaultRoot(home string) string {
	return filepath.Join(home, render.DefaultRootRel)
}

func outsideHome(home, path string) bool {
	rel, err := filepath.Rel(home, path)
	if err != nil {
		return true
	}
	first, _, _ := strings.Cut(rel, string(filepath.Separator))
	return first == ".."
}
