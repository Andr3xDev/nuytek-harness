package backup_test

import (
	"os"
	"path/filepath"
	"testing"
)

func mkBackups(t *testing.T, root string, names ...string) {
	t.Helper()
	for _, n := range names {
		if err := os.MkdirAll(filepath.Join(root, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}
