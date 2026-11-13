package ui

import (
	"context"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

var ansiSequence = regexp.MustCompile(`\x1b\[[0-9;:?]*[A-Za-z]`)

func plain(s string) string { return ansiSequence.ReplaceAllString(s, "") }

// specterRoot is a Root over one available target "Alpha" whose Specter loader never touches the OS.
func specterRoot(t *testing.T) *Root {
	t.Helper()
	tgt := availableTarget()
	tgt.Spec.Name = "Alpha"
	r := NewRoot([]target.Target{tgt}, Deps{SpecterLoad: func(context.Context) specter.Facts { return specter.Facts{} }}, nil)
	r.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return r
}

func selectSpecter(r *Root) {
	press(r, upKey)
}

func TestRoot_InitStartsSpecterLoadAndResultMakesItReady(t *testing.T) {
	r := specterRoot(t)

	r.Init()
	if st := r.specter; st.phase != loading || st.gen != 1 || st.cancel == nil {
		t.Fatalf("after Init: phase = %v, gen = %d, cancel nil = %v; want loading, 1, set", st.phase, st.gen, st.cancel == nil)
	}

	r.Update(specterLoadedMsg{gen: 1, facts: specter.Facts{Home: "/h"}})
	if st := r.specter; st.phase != ready || !st.loaded || st.facts.Home != "/h" {
		t.Fatalf("after result: phase = %v, loaded = %v, facts.Home = %q; want ready, true, /h", st.phase, st.loaded, st.facts.Home)
	}
}

func TestRoot_StaleSpecterResultDropped(t *testing.T) {
	r := specterRoot(t)
	r.Init()
	selectSpecter(r)
	if r.specter.gen != 2 {
		t.Fatalf("gen after entering the screen = %d, want 2", r.specter.gen)
	}

	r.Update(specterLoadedMsg{gen: 1, facts: specter.Facts{Home: "stale"}})
	if st := r.specter; st.phase != loading || st.loaded {
		t.Fatalf("stale result accepted: phase = %v, loaded = %v", st.phase, st.loaded)
	}
	r.Update(specterLoadedMsg{gen: 2, facts: specter.Facts{Home: "fresh"}})
	if st := r.specter; st.phase != ready || st.facts.Home != "fresh" {
		t.Fatalf("fresh result: phase = %v, Home = %q; want ready, fresh", st.phase, st.facts.Home)
	}
}

func TestRoot_SpecterRefreshOnScreenSupersedesRunningLoad(t *testing.T) {
	r := specterRoot(t)
	r.Init()
	selectSpecter(r)
	press(r, enterKey)
	cancelled := 0
	r.specter.cancel = func() { cancelled++ }
	gen := r.specter.gen

	press(r, textKey("r"))
	if r.specter.gen != gen+1 || r.specter.phase != loading || cancelled != 1 {
		t.Fatalf("after r: gen = %d (want %d), phase = %v, cancelled = %d (want 1)", r.specter.gen, gen+1, r.specter.phase, cancelled)
	}
}

func TestRoot_CLIDashboardRefreshAlsoReloadsSpecter(t *testing.T) {
	r := specterRoot(t)
	r.Init()
	r.Update(computedMsg{idx: 0, gen: r.states[0].gen})
	r.Update(specterLoadedMsg{gen: r.specter.gen})
	r.focus = focusPanel
	gen := r.specter.gen

	press(r, textKey("r"))
	if r.specter.gen != gen+1 || r.specter.phase != loading {
		t.Fatalf("after dashboard r: specter gen = %d (want %d), phase = %v, want loading", r.specter.gen, gen+1, r.specter.phase)
	}
	if r.states[0].phase != loading {
		t.Fatalf("target phase = %v, want loading (recomputed together)", r.states[0].phase)
	}
}

func TestRoot_QuitCancelsInFlightSpecterLoad(t *testing.T) {
	var loadCtx context.Context
	r := NewRoot([]target.Target{availableTarget()}, Deps{SpecterLoad: func(ctx context.Context) specter.Facts {
		loadCtx = ctx
		return specter.Facts{}
	}}, nil)
	load := r.loadSpecter()

	if !isQuit(press(r, textKey("q"))) {
		t.Fatal("q did not quit")
	}
	load()

	if loadCtx == nil || loadCtx.Err() == nil {
		t.Fatal("the in-flight Specter load kept a live context after quit")
	}
}

func TestRoot_NilSpecterDepsNeverPanic(t *testing.T) {
	r := NewRoot([]target.Target{availableTarget()}, Deps{}, nil)
	r.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	r.Init()
	selectSpecter(r)
	press(r, enterKey)

	for _, k := range []string{"r", "v", "f", "m", "?", "?"} {
		press(r, textKey(k))
	}
	if got := plain(r.render()); !strings.Contains(got, "Specter") {
		t.Fatalf("Specter screen did not render:\n%s", got)
	}
}

func TestRoot_RunDismissalWithSpecterSelectedReloadsSpecter(t *testing.T) {
	r := specterRoot(t)
	selectSpecter(r)
	r.run = newRun("x", false)
	r.run.finish(plan.Result{})
	gen := r.specter.gen

	press(r, escKey)

	if r.run != nil || r.specter.gen != gen+1 || r.specter.phase != loading {
		t.Fatalf("run = %v, gen = %d (want %d), phase = %v; want the output pane closed and a new Specter load", r.run, r.specter.gen, gen+1, r.specter.phase)
	}
}

func TestRoot_SidebarWalksBetweenSpecterAndTargets(t *testing.T) {
	a, b := availableTarget(), availableTarget()
	r := NewRoot([]target.Target{a, b}, Deps{SpecterLoad: func(context.Context) specter.Facts { return specter.Facts{} }}, nil)

	press(r, upKey)
	if !r.specterSelected || r.cursor != 0 {
		t.Fatalf("up from target 0: specterSelected = %v, cursor = %d; want true, 0", r.specterSelected, r.cursor)
	}
	if _, ok := r.currentTarget(); ok {
		t.Fatal("currentTarget reported a target while Specter is selected")
	}
	press(r, upKey)
	if !r.specterSelected {
		t.Fatal("up from Specter left the Specter entry")
	}
	press(r, downKey)
	if r.specterSelected || r.cursor != 0 {
		t.Fatalf("down from Specter: specterSelected = %v, cursor = %d; want false, 0", r.specterSelected, r.cursor)
	}
	if _, ok := r.currentTarget(); !ok {
		t.Fatal("currentTarget reported none on a target")
	}
	press(r, downKey)
	if r.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", r.cursor)
	}
}

