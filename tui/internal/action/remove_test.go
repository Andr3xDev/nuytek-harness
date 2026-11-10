package action_test

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func TestPlanRemove(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fixture) []manifest.Component
		names []string
		check func(t *testing.T, f *fixture, p plan.Plan, err error)
	}{
		{
			name: "removes only files in both source and target, dirs deepest first plus the root",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "a", 0o644)
				f.writeRepo("claude/agents/sub/b.md", "b", 0o644)
				f.writeRepo("claude/agents/repo-only.md", "r", 0o644)
				f.writeHome(".claude/agents/a.md", "a", 0o644)
				f.writeHome(".claude/agents/sub/b.md", "b", 0o644)
				f.writeHome(".claude/agents/foreign.md", "mine", 0o644)
				f.writeHome(".claude/agents/sub/foreign2.md", "mine", 0o644)
				return []manifest.Component{component("agents", manifest.KindDir)}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				rm := onlyStep[plan.Remove](t, p)
				root := filepath.Join(f.home, ".claude/agents")
				files := slices.Clone(rm.Files)
				slices.Sort(files)
				if want := []string{filepath.Join(root, "a.md"), filepath.Join(root, "sub/b.md")}; !slices.Equal(files, want) {
					t.Errorf("files = %v, want %v", files, want)
				}
				if want := []string{filepath.Join(root, "sub"), root}; !slices.Equal(rm.Dirs, want) {
					t.Errorf("dirs = %v, want deepest first %v", rm.Dirs, want)
				}
				if p.Confirm == "" {
					t.Error("Confirm is empty, remove is destructive")
				}
				if len(stepsOf[plan.RemoveTree](p)) != 0 {
					t.Error("remove must not use RemoveTree")
				}
			},
		},
		{
			name: "file deleted from the repo stays deployed",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "a", 0o644)
				f.writeHome(".claude/agents/a.md", "a", 0o644)
				f.writeHome(".claude/agents/gone.md", "was owned", 0o644)
				return []manifest.Component{component("agents", manifest.KindDir)}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				rm := onlyStep[plan.Remove](t, p)
				if slices.Contains(rm.Files, filepath.Join(f.home, ".claude/agents/gone.md")) {
					t.Errorf("files %v list a file the repo no longer owns", rm.Files)
				}
			},
		},
		{
			name: "file component removes its target",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/settings.json", "{}", 0o644)
				f.writeHome(".claude/settings.json", "{}", 0o644)
				return []manifest.Component{component("settings.json", manifest.KindFile)}
			},
			names: []string{"settings.json"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				rm := onlyStep[plan.Remove](t, p)
				if want := []string{filepath.Join(f.home, ".claude/settings.json")}; !slices.Equal(rm.Files, want) {
					t.Errorf("files = %v, want %v", rm.Files, want)
				}
			},
		},
		{
			name: "nothing deployed gives an empty plan",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "a", 0o644)
				return []manifest.Component{component("agents", manifest.KindDir)}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				if !p.Empty() {
					t.Errorf("plan not empty: %v", p.Rows())
				}
			},
		},
		{
			name: "symlinked component root is refused",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "a", 0o644)
				f.writeHome("elsewhere/a.md", "a", 0o644)
				f.symlink(f.home, ".claude/agents", filepath.Join(f.home, "elsewhere"))
				return []manifest.Component{component("agents", manifest.KindDir)}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				var conflict drift.ErrPathConflict
				if !errors.As(err, &conflict) {
					t.Fatalf("err = %v, want ErrPathConflict", err)
				}
				if len(p.Steps) != 0 {
					t.Errorf("steps = %v, want none", p.Steps)
				}
			},
		},
		{
			name: "kind without a Remove part is refused",
			setup: func(f *fixture) []manifest.Component {
				f.special = map[manifest.Kind]plan.KindOps{"plugins": fakeOps([]plan.Step{}, nil, nil)}
				return []manifest.Component{component("plugins", "plugins")}
			},
			names: []string{"plugins"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				if err == nil {
					t.Fatal("err = nil, want a not-removable error")
				}
				if len(p.Steps) != 0 {
					t.Errorf("steps = %v, want none", p.Steps)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			components := tt.setup(f)

			p, err := action.PlanRemove(context.Background(), f.env(), f.target(components...), tt.names)

			tt.check(t, f, p, err)
		})
	}
}
