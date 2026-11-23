package writer_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
		{name: "timeout with Warn is warned", script: "sleep 30", timeout: 50 * time.Millisecond, warn: true, want: plan.StepWarned},
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

// A clone right after `gh repo create` can fail until the new repo is readable, so the step owns a bounded retry.
func TestExec_RetriesAFailingCommandUpToAttempts(t *testing.T) {
	const succeedOnRun = 3
	tests := []struct {
		name     string
		attempts int
		want     plan.StepStatus
		wantRuns int
	}{
		{"succeeds on a later attempt", 5, plan.StepDone, succeedOnRun},
		{"fails after exhausting the attempts", 2, plan.StepFailed, 2},
		{"zero attempts runs once", 0, plan.StepFailed, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			counter := filepath.Join(t.TempDir(), "runs")
			script := `echo run >> "$1"; [ "$(wc -l < "$1")" -ge 3 ]`
			step := plan.Exec{Argv: []string{"sh", "-c", script, "sh", counter}, Timeout: 5 * time.Second, Attempts: tc.attempts, Backoff: time.Millisecond}

			res, _ := run(t, newWriter(fixedNow), step)

			wantStatus(t, res, tc.want)
			data, err := os.ReadFile(counter)
			if err != nil {
				t.Fatal(err)
			}
			if runs := strings.Count(string(data), "run"); runs != tc.wantRuns {
				t.Errorf("command ran %d times, want %d", runs, tc.wantRuns)
			}
		})
	}
}

func TestExec_DoesNotRetryATimedOutAttempt(t *testing.T) {
	counter := filepath.Join(t.TempDir(), "runs")
	step := plan.Exec{Argv: []string{"sh", "-c", `echo run >> "$1"; sleep 5`, "sh", counter}, Timeout: 200 * time.Millisecond, Attempts: 3, Backoff: time.Millisecond}

	res, _ := run(t, newWriter(fixedNow), step)

	wantStatus(t, res, plan.StepFailed)
	data, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	if runs := strings.Count(string(data), "run"); runs != 1 {
		t.Errorf("command ran %d times, want 1", runs)
	}
}
