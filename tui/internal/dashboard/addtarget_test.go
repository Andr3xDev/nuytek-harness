package dashboard_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/dashboard"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const fakeManifest = `[[component]]
name   = "alpha"
kind   = "file"
source = "alpha.txt"
target = ".fake/alpha.txt"
needs  = []

[[component]]
name   = "beta"
kind   = "file"
source = "beta.txt"
target = ".fake/beta.txt"
needs  = []
`

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A new CLI needs only a manifest plus one Spec composed from generic pieces; no core package changes.
func TestAddTarget_FakeSpecAppearsAndComputes(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	writeFile(t, repo, target.ManifestRel("fake"), fakeManifest)
	writeFile(t, repo, "alpha.txt", "a")
	writeFile(t, repo, "beta.txt", "b")
	writeFile(t, home, ".fake/alpha.txt", "a")

	spec := target.Spec{
		ID:         "fake",
		Name:       "Fake",
		Evaluators: drift.Evaluators(nil),
		Checks: func(manifest.Manifest) ([]check.Check, error) {
			return []check.Check{
				fixedCheck("one", resultOf("one", check.StatusOK)),
				fixedCheck("two", resultOf("two", check.StatusWarning)),
			}, nil
		},
	}
	fsys := osfs.FS{}
	targets := target.Load(repo, []target.Spec{spec}, manifest.Load, fsys)

	if len(targets) != 1 || targets[0].Spec.ID != "fake" {
		t.Fatalf("targets = %+v, want only fake", targets)
	}
	fake := targets[0]
	if fake.Err != nil || fake.Spec.Checks == nil || len(fake.Checks) != 2 {
		t.Fatalf("fake not available: err=%v checks=%d", fake.Err, len(fake.Checks))
	}

	env := check.Env{FS: fsys, Home: home, Repo: repo}
	got := dashboard.Compute(context.Background(), env, fake)

	if got.Counts[check.StatusOK] != 1 || got.Counts[check.StatusWarning] != 1 {
		t.Fatalf("Counts = %v, want 1 ok and 1 warning", got.Counts)
	}
	if len(got.Components) != len(fake.Manifest.Components) || len(got.Components) != 2 {
		t.Fatalf("Components = %+v, want one per manifest entry (2)", got.Components)
	}
	wantStates := map[string]drift.State{"alpha": drift.StateInstalled, "beta": drift.StateNotInstalled}
	for _, c := range got.Components {
		if c.State != wantStates[c.Name] {
			t.Errorf("component %s state = %s, want %s", c.Name, c.State, wantStates[c.Name])
		}
	}
}
