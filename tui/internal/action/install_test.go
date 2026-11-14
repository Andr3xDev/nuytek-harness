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

func TestPlanInstall(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fixture) []manifest.Component
		names []string
		check func(t *testing.T, f *fixture, p plan.Plan, err error)
	}{
		{
			name: "custom file is neither a row, a delete nor in the snapshot",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "new", 0o644)
				f.writeHome(".claude/agents/a.md", "old", 0o644)
				f.writeHome(".claude/agents/custom.md", "mine", 0o644)
				return []manifest.Component{component("agents", manifest.KindDir)}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				if got := rowPaths(p, plan.EffectOverwrite); !slices.Equal(got, []string{"a.md"}) {
					t.Errorf("overwrite rows = %v, want [a.md]", got)
				}
				if got := rowPaths(p, plan.EffectDelete); len(got) != 0 {
					t.Errorf("delete rows = %v, want none", got)
				}
				want := []string{filepath.Join(f.home, ".claude/agents/a.md")}
				if got := snapshotPaths(p); !slices.Equal(got, want) {
					t.Errorf("snapshot paths = %v, want %v", got, want)
				}
				if snap := p.Steps[0].(plan.Snapshot); snap.Store.Label != "home" {
					t.Errorf("snapshot store = %q, want the home store", snap.Store.Label)
				}
			},
		},
		{
			name: "in-sync component gives an empty plan",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "same", 0o644)
				f.writeHome(".claude/agents/a.md", "same", 0o644)
				return []manifest.Component{component("agents", manifest.KindDir)}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				if !p.Empty() || len(stepsOf[plan.Copy](p)) != 0 {
					t.Errorf("plan not empty: %v", p.Steps)
				}
			},
		},
		{
			name: "directory at a file destination is a path conflict",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/settings.json", "{}", 0o644)
				f.mkdir(f.home, ".claude/settings.json")
				return []manifest.Component{component("settings.json", manifest.KindFile)}
			},
			names: []string{"settings.json"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				var conflict drift.ErrPathConflict
				if !errors.As(err, &conflict) {
					t.Fatalf("err = %v, want ErrPathConflict", err)
				}
			},
		},
		{
			name: "symlink at a file destination is replaced as a whole file",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/settings.json", "{}", 0o644)
				f.writeHome("elsewhere.json", "other", 0o644)
				f.symlink(f.home, ".claude/settings.json", filepath.Join(f.home, "elsewhere.json"))
				return []manifest.Component{component("settings.json", manifest.KindFile)}
			},
			names: []string{"settings.json"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				cp := onlyStep[plan.Copy](t, p)
				if !slices.Equal(cp.Files, []string{"."}) || rowPaths(p, plan.EffectOverwrite)[0] != "." {
					t.Errorf("copy files %v rows %v, want the single file \".\" overwritten", cp.Files, p.Rows())
				}
			},
		},
		{
			name: "dirx chmods only top-level scripts lacking the exec bit",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/hooks/lacks.sh", "x", 0o755)
				f.writeRepo("claude/hooks/ok.sh", "y", 0o755)
				f.writeRepo("claude/hooks/sub/inner.sh", "z", 0o755)
				f.writeRepo("claude/hooks/lib.py", "w", 0o644)
				f.writeHome(".claude/hooks/lacks.sh", "x", 0o644)
				f.writeHome(".claude/hooks/ok.sh", "y", 0o755)
				f.writeHome(".claude/hooks/sub/inner.sh", "z", 0o644)
				f.writeHome(".claude/hooks/lib.py", "w", 0o644)
				return []manifest.Component{component("hooks", manifest.KindDirX)}
			},
			names: []string{"hooks"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				chmod := onlyStep[plan.Chmod](t, p)
				want := []string{filepath.Join(f.home, ".claude/hooks/lacks.sh")}
				if !slices.Equal(chmod.Paths, want) {
					t.Errorf("chmod paths = %v, want %v", chmod.Paths, want)
				}
			},
		},
		{
			name: "filex lacking the exec bit is chmoded",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/tool.sh", "x", 0o755)
				f.writeHome(".claude/tool.sh", "x", 0o644)
				c := component("tool.sh", manifest.KindFileX)
				return []manifest.Component{c}
			},
			names: []string{"tool.sh"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				chmod := onlyStep[plan.Chmod](t, p)
				if want := []string{filepath.Join(f.home, ".claude/tool.sh")}; !slices.Equal(chmod.Paths, want) {
					t.Errorf("chmod paths = %v, want %v", chmod.Paths, want)
				}
			},
		},
		{
			name: "missing source names the component and path and plans nothing",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/hooks/a.sh", "x", 0o755)
				return []manifest.Component{component("agents", manifest.KindDir), component("hooks", manifest.KindDirX)}
			},
			names: []string{"agents", "hooks"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				var missing action.ErrMissingSource
				if !errors.As(err, &missing) || missing.Component != "agents" || missing.Path != filepath.Join(f.repo, "claude/agents") {
					t.Fatalf("err = %v, want ErrMissingSource for agents at %s", err, filepath.Join(f.repo, "claude/agents"))
				}
				if len(p.Steps) != 0 {
					t.Errorf("steps = %v, want none", p.Steps)
				}
			},
		},
		{
			name: "component with a missing tool is skipped with no steps",
			setup: func(f *fixture) []manifest.Component {
				f.missingTools = []string{"git"}
				f.writeRepo("claude/agents/a.md", "new", 0o644)
				return []manifest.Component{needing(component("agents", manifest.KindDir), "git")}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				if want := []plan.Skip{{Name: "agents", Reason: "missing: git"}}; !slices.Equal(p.Skipped, want) {
					t.Errorf("skipped = %v, want %v", p.Skipped, want)
				}
				if len(p.Steps) != 0 {
					t.Errorf("steps = %v, want none", p.Steps)
				}
			},
		},
		{
			name: "late kinds are planned last",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "new", 0o644)
				marker := plan.Exec{Argv: []string{"late-kind"}}
				ops := fakeOps([]plan.Step{marker}, nil, nil)
				ops.Late = true
				f.special = map[manifest.Kind]plan.KindOps{"plugins": ops}
				return []manifest.Component{component("plugins", "plugins"), component("agents", manifest.KindDir)}
			},
			names: []string{"plugins", "agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				if len(p.Steps) < 2 {
					t.Fatalf("steps = %v, want a Copy then the special Exec", p.Steps)
				}
				if _, ok := p.Steps[len(p.Steps)-1].(plan.Exec); !ok {
					t.Errorf("last step is %T, want the Late kind's Exec", p.Steps[len(p.Steps)-1])
				}
				if len(stepsOf[plan.Copy](p)) != 1 {
					t.Errorf("want one Copy for agents, steps: %v", p.Steps)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			components := tt.setup(f)

			p, err := action.PlanInstall(context.Background(), f.env(), f.target(components...), tt.names)

			tt.check(t, f, p, err)
		})
	}
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
