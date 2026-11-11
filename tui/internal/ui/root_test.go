package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/dashboard"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

func TestRoot_InitStartsLoadingAndResultMakesReady(t *testing.T) {
	r := newRoot(t, availableTarget())

	if cmd := r.Init(); cmd == nil {
		t.Fatal("Init returned nil Cmd for an available target")
	}
	if got := r.states[0].phase; got != loading {
		t.Fatalf("phase after Init = %v, want loading", got)
	}

	r.Update(computedMsg{idx: 0, gen: r.states[0].gen, dash: dashboard.Dashboard{}})
	if got := r.states[0].phase; got != ready {
		t.Fatalf("phase after result = %v, want ready", got)
	}
}

func TestRoot_RefreshRecomputesAndDropsStaleResult(t *testing.T) {
	r := newRoot(t, availableTarget())
	r.Init()
	r.Update(computedMsg{idx: 0, gen: r.states[0].gen})
	r.focus = focusPanel
	oldGen := r.states[0].gen

	if cmd := press(r, textKey("r")); cmd == nil {
		t.Fatal("r on a ready target returned nil Cmd, want recompute")
	}
	if got := r.states[0].phase; got != loading {
		t.Fatalf("phase after r = %v, want loading", got)
	}
	if got := r.states[0].gen; got != oldGen+1 {
		t.Fatalf("gen after r = %d, want %d", got, oldGen+1)
	}

	r.Update(computedMsg{idx: 0, gen: oldGen})
	if got := r.states[0].phase; got != loading {
		t.Fatalf("phase after stale result = %v, want loading", got)
	}

	r.Update(computedMsg{idx: 0, gen: oldGen + 1})
	if got := r.states[0].phase; got != ready {
		t.Fatalf("phase after fresh result = %v, want ready", got)
	}
}

func TestRoot_SidebarSelectsTargetsAndBoundsCursor(t *testing.T) {
	r := newRoot(t, availableTarget(), availableTarget())
	r.Init()
	r.Update(computedMsg{idx: 0, gen: r.states[0].gen})
	cachedGen := r.states[0].gen

	if cmd := press(r, downKey); cmd == nil || r.cursor != 1 {
		t.Fatalf("down: cmd nil = %v, cursor = %d; want a Cmd and cursor 1", cmd == nil, r.cursor)
	}
	if cmd := press(r, downKey); cmd != nil || r.cursor != 1 {
		t.Fatalf("down at the end: cmd nil = %v, cursor = %d; want no Cmd and cursor 1", cmd == nil, r.cursor)
	}
	if cmd := press(r, upKey); cmd != nil || r.cursor != 0 {
		t.Fatalf("up: cmd nil = %v, cursor = %d; want no Cmd and cursor 0", cmd == nil, r.cursor)
	}
	if st := r.states[0]; st.phase != ready || st.gen != cachedGen {
		t.Fatalf("cached target recomputed: phase = %v, gen = %d, want ready and gen %d", st.phase, st.gen, cachedGen)
	}
}

func TestRoot_RefreshSupersedesRunningComputation(t *testing.T) {
	r := newRoot(t, availableTarget())
	r.Init()
	r.Update(computedMsg{idx: 0, gen: r.states[0].gen})
	r.focus = focusPanel

	press(r, textKey("r"))
	first := r.states[0].gen
	press(r, textKey("r"))
	second := r.states[0].gen
	if second != first+1 {
		t.Fatalf("second r: gen = %d, want %d", second, first+1)
	}

	r.Update(computedMsg{idx: 0, gen: first})
	if got := r.states[0].phase; got != loading {
		t.Fatalf("phase after the superseded result = %v, want loading", got)
	}
	r.Update(computedMsg{idx: 0, gen: second})
	if got := r.states[0].phase; got != ready {
		t.Fatalf("phase after the last result = %v, want ready", got)
	}
}

func TestRoot_PlannedTargetHasNoComputeAndKeepsSidebarFocus(t *testing.T) {
	r := newRoot(t, target.Target{Spec: target.Planned("p", "P")})

	if cmd := r.Init(); cmd != nil {
		t.Fatal("Init returned a Cmd for a planned target")
	}
	press(r, enterKey)
	if r.focus != focusSidebar {
		t.Fatal("enter moved focus to the panel of a planned target")
	}
	if r.activeTab() != nil {
		t.Fatal("planned target exposes an action tab")
	}
	if cmd := press(r, textKey("i")); cmd != nil || r.wizard != nil {
		t.Fatalf("i on a planned target: cmd nil = %v, wizard = %v; want no Cmd and no wizard", cmd == nil, r.wizard)
	}
}

