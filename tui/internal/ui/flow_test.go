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

func newChoiceFlow(t *testing.T, got *[]string, names ...string) *flow {
	t.Helper()
	svc := newRoot(t, availableTarget()).svc
	f := newSelectFlow(svc, "Test", "pick", "nothing", func(_ context.Context, picked []string) (plan.Plan, error) {
		*got = picked
		return applicablePlan(""), nil
	})
	var choices []choice
	for _, n := range names {
		choices = append(choices, choice{name: n})
	}
	f.setChoices(choices)
	return f
}

func TestFlow_LimitOneToggleKeepsOnlyTheLatestPick(t *testing.T) {
	var got []string
	f := newChoiceFlow(t, &got, "a", "b")
	f.limit = 1

	space := tea.KeyPressMsg{Code: tea.KeySpace}
	f.Update(space)
	f.Update(downKey)
	f.Update(space)

	if names := f.names(); len(names) != 1 || names[0] != "b" {
		t.Fatalf("names = %v, want [b]", names)
	}
}

func TestFlow_LimitOneKeepsOnlyTheFirstPreselectedChoice(t *testing.T) {
	var got []string
	f := newChoiceFlow(t, &got)
	f.limit = 1

	f.setChoices([]choice{{name: "a", selected: true}, {name: "b", selected: true}, {name: "c"}})

	if names := f.names(); len(names) != 1 || names[0] != "a" {
		t.Fatalf("names = %v, want [a]", names)
	}
}

func TestFlow_LimitOneIgnoresAllKeyAndHidesIt(t *testing.T) {
	var got []string
	f := newChoiceFlow(t, &got, "a", "b")
	f.limit = 1

	f.Update(textKey("a"))

	if names := f.names(); len(names) != 0 {
		t.Fatalf("names = %v, want none: the all key must do nothing with limit 1", names)
	}
	if claims(f.Keys(), "a") {
		t.Fatal("limit 1 must hide the all key from Keys()")
	}
}

func TestFlow_OptionalEnterWithoutPickPlansWithNilNames(t *testing.T) {
	got := []string{"sentinel"}
	f := newChoiceFlow(t, &got, "a", "b")
	f.optional = true

	cmd := f.Update(enterKey)
	if f.stage != stagePlanning || f.notice != "" {
		t.Fatalf("stage = %v, notice = %q, want planning without a notice", f.stage, f.notice)
	}
	cmd()

	if got != nil {
		t.Fatalf("planner names = %v, want nil", got)
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
