package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

func applicablePlan(confirm string) plan.Plan {
	step := plan.Copy{Rows: []plan.Change{{Effect: plan.EffectCreate, Path: "a"}}}
	return plan.Plan{Title: "t", Steps: []plan.Step{step}, Confirm: confirm}
}

func newTestFlow(t *testing.T, p plan.Plan) *flow {
	t.Helper()
	svc := newRoot(t, availableTarget()).svc
	return newDirectFlow(svc, "Test", "nothing", func(context.Context, []string) (plan.Plan, error) { return p, nil })
}

// planned starts the flow and delivers its plan the way Root would.
func planned(f *flow) {
	f.Update(f.start()())
}

func isRunRequest(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(runRequest)
	return ok
}

func TestFlow_StalePlanAfterEscAndRestartIsDropped(t *testing.T) {
	f := newTestFlow(t, applicablePlan(""))
	stale := f.start()()
	f.Update(escKey)
	f.start()

	f.Update(stale)

	if f.stage != stagePlanning {
		t.Fatalf("stage = %v, want planning: the old plan must be dropped", f.stage)
	}
}

func TestFlow_ConfirmPlanNeedsEnterThenY(t *testing.T) {
	f := newTestFlow(t, applicablePlan("sure?"))
	planned(f)

	if isRunRequest(f.Update(enterKey)) || f.stage != stageConfirm {
		t.Fatalf("enter in preview: stage = %v, want confirm without a run request", f.stage)
	}
	if isRunRequest(f.Update(enterKey)) || f.stage != stageConfirm {
		t.Fatalf("enter in confirm: stage = %v, want it to do nothing", f.stage)
	}
	if !isRunRequest(f.Update(textKey("y"))) {
		t.Fatal("y in confirm must produce a runRequest")
	}
}

func TestFlow_YInPreviewDoesNothing(t *testing.T) {
	f := newTestFlow(t, applicablePlan("sure?"))
	planned(f)

	if isRunRequest(f.Update(textKey("y"))) || f.stage != stagePreview {
		t.Fatalf("y in preview: stage = %v, want preview without a run request", f.stage)
	}
}

func TestFlow_EscInPreviewAndConfirmYieldsNoRunRequest(t *testing.T) {
	f := newTestFlow(t, applicablePlan("sure?"))
	planned(f)
	if isRunRequest(f.Update(escKey)) {
		t.Fatal("esc in preview produced a runRequest")
	}

	planned(f)
	f.Update(enterKey)
	if isRunRequest(f.Update(escKey)) || f.stage != stagePreview {
		t.Fatalf("esc in confirm: stage = %v, want preview without a run request", f.stage)
	}
}

func TestFlow_RunTitleFollowsFlowAction(t *testing.T) {
	f := newTestFlow(t, applicablePlan(""))
	planned(f)

	req, ok := f.Update(enterKey)().(runRequest)

	if !ok || req.plan.Title != "Test" {
		t.Fatalf("run request = %+v (ok=%v), want plan title %q", req, ok, "Test")
	}
	if got := newRun(req.plan.Title, false).boxTitle(); got != "Output - Test (running)" {
		t.Fatalf("boxTitle = %q", got)
	}
}

func TestRun_FollowShowsLastLineOfWrappedOutput(t *testing.T) {
	r := newRun("x", false)
	r.add(plan.Event{Phase: plan.PhaseLine, Text: strings.Repeat("long ", 30)})
	r.finish(plan.Result{Failed: &plan.StepFailure{Step: "boom", Err: errors.New("bad")}})

	view := strings.Join(r.view(newTheme(defaultDark), 12, 3), "\n")

	if !strings.Contains(view, "existed)") {
		t.Fatalf("view does not reach the last line:\n%s", view)
	}
}
