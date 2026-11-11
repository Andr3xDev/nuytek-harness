package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/state"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/testutil"
)

const notExecuted = "not executed"

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWireIODryRunWritesNothing(t *testing.T) {
	home, repo, stateDir := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)

	writeTestFile(t, filepath.Join(home, "dst", "a.txt"), "old")
	writeTestFile(t, filepath.Join(repo, "src", "a.txt"), "new")
	marker := filepath.Join(home, "marker")

	store := state.NewStore(filepath.Join(stateDir, "state.toml"))
	runner, st := wireIO(true, store, time.Now)

	backups := backup.Dir{Path: filepath.Join(home, "backups"), Label: "test", Base: home}
	p := plan.Plan{Steps: []plan.Step{
		plan.Snapshot{Store: backups, Paths: []string{filepath.Join(home, "dst")}},
		plan.Copy{
			Src: filepath.Join(repo, "src"), Dst: filepath.Join(home, "dst"), Files: []string{"a.txt"},
			Rows: []plan.Change{{Effect: plan.EffectOverwrite, Path: "a.txt"}},
		},
		plan.Exec{Argv: []string{"sh", "-c", "touch " + marker}},
	}}

	homeBefore := testutil.TreeHash(t, home)
	repoBefore := testutil.TreeHash(t, repo)
	stateBefore := testutil.TreeHash(t, stateDir)

	var skipped []plan.Event
	result := plan.Apply(context.Background(), runner, p, func(e plan.Event) {
		if e.Phase == plan.PhaseSkipped {
			skipped = append(skipped, e)
		}
	})
	if result.Failed != nil {
		t.Fatalf("apply failed: %+v", result.Failed)
	}
	if len(skipped) != len(p.Steps) {
		t.Fatalf("skipped events = %d, want %d", len(skipped), len(p.Steps))
	}
	for _, e := range skipped {
		if e.Text != notExecuted {
			t.Errorf("step %q detail = %q, want %q", e.Step, e.Text, notExecuted)
		}
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("exec marker exists (err=%v): dry-run executed a step", err)
	}

	if err := st.Save(state.State{RepoPath: repo}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if got := testutil.TreeHash(t, home); got != homeBefore {
		t.Error("HOME changed during dry-run")
	}
	if got := testutil.TreeHash(t, repo); got != repoBefore {
		t.Error("repo changed during dry-run")
	}
	if got := testutil.TreeHash(t, stateDir); got != stateBefore {
		t.Error("state dir changed during dry-run")
	}
}

func TestWireIORealRunsStepsAndUsesFileStore(t *testing.T) {
	dir, stateDir := t.TempDir(), t.TempDir()
	marker := filepath.Join(dir, "marker")
	statePath := filepath.Join(stateDir, "state.toml")

	runner, st := wireIO(false, state.NewStore(statePath), time.Now)

	p := plan.Plan{Steps: []plan.Step{plan.Exec{Argv: []string{"sh", "-c", "touch " + marker}}}}
	if result := plan.Apply(context.Background(), runner, p, func(plan.Event) {}); result.Failed != nil {
		t.Fatalf("apply failed: %+v", result.Failed)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("exec marker missing: %v", err)
	}

	if err := st.Save(state.State{RepoPath: dir}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Errorf("state file not created: %v", err)
	}
}