func TestRoot_SpecterPanelFocusEscAndTabReturnToSidebar(t *testing.T) {
	for name, k := range map[string]tea.KeyPressMsg{"esc": escKey, "tab": {Code: tea.KeyTab}} {
		t.Run(name, func(t *testing.T) {
			r := specterRoot(t)
			selectSpecter(r)
			press(r, enterKey)
			if r.focus != focusPanel {
				t.Fatal("enter did not focus the Specter panel")
			}
			press(r, k)
			if r.focus != focusSidebar || !r.specterSelected {
				t.Fatalf("focus = %v, specterSelected = %v; want sidebar with Specter still selected", r.focus, r.specterSelected)
			}
		})
	}
}

func TestRoot_SpecterIgnoresTabSwitchAndInstall(t *testing.T) {
	r := specterRoot(t)
	selectSpecter(r)
	press(r, enterKey)

	for _, k := range []string{"l", "h", "]", "["} {
		press(r, textKey(k))
		if r.tab != 0 || r.focus != focusPanel {
			t.Fatalf("%q: tab = %d, focus = %v; want 0 and panel", k, r.tab, r.focus)
		}
	}
	press(r, textKey("i"))
	if r.wizard != nil {
		t.Fatal("i opened the install wizard on the Specter screen")
	}
	press(r, escKey)
	press(r, textKey("i"))
	if r.wizard != nil {
		t.Fatal("i opened the install wizard from the sidebar with Specter selected")
	}
}

