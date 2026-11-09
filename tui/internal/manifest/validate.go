package manifest

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

func Validate(m Manifest, allowedKinds []Kind) error {
	seen := map[string]bool{}
	for i, c := range m.Components {
		switch {
		case c.Name == "":
			return fmt.Errorf("component %d: empty name", i)
		case c.Kind == "":
			return fmt.Errorf("component %q: empty kind", c.Name)
		case c.Source == "":
			return fmt.Errorf("component %q: empty source", c.Name)
		case c.Target == "":
			return fmt.Errorf("component %q: empty target", c.Name)
		case !slices.Contains(allowedKinds, c.Kind):
			return fmt.Errorf("component %q: unknown kind %q", c.Name, c.Kind)
		case seen[c.Name]:
			return fmt.Errorf("duplicate component name %q", c.Name)
		}
		seen[c.Name] = true
		if !isRelativeInside(c.Source) {
			return fmt.Errorf("component %q: source %q must be relative without \"..\"", c.Name, c.Source)
		}
		if !isRelativeInside(c.Target) {
			return fmt.Errorf("component %q: target %q must be relative without \"..\"", c.Name, c.Target)
		}
		if slices.Contains(c.Needs, "") {
			return fmt.Errorf("component %q: empty need", c.Name)
		}
	}
	return nil
}

func isRelativeInside(p string) bool {
	return !strings.HasPrefix(p, "/") && !slices.Contains(strings.Split(p, "/"), "..") && path.Clean(p) != "."
}
