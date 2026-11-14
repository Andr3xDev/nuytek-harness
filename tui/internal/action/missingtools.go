package action

import (
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

// ErrMissingTools refuses Hard install before any step exists.
type ErrMissingTools struct{ Tools []manifest.MissingDep }

func (e ErrMissingTools) Error() string {
	names := make([]string, 0, len(e.Tools))
	for _, t := range e.Tools {
		names = append(names, t.Tool)
	}
	return "missing tools: " + strings.Join(names, ", ")
}

// MissingTools reports the needed tools absent from PATH, with their dependents.
func MissingTools(exec check.Exec, m manifest.Manifest) []manifest.MissingDep {
	var absent []string
	for _, tool := range m.AllNeeds() {
		if _, err := exec.LookPath(tool); err != nil {
			absent = append(absent, tool)
		}
	}
	return m.MissingDeps(absent)
}
