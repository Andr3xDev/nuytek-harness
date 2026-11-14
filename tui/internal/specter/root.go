// Package specter models the Specter store root and its openspec registration.
// It is target-independent and reads only through check.FS and check.Exec.
package specter

import (
	"path/filepath"
	"strings"
)

const defaultRootRel = "dev/specter"

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
	if saved != "" {
		root = Root{Path: filepath.Clean(saved), Source: RootSourceSaved}
	}
	root.Relative = !filepath.IsAbs(root.Path)
	root.OutsideHome = !root.Relative && outsideHome(home, root.Path)
	return root
}

func DefaultRoot(home string) string {
	return filepath.Join(home, defaultRootRel)
}

func outsideHome(home, path string) bool {
	rel, err := filepath.Rel(home, path)
	if err != nil {
		return true
	}
	first, _, _ := strings.Cut(rel, string(filepath.Separator))
	return first == ".."
}
