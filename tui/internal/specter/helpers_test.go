package specter_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/state"
)

// fakeExec answers Run by the space-joined command line and records every command it was asked to run.
type fakeExec struct {
	onPath  map[string]bool
	results map[string]check.ExecResult

	mu    sync.Mutex
	calls []string
}

func (f *fakeExec) LookPath(name string) (string, error) {
	f.record("lookpath " + name)
	if f.onPath[name] {
		return "/usr/bin/" + name, nil
	}
	return "", os.ErrNotExist
}

func (f *fakeExec) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	f.record(commandLine(name, args))
	return nil, errors.New("specter must use Run, not Output")
}

func (f *fakeExec) Run(_ context.Context, name string, args ...string) (check.ExecResult, error) {
	line := commandLine(name, args)
	f.record(line)
	res, ok := f.results[line]
	if !ok {
		return check.ExecResult{}, errors.New("unexpected command: " + line)
	}
	return res, nil
}

func (f *fakeExec) record(line string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, line)
}

func (f *fakeExec) callsTo(prefix string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			out = append(out, c)
		}
	}
	return out
}

func commandLine(name string, args []string) string {
	return strings.Join(append([]string{name}, args...), " ")
}

func envWith(t *testing.T, exec check.Exec, root string) check.Env {
	t.Helper()
	return check.Env{
		FS:        osfs.FS{},
		Exec:      exec,
		Home:      t.TempDir(),
		LookupEnv: func(string) (string, bool) { return "", false },
		Saved:     state.State{SpecterRoot: root},
	}
}

func writeStore(t *testing.T, root, dir, storeYAML string) {
	t.Helper()
	path := filepath.Join(root, dir, ".openspec-store", "store.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(storeYAML), 0o644); err != nil {
		t.Fatal(err)
	}
}
