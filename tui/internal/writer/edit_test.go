package writer_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func TestEdit(t *testing.T) {
	setOutput := func(out string) func([]byte) ([]byte, error) {
		return func([]byte) ([]byte, error) { return []byte(out), nil }
	}
	tests := []struct {
		name        string
		old         *string // nil = file missing
		mode        os.FileMode
		edit        func(old []byte) ([]byte, error)
		wantContent string
		wantMode    os.FileMode
		wantFail    bool
		wantSameMod bool // mtime must not move (identical output skips the write)
	}{
		{name: "existing file keeps its mode", old: ptr("old"), mode: 0o640, edit: setOutput("new"), wantContent: "new", wantMode: 0o640},
		{name: "new file gets 0600", edit: setOutput("fresh"), wantContent: "fresh", wantMode: 0o600},
		{name: "identical output skips the write", old: ptr("same"), mode: 0o644, edit: setOutput("same"), wantContent: "same", wantMode: 0o644, wantSameMod: true},
		{
			name: "edit error leaves the file untouched", old: ptr("orig"), mode: 0o644,
			edit:        func([]byte) ([]byte, error) { return nil, errors.New("boom") },
			wantContent: "orig", wantMode: 0o644, wantFail: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "cfg.json")
			past := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
			if tc.old != nil {
				must(t, os.WriteFile(path, []byte(*tc.old), 0o644))
				setMode(t, path, tc.mode)
				must(t, os.Chtimes(path, past, past))
			}
			var gotExists bool
			step := plan.Edit{Path: path, Edit: func(old []byte, exists bool) ([]byte, error) {
				gotExists = exists
				return tc.edit(old)
			}}

			res, _ := run(t, newWriter(fixedNow), step)

			if gotExists != (tc.old != nil) {
				t.Fatalf("exists = %v, want %v", gotExists, tc.old != nil)
			}
			if tc.wantFail {
				wantStatus(t, res, plan.StepFailed)
				if res.Err == nil || !strings.Contains(res.Err.Error(), path) {
					t.Fatalf("err = %v, want it to name %s", res.Err, path)
				}
			} else {
				wantStatus(t, res, plan.StepDone)
			}
			equalTree(t, treeOf(t, dir), fixture{"cfg.json": tc.wantContent})
			if got := modeOf(t, path); got != tc.wantMode {
				t.Fatalf("mode = %o, want %o", got, tc.wantMode)
			}
			if tc.wantSameMod {
				info, err := os.Stat(path)
				must(t, err)
				if !info.ModTime().Equal(past) {
					t.Fatalf("mtime = %v, want %v (write should be skipped)", info.ModTime(), past)
				}
			}
		})
	}
}

func TestEditFailsWhenFileChangesBetweenReadAndRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	must(t, os.WriteFile(path, []byte("old"), 0o644))
	step := plan.Edit{Path: path, Edit: func([]byte, bool) ([]byte, error) {
		must(t, os.WriteFile(path, []byte("other process"), 0o644))
		return []byte("mine"), nil
	}}

	res, _ := run(t, newWriter(fixedNow), step)

	wantStatus(t, res, plan.StepFailed)
	if res.Err == nil || !strings.Contains(res.Err.Error(), "changed while editing, retry") || !strings.Contains(res.Err.Error(), path) {
		t.Fatalf("err = %v, want changed-while-editing naming %s", res.Err, path)
	}
	equalTree(t, treeOf(t, dir), fixture{"cfg.json": "other process"})
}

func TestEditThroughSymlink(t *testing.T) {
	root := t.TempDir()
	build(t, root, fixture{"real/cfg.json": "old"})
	real := filepath.Join(root, "real/cfg.json")
	setMode(t, real, 0o640)
	link := filepath.Join(root, "cfg-link.json")
	must(t, os.Symlink(real, link))
	step := plan.Edit{Path: link, Edit: func([]byte, bool) ([]byte, error) { return []byte("new"), nil }}

	res, _ := run(t, newWriter(fixedNow), step)

	wantStatus(t, res, plan.StepDone)
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link replaced: %v, %v", info, err)
	}
	if b, _ := os.ReadFile(real); string(b) != "new" {
		t.Fatalf("real target = %q, want new", b)
	}
}

func ptr(s string) *string { return &s }
