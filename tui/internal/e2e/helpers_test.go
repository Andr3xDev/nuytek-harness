package e2e_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/targets/claude"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/writer"
)

const (
	skipPluginsEnv = "NUYTEK_SKIP_PLUGINS"
	runInterval    = time.Hour
)

// foundExec finds every tool: the scenarios never probe or run tools through check.Env.
type foundExec struct{}

func (foundExec) LookPath(name string) (string, error) { return "/usr/bin/" + name, nil }

func (foundExec) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("e2e must not probe tool versions")
}

// world is a temp HOME and repo with a small hand-built Claude manifest.
type world struct {
	t          *testing.T
	home, repo string
	target     target.Target
	env        check.Env
	vars       map[string]string
	clock      time.Time
}

func newWorld(t *testing.T) *world {
	t.Helper()
	w := &world{
		t: t, home: t.TempDir(), repo: t.TempDir(), vars: map[string]string{},
		clock: time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC),
	}
	w.target = target.Target{
		Spec: claude.Spec(),
		Manifest: manifest.Manifest{Components: []manifest.Component{
			{Name: "agents", Kind: manifest.KindDir, Source: "claude/agents", Target: ".claude/agents", Pull: true},
			{Name: "hooks", Kind: manifest.KindDirX, Source: "claude/hooks", Target: ".claude/hooks", Pull: true},
			{Name: "mcp", Kind: "mcp", Source: "claude/claude-mcp.json", Target: ".claude.json"},
			{Name: "plugins", Kind: "plugins", Source: "claude/claude-settings.json", Target: ".claude/settings.json"},
		}},
	}
	w.env = check.Env{
		FS: osfs.FS{}, Exec: foundExec{}, Home: w.home, Repo: w.repo,
		LookupEnv: func(name string) (string, bool) { v, ok := w.vars[name]; return v, ok },
	}
	return w
}

func (w *world) skipPlugins() { w.vars[skipPluginsEnv] = "1" }

func (w *world) put(root, rel, content string) string {
	w.t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		w.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		w.t.Fatal(err)
	}
	return path
}

func (w *world) putHome(rel, content string) string { return w.put(w.home, rel, content) }
func (w *world) putRepo(rel, content string) string { return w.put(w.repo, rel, content) }

// run applies p with the real writer; each run gets a later clock instant so stamps never collide.
func (w *world) run(p plan.Plan) plan.Result {
	w.t.Helper()
	w.clock = w.clock.Add(runInterval)
	return plan.Apply(context.Background(), writer.New(func() time.Time { return w.clock }), p, func(plan.Event) {})
}

func (w *world) mustApply(p plan.Plan, err error) plan.Result {
	w.t.Helper()
	if err != nil {
		w.t.Fatalf("plan: %v", err)
	}
	result := w.run(p)
	if result.Failed != nil {
		w.t.Fatalf("apply failed at %q: %v", result.Failed.Step, result.Failed.Err)
	}
	return result
}

func (w *world) homeStore() string { return filepath.Join(w.home, ".claude-backups") }
func (w *world) repoStore() string { return filepath.Join(w.repo, ".refresh-backups") }

// stamps returns the stamp directory names under store.
func (w *world) stamps(store string) []string {
	w.t.Helper()
	items, err := os.ReadDir(store)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		w.t.Fatal(err)
	}
	var names []string
	for _, item := range items {
		names = append(names, item.Name())
	}
	return names
}

// snapshots lists the entries of both stores, newest first.
func (w *world) snapshots() []backup.Entry {
	return backup.List(osfs.FS{}, w.target.Spec.BackupDirs(w.home, w.repo))
}

// files maps the relative path of every regular file under root to its content.
func files(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		out[rel] = string(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
