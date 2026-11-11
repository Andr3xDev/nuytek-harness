package action_test

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func TestPlanRestore(t *testing.T) {
	const homeStamp, repoStamp = "20240101-000000", "20240101T000000Z"
	tests := []struct {
		name        string
		kind        backup.StoreKind
		snapshot    map[string]string // relative to the snapshot, mirrors paths under Base
		current     map[string]string // relative to Base
		wantFiles   []string          // Copy.Files, sorted
		wantTouched []string          // safety snapshot paths relative to Base, sorted
		wantConfirm []string
		wantEmpty   bool
	}{
		{
			name:        "home: changed file is overwritten and safety-snapshotted",
			kind:        backup.StoreHome,
			snapshot:    map[string]string{".claude/agents/a.md": "A"},
			current:     map[string]string{".claude/agents/a.md": "B"},
			wantFiles:   []string{".claude/agents/a.md"},
			wantTouched: []string{".claude/agents/a.md"},
			wantConfirm: []string{homeStamp, "1 overwrite", "0 create"},
		},
		{
			name:        "home: file added later is kept and not snapshotted, absent component untouched",
			kind:        backup.StoreHome,
			snapshot:    map[string]string{".claude/agents/a.md": "A"},
			current:     map[string]string{".claude/agents/a.md": "B", ".claude/agents/new.md": "n", ".claude/skills/s.md": "s", ".claude/sessions/x.jsonl": "x"},
			wantFiles:   []string{".claude/agents/a.md"},
			wantTouched: []string{".claude/agents/a.md"},
		},
		{
			name:        "home: .claude.json is restored whole",
			kind:        backup.StoreHome,
			snapshot:    map[string]string{".claude.json": "A"},
			current:     map[string]string{".claude.json": "B"},
			wantFiles:   []string{".claude.json"},
			wantTouched: []string{".claude.json"},
		},
		{
			name:        "home: missing files are created and counted",
			kind:        backup.StoreHome,
			snapshot:    map[string]string{".claude/agents/a.md": "A", ".claude/agents/b.md": "B"},
			current:     map[string]string{".claude/agents/a.md": "changed"},
			wantFiles:   []string{".claude/agents/a.md", ".claude/agents/b.md"},
			wantTouched: []string{".claude/agents/a.md"},
			wantConfirm: []string{homeStamp, "1 overwrite", "1 create"},
		},
		{
			name:      "home: identical snapshot is empty",
			kind:      backup.StoreHome,
			snapshot:  map[string]string{".claude/agents/a.md": "A"},
			current:   map[string]string{".claude/agents/a.md": "A"},
			wantEmpty: true,
		},
		{
			name:      "home: empty snapshot is empty",
			kind:      backup.StoreHome,
			snapshot:  map[string]string{},
			current:   map[string]string{".claude/agents/a.md": "A"},
			wantEmpty: true,
		},
		{
			name:        "repo: absent component is unchanged and the safety snapshot is in the repo store",
			kind:        backup.StoreRepo,
			snapshot:    map[string]string{"claude/CLAUDE.md": "A"},
			current:     map[string]string{"claude/CLAUDE.md": "B", "claude/sounds/ding.wav": "w"},
			wantFiles:   []string{"claude/CLAUDE.md"},
			wantTouched: []string{"claude/CLAUDE.md"},
			wantConfirm: []string{repoStamp},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			base, store, stamp := f.home, f.homeStore(), homeStamp
			if tt.kind == backup.StoreRepo {
				base, store, stamp = f.repo, f.repoStore(), repoStamp
			}
			snapshot := filepath.Join(store, stamp)
			f.mkdir(store, stamp)
			for rel, content := range tt.snapshot {
				f.write(snapshot, rel, content, 0o644)
			}
			for rel, content := range tt.current {
				f.write(base, rel, content, 0o644)
			}
			entry := backup.Entry{Stamp: stamp, Kind: tt.kind, Base: base, Path: snapshot, Label: "x"}

			p, err := action.PlanRestore(context.Background(), f.env(), f.target(component("agents", manifest.KindDir)), entry)

			requireNoError(t, err)
			if p.Empty() != tt.wantEmpty {
				t.Fatalf("Empty = %v, want %v (rows %v)", p.Empty(), tt.wantEmpty, p.Rows())
			}
			if tt.wantEmpty {
				return
			}
			cp := onlyStep[plan.Copy](t, p)
			files := slices.Clone(cp.Files)
			slices.Sort(files)
			if cp.Src != snapshot || cp.Dst != base || !slices.Equal(files, tt.wantFiles) {
				t.Errorf("copy %s -> %s files %v, want %s -> %s files %v", cp.Src, cp.Dst, files, snapshot, base, tt.wantFiles)
			}
			var wantTouched []string
			for _, rel := range tt.wantTouched {
				wantTouched = append(wantTouched, filepath.Join(base, rel))
			}
			if got := snapshotPaths(p); !slices.Equal(got, wantTouched) {
				t.Errorf("safety snapshot paths = %v, want %v", got, wantTouched)
			}
			if got := p.Steps[0].(plan.Snapshot).Store; got.Kind != tt.kind || got.Path != store {
				t.Errorf("safety store = %+v, want kind %v at %s", got, tt.kind, store)
			}
			if got := rowPaths(p, plan.EffectDelete); len(got) != 0 {
				t.Errorf("delete rows = %v, restore deletes nothing", got)
			}
			for _, want := range tt.wantConfirm {
				if !strings.Contains(p.Confirm, want) {
					t.Errorf("Confirm %q does not contain %q", p.Confirm, want)
				}
			}
			if p.Confirm == "" {
				t.Error("Confirm is empty, restore is destructive")
			}
		})
	}
}

func TestPlanRestoreRefusals(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(f *fixture, snapshot string)
		isLink  bool
		wantMsg string
	}{
		{"entry flagged as link", func(f *fixture, snapshot string) { f.mkdir(snapshot, "") }, true, "symlink"},
		{"snapshot path is a symlink", func(f *fixture, snapshot string) {
			f.writeHome("elsewhere/a.md", "A", 0o644)
			f.symlink(f.homeStore(), "20240101-000000", filepath.Join(f.home, "elsewhere"))
		}, false, "symlink"},
		{"a directory where the snapshot has a file", func(f *fixture, snapshot string) {
			f.write(snapshot, "a.md", "A", 0o644)
			f.writeHome("a.md/inner", "x", 0o644)
		}, false, "move"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			snapshot := filepath.Join(f.homeStore(), "20240101-000000")
			tt.setup(f, snapshot)
			entry := backup.Entry{Stamp: "20240101-000000", Kind: backup.StoreHome, Base: f.home, Path: snapshot, Label: "x", IsLink: tt.isLink}

			p, err := action.PlanRestore(context.Background(), f.env(), f.target(component("agents", manifest.KindDir)), entry)

			if err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
				t.Fatalf("err = %v, want it to contain %q", err, tt.wantMsg)
			}
			if len(p.Steps) != 0 {
				t.Errorf("steps = %v, want none", p.Steps)
			}
		})
	}
}
