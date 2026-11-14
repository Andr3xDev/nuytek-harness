package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
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
		Spec: target.Spec{ID: "t", Name: "T", BackupDirs: func(home, repo string) []backup.Dir {
			return []backup.Dir{{Path: filepath.Join(home, ".bk"), Label: "home", Kind: backup.StoreHome, Base: home}}
		}},
	}
	svc := NewRoot([]target.Target{tg}, Deps{Env: check.Env{FS: osfs.FS{}, Home: home, Repo: t.TempDir()}}, nil).svc
	b := newBackupsTab(svc, tg)
	b.loaded = true
	return b
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
