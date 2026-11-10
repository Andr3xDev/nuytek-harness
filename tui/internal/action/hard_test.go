package action_test

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const unrestorableSentence = "plugin removal cannot be restored from the snapshot"

// hardAgentsFixture holds an agents dir with an owned changed file, an owned unchanged file and a custom file.
func hardAgentsFixture(f *fixture) {
	f.writeRepo("claude/agents/a.md", "new", 0o644)
	f.writeRepo("claude/agents/same.md", "same", 0o644)
	f.writeHome(".claude/agents/a.md", "old", 0o644)
	f.writeHome(".claude/agents/same.md", "same", 0o644)
	f.writeHome(".claude/agents/custom.md", "mine", 0o644)
	f.writeHome(".claude.json", "{}", 0o644)
}

// mcpOps edits ~/.claude.json and deletes the foreign key mine on Hard install.
func mcpOps(f *fixture) plan.KindOps {
	path := filepath.Join(f.home, ".claude.json")
	ops := fakeOps(
		[]plan.Step{plan.Edit{Path: path, Rows: []plan.Change{{Effect: plan.EffectOverwrite, Path: "~/.claude.json#mcpServers.owned"}}}},
		[]plan.Step{plan.Edit{Path: path, Rows: []plan.Change{{Effect: plan.EffectDelete, Path: "~/.claude.json#mcpServers.mine"}}}},
		nil,
	)
	return ops
}

func TestPlanHardInstall(t *testing.T) {
	f := newFixture(t)
	hardAgentsFixture(f)
	f.special = map[manifest.Kind]plan.KindOps{"mcp": mcpOps(f)}
	tg := f.target(component("agents", manifest.KindDir), component("mcp", "mcp"))

	p, err := action.PlanHardInstall(context.Background(), f.env(), tg)

	requireNoError(t, err)
	if len(stepsOf[plan.Snapshot](p)) != 1 {
		t.Fatalf("want exactly one Snapshot, steps: %v", p.Steps)
	}
	paths := snapshotPaths(p)
	for _, want := range []string{filepath.Join(f.home, ".claude/agents"), filepath.Join(f.home, ".claude.json")} {
		if !slices.Contains(paths, want) {
			t.Errorf("snapshot paths %v miss %s", paths, want)
		}
	}
	tree := onlyStep[plan.RemoveTree](t, p)
	if tree.Path != filepath.Join(f.home, ".claude/agents") || !slices.ContainsFunc(tree.Rows, func(c plan.Change) bool {
		return c.Effect == plan.EffectDelete && strings.HasSuffix(c.Path, "custom.md")
	}) {
		t.Errorf("RemoveTree = %+v, want the agents dir with a delete row for custom.md", tree)
	}
	cp := onlyStep[plan.Copy](t, p)
	if got := rowPaths(plan.Plan{Steps: []plan.Step{cp}}, plan.EffectOverwrite); !slices.Equal(got, []string{"a.md"}) {
		t.Errorf("Copy.Rows overwrite = %v, want only the changed a.md", got)
	}
	if !slices.Contains(cp.Files, "same.md") || !slices.Contains(cp.Files, "a.md") {
		t.Errorf("Copy.Files = %v, want every owned file (the tree is wiped first)", cp.Files)
	}
	for _, want := range []string{"agents", "custom.md", "mcp", "mcpServers.mine"} {
		if !strings.Contains(p.Confirm, want) {
			t.Errorf("Confirm %q does not name %q", p.Confirm, want)
		}
	}
	if strings.Contains(p.Confirm, unrestorableSentence) {
		t.Error("Confirm carries the unrestorable sentence without an Unrestorable row")
	}
}

