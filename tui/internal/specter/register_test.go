package specter_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

// homeEnv builds an env whose default root (HOME/dev/specter) lies inside HOME.
func homeEnv(t *testing.T, exec check.Exec) (env check.Env, root string) {
	t.Helper()
	home := t.TempDir()
	root = filepath.Join(home, "dev", "specter")
	env = check.Env{
		FS: osfs.FS{}, Exec: exec, Home: home,
		LookupEnv: func(string) (string, bool) { return "", false },
	}
	return env, root
}

func registerCommand(dir, id string) string {
	return "openspec store register " + dir + " --id " + id + " --yes"
}

func openspecExec(listing check.ExecResult) *fakeExec {
	return &fakeExec{
		onPath:  map[string]bool{"openspec": true},
		results: map[string]check.ExecResult{storeListCommand: listing},
	}
}

func execSteps(p plan.Plan) []plan.Exec {
	var out []plan.Exec
	for _, s := range p.Steps {
		if e, ok := s.(plan.Exec); ok {
			out = append(out, e)
		}
	}
	return out
}

func skippedNames(skipped []plan.Skip) []string {
	names := make([]string, len(skipped))
	for i, s := range skipped {
		names[i] = s.Name
	}
	return names
}

func TestPlanMaintenance_UnregisteredStoreGetsOneRegisterExec(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "a", "id: a\n")
	env.Exec = openspecExec(storeListJSON(root))

	p, err := specter.PlanMaintenance(context.Background(), env)

	if err != nil {
		t.Fatal(err)
	}
	execs := execSteps(p)
	if len(execs) != 1 || len(p.Steps) != 1 {
		t.Fatalf("steps = %+v, want exactly one Exec", p.Steps)
	}
	if got := strings.Join(execs[0].Argv, " "); got != registerCommand(filepath.Join(root, "a"), "a") {
		t.Errorf("argv = %q", got)
	}
	if execs[0].Timeout != specter.RegisterTimeout || execs[0].Warn {
		t.Errorf("Timeout=%v Warn=%v, want RegisterTimeout and Warn=false", execs[0].Timeout, execs[0].Warn)
	}
}

func TestPlanMaintenance_IsNonDestructive(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "a", "id: a\n")
	env.Exec = openspecExec(storeListJSON(root))

	p, err := specter.PlanMaintenance(context.Background(), env)

	if err != nil {
		t.Fatal(err)
	}
	if p.Confirm != "" {
		t.Errorf("Confirm = %q, want empty so enter applies", p.Confirm)
	}
	for _, s := range p.Steps {
		if _, ok := s.(plan.Snapshot); ok {
			t.Error("plan holds a Snapshot, want none")
		}
	}
}

func TestPlanMaintenance_StoreAlreadyRegisteredPlansNothing(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "a", "id: a\n")
	env.Exec = openspecExec(storeListJSON(root, "a"))

	p, err := specter.PlanMaintenance(context.Background(), env)

	if err != nil {
		t.Fatal(err)
	}
	if !p.Empty() || len(p.Steps) != 0 {
		t.Errorf("plan = %+v, want empty", p)
	}
}

func TestPlanMaintenance_StoreRegisteredElsewhereIsSkippedWithReason(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "a", "id: a\n")
	env.Exec = openspecExec(check.ExecResult{Stdout: []byte(`{"stores":[{"id":"a","root":"/old/place/a"}]}`)})

	p, err := specter.PlanMaintenance(context.Background(), env)

	if err != nil {
		t.Fatal(err)
	}
	if len(p.Steps) != 0 {
		t.Errorf("steps = %+v, want none", p.Steps)
	}
	if len(p.Skipped) != 1 || p.Skipped[0].Name != "a" || !strings.Contains(p.Skipped[0].Reason, "/old/place/a") {
		t.Errorf("Skipped = %+v, want a with the other root in the reason", p.Skipped)
	}
}

func TestPlanMaintenance_StoreWithoutIDIsSkipped(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "noid", "name: nothing\n")
	env.Exec = openspecExec(storeListJSON(root))

	p, err := specter.PlanMaintenance(context.Background(), env)

	if err != nil {
		t.Fatal(err)
	}
	if len(p.Steps) != 0 {
		t.Errorf("steps = %+v, want none", p.Steps)
	}
	if len(p.Skipped) != 1 || p.Skipped[0].Name != filepath.Join(root, "noid") || p.Skipped[0].Reason != "no id in store.yaml" {
		t.Errorf("Skipped = %+v, want the folder with reason %q", p.Skipped, "no id in store.yaml")
	}
}

