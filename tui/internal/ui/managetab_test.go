package ui

import (
	"context"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	manageTitleText = "Manage"
	manageHome      = "/fake-home"
	manageViewW     = 80
	manageViewH     = 40
)

type manageProbe struct {
	calls    int
	env      check.Env
	manifest manifest.Manifest
}

// manageRoot builds a Root over one available target whose Manage is fake; no real HOME is read.
func manageRoot(t *testing.T, probe *manageProbe, sections func(call int) []target.ManageSection) *Root {
	t.Helper()
	tg := availableTarget()
	tg.Manifest = manifest.Manifest{Components: []manifest.Component{{Name: "skills", Kind: manifest.KindDir, Source: "claude/skills", Target: ".claude/skills"}}}
	tg.Spec.Manage = func(_ context.Context, env check.Env, m manifest.Manifest) []target.ManageSection {
		probe.calls++
		probe.env, probe.manifest = env, m
		return sections(probe.calls)
	}
	return NewRoot([]target.Target{tg}, Deps{Env: check.Env{Home: manageHome}}, nil)
}

func tabTitles(r *Root) []string {
	var titles []string
	for _, tb := range r.tabs[0] {
		titles = append(titles, tb.Title())
	}
	return titles
}

func manageTabOf(t *testing.T, r *Root) tab {
	t.Helper()
	for _, tb := range r.tabs[0] {
		if tb.Title() == manageTitleText {
			return tb
		}
	}
	t.Fatalf("no %q tab among %v", manageTitleText, tabTitles(r))
	return nil
}

// activate runs the tab's load command to completion and feeds the result back, as the runtime would.
func activate(t *testing.T, r *Root, tb tab) tab {
	t.Helper()
	cmd := tb.Activate(r.targets[0])
	if cmd == nil {
		return tb
	}
	for _, msg := range flattenMsgs(cmd()) {
		tb, _ = tb.Update(msg)
	}
	return tb
}

func flattenMsgs(msg tea.Msg) []tea.Msg {
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			if c != nil {
				out = append(out, flattenMsgs(c())...)
			}
		}
		return out
	}
	return []tea.Msg{msg}
}

func plainSections(titles ...string) func(int) []target.ManageSection {
	return func(int) []target.ManageSection {
		var out []target.ManageSection
		for _, title := range titles {
			out = append(out, target.ManageSection{Title: title, Rows: []target.ManageRow{{Name: title + "-row"}}})
		}
		return out
	}
}

func TestRoot_ManageTabIsFifthAndWraps(t *testing.T) {
	r := manageRoot(t, &manageProbe{}, plainSections("MCPs"))
	want := []string{"Dashboard", "Update", "Pull", "Backups", manageTitleText}
	if got := tabTitles(r); !slices.Equal(got, want) {
		t.Fatalf("tabs = %v, want %v", got, want)
	}

	r.focus, r.tab = focusPanel, 3
	press(r, textKey("l"))
	if r.tab != 4 {
		t.Fatalf("l from Backups: tab = %d, want 4 (Manage)", r.tab)
	}
	press(r, textKey("l"))
	if r.tab != 0 {
		t.Fatalf("l from Manage: tab = %d, want 0 (Dashboard)", r.tab)
	}
	press(r, textKey("h"))
	if r.tab != 4 {
		t.Fatalf("h from Dashboard: tab = %d, want 4 (Manage)", r.tab)
	}
}

func TestRoot_TargetWithoutManageOrUnavailableHasNoManageTab(t *testing.T) {
	plain := newRoot(t, availableTarget())
	if slices.Contains(tabTitles(plain), manageTitleText) {
		t.Fatalf("target without Spec.Manage got a Manage tab: %v", tabTitles(plain))
	}

	unavailable := target.Target{Spec: target.Planned("p", "P")}
	unavailable.Spec.Manage = func(context.Context, check.Env, manifest.Manifest) []target.ManageSection { return nil }
	r := newRoot(t, unavailable)
	if len(r.tabs[0]) != 0 {
		t.Fatalf("unavailable target has tabs: %v", tabTitles(r))
	}
}

