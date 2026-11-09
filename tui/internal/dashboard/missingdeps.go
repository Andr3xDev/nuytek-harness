package dashboard

import (
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

type MissingDep struct {
	Tool       string
	Dependents []string
}

func MissingDeps(m manifest.Manifest, results []check.Result) []MissingDep {
	var missing []MissingDep
	for _, r := range results {
		if r.Status != check.StatusError {
			continue
		}
		dep := MissingDep{Tool: r.Subject}
		for _, c := range m.Components {
			if slices.Contains(c.Needs, r.Subject) {
				dep.Dependents = append(dep.Dependents, c.Name)
			}
		}
		missing = append(missing, dep)
	}
	slices.SortFunc(missing, func(a, b MissingDep) int { return strings.Compare(a.Tool, b.Tool) })
	return missing
}
