package ui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/config"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/state"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

// backupTab builds an empty-list Backups tab over a temp HOME holding the given component target.
func backupTab(t *testing.T, installed bool) *backupsTab {
	t.Helper()
	home := t.TempDir()
	if installed {
		if err := os.MkdirAll(filepath.Join(home, ".claude", "agents"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, ".claude", "agents", "a.md"), []byte("A"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tg := target.Target{
		Manifest: manifest.Manifest{Components: []manifest.Component{{Name: "agents", Kind: manifest.KindDir, Source: "claude/agents", Target: ".claude/agents"}}},
		Spec: target.Spec{ID: "t", Name: "T", BackupDirs: func(home, repo, _ string) []backup.Dir {
			return []backup.Dir{{Path: filepath.Join(home, ".bk"), Label: "home", Kind: backup.StoreHome, Base: home}}
		}},
	}
	svc := NewRoot([]target.Target{tg}, Deps{Env: check.Env{FS: osfs.FS{}, Home: home, Repo: t.TempDir()}}, nil).svc
	b := newBackupsTab(svc, tg)
	b.loaded = true
	return b
}

const (
	oldStamp   = "20260101-101010"
	newerStamp = "20260102-101010"
)

// removeTreeRunner deletes RemoveTree paths for real and reports every other step done.
type removeTreeRunner struct{}

func (removeTreeRunner) Run(_ context.Context, s plan.Step, _ func(string)) plan.StepResult {
	if rt, ok := s.(plan.RemoveTree); ok {
		if err := os.RemoveAll(rt.Path); err != nil {
			return plan.StepResult{Status: plan.StepFailed, Err: err}
		}
	}
	return plan.StepResult{Status: plan.StepDone}
}

// deleteRoot returns a Root focused on a loaded Backups tab holding two snapshots under a temp HOME.
func deleteRoot(t *testing.T) (*Root, *backupsTab, string) {
	t.Helper()
	b := backupTab(t, false)
	home := b.svc.Env.Home
	for _, stamp := range []string{oldStamp, newerStamp} {
		dir := filepath.Join(home, ".bk", stamp)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "f"), make([]byte, 2048), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tg := b.t
	tg.Spec.Checks = availableTarget().Spec.Checks
	r := NewRoot([]target.Target{tg}, Deps{Env: b.svc.Env, Runner: removeTreeRunner{}}, nil)
	r.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	for i, tb := range r.tabs[0] {
		if bt, ok := tb.(*backupsTab); ok {
			r.tab, r.focus = i, focusPanel
			pump(t, r, bt.load())
			return r, bt, home
		}
	}
	t.Fatal("no Backups tab")
	return nil, nil, ""
}

// pump runs cmd and every command it yields until none is left, feeding results to the Root.
func pump(t *testing.T, r *Root, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case nil:
	case tea.BatchMsg:
		for _, c := range msg {
			pump(t, r, c)
		}
	default:
		_, next := r.Update(msg)
		pump(t, r, next)
	}
}

// pressDelete selects the newest snapshot, presses d and delivers the plan.
func pressDelete(t *testing.T, r *Root) {
	t.Helper()
	pump(t, r, press(r, textKey("d")))
}

func TestBackupsTab_DeleteKeyShowsTheConfirmationAtOnce(t *testing.T) {
	r, b, _ := deleteRoot(t)

	pressDelete(t, r)

	if b.flow == nil || b.flow.stage != stageConfirm {
		t.Fatalf("flow = %+v, want the confirm stage with no preview", b.flow)
	}
	if view := r.render(); !strings.Contains(view, newerStamp) {
		t.Fatalf("confirmation does not name the snapshot %s:\n%s", newerStamp, view)
	}
}

func TestBackupsTab_EnterAtDeleteConfirmationDoesNothing(t *testing.T) {
	r, b, home := deleteRoot(t)
	pressDelete(t, r)
	if b.flow == nil || b.flow.stage != stageConfirm {
		t.Fatalf("flow = %+v, want the confirm stage", b.flow)
	}

	pump(t, r, press(r, enterKey))

	if r.run != nil || b.flow == nil || b.flow.stage != stageConfirm {
		t.Fatalf("enter changed state: run=%v flow=%+v, want the confirmation untouched", r.run, b.flow)
	}
	if _, err := os.Stat(filepath.Join(home, ".bk", newerStamp)); err != nil {
		t.Fatalf("snapshot gone after enter: %v", err)
	}
}

func TestBackupsTab_EscAtDeleteConfirmationLeavesTheListUntouched(t *testing.T) {
	r, b, home := deleteRoot(t)
	pressDelete(t, r)
	if b.flow == nil || b.flow.stage != stageConfirm {
		t.Fatalf("flow = %+v, want the confirm stage", b.flow)
	}

	pump(t, r, press(r, escKey))

	if b.flow != nil || r.run != nil {
		t.Fatalf("flow = %+v, run = %v, want back on the list", b.flow, r.run)
	}
	if _, err := os.Stat(filepath.Join(home, ".bk", newerStamp)); err != nil {
		t.Fatalf("snapshot gone after esc: %v", err)
	}
}

func TestBackupsTab_DeleteThenYThenEscClosesTheRunAndReloadsTheList(t *testing.T) {
	r, b, home := deleteRoot(t)
	pressDelete(t, r)

	pump(t, r, press(r, textKey("y")))
	if r.run == nil || !r.run.done {
		t.Fatalf("run = %+v, want a finished run", r.run)
	}
	pump(t, r, press(r, escKey))

	if r.run != nil {
		t.Fatal("esc did not close the output pane")
	}
	if _, err := os.Stat(filepath.Join(home, ".bk", newerStamp)); !os.IsNotExist(err) {
		t.Fatalf("snapshot still exists: %v", err)
	}
	if len(b.rows) != 1 || b.rows[0].entry.Stamp != oldStamp {
		t.Fatalf("rows = %+v, want only %s after the reload", b.rows, oldStamp)
	}
}

func TestBackupsTab_MarkedBackupsAreDeletedTogether(t *testing.T) {
	r, b, home := deleteRoot(t)

	pump(t, r, press(r, textKey("a")))
	pump(t, r, press(r, textKey(" ")))
	if got := b.markedEntries(); len(got) != 1 || got[0].Stamp != oldStamp {
		t.Fatalf("marks = %+v, want only %s after a then space", got, oldStamp)
	}
	pump(t, r, press(r, textKey("a")))
	pressDelete(t, r)
	if b.flow == nil || b.flow.stage != stageConfirm || !strings.Contains(b.flow.p.Confirm, oldStamp) || !strings.Contains(b.flow.p.Confirm, newerStamp) {
		t.Fatalf("flow = %+v, want a confirmation naming both backups", b.flow)
	}
	pump(t, r, press(r, textKey("y")))
	pump(t, r, press(r, escKey))

	for _, stamp := range []string{oldStamp, newerStamp} {
		if _, err := os.Stat(filepath.Join(home, ".bk", stamp)); !os.IsNotExist(err) {
			t.Errorf("%s still exists: %v", stamp, err)
		}
	}
	if len(b.marks) != 0 {
		t.Errorf("marks = %v, want them cleared after the run", b.marks)
	}
}

func TestBackupsTab_OnlyTheMarkedBackupIsDeleted(t *testing.T) {
	r, b, home := deleteRoot(t)

	pump(t, r, press(r, downKey))
	pump(t, r, press(r, textKey(" ")))
	pump(t, r, press(r, upKey))
	pressDelete(t, r)
	pump(t, r, press(r, textKey("y")))
	pump(t, r, press(r, escKey))

	if _, err := os.Stat(filepath.Join(home, ".bk", oldStamp)); !os.IsNotExist(err) {
		t.Errorf("marked %s still exists: %v", oldStamp, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".bk", newerStamp)); err != nil {
		t.Errorf("unmarked %s was deleted: %v", newerStamp, err)
	}
	if len(b.rows) != 1 {
		t.Errorf("rows = %+v, want one left", b.rows)
	}
}

func TestBackupsTab_ListsFromTheConfiguredPathWithItsLabel(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "bk", newerStamp), 0o755); err != nil {
		t.Fatal(err)
	}
	tg := target.Target{Spec: target.Spec{
		ID: "t", Name: "T", DefaultBackupsRel: ".bk",
		BackupDirs: func(home, _, homeStore string) []backup.Dir {
			return []backup.Dir{{Path: homeStore, Label: config.PathLabel(home, homeStore), Kind: backup.StoreHome, Base: home}}
		},
	}}
	env := check.Env{FS: osfs.FS{}, Home: home, Repo: t.TempDir()}
	env.Saved = state.State{Targets: map[string]state.TargetState{"t": {BackupsDir: filepath.Join(home, "bk")}}}
	b := newBackupsTab(NewRoot([]target.Target{tg}, Deps{Env: env}, nil).svc, tg)

	b.Update(b.Activate(tg)())

	if view := b.View(80, 20); !strings.Contains(view, "~/bk") || !strings.Contains(view, newerStamp) {
		t.Fatalf("view = %q, want the snapshot from the configured dir labelled ~/bk", view)
	}
}

func TestBackupsTab_BackupKeyOnEmptyListAppliesWithoutY(t *testing.T) {
	b := backupTab(t, true)
	if !claims(b.Keys(), "b") {
		t.Fatal("Keys() lack the backup binding on an empty list")
	}

	cmd := b.handleKey(textKey("b"))
	if cmd == nil {
		t.Fatal("b on an empty list started no flow")
	}
	b.Update(cmd())
	if b.flow == nil || b.flow.stage != stagePreview {
		t.Fatalf("flow = %+v, want a preview", b.flow)
	}
	if !isRunRequest(b.handleKey(enterKey)) {
		t.Fatal("enter in preview must request the run: a backup needs no y")
	}
}

func TestBackupsTab_BackupKeyWithNothingInstalledShowsNothingToBackUp(t *testing.T) {
	b := backupTab(t, false)

	cmd := b.handleKey(textKey("b"))
	if cmd == nil {
		t.Fatal("b on an empty list started no flow")
	}
	b.Update(cmd())

	if view := b.View(80, 20); !strings.Contains(view, "Nothing to back up.") {
		t.Fatalf("view = %q, want it to say \"Nothing to back up.\"", view)
	}
}
