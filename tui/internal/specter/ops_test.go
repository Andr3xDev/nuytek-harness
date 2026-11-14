package specter_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

var specterComponent = manifest.Component{Name: "specter", Kind: manifest.KindSpecter}

func specterTarget() target.Target {
	return target.Target{
		Manifest: manifest.Manifest{Components: []manifest.Component{specterComponent}},
		Spec: target.Spec{
			ID: "t", Name: "t",
			Special: map[manifest.Kind]plan.KindOps{manifest.KindSpecter: specter.Ops},
			BackupDirs: func(home, _ string) []backup.Dir {
				return []backup.Dir{{Path: filepath.Join(home, ".bk"), Label: "home", Kind: backup.StoreHome, Base: home}}
			},
		},
	}
}

func TestOps_OnlyInstallIsDefined(t *testing.T) {
	if specter.Ops.Install == nil {
		t.Error("Ops.Install is nil, want the register part")
	}
	if specter.Ops.Hard != nil {
		t.Error("Ops.Hard is set, want nil so Hard install runs Install")
	}
	if specter.Ops.Remove != nil {
		t.Error("Ops.Remove is set, want nil (not removable)")
	}
}

func TestOpsInstall_RegistersUnregisteredStore(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "a", "id: a\n")
	env.Exec = openspecExec(storeListJSON(root))

	part, err := specter.Ops.Install(context.Background(), env, specterComponent)

	if err != nil {
		t.Fatal(err)
	}
	if len(part.Steps) != 1 {
		t.Fatalf("steps = %+v, want one register Exec", part.Steps)
	}
	if e, ok := part.Steps[0].(plan.Exec); !ok || strings.Join(e.Argv, " ") != registerCommand(filepath.Join(root, "a"), "a") {
		t.Errorf("step = %+v", part.Steps[0])
	}
}

func TestOpsInstall_RootMissingOrOutsideHomeBecomesSkipNotError(t *testing.T) {
	missing, _ := homeEnv(t, openspecExec(storeListJSON("")))
	outsideRoot := t.TempDir()
	outside := envWith(t, openspecExec(storeListJSON(outsideRoot)), outsideRoot)

	for name, env := range map[string]check.Env{"root missing": missing, "root outside HOME": outside} {
		t.Run(name, func(t *testing.T) {
			part, err := specter.Ops.Install(context.Background(), env, specterComponent)

			if err != nil {
				t.Fatalf("err = %v, want nil so install-all keeps the other components", err)
			}
			if len(part.Steps) != 0 {
				t.Errorf("steps = %+v, want none", part.Steps)
			}
			if len(part.Skipped) != 1 || part.Skipped[0].Name != "specter" || part.Skipped[0].Reason == "" {
				t.Errorf("Skipped = %+v, want specter with a reason", part.Skipped)
			}
		})
	}
}

func TestOpsInstall_UnreadableRootIsNotAdvisedAsMissing(t *testing.T) {
	if os.Geteuid() == 0 || runtime.GOOS == "windows" {
		t.Skip("permissions are not enforced for this user")
	}
	env, root := homeEnv(t, openspecExec(storeListJSON("")))
	writeStore(t, root, "a", "id: a\n")
	if err := os.Chmod(root, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })

	part, err := specter.Ops.Install(context.Background(), env, specterComponent)

	if err != nil {
		t.Fatal(err)
	}
	if len(part.Skipped) != 1 || !strings.Contains(part.Skipped[0].Reason, "unreadable") || strings.Contains(part.Skipped[0].Reason, "clone") {
		t.Errorf("Skipped = %+v, want an unreadable-root reason without a clone hint", part.Skipped)
	}
}

func TestOpsInstall_DuplicateStoreIDIsRegisteredOnce(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "a", "id: a\n")
	writeStore(t, root, "a-copy", "id: a\n")
	env.Exec = openspecExec(storeListJSON(root))

	part, err := specter.Ops.Install(context.Background(), env, specterComponent)

	if err != nil {
		t.Fatal(err)
	}
	if len(part.Steps) != 1 {
		t.Errorf("steps = %+v, want one register", part.Steps)
	}
	if len(part.Skipped) != 1 || !strings.Contains(part.Skipped[0].Reason, "duplicate store id a") {
		t.Errorf("Skipped = %+v, want the duplicate folder skipped", part.Skipped)
	}
}

func TestPlanInstall_RegistersStoreThroughSpecterOps(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "a", "id: a\n")
	env.Exec = openspecExec(storeListJSON(root))

	p, err := action.PlanInstall(context.Background(), env, specterTarget(), []string{"specter"})

	if err != nil {
		t.Fatal(err)
	}
	if got := execSteps(p); len(got) != 1 || strings.Join(got[0].Argv, " ") != registerCommand(filepath.Join(root, "a"), "a") {
		t.Errorf("exec steps = %+v, want the register for a", got)
	}
}

func TestPlanHardInstall_RegistersUnregisteredStoreAndNeverUnregisters(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "a", "id: a\n")
	writeStore(t, root, "b", "id: b\n")
	env.Exec = openspecExec(storeListJSON(root, "a"))

	p, err := action.PlanHardInstall(context.Background(), env, specterTarget())

	if err != nil {
		t.Fatal(err)
	}
	execs := execSteps(p)
	if len(execs) != 1 || strings.Join(execs[0].Argv, " ") != registerCommand(filepath.Join(root, "b"), "b") {
		t.Fatalf("exec steps = %+v, want only the register for b", execs)
	}
	for _, s := range p.Steps {
		if strings.Contains(s.Describe(), "unregister") {
			t.Errorf("step %q unregisters a store", s.Describe())
		}
	}
}

