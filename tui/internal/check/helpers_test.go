package check_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
)

func okCheck(id string) check.Check {
	return check.Check{ID: id, Run: func(context.Context, check.Env) []check.Result {
		return []check.Result{{CheckID: id, Subject: id, Status: check.StatusOK}}
	}}
}

// fakeExec resolves tools from onPath and answers probes from outputs/probeErrs; a tool in hang blocks until ctx ends.
type fakeExec struct {
	onPath    map[string]bool
	outputs   map[string]string
	probeErrs map[string]error
	hang      map[string]bool

	mu   sync.Mutex
	args map[string][]string
}

func (f *fakeExec) LookPath(name string) (string, error) {
	if f.onPath[name] {
		return "/usr/bin/" + name, nil
	}
	return "", os.ErrNotExist
}

func (f *fakeExec) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	f.mu.Lock()
	if f.args == nil {
		f.args = map[string][]string{}
	}
	f.args[name] = args
	f.mu.Unlock()
	if f.hang[name] {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return []byte(f.outputs[name]), f.probeErrs[name]
}

func (f *fakeExec) Run(context.Context, string, ...string) (check.ExecResult, error) {
	return check.ExecResult{}, errors.New("Run is not used by these checks")
}

func (f *fakeExec) argsOf(name string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.args[name]
}

// newHome returns an Env backed by the real filesystem under a temp HOME.
func newHome(t *testing.T, exec check.Exec) check.Env {
	t.Helper()
	return check.Env{FS: osfs.FS{}, Exec: exec, Home: t.TempDir()}
}

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func runCheck(t *testing.T, c check.Check, env check.Env) []check.Result {
	t.Helper()
	return c.Run(context.Background(), env)
}

func resultFor(t *testing.T, results []check.Result, subject string) check.Result {
	t.Helper()
	for _, r := range results {
		if r.Subject == subject {
			return r
		}
	}
	t.Fatalf("no result for subject %q in %+v", subject, results)
	return check.Result{}
}
