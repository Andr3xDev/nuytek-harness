package writer_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/writer"
)

// fixture maps a relative path to file content; "->target" makes a symlink and a trailing "/" a directory.
type fixture map[string]string

func build(t *testing.T, root string, f fixture) {
	t.Helper()
	for rel, v := range f {
		p := filepath.Join(root, rel)
		if strings.HasSuffix(rel, "/") {
			must(t, os.MkdirAll(p, 0o755))
			continue
		}
		must(t, os.MkdirAll(filepath.Dir(p), 0o755))
		if target, ok := strings.CutPrefix(v, "->"); ok {
			must(t, os.Symlink(target, p))
			continue
		}
		must(t, os.WriteFile(p, []byte(v), 0o644))
		must(t, os.Chmod(p, 0o644))
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// treeOf lists files and links under root as relative path to content, or "->target" for a link.
// A missing root gives an empty map.
func treeOf(t *testing.T, root string) fixture {
	t.Helper()
	got := fixture{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && p == root {
				return nil
			}
			return err
		}
		rel, _ := filepath.Rel(root, p)
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			target, err := os.Readlink(p)
			if err != nil {
				return err
			}
			got[rel] = "->" + target
		case d.Type().IsRegular():
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			got[rel] = string(b)
		}
		return nil
	})
	must(t, err)
	return got
}

func equalTree(t *testing.T, got, want fixture) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("tree = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("tree = %v, want %v", got, want)
		}
	}
}

func modeOf(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	must(t, err)
	return info.Mode().Perm()
}

func setMode(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	must(t, os.Chmod(path, mode))
}

var fixedNow = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

func newWriter(now time.Time) *writer.Writer {
	return writer.New(func() time.Time { return now })
}

func run(t *testing.T, w *writer.Writer, s plan.Step) (plan.StepResult, []string) {
	t.Helper()
	var lines []string
	res := w.Run(context.Background(), s, func(l string) { lines = append(lines, l) })
	return res, lines
}

func wantStatus(t *testing.T, res plan.StepResult, want plan.StepStatus) {
	t.Helper()
	if res.Status != want {
		t.Fatalf("status = %q (err %v), want %q", res.Status, res.Err, want)
	}
}
