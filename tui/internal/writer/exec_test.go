package writer_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func TestExec(t *testing.T) {
	const maxElapsed = 5 * time.Second
	tests := []struct {
		name      string
		script    string
		timeout   time.Duration
		warn      bool
		want      plan.StepStatus
		wantLines []string
	}{
		{name: "success streams lines", script: "echo one; echo two", timeout: 5 * time.Second, want: plan.StepDone, wantLines: []string{"one", "two"}},
		{name: "timeout kills the command", script: "sleep 30", timeout: 50 * time.Millisecond, want: plan.StepFailed},
		{name: "nonzero with Warn is warned", script: "echo oops; exit 3", timeout: 5 * time.Second, warn: true, want: plan.StepWarned, wantLines: []string{"oops"}},
		{name: "nonzero without Warn fails", script: "exit 3", timeout: 5 * time.Second, want: plan.StepFailed},
		{name: "stdin is /dev/null", script: "cat; echo done", timeout: 5 * time.Second, want: plan.StepDone, wantLines: []string{"done"}},
		{name: "carriage return is trimmed", script: `printf 'one\r\ntwo\r\n'`, timeout: 5 * time.Second, want: plan.StepDone, wantLines: []string{"one", "two"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			step := plan.Exec{Argv: []string{"sh", "-c", tc.script}, Timeout: tc.timeout, Warn: tc.warn}
			start := time.Now()

			res, lines := run(t, newWriter(fixedNow), step)

			wantStatus(t, res, tc.want)
			if (tc.want == plan.StepFailed || tc.want == plan.StepWarned) && res.Err == nil {
				t.Fatal("want an error")
			}
			if elapsed := time.Since(start); elapsed > maxElapsed {
				t.Fatalf("took %v, want under %v", elapsed, maxElapsed)
			}
			if !reflect.DeepEqual(lines, tc.wantLines) {
				t.Fatalf("lines = %q, want %q", lines, tc.wantLines)
			}
		})
	}
}
