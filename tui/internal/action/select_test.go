package action_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func TestSelect(t *testing.T) {
	f := newFixture(t)
	const kindPlugins, kindMCP = manifest.Kind("plugins"), manifest.Kind("mcp")
	f.special = map[manifest.Kind]plan.KindOps{
		kindMCP:     fakeOps([]plan.Step{}, nil, []plan.Step{}),
		kindPlugins: {Install: fakeOps([]plan.Step{}, nil, nil).Install, Late: true, ExplicitUpdate: true},
	}
	tg := f.target(
		pullable(component("agents", manifest.KindDir)),
		component("codegraph", manifest.KindFileX),
		component("mcp", kindMCP),
		component("plugins", kindPlugins),
	)
	tests := []struct {
		name  string
		names []string
		scope action.Scope
		want  []string
	}{
		{"install all", []string{action.AllNames}, action.ScopeInstall, []string{"agents", "codegraph", "mcp", "plugins"}},
		{"update all skips explicit-update kinds", []string{action.AllNames}, action.ScopeUpdate, []string{"agents", "codegraph", "mcp"}},
		{"update by name accepts an explicit-update kind", []string{"plugins"}, action.ScopeUpdate, []string{"plugins"}},
		{"remove all skips non-removable kinds", []string{action.AllNames}, action.ScopeRemove, []string{"agents", "codegraph", "mcp"}},
		{"pull all keeps pull components only", []string{action.AllNames}, action.ScopePull, []string{"agents"}},
		{"named subset", []string{"mcp", "agents"}, action.ScopeInstall, []string{"agents", "mcp"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := action.Select(tg, tt.names, tt.scope)

			if err != nil {
				t.Fatalf("Select error: %v", err)
			}
			names := make([]string, 0, len(got))
			for _, c := range got {
				names = append(names, c.Name)
			}
			slices.Sort(names)
			if !slices.Equal(names, tt.want) {
				t.Fatalf("selected %v, want %v", names, tt.want)
			}
		})
	}

	t.Run("unknown name lists the valid names", func(t *testing.T) {
		_, err := action.Select(tg, []string{"nope"}, action.ScopeInstall)

		if err == nil {
			t.Fatal("want an error for an unknown name")
		}
		for _, valid := range []string{"nope", "agents", "codegraph", "mcp", "plugins"} {
			if !strings.Contains(err.Error(), valid) {
				t.Errorf("error %q does not mention %q", err, valid)
			}
		}
	})
}
