package ui

import (
	"fmt"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

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
