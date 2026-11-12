package action_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

// recordingFS logs every path the planners read.
type recordingFS struct {
	osfs.FS
	touched *[]string
}

func (r recordingFS) Lstat(p string) (fs.FileInfo, error) {
	*r.touched = append(*r.touched, p)
	return r.FS.Lstat(p)
}

func (r recordingFS) ReadFile(p string) ([]byte, error) {
	*r.touched = append(*r.touched, p)
	return r.FS.ReadFile(p)
}

func (r recordingFS) ReadDir(p string) ([]fs.DirEntry, error) {
	*r.touched = append(*r.touched, p)
	return r.FS.ReadDir(p)
}

// A component without source or target joined to HOME or the repo would address the whole root.
func TestPlanners_ComponentWithoutPathsTouchNoRootDerivedPath(t *testing.T) {
	const kind = manifest.KindSpecter
	planners := map[string]struct {
		run     func(f *fixture, env check.Env) (plan.Plan, error)
		wantErr bool
	}{
		"install": {run: func(f *fixture, env check.Env) (plan.Plan, error) {
			return action.PlanInstall(context.Background(), env, f.target(manifest.Component{Name: "specter", Kind: kind}), []string{"specter"})
		}},
		"remove": {wantErr: true, run: func(f *fixture, env check.Env) (plan.Plan, error) {
			return action.PlanRemove(context.Background(), env, f.target(manifest.Component{Name: "specter", Kind: kind}), []string{"specter"})
		}},
		"hard install": {run: func(f *fixture, env check.Env) (plan.Plan, error) {
			return action.PlanHardInstall(context.Background(), env, f.target(manifest.Component{Name: "specter", Kind: kind}))
		}},
		"backup": {run: func(f *fixture, env check.Env) (plan.Plan, error) {
			return action.PlanBackup(env, f.target(manifest.Component{Name: "specter", Kind: kind}))
		}},
	}
	for name, tt := range planners {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.special = map[manifest.Kind]plan.KindOps{kind: fakeOps([]plan.Step{plan.Exec{Argv: []string{"register"}}}, nil, nil)}
			var touched []string
			env := f.env()
			env.FS = recordingFS{touched: &touched}

			p, err := tt.run(f, env)

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if name == "backup" && !p.Empty() {
				t.Errorf("backup plan = %+v, want empty", p)
			}

			for _, path := range append(touched, snapshotPaths(p)...) {
				if path == f.home || path == f.repo {
					t.Errorf("planner touched %q, a path derived from an empty Source or Target", path)
				}
			}
		})
	}
}