func TestPlanMaintenance_StoreWithUnsafeIDIsSkippedNotRegistered(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "bad", "id: \"--yes ../x\"\n")
	writeStore(t, root, "good", "id: good.store-1\n")
	env.Exec = openspecExec(storeListJSON(root))

	p, err := specter.PlanMaintenance(context.Background(), env)

	if err != nil {
		t.Fatal(err)
	}
	if len(p.Steps) != 1 {
		t.Fatalf("steps = %+v, want only the good store", p.Steps)
	}
	if len(p.Skipped) != 1 || p.Skipped[0].Name != filepath.Join(root, "bad") || !strings.Contains(p.Skipped[0].Reason, "invalid store id") {
		t.Errorf("Skipped = %+v, want the bad folder with an invalid id reason", p.Skipped)
	}
}

func TestPlanMaintenance_PreconditionsYieldEmptyPlanWithReason(t *testing.T) {
	t.Run("root missing names the clone command", func(t *testing.T) {
		env, root := homeEnv(t, openspecExec(storeListJSON("")))

		p, err := specter.PlanMaintenance(context.Background(), env)

		if err != nil {
			t.Fatal(err)
		}
		if len(p.Steps) != 0 {
			t.Errorf("steps = %+v, want none", p.Steps)
		}
		want := "git clone <your-specter-repo-url> " + root + " (set NUYTEK_SPECTER_REPO to fill it in)"
		if notes := strings.Join(p.Notes, "\n"); !strings.Contains(notes, want) {
			t.Errorf("Notes = %q, want %q", notes, want)
		}
	})

	t.Run("root missing uses the configured repo url", func(t *testing.T) {
		env, root := homeEnv(t, openspecExec(storeListJSON("")))
		env.LookupEnv = func(k string) (string, bool) {
			return "https://example.com/specs.git", k == "NUYTEK_SPECTER_REPO"
		}

		p, err := specter.PlanMaintenance(context.Background(), env)

		if err != nil {
			t.Fatal(err)
		}
		want := "git clone https://example.com/specs.git " + root
		notes := strings.Join(p.Notes, "\n")
		if !strings.HasSuffix(notes, want) {
			t.Errorf("Notes = %q, want suffix %q", notes, want)
		}
	})

	t.Run("openspec missing names the tool", func(t *testing.T) {
		env, root := homeEnv(t, &fakeExec{})
		writeStore(t, root, "a", "id: a\n")

		p, err := specter.PlanMaintenance(context.Background(), env)

		if err != nil {
			t.Fatal(err)
		}
		if len(p.Steps) != 0 {
			t.Errorf("steps = %+v, want none", p.Steps)
		}
		if notes := strings.Join(p.Notes, "\n"); !strings.Contains(notes, "openspec") {
			t.Errorf("Notes = %q, want openspec named", notes)
		}
	})

	t.Run("root outside HOME wins over openspec missing", func(t *testing.T) {
		outside := t.TempDir()
		writeStore(t, outside, "a", "id: a\n")
		env := envWith(t, &fakeExec{}, outside)

		p, err := specter.PlanMaintenance(context.Background(), env)

		if err != nil {
			t.Fatal(err)
		}
		if len(p.Steps) != 0 {
			t.Errorf("steps = %+v, want none", p.Steps)
		}
		if notes := strings.Join(p.Notes, "\n"); !strings.Contains(notes, "outside HOME") {
			t.Errorf("Notes = %q, want the outside-HOME reason first", notes)
		}
	})
}

func TestPlanMaintenance_LeavesDriftOnlyStateUntouched(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "a", "id: a\n")
	exec := openspecExec(storeListJSON(root, "a"))
	env.Exec = exec

	if _, err := specter.PlanMaintenance(context.Background(), env); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(env.Home, ".specter-backups")); !os.IsNotExist(err) {
		t.Errorf("stat .specter-backups err = %v, want not exist", err)
	}
	if calls := exec.callsTo("openspec store register"); len(calls) != 0 {
		t.Errorf("planning ran %v, want only the read-only list", calls)
	}
}
