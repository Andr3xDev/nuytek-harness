package drift_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/state"
)

const rootList = "targets/rendered-files.txt"

func TestDiffWith_ComparesTransformedSource(t *testing.T) {
	listedOnly := func(rel string, data []byte) []byte {
		if rel != "listed.md" {
			return data
		}
		return []byte(strings.ReplaceAll(string(data), "@R@", "/home/r"))
	}
	tests := []struct {
		name     string
		setup    func(f fixture)
		wantRows map[string]plan.Effect
	}{
		{
			name: "rendered file of another length is up to date",
			setup: func(f fixture) {
				f.writeSource("s/listed.md", "root @R@", 0o644)
				f.writeTarget("d/listed.md", "root /home/r", 0o644)
			},
			wantRows: map[string]plan.Effect{},
		},
		{
			name: "root changed is an overwrite",
			setup: func(f fixture) {
				f.writeSource("s/listed.md", "root @R@", 0o644)
				f.writeTarget("d/listed.md", "root /home/q", 0o644)
			},
			wantRows: map[string]plan.Effect{"listed.md": plan.EffectOverwrite},
		},
		{
			name: "absent target is a create",
			setup: func(f fixture) {
				f.writeSource("s/listed.md", "root @R@", 0o644)
			},
			wantRows: map[string]plan.Effect{"listed.md": plan.EffectCreate},
		},
		{
			name: "unlisted file is compared as is",
			setup: func(f fixture) {
				f.writeSource("s/other.md", "root @R@", 0o644)
				f.writeTarget("d/other.md", "root /home/r", 0o644)
			},
			wantRows: map[string]plan.Effect{"other.md": plan.EffectOverwrite},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			rows, err := drift.DiffWith(osfs.FS{}, filepath.Join(f.repo, "s"), filepath.Join(f.home, "d"), drift.Filter{}, drift.BlockedFail, listedOnly)

			if err != nil {
				t.Fatal(err)
			}
			got := map[string]plan.Effect{}
			for _, r := range rows {
				got[r.Path] = r.Effect
			}
			if len(got) != len(tt.wantRows) {
				t.Fatalf("rows = %v, want %v", got, tt.wantRows)
			}
			for path, effect := range tt.wantRows {
				if got[path] != effect {
					t.Errorf("row %s effect = %q, want %q", path, got[path], effect)
				}
			}
		})
	}
}

func TestEvaluators_RenderListedFilesBeforeComparing(t *testing.T) {
	const target = ".claude/CLAUDE.md"
	tests := []struct {
		name      string
		saved     string
		deployed  string
		listed    string
		wantState drift.State
	}{
		{"default root, rendered copy of another length", "", "root ~/dev/specter\n", target, drift.StateInstalled},
		{"saved root, rendered copy", "~/work/specs", "root ~/work/specs\n", target, drift.StateInstalled},
		{"saved root differs from the deployed one", "~/work/specs", "root ~/dev/specter\n", target, drift.StateDrift},
		{"file not on the list keeps the byte comparison", "", "root ~/dev/specter\n", ".claude/other.md", drift.StateDrift},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.writeSource("claude/CLAUDE.md", "root {{SPECTER_ROOT}}\n", 0o644)
			f.writeSource(rootList, tt.listed+"\n", 0o644)
			f.writeTarget(target, tt.deployed, 0o644)
			saved := state.State{}
			if tt.saved != "" {
				saved.SpecterRoot = filepath.Join(f.home, strings.TrimPrefix(tt.saved, "~/"))
			}
			env := check.Env{FS: osfs.FS{}, Home: f.home, Repo: f.repo, Saved: saved}
			c := manifest.Component{Name: "claude-md", Kind: manifest.KindFile, Source: "claude/CLAUDE.md", Target: target}

			got := drift.Evaluators(nil)[manifest.KindFile](context.Background(), env, c)

			requireState(t, got, tt.wantState)
		})
	}
}
