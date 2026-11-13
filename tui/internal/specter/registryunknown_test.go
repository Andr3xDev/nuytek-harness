package specter_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

func TestPlanMaintenance_RegistryUnknownPlansNothingAndSaysWhy(t *testing.T) {
	env, root := homeEnv(t, nil)
	writeStore(t, root, "a", "id: a\n")
	env.Exec = openspecExec(check.ExecResult{ExitCode: 1, Stderr: []byte("registry broken")})

	p, err := specter.PlanMaintenance(context.Background(), env)

	if err != nil {
		t.Fatal(err)
	}
	if len(p.Steps) != 0 {
		t.Errorf("steps = %+v, want none while the registry is unknown", p.Steps)
	}
	if notes := strings.Join(p.Notes, "\n"); !strings.Contains(notes, "registry broken") {
		t.Errorf("Notes = %q, want the registry error", notes)
	}
}

func TestPlanFastInit_AlreadyMappedRepoSaysSo(t *testing.T) {
	fx := newInitFixture(t, "a")
	writeFile(t, fx.repoDir+"/.git/config", "[remote \"origin\"]\n\turl = "+mappedRemote+"\n")

	p, err := specter.PlanFastInit(context.Background(), fx.env, "a")

	if err != nil {
		t.Fatal(err)
	}
	if got := reasons(p); !strings.Contains(got, "already mapped to store gamma-app") {
		t.Errorf("reasons = %q, want the existing mapping named", got)
	}
}
