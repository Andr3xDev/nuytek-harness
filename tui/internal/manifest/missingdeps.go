package manifest

import (
	"slices"
	"strings"
)

type MissingDep struct {
	Tool       string
	Dependents []string
}

// MissingDeps pairs each tool with the components that list it in needs, sorted by tool.
func (m Manifest) MissingDeps(tools []string) []MissingDep {
	var missing []MissingDep
	for _, tool := range tools {
		dep := MissingDep{Tool: tool}
		for _, c := range m.Components {
			if slices.Contains(c.Needs, tool) {
				dep.Dependents = append(dep.Dependents, c.Name)
			}
		}
		missing = append(missing, dep)
	}
	slices.SortFunc(missing, func(a, b MissingDep) int { return strings.Compare(a.Tool, b.Tool) })
	return missing
}
