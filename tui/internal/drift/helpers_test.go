package drift_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
)

// fixture is a temp repo and home; component Source resolves under repo, Target under home.
type fixture struct {
	t          *testing.T
	repo, home string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	return fixture{t: t, repo: t.TempDir(), home: t.TempDir()}
}

func (f fixture) writeSource(rel, content string, mode os.FileMode) {
	f.write(f.repo, rel, content, mode)
}
func (f fixture) writeTarget(rel, content string, mode os.FileMode) {
	f.write(f.home, rel, content, mode)
}

func (f fixture) write(root, rel, content string, mode os.FileMode) {
	f.t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		f.t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		f.t.Fatal(err)
	}
}

// symlink skips the test where the platform cannot create links.
func (f fixture) symlink(root, rel, to string) {
	f.t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.Symlink(to, path); err != nil {
		f.t.Skipf("symlinks unavailable: %v", err)
	}
}

func (f fixture) evaluate(kind manifest.Kind, source, target string) drift.Status {
	f.t.Helper()
	evaluate, ok := drift.Evaluators(nil)[kind]
	if !ok {
		f.t.Fatalf("no evaluator for kind %q", kind)
	}
	env := check.Env{FS: osfs.FS{}, Home: f.home, Repo: f.repo}
	return evaluate(context.Background(), env, manifest.Component{Name: "example", Kind: kind, Source: source, Target: target})
}

func requireState(t *testing.T, got drift.Status, want drift.State) {
	t.Helper()
	if got.State != want {
		t.Fatalf("state = %q (reason %q), want %q", got.State, got.Reason, want)
	}
}
