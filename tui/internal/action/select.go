package action

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

type Scope int

const (
	ScopeInstall Scope = iota
	ScopeUpdate
	ScopeRemove
	ScopePull
)

// AllNames expands to every component of the scope.
const AllNames = "all"

// Select resolves names (or AllNames) to the components of the scope, in manifest order.
// The scope narrows AllNames only; a named component is returned and its planner refuses what it cannot do.
func Select(t target.Target, names []string, scope Scope) ([]manifest.Component, error) {
	valid := make([]string, 0, len(t.Manifest.Components))
	for _, c := range t.Manifest.Components {
		valid = append(valid, c.Name)
	}
	for _, name := range names {
		if name != AllNames && !slices.Contains(valid, name) {
			return nil, fmt.Errorf("unknown component %q, valid names: %s", name, strings.Join(valid, ", "))
		}
	}
	var selected []manifest.Component
	for _, c := range t.Manifest.Components {
		inScope := slices.Contains(names, c.Name)
		if slices.Contains(names, AllNames) {
			var err error
			if inScope, err = inAllOf(t, c, scope); err != nil {
				return nil, err
			}
		}
		if inScope {
			selected = append(selected, c)
		}
	}
	return selected, nil
}

func inAllOf(t target.Target, c manifest.Component, scope Scope) (bool, error) {
	switch scope {
	case ScopePull:
		return c.Pull, nil
	case ScopeInstall:
		return true, nil
	}
	ops, err := opsFor(t.Spec, c.Kind)
	if err != nil {
		return false, err
	}
	if scope == ScopeRemove {
		return ops.Remove != nil, nil
	}
	return !ops.ExplicitUpdate, nil
}
