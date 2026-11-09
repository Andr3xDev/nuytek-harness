package target_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

func TestLoad_PlannedTargetDoesNotLoad(t *testing.T) {
	repo := mkManifests(t, "claude")
	var loaded []string
	load := func(p string, _ []manifest.Kind) (manifest.Manifest, error) {
		loaded = append(loaded, p)
		return manifest.Manifest{}, nil
	}

	got := target.Load(repo, []target.Spec{target.Planned("opencode", "OpenCode")}, load, osfs.FS{})

	if len(got) < 1 || got[0].Spec.ID != "opencode" || got[0].Err != nil {
		t.Fatalf("got %+v, want planned opencode target without error first", got)
	}
	if len(loaded) != 0 {
		t.Fatalf("loader called for %v, want no call for a planned target", loaded)
	}
}

func TestLoad_MissingManifestNamesPathAndSkipsLoader(t *testing.T) {
	repo := mkManifests(t)
	called := false
	load := func(string, []manifest.Kind) (manifest.Manifest, error) {
		called = true
		return manifest.Manifest{}, nil
	}

	got := target.Load(repo, []target.Spec{implementedSpec("claude", noChecks)}, load, osfs.FS{})

	want := filepath.Join(repo, "targets", "claude.toml")
	if len(got) < 1 || got[0].Err == nil || !strings.Contains(got[0].Err.Error(), want) {
		t.Fatalf("got %+v, want Err naming %s", got, want)
	}
	if called {
		t.Fatal("loader called for a missing manifest")
	}
}

func TestLoad_LoaderErrorIsReported(t *testing.T) {
	repo := mkManifests(t, "claude")
	boom := errors.New("claude.toml:3: bad field")
	load := func(string, []manifest.Kind) (manifest.Manifest, error) { return manifest.Manifest{}, boom }

	got := target.Load(repo, []target.Spec{implementedSpec("claude", noChecks)}, load, osfs.FS{})

	if len(got) < 1 || !errors.Is(got[0].Err, boom) {
		t.Fatalf("got %+v, want Err wrapping %v", got, boom)
	}
}

func TestLoad_ChecksCompositionErrorIsReported(t *testing.T) {
	repo := mkManifests(t, "claude")
	boom := errors.New("cannot compose")
	load := func(string, []manifest.Kind) (manifest.Manifest, error) { return manifest.Manifest{}, nil }

	got := target.Load(repo, []target.Spec{implementedSpec("claude", func(manifest.Manifest) ([]check.Check, error) { return nil, boom })}, load, osfs.FS{})

	if len(got) < 1 || !errors.Is(got[0].Err, boom) {
		t.Fatalf("got %+v, want Err wrapping %v", got, boom)
	}
}

func TestLoad_LoadsManifestAndChecks(t *testing.T) {
	repo := mkManifests(t, "claude")
	m := manifest.Manifest{Target: "claude", Components: []manifest.Component{{Name: "agents", Kind: manifest.KindFile}}}
	var gotPath string
	var gotKinds []manifest.Kind
	load := func(p string, kinds []manifest.Kind) (manifest.Manifest, error) {
		gotPath, gotKinds = p, kinds
		return m, nil
	}
	checks := func(m manifest.Manifest) ([]check.Check, error) { return []check.Check{okCheck("only")}, nil }

	got := target.Load(repo, []target.Spec{implementedSpec("claude", checks)}, load, osfs.FS{})

	if len(got) < 1 || got[0].Err != nil || got[0].Manifest.Target != "claude" || len(got[0].Checks) != 1 {
		t.Fatalf("got %+v, want loaded manifest and one check", got)
	}
	if gotPath != filepath.Join(repo, "targets", "claude.toml") {
		t.Fatalf("loader path = %q", gotPath)
	}
	if !slices.Equal(gotKinds, []manifest.Kind{manifest.KindFile}) {
		t.Fatalf("loader kinds = %v, want the evaluator keys", gotKinds)
	}
}

func TestLoad_UnregisteredManifestsAreUnavailableAfterRegistryInAlphabeticalOrder(t *testing.T) {
	repo := mkManifests(t, "claude", "zeta", "alpha")
	var loaded []string
	load := func(p string, _ []manifest.Kind) (manifest.Manifest, error) {
		loaded = append(loaded, filepath.Base(p))
		return manifest.Manifest{}, nil
	}

	got := target.Load(repo, []target.Spec{implementedSpec("claude", noChecks), target.Planned("opencode", "OpenCode")}, load, osfs.FS{})

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

func TestLoad_UnreadableTargetsDirIsSurfacedButMissingIsTolerated(t *testing.T) {
	load := func(string, []manifest.Kind) (manifest.Manifest, error) { return manifest.Manifest{}, nil }
	planned := []target.Spec{target.Planned("opencode", "OpenCode")}

	t.Run("missing dir is tolerated", func(t *testing.T) {
		got := target.Load(t.TempDir(), planned, load, osfs.FS{})

		if len(got) != 1 || got[0].Err != nil {
			t.Fatalf("got %+v, want only the planned target without error", got)
		}
	})

	t.Run("targets is a file", func(t *testing.T) {
		repo := t.TempDir()
		if err := os.WriteFile(filepath.Join(repo, "targets"), nil, 0o644); err != nil {
			t.Fatal(err)
		}

		got := target.Load(repo, planned, load, osfs.FS{})

		last := got[len(got)-1]
		if len(got) != 2 || last.Err == nil || !strings.Contains(last.Err.Error(), filepath.Join(repo, "targets")) {
			t.Fatalf("got %+v, want an extra target whose Err names the targets dir", got)
		}
	})
}
