package specter_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/dryrun"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/writer"
)

const mappedRemote = "git@github.com:example/gamma-app.git"

// initFixture is a HOME with a root holding repos.yaml and the given unregistered stores,
// and a git repo under HOME with origin originURL as the working directory.
type initFixture struct {
	env       check.Env
	exec      *fakeExec
	home      string
	root      string
	reposPath string
	repoDir   string
}

func newInitFixture(t *testing.T, storeIDs ...string) initFixture {
	t.Helper()
	exec := openspecExec(check.ExecResult{Stdout: []byte(`{"stores":[]}`)})
	env, root := homeEnv(t, exec)
	for _, id := range storeIDs {
		writeStore(t, root, id, "id: "+id+"\n")
	}
	reposPath := filepath.Join(root, "repos.yaml")
	writeFile(t, reposPath, realReposYAML)
	repoDir := filepath.Join(env.Home, "work", "proj")
	writeFile(t, filepath.Join(repoDir, ".git", "config"), "[remote \"origin\"]\n\turl = "+originURL+"\n")
	env.Cwd = repoDir
	return initFixture{env: env, exec: exec, home: env.Home, root: root, reposPath: reposPath, repoDir: repoDir}
}

func editStep(t *testing.T, p plan.Plan) plan.Edit {
	t.Helper()
	var edits []plan.Edit
	for _, s := range p.Steps {
		if e, ok := s.(plan.Edit); ok {
			edits = append(edits, e)
		}
	}
	if len(edits) != 1 {
		t.Fatalf("plan has %d Edit steps, want 1 (steps: %+v)", len(edits), p.Steps)
	}
	return edits[0]
}

func hasEdit(p plan.Plan) bool {
	for _, s := range p.Steps {
		if _, ok := s.(plan.Edit); ok {
			return true
		}
	}
	return false
}

// reasons joins everything a plan shows to explain what it left out.
func reasons(p plan.Plan) string {
	var b strings.Builder
	for _, s := range p.Skipped {
		b.WriteString(s.Name + " " + s.Reason + "\n")
	}
	b.WriteString(strings.Join(p.Notes, "\n"))
	return b.String()
}

func TestPlanFastInit_UnmappedRepoPlansRegisterThenMappingEdit(t *testing.T) {
	fx := newInitFixture(t, "a")

	p, err := specter.PlanFastInit(context.Background(), fx.env, "a")

	if err != nil {
		t.Fatal(err)
	}
	if len(p.Steps) != 2 {
		t.Fatalf("steps = %+v, want register Exec then repos.yaml Edit", p.Steps)
	}
	if e, ok := p.Steps[0].(plan.Exec); !ok || strings.Join(e.Argv, " ") != registerCommand(filepath.Join(fx.root, "a"), "a") {
		t.Errorf("step 0 = %+v, want the register Exec", p.Steps[0])
	}
	edit := editStep(t, p)
	if edit.Path != fx.reposPath {
		t.Errorf("Edit.Path = %q, want %q", edit.Path, fx.reposPath)
	}
	got, err := edit.Edit([]byte(realReposYAML), true)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(realReposYAML, "stores:\n", "  - remote: "+originURL+"\n    store: a\nstores:\n", 1)
	if string(got) != want {
		t.Errorf("edit result =\n%s\nwant\n%s", got, want)
	}
}

func TestPlanFastInit_IsNonDestructiveAndSnapshotFree(t *testing.T) {
	fx := newInitFixture(t, "a")

	p, err := specter.PlanFastInit(context.Background(), fx.env, "a")

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
	if p.Empty() {
		t.Error("plan is empty, want rows")
	}
}

