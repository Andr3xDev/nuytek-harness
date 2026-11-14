package dryrun

import (
	"context"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const notExecuted = "not executed"

type Runner struct{}

func New() Runner { return Runner{} }

func (Runner) Run(_ context.Context, _ plan.Step, _ func(string)) plan.StepResult {
	return plan.StepResult{Status: plan.StepSkipped, Detail: notExecuted}
}

var _ plan.Runner = Runner{}
