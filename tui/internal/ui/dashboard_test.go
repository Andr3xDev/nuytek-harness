package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/dashboard"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

func TestDashboardLines_MissingDepsAndLastBackup(t *testing.T) {
	th := newTheme(defaultDark)
	d := dashboard.Dashboard{
		MissingDeps: []manifest.MissingDep{{Tool: "jq", Dependents: []string{"hook-x"}}},
		LastBackup:  &backup.Info{Time: time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC), Label: "pre-install"},
	}
	got := strings.Join(dashboardLines(th, d, 80), "\n")
	for _, want := range []string{"jq", "hook-x", "pre-install"} {
		if !strings.Contains(got, want) {
			t.Errorf("view lacks %q", want)
		}
	}

	d.LastBackup = nil
	if got := strings.Join(dashboardLines(th, d, 80), "\n"); !strings.Contains(got, "no backups") {
		t.Error("view lacks \"no backups\" when LastBackup is nil")
	}
}