func TestRoot_RunActiveRefusesKeysAndOnlySetsNotice(t *testing.T) {
	for _, k := range []string{"i", "r", "x", "d", "["} {
		t.Run(k, func(t *testing.T) {
			r := newRoot(t, availableTarget())
			r.Init()
			r.Update(computedMsg{idx: 0, gen: r.states[0].gen})
			r.focus = focusPanel
			r.Update(runRequest{plan: plan.Plan{Title: "run"}})
			if r.run == nil {
				t.Fatal("runRequest did not start a run")
			}
			before := struct {
				cursor, tab, gen int
				focus            focus
				phase            phase
				help             bool
				run              *run
			}{r.cursor, r.tab, r.states[0].gen, r.focus, r.states[0].phase, r.showHelp, r.run}

			if cmd := press(r, textKey(k)); cmd != nil {
				t.Fatal("key returned a Cmd while a run is active")
			}
			after := struct {
				cursor, tab, gen int
				focus            focus
				phase            phase
				help             bool
				run              *run
			}{r.cursor, r.tab, r.states[0].gen, r.focus, r.states[0].phase, r.showHelp, r.run}
			if after != before || r.wizard != nil {
				t.Fatalf("state changed: before %+v, after %+v, wizard %v", before, after, r.wizard)
			}
			if r.notice != runInProgressText {
				t.Fatalf("notice = %q, want %q", r.notice, runInProgressText)
			}
		})
	}
}

func TestRoot_QuitRefusedWhileRunActive(t *testing.T) {
	r := newRoot(t, availableTarget())
	r.Update(runRequest{plan: plan.Plan{Title: "run"}})

	if isQuit(press(r, textKey("q"))) {
		t.Fatal("q quit while a run is active")
	}
	if r.notice != runInProgressText {
		t.Fatalf("notice = %q, want %q", r.notice, runInProgressText)
	}
	if r.ctx.Err() != nil {
		t.Fatal("context cancelled while a run is active")
	}
}

func TestRoot_CtrlCDuringRunCancelsTheRunNotTheApp(t *testing.T) {
	r := newRoot(t, availableTarget())
	r.Update(runRequest{plan: plan.Plan{Title: "run"}})
	cancelled := 0
	r.run.cancel = func() { cancelled++ }

	if isQuit(press(r, ctrlC)) {
		t.Fatal("ctrl+c quit while a run is active")
	}
	if cancelled != 1 || !r.run.cancelling {
		t.Fatalf("cancel calls = %d, cancelling = %v, want 1 and true", cancelled, r.run.cancelling)
	}
	if !strings.Contains(r.run.boxTitle(), "cancelling") {
		t.Fatalf("title = %q, want a cancelling notice", r.run.boxTitle())
	}
	if r.ctx.Err() != nil {
		t.Fatal("app context cancelled; only the run may be")
	}

	r.Update(applyDone{})
	if strings.Contains(r.run.boxTitle(), "cancelling") || !r.run.done {
		t.Fatalf("title = %q, done = %v, want the notice gone after applyDone", r.run.boxTitle(), r.run.done)
	}
}

func TestRoot_QuitKeysQuitAndCancelContext(t *testing.T) {
	for name, k := range map[string]tea.KeyPressMsg{"q": textKey("q"), "ctrl+c": ctrlC} {
		t.Run(name, func(t *testing.T) {
			r := newRoot(t, availableTarget())

			if !isQuit(press(r, k)) {
				t.Fatal("key did not return tea.Quit")
			}
			if r.ctx.Err() == nil {
				t.Fatal("context not cancelled on quit")
			}
		})
	}
}

func TestRoot_HelpToggleAndEscClose(t *testing.T) {
	r := newRoot(t, availableTarget())

	press(r, textKey("?"))
	if !r.showHelp {
		t.Fatal("? did not open help")
	}
	press(r, textKey("?"))
	if r.showHelp {
		t.Fatal("second ? did not close help")
	}
	press(r, textKey("?"))
	press(r, escKey)
	if r.showHelp {
		t.Fatal("esc did not close help")
	}
}

func TestRoot_HelpSwallowsOtherKeysButQuitStillQuits(t *testing.T) {
	r := newRoot(t, availableTarget())
	press(r, textKey("?"))

	press(r, enterKey)
	if r.focus != focusSidebar || !r.showHelp {
		t.Fatal("enter was not swallowed while help is open")
	}
	if !isQuit(press(r, textKey("q"))) {
		t.Fatal("q did not quit while help is open")
	}
}

func TestRoot_EnterFocusesPanelAndEscReturnsToSidebar(t *testing.T) {
	r := newRoot(t, availableTarget())

	press(r, enterKey)
	if r.focus != focusPanel {
		t.Fatal("enter did not focus the panel")
	}
	press(r, escKey)
	if r.focus != focusSidebar {
		t.Fatal("esc did not return focus to the sidebar")
	}
}

func TestRoot_ViewShowsNotAvailableAndComputingText(t *testing.T) {
	const w, h = 100, 30
	planned := newRoot(t, target.Target{Spec: target.Planned("p", "P")})
	planned.Update(tea.WindowSizeMsg{Width: w, Height: h})
	if got := planned.render(); !strings.Contains(got, notAvailable) {
		t.Errorf("planned view lacks %q", notAvailable)
	}

	avail := newRoot(t, availableTarget())
	avail.Init()
	avail.Update(tea.WindowSizeMsg{Width: w, Height: h})
	if got := avail.render(); !strings.Contains(got, computingText) {
		t.Errorf("loading view lacks %q", computingText)
	}
}
