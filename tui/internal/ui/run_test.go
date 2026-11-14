package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const (
	closeHint     = "Press esc to close."
	testBodyH     = 30
	testPopupW    = 80
	longRunEvents = 3000
)

func runWithLines(n int) *run {
	r := newRun("x", false)
	for i := range n {
		r.add(plan.Event{Phase: plan.PhaseLine, Text: fmt.Sprintf("line %d", i)})
	}
	return r
}

// lastContentRow is the box row just above the bottom border.
func lastContentRow(box string) string {
	rows := strings.Split(box, "\n")
	return rows[len(rows)-2]
}

func TestOutputPopup_ShortRunIsContentSized(t *testing.T) {
	r := runWithLines(3)
	r.finish(plan.Result{})

	p := outputPopup(newTheme(defaultDark), r, testPopupW, testBodyH)

	// +1 for the close hint line; these short lines do not wrap.
	if want := len(r.lines) + 1 + borderCells; p.h != want || lipgloss.Height(p.box) != want {
		t.Fatalf("popup h = %d (box %d rows), want %d for %d content lines", p.h, lipgloss.Height(p.box), want, len(r.lines))
	}
}

func TestOutputPopup_RunningFollowsContentUpToBodyHeight(t *testing.T) {
	th := newTheme(defaultDark)
	for _, n := range []int{1, 2} {
		p := outputPopup(th, runWithLines(n), testPopupW, testBodyH)
		if want := runningFloorRows + borderCells; p.h != want || lipgloss.Height(p.box) != want {
			t.Fatalf("%d-line running pane: h = %d (box %d rows), want the floor %d", n, p.h, lipgloss.Height(p.box), want)
		}
	}
	mid := outputPopup(th, runWithLines(10), testPopupW, testBodyH)
	if want := 10 + borderCells; mid.h != want {
		t.Fatalf("10-line running pane: h = %d, want %d", mid.h, want)
	}
	long := outputPopup(th, runWithLines(longRunEvents), testPopupW, testBodyH)
	if want := testBodyH - outputMargin; long.h != want || lipgloss.Height(long.box) != want {
		t.Fatalf("long running pane: h = %d (box %d rows), want %d", long.h, lipgloss.Height(long.box), want)
	}
}

func TestOutputPopup_WidthIsCappedAndFitsNarrowTerminals(t *testing.T) {
	th := newTheme(defaultDark)
	const narrowW = 40
	if p := outputPopup(th, runWithLines(3), 200, testBodyH); p.w != maxOutputWidth || lipgloss.Width(p.box) != maxOutputWidth {
		t.Fatalf("wide terminal: w = %d (box %d), want %d", p.w, lipgloss.Width(p.box), maxOutputWidth)
	}
	if p := outputPopup(th, runWithLines(3), narrowW, testBodyH); p.w != narrowW-outputMargin || lipgloss.Width(p.box) != p.w {
		t.Fatalf("narrow terminal: w = %d (box %d), want %d", p.w, lipgloss.Width(p.box), narrowW-outputMargin)
	}
}

func TestOutputPopup_DegenerateBodyHeightDoesNotPanic(t *testing.T) {
	for bodyH := 0; bodyH <= outputMargin+borderCells+1; bodyH++ {
		running := outputPopup(newTheme(defaultDark), runWithLines(5), testPopupW, bodyH)
		finished := runWithLines(5)
		finished.finish(plan.Result{})
		done := outputPopup(newTheme(defaultDark), finished, testPopupW, bodyH)
		if running.h < borderCells || done.h < borderCells {
			t.Fatalf("bodyH %d: heights %d/%d below the border cells", bodyH, running.h, done.h)
		}
	}
}

func TestOutputPopup_LongRunIsCappedAtBodyHeight(t *testing.T) {
	r := runWithLines(longRunEvents)
	r.finish(plan.Result{})

	p := outputPopup(newTheme(defaultDark), r, testPopupW, testBodyH)

	if p.h > testBodyH || lipgloss.Height(p.box) != p.h {
		t.Fatalf("popup h = %d (box %d rows), want at most the body height %d", p.h, lipgloss.Height(p.box), testBodyH)
	}
}

func TestOutputPopup_FinishedRunEndsWithCloseHint(t *testing.T) {
	failure := plan.Result{Failed: &plan.StepFailure{Step: "boom", Err: errors.New("bad")}}
	cases := map[string]func() *run{
		"done": func() *run {
			r := runWithLines(3)
			r.finish(plan.Result{})
			return r
		},
		"failed": func() *run {
			r := runWithLines(3)
			r.finish(failure)
			return r
		},
		"cancelled": func() *run {
			r := runWithLines(3)
			r.cancel = func() {}
			r.requestCancel()
			r.finish(failure)
			return r
		},
		"long": func() *run {
			r := runWithLines(longRunEvents)
			r.finish(plan.Result{})
			return r
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			p := outputPopup(newTheme(defaultDark), build(), testPopupW, testBodyH)

			if row := lastContentRow(p.box); !strings.Contains(row, closeHint) {
				t.Fatalf("last content row = %q, want it to hold %q", row, closeHint)
			}
		})
	}
}

func TestOutputPopup_RunningHasNoCloseHint(t *testing.T) {
	p := outputPopup(newTheme(defaultDark), runWithLines(3), testPopupW, testBodyH)

	if strings.Contains(p.box, closeHint) {
		t.Fatalf("a running pane must not show %q:\n%s", closeHint, p.box)
	}
}

func TestRun_KeepsOnlyTheLastLinesAndTheirWrappedCache(t *testing.T) {
	r := newRun("x", false)
	th := newTheme(defaultDark)
	const extra = 50
	for i := 0; i < maxOutputLines+extra; i++ {
		r.add(plan.Event{Phase: plan.PhaseLine, Text: fmt.Sprintf("line %d", i)})
		if i == maxOutputLines {
			r.view(th, 40, 10)
		}
	}

	if len(r.lines) != maxOutputLines {
		t.Fatalf("stored lines = %d, want %d", len(r.lines), maxOutputLines)
	}
	if want := fmt.Sprintf("    line %d", extra); r.lines[0].text != want {
		t.Fatalf("first kept line = %q, want %q", r.lines[0].text, want)
	}
	last := r.lines[len(r.lines)-1].text
	if want := fmt.Sprintf("    line %d", maxOutputLines+extra-1); last != want {
		t.Fatalf("last line = %q, want %q", last, want)
	}
	if len(r.view(th, 40, 10)) != 10 {
		t.Fatal("view did not fill the window")
	}
	if len(r.wrapped) != len(r.lines) {
		t.Fatalf("wrapped cache = %d entries, want %d", len(r.wrapped), len(r.lines))
	}
}
