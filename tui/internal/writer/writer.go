package writer

import (
	"context"
	"fmt"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

type Writer struct{ now func() time.Time }

func New(now func() time.Time) *Writer { return &Writer{now: now} }

func (w *Writer) Run(ctx context.Context, s plan.Step, line func(string)) plan.StepResult {
	switch step := s.(type) {
	case plan.Snapshot:
		return w.snapshot(step)
	case plan.Copy:
		return fromError(copyFiles(step))
	case plan.Remove:
		return fromError(removeFiles(step))
	case plan.RemoveTree:
		return fromError(removeTree(step))
	case plan.RemoveMatching:
		return fromError(removeMatching(step))
	case plan.Chmod:
		return fromError(addExecute(step))
	case plan.Edit:
		return fromError(edit(step))
	case plan.Exec:
		return execute(ctx, step, line)
	default:
		return fromError(fmt.Errorf("writer: unknown step type %T", s))
	}
}

func fromError(err error) plan.StepResult {
	if err != nil {
		return plan.StepResult{Status: plan.StepFailed, Err: err}
	}
	return plan.StepResult{Status: plan.StepDone}
}

var _ plan.Runner = (*Writer)(nil)
