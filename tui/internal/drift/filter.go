package drift

import (
	"path"
	"slices"
)

// Filter names base names to leave out of a walk, as exact names or globs.
type Filter struct {
	Names, Globs []string
}

// Excludes reports whether a base name is filtered out.
func (f Filter) Excludes(base string) bool {
	if slices.Contains(f.Names, base) {
		return true
	}
	for _, glob := range f.Globs {
		if ok, _ := path.Match(glob, base); ok {
			return true
		}
	}
	return false
}

var (
	InstallFilter = Filter{Names: []string{"__pycache__"}, Globs: []string{"*.pyc"}}
	PullFilter    = Filter{
		Names: append([]string{".git", "node_modules", "auth.json", "credentials.json", ".credentials.json", ".env", ".DS_Store"}, InstallFilter.Names...),
		Globs: append([]string{".env.*", "*.pem", "*.key", "*.p12", "*.pfx", "*.token"}, InstallFilter.Globs...),
	}
)
