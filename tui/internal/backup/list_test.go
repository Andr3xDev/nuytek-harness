package backup_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
)

func TestList(t *testing.T) {
	mtime := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		setup     func(t *testing.T, home, repo string)
		wantOrder []string // stamps, newest first
		wantLink  string
	}{
		{
			name: "two stores newest first",
			setup: func(t *testing.T, home, repo string) {
				mkBackups(t, home, "20240101-000000", "20240301-120000")
				mkBackups(t, repo, "20240201T000000Z")
			},
			wantOrder: []string{"20240301-120000", "20240201T000000Z", "20240101-000000"},
		},
		{
			name: "collision suffix keeps the stamp whole",
			setup: func(t *testing.T, home, repo string) {
				mkBackups(t, home, "20240101-000000", "20240301-120000-1")
			},
			wantOrder: []string{"20240301-120000-1", "20240101-000000"},
		},
		{
			name: "unparseable name sorts by modification time",
			setup: func(t *testing.T, home, repo string) {
				mkBackups(t, home, "20240101-000000", "manual-copy")
				if err := os.Chtimes(filepath.Join(home, "manual-copy"), mtime, mtime); err != nil {
					t.Fatal(err)
				}
			},
			wantOrder: []string{"manual-copy", "20240101-000000"},
		},
		{
			name: "symlinked store root is skipped",
			setup: func(t *testing.T, home, repo string) {
				real := t.TempDir()
				mkBackups(t, real, "20240101-000000")
				if err := os.Remove(repo); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(real, repo); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				mkBackups(t, home, "20240201-000000")
			},
			wantOrder: []string{"20240201-000000"},
		},
		{
			name: "symlinked entry is listed with IsLink",
			setup: func(t *testing.T, home, repo string) {
				real := t.TempDir()
				if err := os.Symlink(real, filepath.Join(home, "20240101-000000")); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			},
			wantOrder: []string{"20240101-000000"},
			wantLink:  "20240101-000000",
		},
		{
			name:      "no snapshots",
			setup:     func(t *testing.T, home, repo string) {},
			wantOrder: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home, repo := t.TempDir(), t.TempDir()
			tt.setup(t, home, repo)
			dirs := []backup.Dir{
				{Path: home, Label: "home", Kind: backup.StoreHome, Base: "/h"},
				{Path: repo, Label: "repo", Kind: backup.StoreRepo, Base: "/r"},
			}

			got := backup.List(osfs.FS{}, dirs)

			stamps := make([]string, 0, len(got))
			for _, e := range got {
				stamps = append(stamps, e.Stamp)
				if e.IsLink != (e.Stamp == tt.wantLink) {
					t.Errorf("%s IsLink = %v", e.Stamp, e.IsLink)
				}
			}
			if !slices.Equal(stamps, tt.wantOrder) {
				t.Fatalf("stamps = %v, want %v", stamps, tt.wantOrder)
			}
		})
	}

	t.Run("entry carries its store", func(t *testing.T) {
		home, repo := t.TempDir(), t.TempDir()
		mkBackups(t, repo, "20240201T000000Z")

		got := backup.List(osfs.FS{}, []backup.Dir{
			{Path: home, Label: "home", Kind: backup.StoreHome, Base: "/h"},
			{Path: repo, Label: "repo", Kind: backup.StoreRepo, Base: "/r"},
		})

		want := backup.Entry{
			Stamp: "20240201T000000Z", Time: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			Label: "repo", Kind: backup.StoreRepo, Base: "/r", Path: filepath.Join(repo, "20240201T000000Z"),
		}
		if len(got) != 1 || got[0].Stamp != want.Stamp || !got[0].Time.Equal(want.Time) || got[0].Label != want.Label ||
			got[0].Kind != want.Kind || got[0].Base != want.Base || got[0].Path != want.Path {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})
}

func TestSizeAndFilesDoNotFollowSymlinkedDirectory(t *testing.T) {
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "big.md"), []byte("0123456789"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(t.TempDir(), "20240101-000000")
	mkBackups(t, snapshot, "real")
	if err := os.WriteFile(filepath.Join(snapshot, "real", "a.md"), []byte("123"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(snapshot, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	e := backup.Entry{Stamp: "20240101-000000", Path: snapshot}

	if got := backup.Size(osfs.FS{}, e); got != 3 {
		t.Errorf("Size = %d, want 3 (the link target must not count)", got)
	}
	files := backup.Files(osfs.FS{}, e)
	slices.Sort(files)
	if want := []string{"link", "real/a.md"}; !slices.Equal(files, want) {
		t.Errorf("Files = %v, want %v", files, want)
	}
}

func TestSizeAndFiles(t *testing.T) {
	dir := t.TempDir()
	snapshot := filepath.Join(dir, "20240101-000000")
	for rel, content := range map[string]string{".claude/CLAUDE.md": "12345", ".claude.json": "123"} {
		path := filepath.Join(snapshot, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e := backup.Entry{Stamp: "20240101-000000", Path: snapshot}

	if got := backup.Size(osfs.FS{}, e); got != 8 {
		t.Errorf("Size = %d, want the 8 content bytes", got)
	}
	files := backup.Files(osfs.FS{}, e)
	slices.Sort(files)
	if want := []string{".claude.json", ".claude/CLAUDE.md"}; !slices.Equal(files, want) {
		t.Errorf("Files = %v, want %v", files, want)
	}
}