func TestPlanFastInit_RepoWithoutRemoteMapsByHomeRelativePath(t *testing.T) {
	fx := newInitFixture(t, "a")
	writeFile(t, filepath.Join(fx.repoDir, ".git", "config"), "[core]\n\tbare = false\n")

	p, err := specter.PlanFastInit(context.Background(), fx.env, "a")

	if err != nil {
		t.Fatal(err)
	}
	got, err := editStep(t, p).Edit([]byte(realReposYAML), true)
	if err != nil {
		t.Fatal(err)
	}
	if want := "  - path: ~/work/proj\n    store: a\nstores:\n"; !strings.Contains(string(got), want) {
		t.Errorf("edit result =\n%s\nwant it to contain %q", got, want)
	}
}

func TestPlanFastInit_EmptyChoicePlansNoMapping(t *testing.T) {
	fx := newInitFixture(t, "a")

	p, err := specter.PlanFastInit(context.Background(), fx.env, "")

	if err != nil {
		t.Fatal(err)
	}
	if hasEdit(p) {
		t.Error("plan holds an Edit, want registration only")
	}
	if len(execSteps(p)) != 1 {
		t.Errorf("steps = %+v, want the register Exec", p.Steps)
	}
}

func TestPlanFastInit_MappedRepoPlansNoEdit(t *testing.T) {
	fx := newInitFixture(t, "gamma-app")
	writeFile(t, filepath.Join(fx.repoDir, ".git", "config"), "[remote \"origin\"]\n\turl = "+mappedRemote+"\n")

	p, err := specter.PlanFastInit(context.Background(), fx.env, "gamma-app")

	if err != nil {
		t.Fatal(err)
	}
	if hasEdit(p) {
		t.Error("plan holds an Edit for an already mapped repo")
	}
}

func TestPlanFastInit_MappingIsSkippedWithReasonWhileRegistrationStays(t *testing.T) {
	tests := []struct {
		name       string
		arrange    func(fx *initFixture)
		mapTo      string
		wantReason string
	}{
		{"not in a git repo", func(fx *initFixture) { fx.env.Cwd = fx.home }, "a", "git"},
		{"repos.yaml missing", func(fx *initFixture) {
			if err := os.Remove(fx.reposPath); err != nil {
				t.Fatal(err)
			}
		}, "a", "repos.yaml"},
		{"repos.yaml invalid", func(fx *initFixture) { writeFile(t, fx.reposPath, "repos: [unclosed\n  - : :") }, "a", "repos.yaml invalid"},
		{"store to map to is unknown", func(*initFixture) {}, "ghost", "ghost"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := newInitFixture(t, "a")
			tt.arrange(&fx)

			p, err := specter.PlanFastInit(context.Background(), fx.env, tt.mapTo)

			if err != nil {
				t.Fatal(err)
			}
			if hasEdit(p) {
				t.Error("plan holds an Edit, want the mapping skipped")
			}
			if len(execSteps(p)) != 1 {
				t.Errorf("steps = %+v, want the register Exec kept", p.Steps)
			}
			if got := reasons(p); !strings.Contains(got, tt.wantReason) {
				t.Errorf("reasons = %q, want %q mentioned", got, tt.wantReason)
			}
		})
	}
}

func TestPlanFastInit_NoStoreToChooseGivesReason(t *testing.T) {
	fx := newInitFixture(t)

	p, err := specter.PlanFastInit(context.Background(), fx.env, "")

	if err != nil {
		t.Fatal(err)
	}
	if hasEdit(p) {
		t.Error("plan holds an Edit, want none")
	}
	if got := reasons(p); !strings.Contains(got, "no registered store") {
		t.Errorf("reasons = %q, want %q", got, "no registered store to choose")
	}
}

func TestPlanFastInit_RootOutsideHomeIsEmptyWithReason(t *testing.T) {
	outside := t.TempDir()
	writeStore(t, outside, "a", "id: a\n")
	writeFile(t, filepath.Join(outside, "repos.yaml"), realReposYAML)
	env := envWith(t, openspecExec(check.ExecResult{Stdout: []byte(`{"stores":[]}`)}), outside)

	p, err := specter.PlanFastInit(context.Background(), env, "a")

	if err != nil {
		t.Fatal(err)
	}
	if len(p.Steps) != 0 {
		t.Errorf("steps = %+v, want none", p.Steps)
	}
	if got := reasons(p); !strings.Contains(got, "outside HOME") {
		t.Errorf("reasons = %q, want the outside-HOME reason", got)
	}
}

