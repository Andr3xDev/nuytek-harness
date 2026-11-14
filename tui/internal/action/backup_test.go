package action_test

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func TestPlanBackup(t *testing.T) {
	shared := func(name string) manifest.Component {
		return manifest.Component{Name: name, Kind: manifest.KindFile, Source: "claude/settings.json", Target: ".claude/settings.json"}
	}
	tests := []struct {
		name       string
		components []manifest.Component
		home       map[string]string // relative to HOME
		wantPaths  []string          // relative to HOME, sorted
	}{
		{
			name:       "only existing component paths are backed up",
			components: []manifest.Component{component("agents", manifest.KindDir), component("skills", manifest.KindDir)},
			home:       map[string]string{".claude/agents/a.md": "A", ".claude/other/x": "x"},
			wantPaths:  []string{".claude/agents"},
		},
		{
			name:       "components sharing a destination are listed once",
			components: []manifest.Component{shared("settings"), shared("plugins")},
			home:       map[string]string{".claude/settings.json": "{}"},
			wantPaths:  []string{".claude/settings.json"},
		},
		{
			name:       "special-kind component is included",
			components: []manifest.Component{{Name: "mcp", Kind: "mcp", Target: ".claude.json"}},
			home:       map[string]string{".claude.json": "{}"},
			wantPaths:  []string{".claude.json"},
		},
		{
			name:       "component without a target is skipped",
			components: []manifest.Component{{Name: "specter", Kind: "specter"}, component("agents", manifest.KindDir)},
			home:       map[string]string{".claude/agents/a.md": "A"},
			wantPaths:  []string{".claude/agents"},
		},
		{
			name: "path nested below another backed-up path is dropped",
			components: []manifest.Component{
				component("agents", manifest.KindDir),
				{Name: "one", Kind: manifest.KindFile, Source: "claude/a.md", Target: ".claude/agents/a.md"},
			},
			home:      map[string]string{".claude/agents/a.md": "A"},
			wantPaths: []string{".claude/agents"},
		},
		{
			name:       "nothing installed is an empty plan",
			components: []manifest.Component{component("agents", manifest.KindDir)},
			home:       map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			for rel, content := range tt.home {
				f.writeHome(rel, content, 0o644)
			}

			p, err := action.PlanBackup(f.env(), f.target(tt.components...))

			if err != nil {
				t.Fatal(err)
			}
			if p.Title != "Backup" || p.Confirm != "" {
				t.Errorf("title = %q confirm = %q, want Backup and no confirmation", p.Title, p.Confirm)
			}
			if len(tt.wantPaths) == 0 {
				if !p.Empty() {
					t.Fatalf("plan not empty: %v", p.Steps)
				}
				return
			}
			snap := onlyStep[plan.Snapshot](t, p)
			want := make([]string, len(tt.wantPaths))
			for i, rel := range tt.wantPaths {
				want[i] = filepath.Join(f.home, rel)
			}
			if got := snapshotPaths(p); !slices.Equal(got, want) {
				t.Errorf("snapshot paths = %v, want %v", got, want)
			}
			if snap.Store.Kind != backup.StoreHome {
				t.Errorf("store kind = %v, want home", snap.Store.Kind)
			}
			rowPaths := make([]string, len(snap.Rows))
			for i, row := range snap.Rows {
				rowPaths[i] = row.Path
			}
			if !slices.Equal(rowPaths, snap.Paths) {
				t.Errorf("row paths = %v, want snapshot paths %v", rowPaths, snap.Paths)
			}
		})
	}
}
