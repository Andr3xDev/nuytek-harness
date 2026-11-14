package ui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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

// boxRows returns the indices of the view rows that carry a border glyph.
func boxRows(view string) []int {
	var rows []int
	for i, line := range strings.Split(view, "\n") {
		if strings.ContainsAny(line, "╭╰│") {
			rows = append(rows, i)
		}
	}
	return rows
}

func TestFlow_ConfirmIsACompactBoxCenteredInThePanel(t *testing.T) {
	const panelW, panelH = 60, 20
	f := newTestFlow(t, applicablePlan("Delete everything?"))
	planned(f)
	f.Update(enterKey)

	view := f.View(panelW, panelH)
	rows := boxRows(view)

	if len(rows) == 0 {
		t.Fatalf("confirmation has no box:\n%s", view)
	}
	// Title, confirm text and one spare row, plus the borders.
	if height := len(rows); height > 3+borderCells {
		t.Fatalf("box height = %d, want at most content plus borders (%d)", height, 3+borderCells)
	}
	first, last := rows[0], rows[len(rows)-1]
	if gap := first - (panelH - 1 - last); gap < -1 || gap > 1 {
		t.Fatalf("box rows %d..%d in a %d-row panel: not vertically centered", first, last, panelH)
	}
	row := strings.Split(view, "\n")[first]
	left := lipgloss.Width(row[:strings.IndexAny(row, "╭")])
	right := panelW - left - lipgloss.Width(strings.TrimSpace(row))
	if gap := left - right; gap < -1 || gap > 1 {
		t.Fatalf("box margins left=%d right=%d: not horizontally centered", left, right)
	}
}

func TestFlow_ConfirmStaysWithinANarrowOrShortPanel(t *testing.T) {
	cases := map[string][2]int{"narrow": {30, 20}, "short": {60, 4}}
	for name, size := range cases {
		t.Run(name, func(t *testing.T) {
			f := newTestFlow(t, applicablePlan("Delete everything?"))
			planned(f)
			f.Update(enterKey)

			view := f.View(size[0], size[1])

			if lipgloss.Width(view) > size[0] || lipgloss.Height(view) > size[1] {
				t.Fatalf("view is %dx%d, want within %dx%d", lipgloss.Width(view), lipgloss.Height(view), size[0], size[1])
			}
			if len(boxRows(view)) == 0 {
				t.Fatalf("confirmation has no box:\n%s", view)
			}
		})
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
