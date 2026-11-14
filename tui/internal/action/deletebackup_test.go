package action_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func TestPlanDeleteBackup(t *testing.T) {
	const stamp = "20240101-000000"
	tests := []struct {
		name    string
		setup   func(f *fixture) backup.Entry
		wantErr bool
	}{
		{
			name: "ok",
			setup: func(f *fixture) backup.Entry {
				f.write(filepath.Join(f.homeStore(), stamp), ".claude/a.md", strings.Repeat("x", 2048), 0o644)
				return backup.Entry{Stamp: stamp, Path: filepath.Join(f.homeStore(), stamp)}
			},
		},
		{
			name: "symlinked entry",
			setup: func(f *fixture) backup.Entry {
				f.mkdir(f.home, "real")
				f.symlink(f.homeStore(), stamp, filepath.Join(f.home, "real"))
				return backup.Entry{Stamp: stamp, Path: filepath.Join(f.homeStore(), stamp)}
			},
			wantErr: true,
		},
		{
			name: "symlinked store root",
			setup: func(f *fixture) backup.Entry {
				f.mkdir(f.home, "real/"+stamp)
				f.symlink(f.home, homeStoreDir, filepath.Join(f.home, "real"))
				return backup.Entry{Stamp: stamp, Path: filepath.Join(f.homeStore(), stamp)}
			},
			wantErr: true,
		},
		{
			name: "foreign directory outside the stores",
			setup: func(f *fixture) backup.Entry {
				f.mkdir(f.home, "foreign/"+stamp)
				return backup.Entry{Stamp: stamp, Path: filepath.Join(f.home, "foreign", stamp)}
			},
			wantErr: true,
		},
		{
			name: "dot-dot stamp",
			setup: func(f *fixture) backup.Entry {
				f.mkdir(f.homeStore(), stamp)
				return backup.Entry{Stamp: "..", Path: filepath.Join(f.homeStore(), "..")}
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			entry := tt.setup(f)

			p, err := action.PlanDeleteBackup(f.env(), f.target(), entry)

			if tt.wantErr {
				if err == nil || len(p.Steps) != 0 {
					t.Fatalf("plan %v err %v, want a refusal with no steps", p.Steps, err)
				}
				return
			}
			requireNoError(t, err)
			rm := onlyStep[plan.RemoveTree](t, p)
			if len(p.Steps) != 1 {
				t.Errorf("steps = %v, want only the RemoveTree (no Snapshot)", p.Steps)
			}
			if rm.Path != entry.Path || len(rm.Rows) != 1 || rm.Rows[0].Effect != plan.EffectDelete || rm.Rows[0].Path != entry.Path {
				t.Errorf("RemoveTree = %+v, want one delete row for %s", rm, entry.Path)
			}
			if !reflect.DeepEqual(p.Rows(), rm.Rows) {
				t.Errorf("plan rows %v differ from the supplied rows %v", p.Rows(), rm.Rows)
			}
			if !strings.Contains(p.Confirm, stamp) || !strings.Contains(p.Confirm, strconv.FormatInt(backup.Size(f.env().FS, entry), 10)) {
				t.Errorf("Confirm %q must name the stamp and the size as a plain byte count (backup.Size)", p.Confirm)
			}
			if _, err := os.Lstat(entry.Path); err != nil {
				t.Errorf("planning must not delete anything: %v", err)
			}
		})
	}
}

func homeStoreEntries(f *fixture, stamps ...string) []backup.Entry {
	var entries []backup.Entry
	for _, stamp := range stamps {
		f.write(filepath.Join(f.homeStore(), stamp), ".claude/a.md", "x", 0o644)
		entries = append(entries, backup.Entry{Stamp: stamp, Path: filepath.Join(f.homeStore(), stamp)})
	}
	return entries
}

func TestPlanDeleteBackup_ConfirmCapsTheNamedStamps(t *testing.T) {
	stamps := []string{"20240101-000000", "20240102-000000", "20240103-000000", "20240104-000000", "20240105-000000"}
	f := newFixture(t)
	entries := homeStoreEntries(f, stamps...)

	p, err := action.PlanDeleteBackup(f.env(), f.target(), entries...)

	requireNoError(t, err)
	if len(p.Steps) != len(stamps) {
		t.Errorf("plan steps %v, want one RemoveTree per entry", p.Steps)
	}
	if !strings.Contains(p.Confirm, "Delete 5 backups") || !strings.Contains(p.Confirm, "and 2 more") ||
		!strings.Contains(p.Confirm, stamps[2]) || strings.Contains(p.Confirm, stamps[3]) {
		t.Errorf("Confirm %q, want the count, the first 3 stamps and \"and 2 more\"", p.Confirm)
	}

	entries[1].Stamp = ".."
	if p, err := action.PlanDeleteBackup(f.env(), f.target(), entries...); err == nil || len(p.Steps) != 0 {
		t.Errorf("plan %v err %v, want one invalid entry to refuse the whole plan", p.Steps, err)
	}
}
