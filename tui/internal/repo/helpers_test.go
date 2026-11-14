package repo_test

import (
	"os"
	"path/filepath"
	"testing"
)

const testMarker = "targets/claude.toml"

func mkRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "targets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, testMarker), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}
