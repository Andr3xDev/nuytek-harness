package app_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/adapters/fsx"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/app"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/domain"
)

func mkManifests(t *testing.T, names ...string) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "targets"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(repo, "targets", n+".toml"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

func implementedSpec(id string, checks func(domain.Manifest) ([]app.Check, error)) app.TargetSpec {
	return app.TargetSpec{ID: id, Name: id, Checks: checks,
		Evaluators: map[domain.Kind]app.Evaluator{domain.KindFile: nil}}
}

func noChecks(domain.Manifest) ([]app.Check, error) { return nil, nil }

func TestLoadTargets_PlannedTargetDoesNotLoad(t *testing.T) {
	repo := mkManifests(t, "claude")
	var loaded []string
	load := func(p string, _ []domain.Kind) (domain.Manifest, error) {
		loaded = append(loaded, p)
		return domain.Manifest{}, nil
	}

	got := app.LoadTargets(repo, []app.TargetSpec{app.Planned("opencode", "OpenCode")}, load, fsx.FS{})

	if len(got) < 1 || got[0].Spec.ID != "opencode" || got[0].Err != nil {
		t.Fatalf("got %+v, want planned opencode target without error first", got)
	}
	if len(loaded) != 0 {
		t.Fatalf("loader called for %v, want no call for a planned target", loaded)
	}
}

func TestLoadTargets_MissingManifestNamesPathAndSkipsLoader(t *testing.T) {
	repo := mkManifests(t)
	called := false
	load := func(string, []domain.Kind) (domain.Manifest, error) { called = true; return domain.Manifest{}, nil }

	got := app.LoadTargets(repo, []app.TargetSpec{implementedSpec("claude", noChecks)}, load, fsx.FS{})

	want := filepath.Join(repo, "targets", "claude.toml")
	if len(got) < 1 || got[0].Err == nil || !strings.Contains(got[0].Err.Error(), want) {
		t.Fatalf("got %+v, want Err naming %s", got, want)
	}
	if called {
		t.Fatal("loader called for a missing manifest")
	}
}

func TestLoadTargets_LoaderErrorIsReported(t *testing.T) {
	repo := mkManifests(t, "claude")
	boom := errors.New("claude.toml:3: bad field")
	load := func(string, []domain.Kind) (domain.Manifest, error) { return domain.Manifest{}, boom }

	got := app.LoadTargets(repo, []app.TargetSpec{implementedSpec("claude", noChecks)}, load, fsx.FS{})

	if len(got) < 1 || !errors.Is(got[0].Err, boom) {
		t.Fatalf("got %+v, want Err wrapping %v", got, boom)
	}
}

func TestLoadTargets_ChecksCompositionErrorIsReported(t *testing.T) {
	repo := mkManifests(t, "claude")
	boom := errors.New("cannot compose")
	load := func(string, []domain.Kind) (domain.Manifest, error) { return domain.Manifest{}, nil }

	got := app.LoadTargets(repo, []app.TargetSpec{implementedSpec("claude", func(domain.Manifest) ([]app.Check, error) { return nil, boom })}, load, fsx.FS{})

	if len(got) < 1 || !errors.Is(got[0].Err, boom) {
		t.Fatalf("got %+v, want Err wrapping %v", got, boom)
	}
}

func TestLoadTargets_LoadsManifestAndChecks(t *testing.T) {
	repo := mkManifests(t, "claude")
	manifest := domain.Manifest{Target: "claude", Components: []domain.Component{{Name: "agents", Kind: domain.KindFile}}}
	var gotPath string
	var gotKinds []domain.Kind
	load := func(p string, kinds []domain.Kind) (domain.Manifest, error) {
		gotPath, gotKinds = p, kinds
		return manifest, nil
	}
	checks := func(m domain.Manifest) ([]app.Check, error) { return []app.Check{okCheck("only")}, nil }

	got := app.LoadTargets(repo, []app.TargetSpec{implementedSpec("claude", checks)}, load, fsx.FS{})

	if len(got) < 1 || got[0].Err != nil || got[0].Manifest.Target != "claude" || len(got[0].Checks) != 1 {
		t.Fatalf("got %+v, want loaded manifest and one check", got)
	}
	if gotPath != filepath.Join(repo, "targets", "claude.toml") {
		t.Fatalf("loader path = %q", gotPath)
	}
	if !slices.Equal(gotKinds, []domain.Kind{domain.KindFile}) {
		t.Fatalf("loader kinds = %v, want the evaluator keys", gotKinds)
	}
}

func TestLoadTargets_UnregisteredManifestsAreUnavailableAfterRegistryInAlphabeticalOrder(t *testing.T) {
	repo := mkManifests(t, "claude", "zeta", "alpha")
	var loaded []string
	load := func(p string, _ []domain.Kind) (domain.Manifest, error) {
		loaded = append(loaded, filepath.Base(p))
		return domain.Manifest{}, nil
	}

	got := app.LoadTargets(repo, []app.TargetSpec{implementedSpec("claude", noChecks), app.Planned("opencode", "OpenCode")}, load, fsx.FS{})

	var ids []string
	for _, tg := range got {
		ids = append(ids, tg.Spec.ID)
	}
	if want := []string{"claude", "opencode", "alpha", "zeta"}; !slices.Equal(ids, want) {
		t.Fatalf("target order = %v, want %v", ids, want)
	}
	for _, tg := range got[2:] {
		if tg.Spec.Checks != nil {
			t.Fatalf("unregistered target %q has checks, want unavailable", tg.Spec.ID)
		}
	}
	if !slices.Equal(loaded, []string{"claude.toml"}) {
		t.Fatalf("loader calls = %v, want only claude.toml", loaded)
	}
}

func TestLoadTargets_UnreadableTargetsDirIsSurfacedButMissingIsTolerated(t *testing.T) {
	load := func(string, []domain.Kind) (domain.Manifest, error) { return domain.Manifest{}, nil }
	planned := []app.TargetSpec{app.Planned("opencode", "OpenCode")}

	t.Run("missing dir is tolerated", func(t *testing.T) {
		got := app.LoadTargets(t.TempDir(), planned, load, fsx.FS{})

		if len(got) != 1 || got[0].Err != nil {
			t.Fatalf("got %+v, want only the planned target without error", got)
		}
	})

	t.Run("targets is a file", func(t *testing.T) {
		repo := t.TempDir()
		if err := os.WriteFile(filepath.Join(repo, "targets"), nil, 0o644); err != nil {
			t.Fatal(err)
		}

		got := app.LoadTargets(repo, planned, load, fsx.FS{})

		last := got[len(got)-1]
		if len(got) != 2 || last.Err == nil || !strings.Contains(last.Err.Error(), filepath.Join(repo, "targets")) {
			t.Fatalf("got %+v, want an extra target whose Err names the targets dir", got)
		}
	})
}
