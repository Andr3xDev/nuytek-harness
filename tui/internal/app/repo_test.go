package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/adapters/fsx"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/app"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/domain"
)

const testMarker = "claude.toml"

func mkRepo(t *testing.T) string {
	t.Helper()
	return mkManifests(t, "claude")
}

func TestResolveRepo_Precedence(t *testing.T) {
	flagRepo, envRepo, stateRepo, cwdRepo := mkRepo(t), mkRepo(t), mkRepo(t), mkRepo(t)
	deep := filepath.Join(cwdRepo, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	st := domain.State{RepoPath: stateRepo}

	tests := []struct {
		name             string
		flag, env, cwd   string
		state            domain.State
		wantPath, source string
	}{
		{"flag wins", flagRepo, envRepo, deep, st, flagRepo, app.SourceFlag},
		{"env over state", "", envRepo, deep, st, envRepo, app.SourceEnv},
		{"state over cwd", "", "", deep, st, stateRepo, app.SourceState},
		{"walk up from cwd", "", "", deep, domain.State{}, cwdRepo, app.SourceCwd},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := app.ResolveRepo(tt.flag, tt.env, tt.state, tt.cwd, testMarker, fsx.FS{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Path != tt.wantPath || got.Source != tt.source || got.Warning != "" {
				t.Fatalf("got %+v, want path %q source %q no warning", got, tt.wantPath, tt.source)
			}
		})
	}
}

func TestResolveRepo_RelativeFlagAndEnvBecomeAbsolute(t *testing.T) {
	repo := mkRepo(t)
	t.Chdir(filepath.Dir(repo))
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(wd, filepath.Base(repo))

	for name, resolve := range map[string]func() (app.Resolution, error){
		"flag": func() (app.Resolution, error) {
			return app.ResolveRepo(filepath.Base(repo), "", domain.State{}, wd, testMarker, fsx.FS{})
		},
		"env": func() (app.Resolution, error) {
			return app.ResolveRepo("", filepath.Base(repo), domain.State{}, wd, testMarker, fsx.FS{})
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

func TestResolveRepo_ExplicitSourceWithoutMarkerIsHardError(t *testing.T) {
	bare, repo := t.TempDir(), mkRepo(t)

	t.Run("flag", func(t *testing.T) {
		_, err := app.ResolveRepo(bare, "", domain.State{}, repo, testMarker, fsx.FS{})
		if err == nil || !strings.Contains(err.Error(), "--repo") || !strings.Contains(err.Error(), bare) || !strings.Contains(err.Error(), testMarker) {
			t.Fatalf("got %v, want error naming --repo, %s and %s", err, bare, testMarker)
		}
	})

	t.Run("env", func(t *testing.T) {
		_, err := app.ResolveRepo("", bare, domain.State{}, repo, testMarker, fsx.FS{})
		if err == nil || !strings.Contains(err.Error(), "NUYTEK_HARNESS_REPO") || !strings.Contains(err.Error(), bare) || !strings.Contains(err.Error(), testMarker) {
			t.Fatalf("got %v, want error naming NUYTEK_HARNESS_REPO, %s and %s", err, bare, testMarker)
		}
	})
}

func TestResolveRepo_StaleStateWarnsAndFallsThrough(t *testing.T) {
	cwdRepo := mkRepo(t)
	stale := filepath.Join(t.TempDir(), "moved")

	got, err := app.ResolveRepo("", "", domain.State{RepoPath: stale}, cwdRepo, testMarker, fsx.FS{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Path != cwdRepo || got.Source != app.SourceCwd {
		t.Fatalf("got %+v, want cwd repo %q", got, cwdRepo)
	}
	if !strings.Contains(got.Warning, stale) {
		t.Fatalf("Warning = %q, want it to name %q", got.Warning, stale)
	}
}

func TestResolveRepo_NothingResolvesListsFourSources(t *testing.T) {
	_, err := app.ResolveRepo("", "", domain.State{}, t.TempDir(), testMarker, fsx.FS{})

	if err == nil {
		t.Fatal("want error, got nil")
	}
	for _, source := range []string{"--repo", "NUYTEK_HARNESS_REPO", "repo_path", "current directory"} {
		if !strings.Contains(err.Error(), source) {
			t.Fatalf("error %q does not list source %q", err, source)
		}
	}
}
