package plan_test

import (
	"slices"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func TestStepTouches(t *testing.T) {
	tests := []struct {
		name      string
		step      plan.Step
		wantPaths []string
		wantGlobs []string
	}{
		{"copy touches only overwritten destinations", plan.Copy{
			Src: "/src", Dst: "/dst", Files: []string{"new.md", "old.md", "same.md"},
			Rows: []plan.Change{
				{Effect: plan.EffectCreate, Path: "new.md"},
				{Effect: plan.EffectOverwrite, Path: "old.md"},
				{Effect: plan.EffectSkip, Path: "same.md"},
			},
		}, []string{"/dst/old.md"}, nil},
		{"copy of a single file touches Dst", plan.Copy{
			Src: "/src/f", Dst: "/dst/f", Files: []string{"."},
			Rows: []plan.Change{{Effect: plan.EffectOverwrite, Path: "."}},
		}, []string{"/dst/f"}, nil},
		{"remove touches its files", plan.Remove{Files: []string{"/a", "/b"}, Dirs: []string{"/d"}}, []string{"/a", "/b"}, nil},
		{"remove tree touches the whole path", plan.RemoveTree{Path: "/home/.tool"}, []string{"/home/.tool"}, nil},
		{"edit touches its path", plan.Edit{Path: "/home/.claude.json"}, []string{"/home/.claude.json"}, nil},
		{"chmod touches nothing", plan.Chmod{Paths: []string{"/a.sh"}}, nil, nil},
		{"exec touches nothing", plan.Exec{Argv: []string{"claude", "plugin", "update"}}, nil, nil},
		{"remove matching touches its globs", plan.RemoveMatching{Globs: []string{"/x/*.md"}}, nil, []string{"/x/*.md"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.step.Touches()
			if !slices.Equal(got.Paths, tt.wantPaths) || !slices.Equal(got.Globs, tt.wantGlobs) {
				t.Fatalf("Touches = %+v, want paths %v globs %v", got, tt.wantPaths, tt.wantGlobs)
			}
		})
	}
}

func TestSnapshotChangesReturnsRows(t *testing.T) {
	rows := []plan.Change{{Effect: plan.EffectCreate, Path: ".claude/agents"}}
	if got := (plan.Snapshot{Rows: rows}).Changes(); !slices.Equal(got, rows) {
		t.Fatalf("Changes = %+v, want %+v", got, rows)
	}
}

func TestExecChanges(t *testing.T) {
	t.Run("nil rows give one run row from argv", func(t *testing.T) {
		got := plan.Exec{Argv: []string{"claude", "plugin", "update", "x"}}.Changes()
		if len(got) != 1 || got[0].Effect != plan.EffectRun || got[0].Path != "claude plugin update x" {
			t.Fatalf("Changes = %+v, want one EffectRun row \"claude plugin update x\"", got)
		}
	})

	t.Run("supplied rows are returned", func(t *testing.T) {
		rows := []plan.Change{{Effect: plan.EffectDelete, Path: "plugin x", Unrestorable: true}}
		got := plan.Exec{Argv: []string{"claude"}, Rows: rows}.Changes()
		if !slices.Equal(got, rows) {
			t.Fatalf("Changes = %+v, want %+v", got, rows)
		}
	})
}
