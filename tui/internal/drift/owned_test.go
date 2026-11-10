package drift_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
)

func TestOwnedFiles(t *testing.T) {
	f := newFixture(t)
	f.writeSource("s/a.md", "a", 0o644)
	f.writeSource("s/sub/b.md", "b", 0o644)
	f.writeSource("s/sub/c.pyc", "x", 0o644)
	f.writeSource("s/__pycache__/x.pyc", "x", 0o644)
	f.writeSource("s/emptydir/.keep", "", 0o644)
	f.symlink(f.repo, "s/l", "a.md")

	got, err := drift.OwnedFiles(osfs.FS{}, filepath.Join(f.repo, "s"), drift.InstallFilter)

	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	want := []string{"a.md", "emptydir", "emptydir/.keep", "l", "sub", "sub/b.md"}
	if !slices.Equal(got, want) {
		t.Fatalf("OwnedFiles = %v, want %v", got, want)
	}
}

func TestOwnedFilesAndExtrasPropagateReadErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	f := newFixture(t)
	f.writeSource("s/sub/a.md", "a", 0o644)
	f.writeTarget("d/sub/a.md", "a", 0o644)
	for _, locked := range []string{filepath.Join(f.repo, "s/sub"), filepath.Join(f.home, "d/sub")} {
		if err := os.Chmod(locked, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	}
	src, dst := filepath.Join(f.repo, "s"), filepath.Join(f.home, "d")

	if _, err := drift.OwnedFiles(osfs.FS{}, src, drift.InstallFilter); err == nil {
		t.Error("OwnedFiles error = nil, want the ReadDir error")
	}
	if _, err := drift.Extras(osfs.FS{}, src, dst, drift.InstallFilter); err == nil {
		t.Error("Extras error = nil, want the ReadDir error")
	}
}

func TestExtras(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f fixture)
		want  []string
	}{
		{
			name: "custom file, custom empty directory, bytecode and an unfollowed link",
			setup: func(f fixture) {
				f.writeSource("s/a.md", "a", 0o644)
				f.writeSource("s/sub/b.md", "b", 0o644)
				f.writeTarget("d/a.md", "a", 0o644)
				f.writeTarget("d/sub/b.md", "b", 0o644)
				f.writeTarget("d/custom.md", "mine", 0o644)
				f.writeTarget("d/stale.pyc", "x", 0o644)
				if err := os.MkdirAll(filepath.Join(f.home, "d/customdir"), 0o755); err != nil {
					t.Fatal(err)
				}
				f.writeTarget("elsewhere/inner.md", "i", 0o644)
				f.symlink(f.home, "d/link", filepath.Join(f.home, "elsewhere"))
			},
			want: []string{"custom.md", "customdir", "link", "stale.pyc"},
		},
		{
			name: "identical tree",
			setup: func(f fixture) {
				f.writeSource("s/a.md", "a", 0o644)
				f.writeSource("s/sub/b.md", "b", 0o644)
				f.writeTarget("d/a.md", "a", 0o644)
				f.writeTarget("d/sub/b.md", "b", 0o644)
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			got, err := drift.Extras(osfs.FS{}, filepath.Join(f.repo, "s"), filepath.Join(f.home, "d"), drift.InstallFilter)

			if err != nil {
				t.Fatal(err)
			}
			slices.Sort(got)
			if len(got) != len(tt.want) || (len(got) > 0 && !slices.Equal(got, tt.want)) {
				t.Fatalf("Extras = %v, want %v", got, tt.want)
			}
		})
	}
}
