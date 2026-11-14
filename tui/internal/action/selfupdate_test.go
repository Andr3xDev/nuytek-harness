package action_test

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

// statusExec answers the porcelain probe with a canned result and records every call.
type statusExec struct {
	fakeExec
	result check.ExecResult
	err    error
	calls  [][]string
}

func (s *statusExec) Run(_ context.Context, name string, args ...string) (check.ExecResult, error) {
	s.calls = append(s.calls, append([]string{name}, args...))
	return s.result, s.err
}

func selfUpdateEnv(f *fixture, ex *statusExec) check.Env {
	env := f.env()
	env.Exec = ex
	return env
}

func TestPlanSelfUpdateClean(t *testing.T) {
	f := newFixture(t)
	ex := &statusExec{}
	p, err := action.PlanSelfUpdate(context.Background(), selfUpdateEnv(f, ex))
	if err != nil {
		t.Fatal(err)
	}

	wantProbe := []string{"git", "-C", f.repo, "status", "--porcelain", "--untracked-files=no"}
	if len(ex.calls) != 1 || !reflect.DeepEqual(ex.calls[0], wantProbe) {
		t.Errorf("probe calls = %v, want [%v]", ex.calls, wantProbe)
	}

	execs := stepsOf[plan.Exec](p)
	if len(execs) != 2 {
		t.Fatalf("plan has %d exec steps, want 2 (steps: %v)", len(execs), p.Steps)
	}
	wantArgv := [][]string{
		{"env", "GIT_TERMINAL_PROMPT=0", "git", "-C", f.repo, "pull", "--ff-only"},
		{"bash", filepath.Join(f.repo, "scripts", "bootstrap-tui.sh")},
	}
	wantTimeouts := []time.Duration{2 * time.Minute, 5 * time.Minute}
	for i, s := range execs {
		if !reflect.DeepEqual(s.Argv, wantArgv[i]) {
			t.Errorf("step %d argv = %v, want %v", i, s.Argv, wantArgv[i])
		}
		if s.Timeout != wantTimeouts[i] {
			t.Errorf("step %d timeout = %v, want %v", i, s.Timeout, wantTimeouts[i])
		}
	}
	if p.Confirm == "" {
		t.Error("Confirm is empty, want a confirmation prompt")
	}
	notes := strings.Join(p.Notes, "\n")
	for _, want := range []string{f.repo, "restart the TUI afterwards"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes %q missing %q", notes, want)
		}
	}
}

func TestPlanSelfUpdateRefuses(t *testing.T) {
	tests := []struct {
		name    string
		ex      *statusExec
		wantErr string
	}{
		{
			name:    "dirty tree",
			ex:      &statusExec{result: check.ExecResult{Stdout: []byte(" M tui/main.go\n")}},
			wantErr: "uncommitted changes",
		},
		{
			name:    "non-zero exit carries stderr",
			ex:      &statusExec{result: check.ExecResult{ExitCode: 128, Stderr: []byte("fatal: not a git repository")}},
			wantErr: "fatal: not a git repository",
		},
		{
			name:    "command did not start",
			ex:      &statusExec{err: errors.New("exec: git: not found")},
			wantErr: "git: not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := action.PlanSelfUpdate(context.Background(), selfUpdateEnv(newFixture(t), tt.ex))
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
			if len(p.Steps) != 0 || p.Confirm != "" {
				t.Errorf("plan = %+v, want zero plan on refusal", p)
			}
		})
	}
}
