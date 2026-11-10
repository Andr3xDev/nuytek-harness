package writer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func TestRemove(t *testing.T) {
	root := t.TempDir()
	build(t, root, fixture{
		"a": "A", "sub/b": "B", "foreign": "F", "keep/x": "X", "empty/": "",
	})
	step := plan.Remove{
		Files: []string{filepath.Join(root, "a"), filepath.Join(root, "sub/b")},
		Dirs:  []string{filepath.Join(root, "sub"), filepath.Join(root, "keep"), filepath.Join(root, "empty")},
	}

	res, _ := run(t, newWriter(fixedNow), step)

	wantStatus(t, res, plan.StepDone)
	equalTree(t, treeOf(t, root), fixture{"foreign": "F", "keep/x": "X"})
	for _, gone := range []string{"sub", "empty"} {
		if _, err := os.Lstat(filepath.Join(root, gone)); !os.IsNotExist(err) {
			t.Fatalf("%s should be removed, Lstat err = %v", gone, err)
		}
	}
}

func TestRemoveTree(t *testing.T) {
	t.Run("removes custom and owned files", func(t *testing.T) {
		root := t.TempDir()
		build(t, root, fixture{"tree/owned": "O", "tree/custom": "C", "tree/d/deep": "D", "other": "keep"})

		res, _ := run(t, newWriter(fixedNow), plan.RemoveTree{Path: filepath.Join(root, "tree")})

		wantStatus(t, res, plan.StepDone)
		equalTree(t, treeOf(t, root), fixture{"other": "keep"})
	})
	t.Run("symlink path fails and touches nothing", func(t *testing.T) {
		root := t.TempDir()
		build(t, root, fixture{"real/f": "F", "ln": "->real"})

		res, _ := run(t, newWriter(fixedNow), plan.RemoveTree{Path: filepath.Join(root, "ln")})

		wantStatus(t, res, plan.StepFailed)
		if res.Err == nil {
			t.Fatal("want an error")
		}
		equalTree(t, treeOf(t, root), fixture{"real/f": "F", "ln": "->real"})
	})
	t.Run("link inside the tree is removed as a link", func(t *testing.T) {
		root := t.TempDir()
		build(t, root, fixture{"outside/f": "F", "tree/own": "O", "tree/ln": "->" + filepath.Join(root, "outside")})

		res, _ := run(t, newWriter(fixedNow), plan.RemoveTree{Path: filepath.Join(root, "tree")})

		wantStatus(t, res, plan.StepDone)
		equalTree(t, treeOf(t, root), fixture{"outside/f": "F"})
	})
}

func TestRemoveMatching(t *testing.T) {
	tests := []struct {
		name string
		glob string
		want fixture
	}{
		{"only matches removed", "a*.txt", fixture{"keep.md": "K", "sub/a3.txt": "3"}},
		{"zero matches is done", "zzz*", fixture{"a1.txt": "1", "a2.txt": "2", "keep.md": "K", "sub/a3.txt": "3"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			build(t, root, fixture{"a1.txt": "1", "a2.txt": "2", "keep.md": "K", "sub/a3.txt": "3"})

			res, _ := run(t, newWriter(fixedNow), plan.RemoveMatching{Globs: []string{filepath.Join(root, tc.glob)}})

			wantStatus(t, res, plan.StepDone)
			equalTree(t, treeOf(t, root), tc.want)
		})
	}
}

func TestChmod(t *testing.T) {
	tests := []struct {
		name       string
		from, want os.FileMode
	}{
		{"read for all gets x for all", 0o644, 0o755},
		{"x only where r is set", 0o604, 0o705},
		{"no read bit unchanged", 0o200, 0o200},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			build(t, root, fixture{"f": "F"})
			p := filepath.Join(root, "f")
			setMode(t, p, tc.from)

			res, _ := run(t, newWriter(fixedNow), plan.Chmod{Paths: []string{p}})

			wantStatus(t, res, plan.StepDone)
			if got := modeOf(t, p); got != tc.want {
				t.Fatalf("mode = %o, want %o", got, tc.want)
			}
		})
	}
}

type unknownStep struct{}

func (unknownStep) Describe() string       { return "unknown" }
func (unknownStep) Changes() []plan.Change { return nil }
func (unknownStep) Touches() plan.Touch    { return plan.Touch{} }

func TestRunRejectsUnknownStep(t *testing.T) {
	res, _ := run(t, newWriter(fixedNow), unknownStep{})

	wantStatus(t, res, plan.StepFailed)
	if res.Err == nil {
		t.Fatal("want an error")
	}
}
