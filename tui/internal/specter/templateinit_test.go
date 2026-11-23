package specter_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

const (
	ghLogin       = "octo"
	ghAuthCommand = "gh auth status"
	ghUserCommand = "gh api user --jq .login"
	ghViewCommand = "gh repo view " + ghLogin + "/specter"
	// Real gh stderr for an absent repo; a bad token or network failure never contains it.
	ghNotFoundStderr = "GraphQL: Could not resolve to a Repository with the name 'octo/specter'. (repository)"
)

// templateFixture has every tool installed, gh authenticated as ghLogin, no repo `specter` and a missing root whose parent exists.
type templateFixture struct {
	env  check.Env
	exec *fakeExec
	root string
}

func newTemplateFixture(t *testing.T) templateFixture {
	t.Helper()
	exec := &fakeExec{
		onPath: map[string]bool{"git": true, "openspec": true, "gh": true},
		results: map[string]check.ExecResult{
			ghAuthCommand:    {},
			ghUserCommand:    {Stdout: []byte(ghLogin + "\n")},
			ghViewCommand:    {ExitCode: 1, Stderr: []byte(ghNotFoundStderr)},
			storeListCommand: storeListJSON(""),
		},
	}
	env, root := homeEnv(t, exec)
	if err := os.MkdirAll(filepath.Dir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	return templateFixture{env: env, exec: exec, root: root}
}

func blockedReason(t *testing.T, err error) string {
	t.Helper()
	var blocked specter.ErrBlocked
	if !errors.As(err, &blocked) {
		t.Fatalf("err = %v (%T), want specter.ErrBlocked", err, err)
	}
	return blocked.Reason
}

func TestInitBlocker(t *testing.T) {
	tests := []struct {
		name       string
		arrange    func(fx *templateFixture)
		wantReason []string // every fragment must appear; nil means no blocker
		wantNoGh   bool
	}{
		{"all dependencies pass and the name is free", func(*templateFixture) {}, nil, false},
		{"root outside HOME", func(fx *templateFixture) { fx.env.Saved.SpecterRoot = t.TempDir() }, []string{"outside HOME"}, true},
		{"git missing", func(fx *templateFixture) { delete(fx.exec.onPath, "git") }, []string{"git", "not found"}, true},
		{"openspec missing", func(fx *templateFixture) { delete(fx.exec.onPath, "openspec") }, []string{"openspec", "not found"}, true},
		{"gh missing", func(fx *templateFixture) { delete(fx.exec.onPath, "gh") }, []string{"gh", "not found"}, true},
		{"gh not authenticated", func(fx *templateFixture) { fx.exec.results[ghAuthCommand] = check.ExecResult{ExitCode: 1} }, []string{"gh", "not authenticated"}, false},
		{"parent missing", func(fx *templateFixture) {
			fx.env.Saved.SpecterRoot = filepath.Join(fx.env.Home, "absent", "specter")
			fx.root = fx.env.Saved.SpecterRoot
		}, []string{"parent", "absent"}, false},
		{"repo already exists names the clone line", func(fx *templateFixture) {
			fx.exec.results[ghViewCommand] = check.ExecResult{}
		}, []string{"already exists", "gh repo clone " + ghLogin + "/specter {root}"}, false},
		{"any other gh failure blocks with its stderr line", func(fx *templateFixture) {
			fx.exec.results[ghViewCommand] = check.ExecResult{ExitCode: 1, Stderr: []byte("HTTP 401: Bad credentials (https://api.github.com/graphql)\nTry authenticating with: gh auth login")}
		}, []string{"HTTP 401: Bad credentials"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := newTemplateFixture(t)
			tt.arrange(&fx)

			err := specter.InitBlocker(context.Background(), fx.env, specter.ResolveRoot(fx.env.Saved.SpecterRoot, fx.env.Home))

			if tt.wantReason == nil {
				if err != nil {
					t.Fatalf("err = %v, want no blocker", err)
				}
				return
			}
			reason := blockedReason(t, err)
			for _, fragment := range tt.wantReason {
				fragment = strings.ReplaceAll(fragment, "{root}", fx.root)
				if !strings.Contains(reason, fragment) {
					t.Errorf("reason = %q, want %q mentioned", reason, fragment)
				}
			}
			if tt.wantNoGh {
				if calls := append(fx.exec.callsTo("gh"), fx.exec.callsTo("git")...); len(calls) != 0 {
					t.Errorf("ran %v, want no gh or git command before the blocker", calls)
				}
			}
		})
	}
}

func TestInitBlocker_ReportsOnlyTheFirstMissingTool(t *testing.T) {
	fx := newTemplateFixture(t)
	delete(fx.exec.onPath, "git")
	delete(fx.exec.onPath, "gh")

	err := specter.InitBlocker(context.Background(), fx.env, specter.ResolveRoot(fx.env.Saved.SpecterRoot, fx.env.Home))

	reason := blockedReason(t, err)
	if !strings.Contains(reason, "git") || strings.Contains(reason, "gh") {
		t.Errorf("reason = %q, want git alone", reason)
	}
}

func TestPlanFastInit_MissingRootPlansCreateThenCloneWithNoSnapshot(t *testing.T) {
	fx := newTemplateFixture(t)

	p, err := specter.PlanFastInit(context.Background(), fx.env, "")

	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"gh repo create " + ghLogin + "/specter --template Andr3xDev/specter-template --private",
		"gh repo clone " + ghLogin + "/specter " + fx.root,
	}
	if len(p.Steps) != len(want)+1 {
		t.Fatalf("steps = %+v, want the create and clone Exec steps then the repos.yaml check", p.Steps)
	}
	for i, line := range want {
		exec, ok := p.Steps[i].(plan.Exec)
		if !ok || strings.Join(exec.Argv, " ") != line {
			t.Errorf("step %d = %+v, want Exec %q", i, p.Steps[i], line)
		}
	}
}

