// Package specter models the Specter store root and its openspec registration.
// It is target-independent and reads only through check.FS and check.Exec.
package specter

import (
	"path/filepath"
	"strings"
)

const (
	rootEnvVar     = "NUYTEK_SPECTER_ROOT"
	defaultRootRel = "dev/specter"
)

type RootSource string

const (
	RootSourceEnv     RootSource = "env"
	RootSourceDefault RootSource = "default"
)

type Root struct {
	Path        string
	Source      RootSource
	OutsideHome bool
	Relative    bool // a relative path is a missing root, never "outside home"
}

// ResolveRoot never touches the filesystem; an empty env value counts as unset.
func ResolveRoot(lookupEnv func(string) (string, bool), home string) Root {
	root := Root{Path: filepath.Join(home, defaultRootRel), Source: RootSourceDefault}
	if v, ok := lookupEnv(rootEnvVar); ok && v != "" {
		root = Root{Path: filepath.Clean(v), Source: RootSourceEnv}
	}
	root.Relative = !filepath.IsAbs(root.Path)
	root.OutsideHome = !root.Relative && outsideHome(home, root.Path)
	return root
}

func outsideHome(home, path string) bool {
	rel, err := filepath.Rel(home, path)
	if err != nil {
		return true
	}
	first, _, _ := strings.Cut(rel, string(filepath.Separator))
	return first == ".."
}
