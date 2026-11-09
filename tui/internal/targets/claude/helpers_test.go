package claude

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
)

// newEnv returns an Env over the real filesystem with a temp HOME and a temp repo.
func newEnv(t *testing.T) check.Env {
	t.Helper()
	return check.Env{FS: osfs.FS{}, Home: t.TempDir(), Repo: t.TempDir()}
}

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

func requireState(t *testing.T, got drift.Status, want drift.State) {
	t.Helper()
	if got.State != want {
		t.Fatalf("state = %q (reason %q), want %q", got.State, got.Reason, want)
	}
}
