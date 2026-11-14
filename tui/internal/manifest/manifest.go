package manifest

import "slices"

type Component struct {
	Name           string
	Kind           Kind
	Source, Target string
	Needs          []string
}

type Manifest struct {
	Path       string
	Components []Component
}

func (m Manifest) TargetOf(name string) (string, bool) {
	for _, c := range m.Components {
		if c.Name == name {
			return c.Target, true
		}
	}
	return "", false
}

func (m Manifest) AllNeeds() []string {
	var needs []string
	for _, c := range m.Components {
		needs = append(needs, c.Needs...)
	}
	slices.Sort(needs)
	return slices.Compact(needs)
}