func TestPlanFastInit_CloneRetriesAndReposYAMLIsVerifiedAfterwards(t *testing.T) {
	fx := newTemplateFixture(t)

	p, err := specter.PlanFastInit(context.Background(), fx.env, "")

	if err != nil || len(p.Steps) != 3 {
		t.Fatalf("plan = %+v (err %v), want create, clone and verify steps", p.Steps, err)
	}
	create, clone := p.Steps[0].(plan.Exec), p.Steps[1].(plan.Exec)
	if create.Attempts > 1 {
		t.Errorf("create Attempts = %d, want one run: repeating a create would hit an existing repo", create.Attempts)
	}
	if clone.Attempts < 2 || clone.Backoff <= 0 {
		t.Errorf("clone Attempts = %d, Backoff = %v, want a bounded retry with a wait", clone.Attempts, clone.Backoff)
	}
	verify, ok := p.Steps[2].(plan.Edit)
	if !ok || verify.Path != filepath.Join(fx.root, "repos.yaml") {
		t.Fatalf("step 3 = %+v, want an Edit of <root>/repos.yaml", p.Steps[2])
	}
	if _, err := verify.Edit(nil, false); err == nil ||
		!strings.Contains(err.Error(), "repos.yaml") || !strings.Contains(err.Error(), "git -C "+fx.root+" pull") ||
		strings.Contains(err.Error(), "gh repo clone") {
		t.Errorf("missing repos.yaml error = %v, want repos.yaml named and a git pull to recover", err)
	}
	if got, err := verify.Edit([]byte("repos: []\n"), true); err != nil || string(got) != "repos: []\n" {
		t.Errorf("existing repos.yaml: got %q, err %v, want it kept as is", got, err)
	}
}

