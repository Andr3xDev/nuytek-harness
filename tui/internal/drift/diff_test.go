package drift_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(f fixture)
		filter       drift.Filter
		blocked      drift.Blocked
		wantRows     map[string]plan.Effect // path -> effect, exact set
		wantNotes    map[string]string
		wantConflict string // suffix of ErrPathConflict.Path; empty = no error
	}{
		{
			name: "target-only file yields no row",
			setup: func(f fixture) {
				f.writeSource("s/a.md", "a", 0o644)
				f.writeTarget("d/a.md", "a", 0o644)
				f.writeTarget("d/custom.md", "mine", 0o644)
			},
			wantRows: map[string]plan.Effect{},
		},
		{
			name: "missing is create and different is overwrite",
			setup: func(f fixture) {
				f.writeSource("s/new.md", "n", 0o644)
				f.writeSource("s/sub/old.md", "new", 0o644)
				f.writeSource("s/same.md", "s", 0o644)
				f.writeTarget("d/sub/old.md", "old", 0o644)
				f.writeTarget("d/same.md", "s", 0o644)
			},
			wantRows: map[string]plan.Effect{"new.md": plan.EffectCreate, "sub/old.md": plan.EffectOverwrite},
		},
		{
			name: "link in source is a link change",
			setup: func(f fixture) {
				f.writeSource("s/a.md", "a", 0o644)
				f.symlink(f.repo, "s/l", "a.md")
			},
			wantRows:  map[string]plan.Effect{"a.md": plan.EffectCreate, "l": plan.EffectCreate},
			wantNotes: map[string]string{"l": "link"},
		},
		{
			name: "filter is respected",
			setup: func(f fixture) {
				f.writeSource("s/a.md", "a", 0o644)
				f.writeSource("s/__pycache__/x.pyc", "x", 0o644)
				f.writeSource("s/b.pyc", "x", 0o644)
			},
			filter:   drift.InstallFilter,
			wantRows: map[string]plan.Effect{"a.md": plan.EffectCreate},
		},
		{
			name: "file where a directory is expected",
			setup: func(f fixture) {
				f.writeSource("s/sub/a.md", "a", 0o644)
				f.writeTarget("d/sub", "i am a file", 0o644)
			},
			wantConflict: "d/sub",
		},
		{
			name: "directory where a file is expected",
			setup: func(f fixture) {
				f.writeSource("s/a.md", "a", 0o644)
				f.writeTarget("d/a.md/inner", "x", 0o644)
			},
			wantConflict: "d/a.md",
		},
		{
			name: "symlink where a directory is expected",
			setup: func(f fixture) {
				f.writeSource("s/sub/a.md", "a", 0o644)
				f.writeTarget("elsewhere/keep", "k", 0o644)
				f.symlink(f.home, "d/sub", filepath.Join(f.home, "elsewhere"))
			},
			wantConflict: "d/sub",
		},
		{
			name: "symlinked root",
			setup: func(f fixture) {
				f.writeSource("s/a.md", "a", 0o644)
				f.writeTarget("elsewhere/keep", "k", 0o644)
				f.symlink(f.home, "d", filepath.Join(f.home, "elsewhere"))
			},
			wantConflict: "d",
		},
		{
			name: "replace policy turns a blocker below the root into an overwrite",
			setup: func(f fixture) {
				f.writeSource("s/sub/a.md", "a", 0o644)
				f.writeTarget("d/sub", "i am a file", 0o644)
			},
			blocked:   drift.BlockedReplace,
			wantRows:  map[string]plan.Effect{"sub": plan.EffectOverwrite, "sub/a.md": plan.EffectCreate},
			wantNotes: map[string]string{"sub": "replaces file"},
		},
		{
			name: "replace policy still refuses a symlinked root",
			setup: func(f fixture) {
				f.writeSource("s/a.md", "a", 0o644)
				f.writeTarget("elsewhere/keep", "k", 0o644)
				f.symlink(f.home, "d", filepath.Join(f.home, "elsewhere"))
			},
			blocked:      drift.BlockedReplace,
			wantConflict: "d",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			rows, err := drift.Diff(osfs.FS{}, filepath.Join(f.repo, "s"), filepath.Join(f.home, "d"), tt.filter, tt.blocked)

			if tt.wantConflict != "" {
				var conflict drift.ErrPathConflict
				if !errors.As(err, &conflict) || !strings.HasSuffix(conflict.Path, tt.wantConflict) {
					t.Fatalf("err = %v, want ErrPathConflict ending in %q", err, tt.wantConflict)
				}
				return
			}
			if err != nil {
				t.Fatalf("Diff error: %v", err)
			}
			got := map[string]plan.Effect{}
			for _, r := range rows {
				got[r.Path] = r.Effect
				if want, ok := tt.wantNotes[r.Path]; ok && r.Note != want {
					t.Errorf("row %s note = %q, want %q", r.Path, r.Note, want)
				}
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
