package action_test

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const notPulledNote = "not pulled: not in repo"

func TestPlanPull(t *testing.T) {
	agents := pullable(component("agents", manifest.KindDir))
	tests := []struct {
		name  string
		setup func(f *fixture) []manifest.Component
		names []string
		check func(t *testing.T, f *fixture, p plan.Plan, err error)
	}{
		{
			name: "overwrite becomes one copy, deployed-only is a skip row, repo-only is untouched",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "old", 0o644)
				f.writeRepo("claude/agents/b.md", "repo-only", 0o644)
				f.writeHome(".claude/agents/a.md", "new", 0o644)
				f.writeHome(".claude/agents/custom.md", "mine", 0o644)
				return []manifest.Component{agents}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				cp := onlyStep[plan.Copy](t, p)
				if cp.Src != filepath.Join(f.home, ".claude/agents") || cp.Dst != filepath.Join(f.repo, "claude/agents") {
					t.Errorf("copy %s -> %s, want deployed to repo source", cp.Src, cp.Dst)
				}
				if !slices.Equal(cp.Files, []string{"a.md"}) {
					t.Errorf("copy files = %v, want [a.md]", cp.Files)
				}
				if got := rowPaths(p, plan.EffectSkip); !slices.Equal(got, []string{"custom.md"}) {
					t.Errorf("skip rows = %v, want [custom.md]", got)
				}
				for _, r := range p.Rows() {
					if r.Effect == plan.EffectSkip && r.Note != notPulledNote {
						t.Errorf("skip row note = %q, want %q", r.Note, notPulledNote)
					}
					if r.Path == "b.md" {
						t.Error("repo-only b.md appears in a row")
					}
				}
			},
		},
		{
			name: "secrets never become rows",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "old", 0o644)
				f.writeHome(".claude/agents/a.md", "new", 0o644)
				f.writeHome(".claude/agents/.env", "SECRET=1", 0o644)
				f.writeHome(".claude/agents/key.pem", "k", 0o644)
				return []manifest.Component{agents}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				for _, r := range p.Rows() {
					if r.Path == ".env" || r.Path == "key.pem" {
						t.Errorf("secret %s appears as a row", r.Path)
					}
				}
			},
		},
		{
			name: "deployed symlink is a skip row and never copied",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "old", 0o644)
				f.writeRepo("claude/agents/ln.md", "old", 0o644)
				f.writeHome(".claude/agents/a.md", "new", 0o644)
				f.symlink(f.home, ".claude/agents/ln.md", "/etc/hostname")
				return []manifest.Component{agents}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				cp := onlyStep[plan.Copy](t, p)
				if !slices.Equal(cp.Files, []string{"a.md"}) {
					t.Errorf("copy files = %v, want [a.md]", cp.Files)
				}
				if got := rowPaths(p, plan.EffectSkip); !slices.Equal(got, []string{"ln.md"}) {
					t.Errorf("skip rows = %v, want [ln.md]", got)
				}
				for _, r := range p.Rows() {
					if r.Effect == plan.EffectSkip && r.Note != "not pulled: symlink" {
						t.Errorf("skip row note = %q, want %q", r.Note, "not pulled: symlink")
					}
				}
			},
		},
		{
			name: "snapshot holds only the existing repo files of overwrite rows, in the repo store",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "old", 0o644)
				f.writeRepo("claude/agents/b.md", "repo-only", 0o644)
				f.writeHome(".claude/agents/a.md", "new", 0o644)
				f.writeHome(".claude/agents/custom.md", "mine", 0o644)
				return []manifest.Component{agents}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				if want := []string{filepath.Join(f.repo, "claude/agents/a.md")}; !slices.Equal(snapshotPaths(p), want) {
					t.Errorf("snapshot paths = %v, want %v", snapshotPaths(p), want)
				}
				if kind := p.Steps[0].(plan.Snapshot).Store.Kind; kind != backup.StoreRepo {
					t.Errorf("snapshot store kind = %v, want StoreRepo", kind)
				}
			},
		},
		{
			name: "missing repo source gives only skip rows",
			setup: func(f *fixture) []manifest.Component {
				f.writeHome(".claude/agents/custom.md", "mine", 0o644)
				return []manifest.Component{agents}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				rows := p.Rows()
				if len(rows) != 1 || rows[0].Effect != plan.EffectSkip || rows[0].Path != "custom.md" {
					t.Errorf("rows = %+v, want one skip row for custom.md", rows)
				}
				if !p.Empty() {
					t.Error("plan with only skip rows must be Empty")
				}
				if len(stepsOf[plan.Snapshot](p)) != 0 {
					t.Error("nothing to overwrite, so no snapshot")
				}
			},
		},
		{
			name: "explicit non-pull name is refused",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "old", 0o644)
				f.writeHome(".claude/agents/a.md", "new", 0o644)
				return []manifest.Component{component("agents", manifest.KindDir)}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				if err == nil {
					t.Fatal("err = nil, want a not-pulled error")
				}
				if len(p.Steps) != 0 {
					t.Errorf("steps = %v, want none", p.Steps)
				}
			},
		},
		{
			name: "target without a repo store is an error",
			setup: func(f *fixture) []manifest.Component {
				f.noRepoStore = true
				f.writeHome(".claude/agents/a.md", "new", 0o644)
				return []manifest.Component{agents}
			},
			names: []string{"agents"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				if err == nil || !strings.Contains(err.Error(), "backup store") {
					t.Fatalf("err = %v, want a 'no backup store' error", err)
				}
			},
		},
		{
			name: "missing deployed path aborts naming it and plans nothing",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "old", 0o644)
				f.writeHome(".claude/agents/a.md", "new", 0o644)
				return []manifest.Component{agents, pullable(component("hooks", manifest.KindDirX))}
			},
			names: []string{"agents", "hooks"},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				var missing action.ErrMissingDeployed
				if !errors.As(err, &missing) || !slices.Equal(missing.Names, []string{"hooks"}) {
					t.Fatalf("err = %v, want ErrMissingDeployed{[hooks]}", err)
				}
				if len(p.Steps) != 0 {
					t.Errorf("steps = %v, want none, not even for agents", p.Steps)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			components := tt.setup(f)

			p, err := action.PlanPull(context.Background(), f.env(), f.target(components...), tt.names)

			tt.check(t, f, p, err)
		})
	}
}
