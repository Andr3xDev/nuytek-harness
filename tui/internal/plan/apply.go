package plan

import (
	"context"
	"fmt"
)

type StepStatus string

const (
	StepDone    StepStatus = "done"
	StepSkipped StepStatus = "skipped"
	StepWarned  StepStatus = "warned"
	StepFailed  StepStatus = "failed"
)

type StepResult struct {
	Status StepStatus
	Detail string // the Snapshot step reports its stamp here
	Err    error
}

type Runner interface {
	Run(ctx context.Context, s Step, line func(string)) StepResult
}

type Phase string

const (
	PhaseStart   Phase = "start"
	PhaseLine    Phase = "line"
	PhaseDone    Phase = "done"
	PhaseSkipped Phase = "skipped"
	PhaseWarned  Phase = "warned"
	PhaseFailed  Phase = "failed"
)

type Event struct {
	Index, Total int
	Step         string
	Phase        Phase
	Text         string
}

type StepFailure struct {
	Step string
	Err  error
}

type Result struct {
	Stamp  string
	Warned []string
	Failed *StepFailure
}

// Apply runs the steps in order and stops at the first failure; warned steps are collected.
func Apply(ctx context.Context, r Runner, p Plan, emit func(Event)) Result {
	var result Result
	for i, step := range p.Steps {
		name := step.Describe()
		if err := ctx.Err(); err != nil {
			result.Failed = &StepFailure{Step: name, Err: err}
			return result
		}
		send := func(phase Phase, text string) {
			emit(Event{Index: i + 1, Total: len(p.Steps), Step: name, Phase: phase, Text: text})
		}
		send(PhaseStart, "")
		outcome := r.Run(ctx, step, func(line string) { send(PhaseLine, line) })
		if _, ok := step.(Snapshot); ok {
			result.Stamp = outcome.Detail
		}
		switch outcome.Status {
		case StepDone, StepSkipped, StepWarned, StepFailed:
		default:
			outcome.Err = fmt.Errorf("unknown step status %q", outcome.Status)
			outcome.Status = StepFailed
		}
		// Phase values equal StepStatus values, so the status names the closing event.
		text := ""
		if outcome.Err != nil {
			text = outcome.Err.Error()
		} else if outcome.Status == StepSkipped {
			text = outcome.Detail
		}
		send(Phase(outcome.Status), text)
		switch outcome.Status {
		case StepWarned:
			result.Warned = append(result.Warned, name+": "+text)
		case StepFailed:
			result.Failed = &StepFailure{Step: name, Err: outcome.Err}
			return result
		}
	}
	return result
}