func TestPlanRemove_SpecterIsRejectedAndNothingChanges(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "a", "id: a\n")
	exec := openspecExec(storeListJSON(root, "a"))
	env.Exec = exec

	p, err := action.PlanRemove(context.Background(), env, specterTarget(), []string{"specter"})

	if err == nil || !strings.Contains(err.Error(), "specter") {
		t.Errorf("err = %v, want a refusal naming specter", err)
	}
	if len(p.Steps) != 0 {
		t.Errorf("steps = %+v, want none", p.Steps)
	}
	if calls := exec.callsTo("openspec store unregister"); len(calls) != 0 {
		t.Errorf("remove ran %v", calls)
	}
}

func evaluate(t *testing.T, env check.Env) drift.Status {
	t.Helper()
	return specter.Evaluate(context.Background(), env, specterComponent)
}

func TestEvaluate_SkippedWhenAPrerequisiteIsAbsent(t *testing.T) {
	t.Run("openspec missing", func(t *testing.T) {
		env, root := homeEnv(t, &fakeExec{})
		writeStore(t, root, "a", "id: a\n")

		got := evaluate(t, env)

		if got.State != drift.StateSkipped || !strings.Contains(got.Reason, "openspec not found") {
			t.Errorf("status = %+v, want skipped: openspec not found", got)
		}
	})
	t.Run("root missing", func(t *testing.T) {
		env, _ := homeEnv(t, openspecExec(storeListJSON("")))

		got := evaluate(t, env)

		if got.State != drift.StateSkipped || !strings.Contains(got.Reason, "specter root not found") {
			t.Errorf("status = %+v, want skipped: specter root not found", got)
		}
	})
	t.Run("root outside HOME", func(t *testing.T) {
		outsideRoot := t.TempDir()
		writeStore(t, outsideRoot, "a", "id: a\n")
		env := envWith(t, openspecExec(storeListJSON(outsideRoot, "a")), outsideRoot)

		got := evaluate(t, env)

		if got.State != drift.StateSkipped || !strings.Contains(got.Reason, "outside HOME") {
			t.Errorf("status = %+v, want skipped: outside HOME", got)
		}
	})
	t.Run("root without store folders", func(t *testing.T) {
		env, root := homeEnv(t, nil)
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		env.Exec = openspecExec(storeListJSON(root))

		got := evaluate(t, env)

		if got.State != drift.StateSkipped || !strings.Contains(got.Reason, "no stores found") {
			t.Errorf("status = %+v, want skipped: no stores found", got)
		}
	})
}

func TestEvaluate_UnknownWhenStateCannotBeRead(t *testing.T) {
	t.Run("registry query fails", func(t *testing.T) {
		env, root := homeEnv(t, nil)
		writeStore(t, root, "a", "id: a\n")
		env.Exec = openspecExec(check.ExecResult{ExitCode: 1, Stderr: []byte("boom")})

		got := evaluate(t, env)

		if got.State != drift.StateUnknown || !strings.Contains(got.Reason, "boom") {
			t.Errorf("status = %+v, want unknown naming the error", got)
		}
	})
	t.Run("store folder unreadable", func(t *testing.T) {
		if os.Geteuid() == 0 || runtime.GOOS == "windows" {
			t.Skip("permission bits are not enforced")
		}
		env, root := homeEnv(t, nil)
		writeStore(t, root, "a", "id: a\n")
		locked := filepath.Join(root, "a", ".openspec-store")
		if err := os.Chmod(locked, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
		env.Exec = openspecExec(storeListJSON(root, "a"))

		got := evaluate(t, env)

		if got.State != drift.StateUnknown || got.Reason == "" {
			t.Errorf("status = %+v, want unknown with a reason", got)
		}
	})
}

func TestEvaluate_InstalledOnlyWhenEveryFolderIsRegisteredAtItself(t *testing.T) {
	tests := []struct {
		name    string
		listing func(root string) check.ExecResult
		want    drift.State

		wantReason string
	}{
		{"all registered at their folders", func(root string) check.ExecResult { return storeListJSON(root, "a", "b") }, drift.StateInstalled, ""},
		{"one unregistered", func(root string) check.ExecResult { return storeListJSON(root, "a") }, drift.StateNotInstalled, ""},
		{"one registered elsewhere", func(root string) check.ExecResult {
			return check.ExecResult{Stdout: []byte(`{"stores":[{"id":"a","root":"` + filepath.Join(root, "a") + `"},{"id":"b","root":"/old/b"}]}`)}
		}, drift.StateSkipped, "registered at /old/b"},
		{"one elsewhere and one unregistered", func(root string) check.ExecResult {
			return check.ExecResult{Stdout: []byte(`{"stores":[{"id":"b","root":"/old/b"}]}`)}
		}, drift.StateNotInstalled, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, root := homeEnv(t, nil)
			writeStore(t, root, "a", "id: a\n")
			writeStore(t, root, "b", "id: b\n")
			env.Exec = openspecExec(tt.listing(root))

			got := evaluate(t, env)

			if got.State != tt.want || got.Reason != tt.wantReason {
				t.Errorf("state = %q (reason %q), want %q (reason %q)", got.State, got.Reason, tt.want, tt.wantReason)
			}
			if got.Name != "specter" {
				t.Errorf("Name = %q, want specter", got.Name)
			}
		})
	}
}