func TestPlanHardInstallUnrestorableSentence(t *testing.T) {
	f := newFixture(t)
	hardAgentsFixture(f)
	removal := plan.Exec{Argv: []string{"claude", "plugin", "uninstall", "mine@local"}, Warn: true, Rows: []plan.Change{
		{Effect: plan.EffectDelete, Path: "plugin mine@local", Unrestorable: true},
	}}
	f.special = map[manifest.Kind]plan.KindOps{"plugins": fakeOps([]plan.Step{}, []plan.Step{removal}, nil)}
	tg := f.target(component("agents", manifest.KindDir), component("plugins", "plugins"))

	p, err := action.PlanHardInstall(context.Background(), f.env(), tg)

	requireNoError(t, err)
	if !strings.Contains(p.Confirm, unrestorableSentence) || !strings.Contains(p.Confirm, "plugin mine@local") {
		t.Errorf("Confirm %q must list the plugin row and the unrestorable sentence", p.Confirm)
	}
}

func TestPlanHardInstallShapes(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fixture) []manifest.Component
		check func(t *testing.T, f *fixture, p plan.Plan, err error)
	}{
		{
			name: "missing tool refuses with zero steps and no snapshot",
			setup: func(f *fixture) []manifest.Component {
				f.missingTools = []string{"git"}
				hardAgentsFixture(f)
				return []manifest.Component{needing(component("agents", manifest.KindDir), "git")}
			},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				var missing action.ErrMissingTools
				if !errors.As(err, &missing) || len(missing.Tools) != 1 || missing.Tools[0].Tool != "git" {
					t.Fatalf("err = %v, want ErrMissingTools{git}", err)
				}
				if len(p.Steps) != 0 {
					t.Errorf("steps = %v, want none", p.Steps)
				}
			},
		},
		{
			name: "in-sync component adds no step",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "same", 0o644)
				f.writeHome(".claude/agents/a.md", "same", 0o644)
				return []manifest.Component{component("agents", manifest.KindDir)}
			},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				if len(p.Steps) != 0 || !p.Empty() {
					t.Errorf("steps = %v, want none", p.Steps)
				}
			},
		},
		{
			name: "symlinked component root is a path conflict",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "a", 0o644)
				f.writeHome("elsewhere/keep.md", "k", 0o644)
				f.symlink(f.home, ".claude/agents", filepath.Join(f.home, "elsewhere"))
				return []manifest.Component{component("agents", manifest.KindDir)}
			},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				var conflict drift.ErrPathConflict
				if !errors.As(err, &conflict) {
					t.Fatalf("err = %v, want ErrPathConflict", err)
				}
			},
		},
		{
			name: "missing directory target is a plain install",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/agents/a.md", "a", 0o644)
				return []manifest.Component{component("agents", manifest.KindDir)}
			},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				if len(stepsOf[plan.RemoveTree](p)) != 0 || len(stepsOf[plan.Copy](p)) != 1 {
					t.Errorf("steps = %v, want one Copy and no RemoveTree", p.Steps)
				}
			},
		},
		{
			name: "file component gets its install steps",
			setup: func(f *fixture) []manifest.Component {
				f.writeRepo("claude/settings.json", "new", 0o644)
				f.writeHome(".claude/settings.json", "old", 0o644)
				return []manifest.Component{component("settings.json", manifest.KindFile)}
			},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				cp := onlyStep[plan.Copy](t, p)
				if !slices.Equal(cp.Files, []string{"."}) || len(stepsOf[plan.RemoveTree](p)) != 0 {
					t.Errorf("steps = %v, want one whole-file Copy and no RemoveTree", p.Steps)
				}
			},
		},
		{
			name: "kind without a Hard part runs its Install part",
			setup: func(f *fixture) []manifest.Component {
				register := plan.Exec{Argv: []string{"register"}}
				f.special = map[manifest.Kind]plan.KindOps{"specter": fakeOps([]plan.Step{register}, nil, nil)}
				return []manifest.Component{component("specter", "specter")}
			},
			check: func(t *testing.T, f *fixture, p plan.Plan, err error) {
				requireNoError(t, err)
				if ex := onlyStep[plan.Exec](t, p); !slices.Equal(ex.Argv, []string{"register"}) {
					t.Errorf("exec argv = %v, want [register]", ex.Argv)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			components := tt.setup(f)

			p, err := action.PlanHardInstall(context.Background(), f.env(), f.target(components...))

			tt.check(t, f, p, err)
		})
	}
}
