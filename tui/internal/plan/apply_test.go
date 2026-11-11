package plan_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const (
	snapshotStamp = "20240101-000000"
	skippedDetail = "not executed"
)

// recordingRunner reports statuses by step description and records the order of runs.
type recordingRunner struct {
	status map[string]plan.StepStatus
	ran    []string
}

func (r *recordingRunner) Run(_ context.Context, s plan.Step, line func(string)) plan.StepResult {
	r.ran = append(r.ran, s.Describe())
	line("out " + s.Describe())
	result := plan.StepResult{Status: plan.StepDone}
	if status, ok := r.status[s.Describe()]; ok {
		result.Status = status
		if status == plan.StepFailed || status == plan.StepWarned {
			result.Err = errors.New("boom")
		}
		if status == plan.StepSkipped {
			result.Detail = skippedDetail
		}
	}
	if _, ok := s.(plan.Snapshot); ok {
		result.Detail = snapshotStamp
	}
	return result
}

func execStep(name string) plan.Exec { return plan.Exec{Argv: []string{name}} }

func TestApply(t *testing.T) {
	snap := plan.Snapshot{Store: backup.Dir{Label: "home"}}
	tests := []struct {
		name       string
		steps      []plan.Step
		status     map[string]plan.StepStatus
		wantRan    []string
		wantEvents []string // "phase:step" or "line:step:text"
		wantStamp  string
		wantWarned int
		wantFailed string
	}{
		{
			name:    "runs in order with one start and one end per step",
			steps:   []plan.Step{snap, execStep("a"), execStep("b")},
			wantRan: []string{"snapshot into home", "a", "b"},
			wantEvents: []string{
				"start:snapshot into home", "line:snapshot into home:out snapshot into home", "done:snapshot into home",
				"start:a", "line:a:out a", "done:a",
				"start:b", "line:b:out b", "done:b",
			},
			wantStamp: snapshotStamp,
		},
		{
			name:       "stops at the first failed step",
			steps:      []plan.Step{execStep("a"), execStep("b"), execStep("c")},
			status:     map[string]plan.StepStatus{"b": plan.StepFailed},
			wantRan:    []string{"a", "b"},
			wantEvents: []string{"start:a", "line:a:out a", "done:a", "start:b", "line:b:out b", "failed:b"},
			wantFailed: "b",
		},
		{
			name:       "a warned step is collected and the run continues",
			steps:      []plan.Step{execStep("a"), execStep("b")},
			status:     map[string]plan.StepStatus{"a": plan.StepWarned},
			wantRan:    []string{"a", "b"},
			wantEvents: []string{"start:a", "line:a:out a", "warned:a", "start:b", "line:b:out b", "done:b"},
			wantWarned: 1,
		},
		{
			name:       "a skipped step emits its detail as the closing text",
			steps:      []plan.Step{execStep("a")},
			status:     map[string]plan.StepStatus{"a": plan.StepSkipped},
			wantRan:    []string{"a"},
			wantEvents: []string{"start:a", "line:a:out a", "skipped:a:" + skippedDetail},
		},
		{
			name:       "an unknown status is treated as failed",
			steps:      []plan.Step{execStep("a"), execStep("b")},
			status:     map[string]plan.StepStatus{"a": "bogus"},
			wantRan:    []string{"a"},
			wantEvents: []string{"start:a", "line:a:out a", "failed:a"},
			wantFailed: "a",
		},
		{
			name:       "a failed snapshot aborts before any other step runs",
			steps:      []plan.Step{snap, execStep("a")},
			status:     map[string]plan.StepStatus{"snapshot into home": plan.StepFailed},
			wantRan:    []string{"snapshot into home"},
			wantEvents: []string{"start:snapshot into home", "line:snapshot into home:out snapshot into home", "failed:snapshot into home"},
			wantStamp:  snapshotStamp,
			wantFailed: "snapshot into home",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &recordingRunner{status: tt.status}
			var events []string

			result := plan.Apply(context.Background(), runner, plan.Plan{Steps: tt.steps}, func(e plan.Event) {
				if e.Phase == plan.PhaseLine {
					events = append(events, fmt.Sprintf("line:%s:%s", e.Step, e.Text))
					return
				}
				if e.Phase == plan.PhaseSkipped {
					events = append(events, fmt.Sprintf("skipped:%s:%s", e.Step, e.Text))
					return
				}
				events = append(events, fmt.Sprintf("%s:%s", e.Phase, e.Step))
			})

			if !slices.Equal(runner.ran, tt.wantRan) {
				t.Errorf("ran = %v, want %v", runner.ran, tt.wantRan)
			}
			if !slices.Equal(events, tt.wantEvents) {
				t.Errorf("events = %v, want %v", events, tt.wantEvents)
			}
			if result.Stamp != tt.wantStamp {
				t.Errorf("Stamp = %q, want %q", result.Stamp, tt.wantStamp)
			}
			if len(result.Warned) != tt.wantWarned {
				t.Errorf("Warned = %v, want %d entries", result.Warned, tt.wantWarned)
			}
			if tt.wantWarned > 0 && result.Warned[0] != "a: boom" {
				t.Errorf("Warned[0] = %q, want %q", result.Warned[0], "a: boom")
			}
			if (result.Failed == nil) != (tt.wantFailed == "") || (result.Failed != nil && result.Failed.Step != tt.wantFailed) {
				t.Errorf("Failed = %+v, want step %q", result.Failed, tt.wantFailed)
			}
		})
	}
}

func TestApplyStopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runner := &recordingRunner{}

	result := plan.Apply(ctx, runner, plan.Plan{Steps: []plan.Step{execStep("a")}}, func(plan.Event) {})

	if len(runner.ran) != 0 {
		t.Errorf("ran = %v, want no step", runner.ran)
	}
	if result.Failed == nil || result.Failed.Step != "a" || !errors.Is(result.Failed.Err, context.Canceled) {
		t.Errorf("Failed = %+v, want step a with context.Canceled", result.Failed)
	}
}
