package ui

import (
	"slices"
	"testing"

	"charm.land/bubbles/v2/key"
)

func TestKeyMap_SpecterKeysAreDistinctAndFreeOfNavigationKeys(t *testing.T) {
	k := newKeyMap()
	specter := map[string]key.Binding{"Validate": k.Validate, "FastInit": k.FastInit, "Maintenance": k.Maintenance, "Refresh": k.Refresh}
	navigation := slices.Concat(k.Back.Keys(), k.PrevTab.Keys(), k.NextTab.Keys(), k.Quit.Keys(), k.Focus.Keys())

	seen := map[string]string{}
	for name, b := range specter {
		for _, pressed := range b.Keys() {
			if other, dup := seen[pressed]; dup {
				t.Errorf("key %q bound to both %s and %s", pressed, other, name)
			}
			seen[pressed] = name
			if slices.Contains(navigation, pressed) {
				t.Errorf("%s key %q collides with a navigation or quit key", name, pressed)
			}
		}
	}
}

func TestKeyMap_HelpIsContextAware(t *testing.T) {
	k := newKeyMap()
	desc := func(onSpecter bool) []string {
		var out []string
		for _, group := range k.helpGroups(nil, onSpecter) {
			for _, b := range group {
				out = append(out, b.Help().Desc)
			}
		}
		return out
	}

	elsewhere, onSpecter := desc(false), desc(true)
	if slices.Contains(elsewhere, "maintenance") || slices.Contains(elsewhere, "validate") || slices.Contains(elsewhere, "fast init") || !slices.Contains(elsewhere, "mode") {
		t.Errorf("help outside Specter = %v, want mode and no Specter keys", elsewhere)
	}
	if slices.Contains(onSpecter, "mode") || slices.Contains(onSpecter, "install") || !slices.Contains(onSpecter, "maintenance") {
		t.Errorf("help on Specter = %v, want maintenance and no mode or install", onSpecter)
	}
}

func TestKeyMap_SpecterKeysHaveHelpAndAppearInHelpGroups(t *testing.T) {
	k := newKeyMap()
	var all []key.Binding
	for _, group := range k.helpGroups(nil, true) {
		all = append(all, group...)
	}

	for name, want := range map[string]key.Binding{"Validate": k.Validate, "FastInit": k.FastInit, "Maintenance": k.Maintenance} {
		if want.Help().Desc == "" {
			t.Errorf("%s has no help text", name)
		}
		if !slices.ContainsFunc(all, func(b key.Binding) bool { return b.Help() == want.Help() }) {
			t.Errorf("%s is missing from helpGroups", name)
		}
	}
}
