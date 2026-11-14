package specter

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

// Internal package: openspecJSON is unexported by design (D2).

type runExec struct {
	res     check.ExecResult
	err     error
	gotName string
	gotArgs []string
}

func (r *runExec) LookPath(string) (string, error) { return "", os.ErrNotExist }
func (r *runExec) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("specter must use Run, not Output")
}
func (r *runExec) Run(_ context.Context, name string, args ...string) (check.ExecResult, error) {
	r.gotName, r.gotArgs = name, args
	return r.res, r.err
}

func TestOpenspecJSON(t *testing.T) {
	type payload struct {
		Valid bool `json:"valid"`
	}
	tests := []struct {
		name         string
		res          check.ExecResult
		runErr       error
		allowNonZero bool
		wantErr      bool
		wantValid    bool
		errContains  []string
	}{
		{name: "exit 0 with json is ok", res: check.ExecResult{Stdout: []byte(`{"valid":true}`)}, wantValid: true},
		{name: "exit 0 with non-json is a tool failure", res: check.ExecResult{Stdout: []byte("Warning: node")}, wantErr: true, errContains: []string{"Warning: node"}},
		{name: "non-zero with json and allowNonZero is a result", res: check.ExecResult{Stdout: []byte(`{"valid":false}`), ExitCode: 1}, allowNonZero: true},
		{name: "non-zero with json but no allowNonZero is a tool failure", res: check.ExecResult{Stdout: []byte(`{"valid":false}`), Stderr: []byte("boom"), ExitCode: 2}, wantErr: true, errContains: []string{"boom", "exit 2"}},
		{name: "non-zero with non-json stdout is a tool failure even when allowed", res: check.ExecResult{Stdout: []byte("nope"), Stderr: []byte("bad flag"), ExitCode: 3}, allowNonZero: true, wantErr: true, errContains: []string{"bad flag", "exit 3"}},
		{name: "run error is a tool failure naming the command", runErr: errors.New("deadline exceeded"), wantErr: true, errContains: []string{"openspec", "deadline exceeded"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exec := &runExec{res: tc.res, err: tc.runErr}
			var out payload

			err := openspecJSON(context.Background(), exec, &out, tc.allowNonZero, "validate", "--json")

			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			for _, want := range tc.errContains {
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Errorf("err %v does not contain %q", err, want)
				}
			}
			if out.Valid != tc.wantValid {
				t.Errorf("out.Valid = %v, want %v", out.Valid, tc.wantValid)
			}
			if exec.gotName != "openspec" || strings.Join(exec.gotArgs, " ") != "validate --json" {
				t.Errorf("ran %q %v, want openspec [validate --json]", exec.gotName, exec.gotArgs)
			}
		})
	}
}

func TestSnippet_DoesNotCutAMultibyteRune(t *testing.T) {
	got := snippet([]byte("aéé"), 2) // 'é' is 2 bytes: a 2-byte cut would split the first one

	if got != "a" {
		t.Errorf("snippet = %q, want %q", got, "a")
	}
}