func TestPlanFastInit_RootThatIsNotADirectoryIsBlocked(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(t *testing.T, root string)
	}{
		{"regular file", func(t *testing.T, root string) {
			if err := os.WriteFile(root, []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"dangling symlink", func(t *testing.T, root string) {
			if err := os.Symlink(filepath.Join(filepath.Dir(root), "absent-target"), root); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := newTemplateFixture(t)
			tt.arrange(t, fx.root)

			p, planErr := specter.PlanFastInit(context.Background(), fx.env, "")
			blockerErr := specter.InitBlocker(context.Background(), fx.env, specter.ResolveRoot(fx.env.Saved.SpecterRoot, fx.env.Home))

			for name, err := range map[string]error{"PlanFastInit": planErr, "InitBlocker": blockerErr} {
				if reason := blockedReason(t, err); !strings.Contains(reason, fx.root) {
					t.Errorf("%s reason = %q, want the root named", name, reason)
				}
			}
			if len(p.Steps) != 0 {
				t.Errorf("steps = %+v, want none: no gh create over a path that is in the way", p.Steps)
			}
		})
	}
}

func TestInitBlocker_CloneHintQuotesARootWithShellSpecialChars(t *testing.T) {
	fx := newTemplateFixture(t)
	fx.root = filepath.Join(fx.env.Home, "my $specs", "it's", "specter")
	wantQuoted := "'" + strings.ReplaceAll(fx.root, "'", `'\''`) + "'"
	fx.env.Saved.SpecterRoot = fx.root
	if err := os.MkdirAll(filepath.Dir(fx.root), 0o755); err != nil {
		t.Fatal(err)
	}
	fx.exec.results[ghViewCommand] = check.ExecResult{}

	err := specter.InitBlocker(context.Background(), fx.env, specter.ResolveRoot(fx.root, fx.env.Home))

	if reason := blockedReason(t, err); !strings.Contains(reason, wantQuoted) {
		t.Errorf("reason = %q, want the root single-quoted in the clone line", reason)
	}
}

func TestPlanFastInit_ExistingRootPlansNoCreateAndRunsNoGh(t *testing.T) {
	fx := newTemplateFixture(t)
	if err := os.MkdirAll(fx.root, 0o755); err != nil {
		t.Fatal(err)
	}

	p, err := specter.PlanFastInit(context.Background(), fx.env, "")

	if err != nil {
		t.Fatal(err)
	}
	for _, e := range execSteps(p) {
		if e.Argv[0] == "gh" {
			t.Errorf("step %v, want no gh step when the root exists", e.Argv)
		}
	}
	if calls := fx.exec.callsTo("gh"); len(calls) != 0 {
		t.Errorf("ran %v, want no gh command when the root exists", calls)
	}
}

func TestPlanFastInit_BlockedReturnsErrBlockedAndNoGhCommand(t *testing.T) {
	t.Run("root missing and gh absent", func(t *testing.T) {
		fx := newTemplateFixture(t)
		delete(fx.exec.onPath, "gh")

		p, err := specter.PlanFastInit(context.Background(), fx.env, "")

		if reason := blockedReason(t, err); !strings.Contains(reason, "gh") {
			t.Errorf("reason = %q, want gh named", reason)
		}
		if len(p.Steps) != 0 {
			t.Errorf("steps = %+v, want none", p.Steps)
		}
		if calls := append(fx.exec.callsTo("gh"), fx.exec.callsTo("git")...); len(calls) != 0 {
			t.Errorf("ran %v, want no git or gh command", calls)
		}
	})

	t.Run("openspec missing with an existing root", func(t *testing.T) {
		fx := newTemplateFixture(t)
		if err := os.MkdirAll(fx.root, 0o755); err != nil {
			t.Fatal(err)
		}
		delete(fx.exec.onPath, "openspec")

		p, err := specter.PlanFastInit(context.Background(), fx.env, "")

		if reason := blockedReason(t, err); !strings.Contains(reason, "openspec") {
			t.Errorf("reason = %q, want openspec named", reason)
		}
		if len(p.Steps) != 0 {
			t.Errorf("steps = %+v, want none", p.Steps)
		}
	})
}

func TestPlanMaintenance_MissingRootNoteDoesNotAdviseCloning(t *testing.T) {
	env, root := homeEnv(t, openspecExec(storeListJSON("")))

	p, err := specter.PlanMaintenance(context.Background(), env)

	if err != nil {
		t.Fatal(err)
	}
	notes := strings.Join(p.Notes, "\n")
	if !strings.Contains(notes, root) {
		t.Errorf("Notes = %q, want the missing root named", notes)
	}
	for _, gone := range []string{"git clone", "NUYTEK_SPECTER_REPO"} {
		if strings.Contains(notes, gone) {
			t.Errorf("Notes = %q, want no %q hint", notes, gone)
		}
	}
}
