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
	for _, k := range []string{"i", "r", "x", "d", "[", "h", "l"} {
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
	if keys, _ := r.keybarKeys(); !claims(keys, "ctrl+c") {
		t.Fatalf("keybar keys = %v, want ctrl+c listed while a run is active", keys)
	}

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

func TestRoot_PanelSwitchesTabsWithHLAndArrowsAndWraps(t *testing.T) {
	right, left := tea.KeyPressMsg{Code: tea.KeyRight}, tea.KeyPressMsg{Code: tea.KeyLeft}
	tests := []struct {
		name string
		key  tea.KeyPressMsg
		from int
		want int
	}{
		{"l next", textKey("l"), 0, 1},
		{"right next", right, 0, 1},
		{"h prev", textKey("h"), 1, 0},
		{"left prev", left, 1, 0},
		{"h wraps to last", textKey("h"), 0, 3},
		{"l wraps to first", textKey("l"), 3, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newRoot(t, availableTarget())
			r.focus = focusPanel
			r.tab = tc.from
			press(r, tc.key)
			if r.tab != tc.want || r.focus != focusPanel {
				t.Fatalf("tab = %d, focus = %v; want tab %d in the panel", r.tab, r.focus, tc.want)
			}
		})
	}
}

func TestRoot_PanelTabKeyReturnsToSidebar(t *testing.T) {
	r := newRoot(t, availableTarget())
	r.focus = focusPanel
	press(r, tea.KeyPressMsg{Code: tea.KeyTab})
	if r.focus != focusSidebar {
		t.Fatal("tab did not return focus to the sidebar")
	}
}

func TestKeyMap_StepMapsEveryMovementSpelling(t *testing.T) {
	k := newKeyMap()
	tests := []struct {
		msg  tea.KeyPressMsg
		want int
	}{
		{textKey("j"), 1},
		{upKey, -1},
		{textKey("J"), jumpRows},
		{tea.KeyPressMsg{Code: 'j', Mod: tea.ModShift}, jumpRows},
		{textKey("K"), -jumpRows},
		{tea.KeyPressMsg{Code: 'k', Mod: tea.ModShift}, -jumpRows},
		{textKey("x"), 0},
	}
	for _, tt := range tests {
		if got := k.step(tt.msg); got != tt.want {
			t.Errorf("step(%q) = %d, want %d", tt.msg.String(), got, tt.want)
		}
	}
}

func TestRoot_MoveSidebarClampsAndEntersSpecterAtTheTop(t *testing.T) {
	const targets = 7
	tests := []struct {
		name         string
		cursor       int
		entry        entry
		delta        int
		wantCursor   int
		wantEntry    entry
		wantNoAction bool
	}{
		{name: "down one", cursor: 1, delta: 1, wantCursor: 2},
		{name: "jump down", cursor: 1, delta: jumpRows, wantCursor: 6},
		{name: "jump down clamps", cursor: 4, delta: jumpRows, wantCursor: 6},
		{name: "no-op at last target", cursor: 6, delta: jumpRows, wantCursor: 6, wantNoAction: true},
		{name: "up one", cursor: 3, delta: -1, wantCursor: 2},
		{name: "jump up", cursor: 6, delta: -jumpRows, wantCursor: 1},
		{name: "jump up clamps at first target", cursor: 3, delta: -jumpRows, wantCursor: 0},
		{name: "up from first target enters Config", cursor: 0, delta: -1, wantCursor: 0, wantEntry: entryConfig},
		{name: "jump up from first target enters Config", cursor: 0, delta: -jumpRows, wantCursor: 0, wantEntry: entryConfig},
		{name: "up from Config enters Specter", cursor: 0, entry: entryConfig, delta: -1, wantCursor: 0, wantEntry: entrySpecter},
		{name: "down from Config returns to the first target", cursor: 0, entry: entryConfig, delta: 1, wantCursor: 0},
		{name: "down from Specter enters Config", cursor: 0, entry: entrySpecter, delta: 1, wantCursor: 0, wantEntry: entryConfig},
		{name: "up on Specter stays", cursor: 0, entry: entrySpecter, delta: -1, wantCursor: 0, wantEntry: entrySpecter, wantNoAction: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := make([]target.Target, targets)
			for i := range ts {
				ts[i] = target.Target{Spec: target.Planned("p", "P")}
			}
			r := newRoot(t, ts...)
			r.cursor, r.entry = tt.cursor, tt.entry

			cmd := r.moveSidebar(tt.delta)

			if r.cursor != tt.wantCursor || r.entry != tt.wantEntry {
				t.Errorf("cursor %d entry %v, want %d %v", r.cursor, r.entry, tt.wantCursor, tt.wantEntry)
			}
			if tt.wantNoAction && cmd != nil {
				t.Error("a move that changes nothing must not trigger a recompute")
			}
		})
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