func TestRoot_SpecterPanelHasNoTabRow(t *testing.T) {
	r := specterRoot(t)
	r.Init()
	selectSpecter(r)

	view := plain(r.render())
	for _, title := range []string{"Update", "Backups"} {
		if strings.Contains(view, title) {
			t.Errorf("Specter view shows CLI tab %q", title)
		}
	}
}

func TestRoot_KeybarFollowsFocusOnSpecter(t *testing.T) {
	r := specterRoot(t)
	selectSpecter(r)

	keys, _ := r.keybarKeys()
	if !claims(keys, "enter") || claims(keys, "i") {
		t.Errorf("sidebar keybar on Specter = %v, want enter and no install", keys)
	}
	press(r, enterKey)
	keys, _ = r.keybarKeys()
	if !claims(keys, "esc") || !claims(keys, "r") || claims(keys, "i") {
		t.Errorf("panel keybar on Specter = %v, want esc and r and no install", keys)
	}
}

func TestRoot_SidebarListsSpecterThenSeparatorThenTargets(t *testing.T) {
	r := specterRoot(t)

	var rows []string
	for _, line := range strings.Split(plain(r.render()), "\n") {
		runes := []rune(line)
		rows = append(rows, string(runes[:min(len(runes), sidebarWidth)]))
	}
	index := func(sub string, from int) int {
		for i := from; i < len(rows); i++ {
			if strings.Contains(rows[i], sub) {
				return i
			}
		}
		return -1
	}
	// the box's own top border is made of ─ too, so the separator is searched below the Specter row
	specterRow := index("Specter", 0)
	separatorRow, targetRow := index("──", specterRow+1), index("Alpha", 0)
	if specterRow < 0 || separatorRow < 0 || targetRow < 0 || !(specterRow < separatorRow && separatorRow < targetRow) {
		t.Fatalf("sidebar rows: Specter %d, separator %d, Alpha %d; want strictly increasing and present\n%s", specterRow, separatorRow, targetRow, strings.Join(rows, "\n"))
	}
}

func TestDashboardTab_PrependsSpecterLineAndSeparator(t *testing.T) {
	r := specterRoot(t)
	r.Init()
	r.Update(computedMsg{idx: 0, gen: r.states[0].gen})
	dash := r.tabs[0][0]

	r.specter.phase, r.specter.loaded, r.specter.facts = loading, false, specter.Facts{}
	lines := strings.Split(plain(dash.View(60, 20)), "\n")
	if !strings.Contains(lines[0], "Specter") || !strings.Contains(lines[0], "pending") || !strings.Contains(lines[1], "─") {
		t.Fatalf("loading view starts %q / %q; want a pending Specter line and a separator", lines[0], lines[1])
	}
	if len(lines) < 3 || strings.TrimSpace(lines[2]) == "" {
		t.Fatalf("dashboard content missing under the separator: %q", lines)
	}

	facts := specter.Facts{}
	sum := specter.Summary(facts, specter.Checks(facts))
	r.specter.phase, r.specter.loaded, r.specter.facts = ready, true, facts
	lines = strings.Split(plain(dash.View(60, 20)), "\n")
	if !strings.Contains(lines[0], statusGlyph(sum.Status)) || !strings.Contains(lines[0], sum.Text) || strings.Contains(lines[0], "pending") {
		t.Fatalf("ready Specter line = %q, want glyph %q and text %q", lines[0], statusGlyph(sum.Status), sum.Text)
	}
	if !strings.Contains(lines[1], "─") {
		t.Fatalf("separator line = %q", lines[1])
	}
}