func TestManageTab_ListsDoctorFromCacheAndLoadedSectionsWithTargetInputs(t *testing.T) {
	probe := &manageProbe{}
	r := manageRoot(t, probe, plainSections("MCPs", "Plugins", "Skills"))
	r.states[0].loaded = true
	r.states[0].dash.Results = []check.Result{{CheckID: "doctor", Subject: "jq-subject", Status: check.StatusError, Message: "jq missing"}}

	tb := activate(t, r, manageTabOf(t, r))
	view := tb.View(manageViewW, manageViewH)

	for _, want := range []string{"Doctor", "jq-subject", "MCPs", "Plugins", "Skills"} {
		if !strings.Contains(view, want) {
			t.Errorf("view lacks %q:\n%s", want, view)
		}
	}
	if probe.calls != 1 {
		t.Errorf("Manage called %d times, want 1", probe.calls)
	}
	if probe.env.Home != manageHome || len(probe.manifest.Components) != 1 {
		t.Errorf("Manage got env.Home %q and %d components, want %q and the target manifest", probe.env.Home, len(probe.manifest.Components), manageHome)
	}
}

func TestManageTab_ShowsEmptyAndUnknownReasons(t *testing.T) {
	r := manageRoot(t, &manageProbe{}, func(int) []target.ManageSection {
		return []target.ManageSection{
			{Title: "MCPs", Empty: "no MCP servers configured"},
			{Title: "Plugins", Unknown: "invalid JSON at byte 7"},
		}
	})
	r.states[0].loaded = true

	view := activate(t, r, manageTabOf(t, r)).View(manageViewW, manageViewH)

	for _, want := range []string{"no MCP servers configured", "invalid JSON at byte 7"} {
		if !strings.Contains(view, want) {
			t.Errorf("view lacks %q:\n%s", want, view)
		}
	}
}

func TestManageTab_PreviewFollowsSelectedRow(t *testing.T) {
	r := manageRoot(t, &manageProbe{}, func(int) []target.ManageSection {
		return []target.ManageSection{{Title: "Skills", Rows: []target.ManageRow{
			{Name: "alpha", Preview: "PREVIEW-ALPHA"},
			{Name: "beta", Preview: "PREVIEW-BETA"},
		}}}
	})
	r.states[0].loaded = true
	tb := activate(t, r, manageTabOf(t, r))

	if view := tb.View(manageViewW, manageViewH); !strings.Contains(view, "PREVIEW-ALPHA") || strings.Contains(view, "PREVIEW-BETA") {
		t.Fatalf("first row selected, view:\n%s", view)
	}
	tb, _ = tb.Update(downKey)
	if view := tb.View(manageViewW, manageViewH); !strings.Contains(view, "PREVIEW-BETA") || strings.Contains(view, "PREVIEW-ALPHA") {
		t.Fatalf("after down, view:\n%s", view)
	}
}

func TestManageTab_ExposesOnlyMoveAndIgnoresEveryOtherKey(t *testing.T) {
	r := manageRoot(t, &manageProbe{}, plainSections("MCPs"))
	r.states[0].loaded = true
	tb := activate(t, r, manageTabOf(t, r))

	keys := tb.Keys()
	if len(keys) != 1 || !claims(keys, "up") || !claims(keys, "down") {
		t.Fatalf("Keys() = %v, want only the Move binding", keys)
	}
	before := tb.View(manageViewW, manageViewH)
	for _, k := range []tea.KeyPressMsg{textKey("b"), textKey("d"), textKey("r"), textKey("i"), enterKey} {
		var cmd tea.Cmd
		tb, cmd = tb.Update(k)
		if cmd != nil {
			t.Errorf("key %q returned a command", k.String())
		}
	}
	if after := tb.View(manageViewW, manageViewH); after != before {
		t.Fatalf("view changed on action keys:\n%s\n---\n%s", before, after)
	}
}