// fakeOpenspec writes a script that records its arguments to the returned log, standing in for the real CLI.
func fakeOpenspec(t *testing.T) (script, log string) {
	t.Helper()
	dir := t.TempDir()
	script, log = filepath.Join(dir, "openspec"), filepath.Join(dir, "calls.log")
	body := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + log + "'\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return script, log
}

// pointExecAt swaps the PATH lookup of `openspec` for the script, so the test never relies on or alters PATH.
func pointExecAt(t *testing.T, p plan.Plan, script string) {
	t.Helper()
	for i, s := range p.Steps {
		if e, ok := s.(plan.Exec); ok {
			if e.Argv[0] != "openspec" {
				t.Fatalf("exec argv[0] = %q, want openspec", e.Argv[0])
			}
			e.Argv = append([]string{script}, e.Argv[1:]...)
			p.Steps[i] = e
		}
	}
}

func readFileOrEmpty(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestFastInit_ApplyRegistersMapsAndCreatesNoSnapshot(t *testing.T) {
	fx := newInitFixture(t, "alpha")
	p, err := specter.PlanFastInit(context.Background(), fx.env, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	script, log := fakeOpenspec(t)
	pointExecAt(t, p, script)

	result := plan.Apply(context.Background(), writer.New(time.Now), p, func(plan.Event) {})

	if result.Failed != nil {
		t.Fatalf("apply failed: %+v", result.Failed)
	}
	if want := "store register " + filepath.Join(fx.root, "alpha") + " --id alpha --yes\n"; readFileOrEmpty(t, log) != want {
		t.Errorf("fake openspec saw %q, want %q", readFileOrEmpty(t, log), want)
	}
	want := strings.Replace(realReposYAML, "stores:\n", "  - remote: "+originURL+"\n    store: alpha\nstores:\n", 1)
	if got := readFileOrEmpty(t, fx.reposPath); got != want {
		t.Errorf("repos.yaml =\n%s\nwant (all comments and flow maps byte-identical)\n%s", got, want)
	}
	if result.Stamp != "" {
		t.Errorf("Stamp = %q, want no snapshot", result.Stamp)
	}
	if _, err := os.Stat(filepath.Join(fx.home, ".specter-backups")); !os.IsNotExist(err) {
		t.Errorf("stat .specter-backups err = %v, want not exist", err)
	}
}

func TestFastInit_DryRunRunsNothingAndListsStepsAsNotExecuted(t *testing.T) {
	fx := newInitFixture(t, "alpha")
	p, err := specter.PlanFastInit(context.Background(), fx.env, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	script, log := fakeOpenspec(t)
	pointExecAt(t, p, script)
	var events []plan.Event

	result := plan.Apply(context.Background(), dryrun.New(), p, func(e plan.Event) { events = append(events, e) })

	if result.Failed != nil {
		t.Fatalf("dry-run failed: %+v", result.Failed)
	}
	if got := readFileOrEmpty(t, log); got != "" {
		t.Errorf("fake openspec ran under dry-run: %q", got)
	}
	if got := readFileOrEmpty(t, fx.reposPath); got != realReposYAML {
		t.Errorf("repos.yaml changed under dry-run:\n%s", got)
	}
	closing := 0
	for _, e := range events {
		if e.Phase == plan.PhaseStart || e.Phase == plan.PhaseLine {
			continue
		}
		closing++
		if e.Phase != plan.PhaseSkipped || e.Text != "not executed" {
			t.Errorf("event %+v, want skipped with text %q", e, "not executed")
		}
	}
	if closing != len(p.Steps) {
		t.Errorf("%d closing events for %d steps", closing, len(p.Steps))
	}
}
