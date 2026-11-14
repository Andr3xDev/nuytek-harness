package claude

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/dashboard"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	componentSpecter = "specter"
	toolOpenspec     = "openspec"
	storeListCommand = "openspec store list --json"
)

// specterExec resolves every tool except those in missing and answers only `openspec store list --json`.
type specterExec struct {
	missing []string
	listing string
}

func (e specterExec) LookPath(name string) (string, error) {
	if slices.Contains(e.missing, name) {
		return "", errors.New("not found")
	}
	return "/usr/bin/" + name, nil
}

func (specterExec) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected Output")
}

func (e specterExec) Run(_ context.Context, name string, args ...string) (check.ExecResult, error) {
	if line := strings.Join(append([]string{name}, args...), " "); line == storeListCommand {
		return check.ExecResult{Stdout: []byte(e.listing)}, nil
	}
	return check.ExecResult{}, errors.New("unexpected command")
}

func realTarget(t *testing.T) target.Target {
	t.Helper()
	return target.Target{Spec: Spec(), Manifest: loadRealManifest(t)}
}

// specterWorld is a temp HOME with a specter root holding one store folder.
func specterWorld(t *testing.T, exec specterExec) (check.Env, string) {
	t.Helper()
	env := newEnv(t)
	root := filepath.Join(env.Home, "dev", "specter")
	writeFile(t, root, "a/.openspec-store/store.yaml", "id: a\n")
	env.Exec = exec
	env.LookupEnv = func(string) (string, bool) { return "", false }
	return env, root
}

func registered(root string) string {
	return `{"stores":[{"id":"a","root":"` + filepath.Join(root, "a") + `"}],"status":[]}`
}

func TestRealManifest_WizardBlocksSpecterWhenOpenspecIsMissing(t *testing.T) {
	m := loadRealManifest(t)

	missing := action.MissingTools(specterExec{missing: []string{toolOpenspec}}, m)

	var dependents []string
	for _, dep := range missing {
		if dep.Tool == toolOpenspec {
			dependents = dep.Dependents
		}
	}
	if !slices.Contains(dependents, componentSpecter) {
		t.Errorf("openspec dependents = %v, want them to include %q", dependents, componentSpecter)
	}
}

func TestRealManifest_InstallPlanRegistersTheUnregisteredStore(t *testing.T) {
	env, root := specterWorld(t, specterExec{listing: `{"stores":[],"status":[]}`})

	p, err := action.PlanInstall(context.Background(), env, realTarget(t), []string{componentSpecter})

	if err != nil {
		t.Fatalf("PlanInstall: %v", err)
	}
	want := "openspec store register " + filepath.Join(root, "a") + " --id a --yes"
	var got []string
	for _, s := range p.Steps {
		if e, ok := s.(plan.Exec); ok {
			got = append(got, strings.Join(e.Argv, " "))
		}
	}
	if !slices.Equal(got, []string{want}) {
		t.Errorf("exec steps = %v, want [%s]", got, want)
	}
}

func TestRealManifest_InstallPlanSkipsSpecterWhenOpenspecIsMissing(t *testing.T) {
	env, _ := specterWorld(t, specterExec{missing: []string{toolOpenspec}})

	p, err := action.PlanInstall(context.Background(), env, realTarget(t), []string{componentSpecter})

	if err != nil {
		t.Fatalf("PlanInstall: %v", err)
	}
	if len(p.Steps) != 0 || len(p.Skipped) != 1 || p.Skipped[0].Name != componentSpecter || !strings.Contains(p.Skipped[0].Reason, toolOpenspec) {
		t.Errorf("plan = steps %v, skipped %+v; want no steps and specter skipped for missing openspec", p.Steps, p.Skipped)
	}
}

// The Update tab lists only drift components, so never reporting drift keeps specter out of it.
func TestRealManifest_DashboardShowsSpecterNeverDrift(t *testing.T) {
	tests := []struct {
		name         string
		unregistered bool
		openspecGone bool
		want         drift.State
	}{
		{"registered store", false, false, drift.StateInstalled},
		{"unregistered store", true, false, drift.StateNotInstalled},
		{"openspec missing", false, true, drift.StateSkipped},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := specterExec{listing: `{"stores":[],"status":[]}`}
			if tt.openspecGone {
				exec.missing = []string{toolOpenspec}
			}
			env, root := specterWorld(t, exec)
			if !tt.unregistered {
				// The store root lives under the temp HOME, known only after the world is built.
				exec.listing = registered(root)
				env.Exec = exec
			}

			d := dashboard.Compute(context.Background(), env, realTarget(t))

			var got *drift.Status
			for i := range d.Components {
				if d.Components[i].Name == componentSpecter {
					got = &d.Components[i]
				}
			}
			if got == nil {
				t.Fatalf("dashboard components %+v have no %q", d.Components, componentSpecter)
			}
			if got.State != tt.want {
				t.Errorf("specter state = %q, want %q (never %q)", got.State, tt.want, drift.StateDrift)
			}
		})
	}
}

// TestRootFiles_MatchTheDeployingComponent pins D6: HomePath is the deploying component target joined with the file's path under its source.
func TestRootFiles_MatchTheDeployingComponent(t *testing.T) {
	m := loadRealManifest(t)
	for _, f := range specter.RootFiles {
		t.Run(f.Label, func(t *testing.T) {
			var homePaths []string
			for _, c := range m.Components {
				if c.Source == "" || c.Target == "" {
					continue
				}
				if rel, ok := strings.CutPrefix(f.RepoPath, c.Source+"/"); ok {
					homePaths = append(homePaths, filepath.Join(c.Target, rel))
				} else if f.RepoPath == c.Source {
					homePaths = append(homePaths, c.Target)
				}
			}
			if !slices.Contains(homePaths, f.HomePath) {
				t.Errorf("HomePath %q not produced by any component for %q (candidates %v)", f.HomePath, f.RepoPath, homePaths)
			}
		})
	}
}