func TestManageTab_DropsALoadSupersededByANewerActivation(t *testing.T) {
	r := manageRoot(t, &manageProbe{}, func(call int) []target.ManageSection {
		return []target.ManageSection{{Title: "Load" + string(rune('0'+call))}}
	})
	r.states[0].loaded = true
	tb := manageTabOf(t, r)

	first := tb.Activate(r.targets[0])
	second := tb.Activate(r.targets[0])
	// Run in activation order so the fake numbers the loads by activation; deliver them reversed.
	firstMsg, secondMsg := first(), second()
	tb, _ = tb.Update(secondMsg)
	tb, _ = tb.Update(firstMsg)

	view := tb.View(manageViewW, manageViewH)
	if !strings.Contains(view, "Load2") || strings.Contains(view, "Load1") {
		t.Fatalf("stale load applied, view:\n%s", view)
	}
}

func TestManageTab_ShowsComputingOnceBeforeDashboardAndManageLoad(t *testing.T) {
	r := manageRoot(t, &manageProbe{}, plainSections("MCPs"))
	tb := manageTabOf(t, r)

	view := tb.View(manageViewW, manageViewH)
	if n := strings.Count(view, computingText); n != 1 {
		t.Fatalf("%q shown %d times before any load, want 1:\n%s", computingText, n, view)
	}
	if !strings.Contains(view, manageLoadingText) {
		t.Fatalf("view lacks %q before Manage loads:\n%s", manageLoadingText, view)
	}

	r.states[0].loaded = true
	view = tb.View(manageViewW, manageViewH)
	if strings.Contains(view, computingText) || !strings.Contains(view, manageLoadingText) {
		t.Fatalf("dashboard loaded, Manage not, view:\n%s", view)
	}

	view = activate(t, r, tb).View(manageViewW, manageViewH)
	if strings.Contains(view, computingText) || strings.Contains(view, manageLoadingText) {
		t.Fatalf("everything loaded, view still shows a loading line:\n%s", view)
	}
}

func TestManageTab_ClampsCursorWhenANewLoadHasFewerRows(t *testing.T) {
	r := manageRoot(t, &manageProbe{}, func(call int) []target.ManageSection {
		rows := []target.ManageRow{{Name: "alpha"}, {Name: "beta"}, {Name: "gamma"}}
		if call > 1 {
			rows = rows[:1]
		}
		return []target.ManageSection{{Title: "Skills", Rows: rows}}
	})
	r.states[0].loaded = true
	tb := activate(t, r, manageTabOf(t, r))
	tb, _ = tb.Update(downKey)
	tb, _ = tb.Update(downKey)

	tb = activate(t, r, tb)

	if got := tb.(*manageTab).cursor; got != 0 {
		t.Fatalf("cursor = %d after a load with one row, want 0", got)
	}
	if view := tb.View(manageViewW, manageViewH); !strings.Contains(view, "alpha") {
		t.Fatalf("view lacks the remaining row:\n%s", view)
	}
}

func TestManageTab_SmallHeightKeepsPreviewWithinCap(t *testing.T) {
	const height = 9
	preview := strings.TrimSuffix(strings.Repeat("preview line\n", 30), "\n")
	r := manageRoot(t, &manageProbe{}, func(int) []target.ManageSection {
		return []target.ManageSection{{Title: "Skills", Rows: []target.ManageRow{{Name: "alpha", Preview: preview}}}}
	})
	r.states[0].loaded = true

	view := activate(t, r, manageTabOf(t, r)).View(manageViewW, height)

	lines := strings.Split(view, "\n")
	if len(lines) > height {
		t.Fatalf("view has %d lines, want at most %d:\n%s", len(lines), height, view)
	}
	if n := strings.Count(view, "preview line"); n != height/3 {
		t.Fatalf("preview shows %d lines, want the cap of %d:\n%s", n, height/3, view)
	}
}

func TestManageTab_DoctorFollowsDashboardRefresh(t *testing.T) {
	r := manageRoot(t, &manageProbe{}, plainSections("MCPs"))
	st := &r.states[0]
	st.loaded = true
	st.dash.Results = []check.Result{{Subject: "old-subject", Status: check.StatusError}}
	tb := activate(t, r, manageTabOf(t, r))

	st.dash.Results = []check.Result{{Subject: "new-subject", Status: check.StatusOK}}
	tb, _ = tb.Update(dashboardReady{st: st})

	view := tb.View(manageViewW, manageViewH)
	if !strings.Contains(view, "new-subject") || strings.Contains(view, "old-subject") {
		t.Fatalf("Doctor not refreshed, view:\n%s", view)
	}
}
