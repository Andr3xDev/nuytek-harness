package app_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/adapters/fsx"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/app"
)

func mkBackups(t *testing.T, root string, names ...string) {
	t.Helper()
	for _, n := range names {
		if err := os.MkdirAll(filepath.Join(root, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLastBackup(t *testing.T) {
	t.Run("newest across two dirs keeps its label", func(t *testing.T) {
		a, b := t.TempDir(), t.TempDir()
		mkBackups(t, a, "20240101-000000", "20240301-120000")
		mkBackups(t, b, "20240201-000000")

		got := app.LastBackup(fsx.FS{}, []app.BackupDir{{Path: a, Label: "claude"}, {Path: b, Label: "repo"}})

		want := time.Date(2024, 3, 1, 12, 0, 0, 0, time.Local)
		if got == nil || got.Label != "claude" || !got.Time.Equal(want) {
			t.Fatalf("got %+v, want label claude at %v", got, want)
		}
	})

	t.Run("no dirs", func(t *testing.T) {
		if got := app.LastBackup(fsx.FS{}, nil); got != nil {
			t.Fatalf("got %+v, want nil", got)
		}
	})

	t.Run("empty dir", func(t *testing.T) {
		if got := app.LastBackup(fsx.FS{}, []app.BackupDir{{Path: t.TempDir(), Label: "claude"}}); got != nil {
			t.Fatalf("got %+v, want nil", got)
		}
	})

	t.Run("missing dir is skipped", func(t *testing.T) {
		b := t.TempDir()
		mkBackups(t, b, "20240201-000000")

		got := app.LastBackup(fsx.FS{}, []app.BackupDir{{Path: filepath.Join(t.TempDir(), "absent"), Label: "claude"}, {Path: b, Label: "repo"}})

		if got == nil || got.Label != "repo" {
			t.Fatalf("got %+v, want label repo", got)
		}
	})

	t.Run("unstamped name falls back to mtime", func(t *testing.T) {
		dir := t.TempDir()
		mkBackups(t, dir, "20240101-000000", "manual-copy")
		mtime := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
		if err := os.Chtimes(filepath.Join(dir, "manual-copy"), mtime, mtime); err != nil {
			t.Fatal(err)
		}

		got := app.LastBackup(fsx.FS{}, []app.BackupDir{{Path: dir, Label: "claude"}})

		if got == nil || !got.Time.Equal(mtime) {
			t.Fatalf("got %+v, want time %v", got, mtime)
		}
	})

	t.Run("non-directory entries are ignored", func(t *testing.T) {
		dir := t.TempDir()
		mkBackups(t, dir, "20240101-000000")
		if err := os.WriteFile(filepath.Join(dir, "20250101-000000"), nil, 0o644); err != nil {
			t.Fatal(err)
		}

		got := app.LastBackup(fsx.FS{}, []app.BackupDir{{Path: dir, Label: "claude"}})

		want := time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local)
		if got == nil || !got.Time.Equal(want) {
			t.Fatalf("got %+v, want time %v", got, want)
		}
	})
}
