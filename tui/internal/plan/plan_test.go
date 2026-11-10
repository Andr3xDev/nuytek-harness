package plan_test

import (
	"slices"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

var store = &backup.Dir{Path: "/home/.bk", Label: "home"}

func overwriteCopy(dst string, files ...string) plan.Copy {
	rows := make([]plan.Change, 0, len(files))
	for _, f := range files {
		rows = append(rows, plan.Change{Effect: plan.EffectOverwrite, Path: f})
	}
	return plan.Copy{Src: "/src", Dst: dst, Files: files, Rows: rows}
}

func snapshotOf(t *testing.T, p plan.Plan) plan.Snapshot {
	t.Helper()
	if len(p.Steps) == 0 {
		t.Fatal("plan has no steps, want a leading Snapshot")
	}
	s, ok := p.Steps[0].(plan.Snapshot)
	if !ok {
		t.Fatalf("Steps[0] is %T, want plan.Snapshot", p.Steps[0])
	}
	return s
}

func TestNewSnapshotFirst(t *testing.T) {
	steps := []plan.Step{
		overwriteCopy("/home/.tool", "a.md"),
		plan.Edit{Path: "/home/.claude.json"},
		plan.Remove{Files: []string{"/home/x"}},
	}
	p := plan.New("t", store, steps)
	snap := snapshotOf(t, p)
	for _, s := range steps {
		for _, path := range s.Touches().Paths {
			if !slices.Contains(snap.Paths, path) {
				t.Errorf("snapshot paths %v miss %s", snap.Paths, path)
			}
		}
	}
	if snap.Store != *store {
		t.Errorf("snapshot store = %+v, want %+v", snap.Store, *store)
	}
}

func TestNewSnapshotContent(t *testing.T) {
	tests := []struct {
		name      string
		store     *backup.Dir
		steps     []plan.Step
		wantPaths []string
		wantNone  bool
	}{
		{"overwritten file only, never a custom file", store, []plan.Step{
			plan.Copy{Src: "/s", Dst: "/d", Files: []string{"new.md", "old.md"}, Rows: []plan.Change{
				{Effect: plan.EffectCreate, Path: "new.md"}, {Effect: plan.EffectOverwrite, Path: "old.md"},
			}},
		}, []string{"/d/old.md"}, false},
		{"union of steps gives one snapshot", store, []plan.Step{
			overwriteCopy("/d", "a.md"), plan.Edit{Path: "/h/.claude.json"},
		}, []string{"/d/a.md", "/h/.claude.json"}, false},
		{"ancestor covers copy rows below it", store, []plan.Step{
			plan.RemoveTree{Path: "/d"}, overwriteCopy("/d", "a.md", "sub/b.md"),
		}, []string{"/d"}, false},
		{"ancestor match respects path boundary", store, []plan.Step{
			plan.RemoveTree{Path: "/d"}, plan.Edit{Path: "/d-x"}, plan.Edit{Path: "/d/a"},
		}, []string{"/d", "/d-x"}, false},
		{"empty union gives no snapshot", store, []plan.Step{
			plan.Copy{Src: "/s", Dst: "/d", Files: []string{"n.md"}, Rows: []plan.Change{{Effect: plan.EffectCreate, Path: "n.md"}}},
		}, nil, true},
		{"nil store gives no snapshot", nil, []plan.Step{plan.Edit{Path: "/h/.claude.json"}}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := plan.New("t", tt.store, tt.steps)
			if tt.wantNone {
				if len(p.Steps) != len(tt.steps) {
					t.Fatalf("got %d steps, want %d with no Snapshot", len(p.Steps), len(tt.steps))
				}
				if _, ok := p.Steps[0].(plan.Snapshot); ok {
					t.Fatal("Steps[0] is a Snapshot, want none")
				}
				return
			}
			snapshots := 0
			for _, s := range p.Steps {
				if _, ok := s.(plan.Snapshot); ok {
					snapshots++
				}
			}
			if snapshots != 1 {
				t.Fatalf("got %d Snapshot steps, want 1", snapshots)
			}
			got := slices.Sorted(slices.Values(snapshotOf(t, p).Paths))
			if !slices.Equal(got, tt.wantPaths) {
				t.Fatalf("snapshot paths = %v, want %v", got, tt.wantPaths)
			}
		})
	}
}

func TestPlanEmpty(t *testing.T) {
	tests := []struct {
		name  string
		steps []plan.Step
		want  bool
	}{
		{"skip rows only", []plan.Step{plan.Exec{Argv: []string{"x"}, Rows: []plan.Change{{Effect: plan.EffectSkip}}}}, true},
		{"one create row", []plan.Step{plan.Copy{Rows: []plan.Change{{Effect: plan.EffectCreate, Path: "a"}}}}, false},
		{"no rows", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := plan.New("t", nil, tt.steps).Empty(); got != tt.want {
				t.Fatalf("Empty = %v, want %v", got, tt.want)
			}
		})
	}
}
