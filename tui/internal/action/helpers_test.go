package action_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	homeStoreDir = ".bk"
	repoStoreDir = ".bkr"
)

// fakeExec reports the listed tools as missing from PATH and fails any version probe.
type fakeExec struct{ missing []string }

func (f fakeExec) LookPath(name string) (string, error) {
	if slices.Contains(f.missing, name) {
		return "", errors.New("not found")
	}
	return "/usr/bin/" + name, nil
}

func (fakeExec) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("action must not probe tool versions")
}

func (fakeExec) Run(context.Context, string, ...string) (check.ExecResult, error) {
	return check.ExecResult{}, errors.New("action must not run tools")
}

// fixture is a temp HOME and repo; component Source resolves under repo, Target under home.
type fixture struct {
	t            *testing.T
	repo, home   string
	missingTools []string
	noRepoStore  bool
	special      map[manifest.Kind]plan.KindOps
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	return &fixture{t: t, repo: t.TempDir(), home: t.TempDir()}
}

func (f *fixture) writeRepo(rel, content string, mode os.FileMode) {
	f.write(f.repo, rel, content, mode)
}
func (f *fixture) writeHome(rel, content string, mode os.FileMode) {
	f.write(f.home, rel, content, mode)
}

func (f *fixture) write(root, rel, content string, mode os.FileMode) {
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

func (f *fixture) mkdir(root, rel string) {
	f.t.Helper()
	if err := os.MkdirAll(filepath.Join(root, rel), 0o755); err != nil {
		f.t.Fatal(err)
	}
}

// symlink skips the test where the platform cannot create links.
func (f *fixture) symlink(root, rel, to string) {
	f.t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.Symlink(to, path); err != nil {
		f.t.Skipf("symlinks unavailable: %v", err)
	}
}

func (f *fixture) env() check.Env {
	return check.Env{
		FS: osfs.FS{}, Exec: fakeExec{missing: f.missingTools}, Home: f.home, Repo: f.repo,
		LookupEnv: func(string) (string, bool) { return "", false },
	}
}

func (f *fixture) homeStore() string { return filepath.Join(f.home, homeStoreDir) }
func (f *fixture) repoStore() string { return filepath.Join(f.repo, repoStoreDir) }

func (f *fixture) target(components ...manifest.Component) target.Target {
	return target.Target{
		Manifest: manifest.Manifest{Components: components},
		Spec: target.Spec{
			ID: "t", Name: "t", Special: f.special,
			BackupDirs: func(home, repo, _ string) []backup.Dir {
				dirs := []backup.Dir{{Path: f.homeStore(), Label: "home", Kind: backup.StoreHome, Base: home}}
				if !f.noRepoStore {
					dirs = append(dirs, backup.Dir{Path: f.repoStore(), Label: "repo", Kind: backup.StoreRepo, Base: repo})
				}
				return dirs
			},
		},
	}
}

func component(name string, kind manifest.Kind) manifest.Component {
	return manifest.Component{Name: name, Kind: kind, Source: "claude/" + name, Target: ".claude/" + name}
}

func pullable(c manifest.Component) manifest.Component {
	c.Pull = true
	return c
}

func needing(c manifest.Component, tools ...string) manifest.Component {
	c.Needs = tools
	return c
}

func stepsOf[T plan.Step](p plan.Plan) []T {
	var out []T
	for _, s := range p.Steps {
		if v, ok := s.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

func onlyStep[T plan.Step](t *testing.T, p plan.Plan) T {
	t.Helper()
	got := stepsOf[T](p)
	if len(got) != 1 {
		t.Fatalf("plan has %d steps of type %T, want 1 (steps: %v)", len(got), *new(T), p.Steps)
	}
	return got[0]
}

func snapshotPaths(p plan.Plan) []string {
	if len(p.Steps) == 0 {
		return nil
	}
	s, ok := p.Steps[0].(plan.Snapshot)
	if !ok {
		return nil
	}
	paths := slices.Clone(s.Paths)
	slices.Sort(paths)
	return paths
}

func rowPaths(p plan.Plan, effect plan.Effect) []string {
	var paths []string
	for _, r := range p.Rows() {
		if r.Effect == effect {
			paths = append(paths, r.Path)
		}
	}
	slices.Sort(paths)
	return paths
}

// fakeOps returns kind ops whose parts hold the given steps.
func fakeOps(install, hard, remove []plan.Step) plan.KindOps {
	part := func(steps []plan.Step) plan.PartFunc {
		if steps == nil {
			return nil
		}
		return func(context.Context, check.Env, manifest.Component) (plan.Part, error) {
			return plan.Part{Steps: steps}, nil
		}
	}
	return plan.KindOps{Install: part(install), Hard: part(hard), Remove: part(remove)}
}
