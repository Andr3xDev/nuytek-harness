package repo_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/repo"
)

func TestResolve_Precedence(t *testing.T) {
	flagRepo, envRepo, stateRepo, cwdRepo := mkRepo(t), mkRepo(t), mkRepo(t), mkRepo(t)
	deep := filepath.Join(cwdRepo, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name                  string
		flag, env, cwd, state string
		wantPath, source      string
	}{
		{"flag wins", flagRepo, envRepo, deep, stateRepo, flagRepo, repo.SourceFlag},
		{"env over state", "", envRepo, deep, stateRepo, envRepo, repo.SourceEnv},
		{"state over cwd", "", "", deep, stateRepo, stateRepo, repo.SourceState},
		{"walk up from cwd", "", "", deep, "", cwdRepo, repo.SourceCwd},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.Resolve(tt.flag, tt.env, tt.state, tt.cwd, testMarker, osfs.FS{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Path != tt.wantPath || got.Source != tt.source || got.Warning != "" {
				t.Fatalf("got %+v, want path %q source %q no warning", got, tt.wantPath, tt.source)
			}
		})
	}
}

func TestFindUp(t *testing.T) {
	root := mkRepo(t)
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, from, marker string
		want               string
		wantOK             bool
	}{
		{"hit at from", root, testMarker, root, true},
		{"hit in a parent", deep, testMarker, root, true},
		{"miss up to the filesystem root", deep, "no-such-marker-nuytek", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := repo.FindUp(osfs.FS{}, tt.from, tt.marker)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("FindUp = %q, %v, want %q, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestResolve_RelativeFlagAndEnvBecomeAbsolute(t *testing.T) {
	dir := mkRepo(t)
	t.Chdir(filepath.Dir(dir))
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(wd, filepath.Base(dir))

	for name, resolve := range map[string]func() (repo.Resolution, error){
		"flag": func() (repo.Resolution, error) {
			return repo.Resolve(filepath.Base(dir), "", "", wd, testMarker, osfs.FS{})
		},
		"env": func() (repo.Resolution, error) {
			return repo.Resolve("", filepath.Base(dir), "", wd, testMarker, osfs.FS{})
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := resolve()
			if err != nil || got.Path != want {
				t.Fatalf("got %+v, %v, want absolute path %q", got, err, want)
			}
		})
	}
}

func TestResolve_ExplicitSourceWithoutMarkerIsHardError(t *testing.T) {
	bare, dir := t.TempDir(), mkRepo(t)

	t.Run("flag", func(t *testing.T) {
		_, err := repo.Resolve(bare, "", "", dir, testMarker, osfs.FS{})
		if err == nil || !strings.Contains(err.Error(), "--repo") || !strings.Contains(err.Error(), bare) || !strings.Contains(err.Error(), testMarker) {
			t.Fatalf("got %v, want error naming --repo, %s and %s", err, bare, testMarker)
		}
	})

	t.Run("env", func(t *testing.T) {
		_, err := repo.Resolve("", bare, "", dir, testMarker, osfs.FS{})
		if err == nil || !strings.Contains(err.Error(), "NUYTEK_HARNESS_REPO") || !strings.Contains(err.Error(), bare) || !strings.Contains(err.Error(), testMarker) {
			t.Fatalf("got %v, want error naming NUYTEK_HARNESS_REPO, %s and %s", err, bare, testMarker)
		}
	})
}

func TestResolve_StaleStateWarnsAndFallsThrough(t *testing.T) {
	cwdRepo := mkRepo(t)
	stale := filepath.Join(t.TempDir(), "moved")

	got, err := repo.Resolve("", "", stale, cwdRepo, testMarker, osfs.FS{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Path != cwdRepo || got.Source != repo.SourceCwd {
		t.Fatalf("got %+v, want cwd repo %q", got, cwdRepo)
	}
	if !strings.Contains(got.Warning, stale) {
		t.Fatalf("Warning = %q, want it to name %q", got.Warning, stale)
	}
}

func TestResolve_NothingResolvesListsFourSources(t *testing.T) {
	_, err := repo.Resolve("", "", "", t.TempDir(), testMarker, osfs.FS{})

	if err == nil {
		t.Fatal("want error, got nil")
	}
	for _, source := range []string{"--repo", "NUYTEK_HARNESS_REPO", "repo_path", "current directory"} {
		if !strings.Contains(err.Error(), source) {
			t.Fatalf("error %q does not list source %q", err, source)
		}
	}
}
