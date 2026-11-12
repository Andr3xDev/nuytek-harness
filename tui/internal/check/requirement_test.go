package check_test

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

func TestRequirements(t *testing.T) {
	exec := &fakeExec{onPath: map[string]bool{"jq": true}}
	env := newHome(t, exec)
	script := filepath.Join(env.Home, "run.py")
	tool := filepath.Join(env.Home, "tool.sh")
	execTool := filepath.Join(env.Home, "exec.sh")
	sound := filepath.Join(env.Home, "done.oga")
	writeFile(t, script, "print()", 0o644)
	writeFile(t, tool, "#!/bin/sh", 0o644)
	writeFile(t, execTool, "#!/bin/sh", 0o755)
	writeFile(t, sound, "x", 0o644)

	tests := []struct {
		name        string
		req         check.Requirement
		want        check.Status
		wantMessage string
	}{
		{"interpreter script without exec bit", check.Requirement{Subject: "guard", Path: script}, check.StatusOK, ""},
		{"direct exec without exec bit", check.Requirement{Subject: "stop", Path: tool, MustExec: true}, check.StatusError, tool},
		{"direct exec with exec bit", check.Requirement{Subject: "stop-ok", Path: execTool, MustExec: true}, check.StatusOK, ""},
		{"missing script", check.Requirement{Subject: "gone", Path: filepath.Join(env.Home, "gone.py")}, check.StatusError, "gone.py"},
		{"bare command not on PATH", check.Requirement{Subject: "cg", Command: "codegraph"}, check.StatusError, "codegraph"},
		{"command on PATH", check.Requirement{Subject: "jq", Command: "jq"}, check.StatusOK, ""},
		{"sound present", check.Requirement{Subject: "sound-ok", Path: sound}, check.StatusOK, ""},
		{"sound missing", check.Requirement{Subject: "sound-gone", Path: filepath.Join(env.Home, "done.wav")}, check.StatusError, "done.wav"},
		{"unanalyzed skips the other checks", check.Requirement{Subject: "complex", Path: filepath.Join(env.Home, "gone.py"), Unanalyzed: "not analyzed"}, check.StatusUnknown, "not analyzed"},
	}
	reqs := make([]check.Requirement, len(tests))
	for i, tt := range tests {
		reqs[i] = tt.req
	}

	got := runCheck(t, check.Requirements("hooks", func(check.Env) ([]check.Requirement, error) { return reqs, nil }), env)

	if len(got) != len(tests) {
		t.Fatalf("got %d results, want %d: %+v", len(got), len(tests), got)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := resultFor(t, got, tt.req.Subject)
			if r.Status != tt.want {
				t.Errorf("status = %q, want %q (%+v)", r.Status, tt.want, r)
			}
			if r.CheckID != "hooks" {
				t.Errorf("CheckID = %q, want hooks", r.CheckID)
			}
			if !strings.Contains(r.Message, tt.wantMessage) {
				t.Errorf("message %q lacks %q", r.Message, tt.wantMessage)
			}
		})
	}
}

func TestRequirements_ExtractErrorIsSingleUnknown(t *testing.T) {
	extract := func(check.Env) ([]check.Requirement, error) { return nil, errors.New("settings unparseable") }

	got := runCheck(t, check.Requirements("hooks", extract), newHome(t, &fakeExec{}))

	if len(got) != 1 || got[0].Status != check.StatusUnknown || !strings.Contains(got[0].Message, "settings unparseable") {
		t.Fatalf("got %+v, want one unknown result carrying the extract error", got)
	}
}

func TestRequirements_NotFoundCasesCarrySuggestion(t *testing.T) {
	env := newHome(t, &fakeExec{})
	reqs := []check.Requirement{
		{Subject: "cmd", Command: "codegraph"},
		{Subject: "file", Path: filepath.Join(env.Home, "gone.py")},
		{Subject: "ok-file", Path: env.Home},
	}

	got := runCheck(t, check.Requirements("hooks", func(check.Env) ([]check.Requirement, error) { return reqs, nil }), env)

	if s := resultFor(t, got, "cmd").Suggestion; !strings.Contains(s, "install codegraph") {
		t.Errorf("command suggestion = %q, want install codegraph", s)
	}
	if s := resultFor(t, got, "file").Suggestion; !strings.Contains(s, "install") {
		t.Errorf("path suggestion = %q, want it to point at install", s)
	}
	if s := resultFor(t, got, "ok-file").Suggestion; s != "" {
		t.Errorf("ok suggestion = %q, want none", s)
	}
}
