package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/dashboard"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

func TestRenderBox_LongTitleKeepsEveryLineAtWidth(t *testing.T) {
	const w = 20
	out := renderBox(newTheme(defaultDark), strings.Repeat("T", 100), []string{"x"}, w, 5, true)

	for i, line := range strings.Split(out, "\n") {
		if got := lipgloss.Width(line); got != w {
			t.Errorf("line %d width = %d, want %d: %q", i, got, w, line)
		}
	}
}

func TestRoot_DryRunBadgeShown(t *testing.T) {
	r := NewRoot([]target.Target{availableTarget()}, check.Env{}, true)
	r.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if out := r.render(); !strings.Contains(out, dryRunLabel) {
		t.Fatalf("render lacks %q:\n%s", dryRunLabel, out)
	}
}

func TestRoot_RenderFitsTerminalWidth(t *testing.T) {
	long := availableTarget()
	long.Spec.Name = strings.Repeat("Name", 20)
	notAvail := target.Target{Spec: target.Spec{ID: "p", Name: "Planned target with a long name"}}

	for _, tc := range []struct {
		name  string
		w, h  int
		ready bool
		help  bool
	}{
		{"help 80x24", 80, 24, false, true},
		{"help 40x8", 40, 8, false, true},
		{"80x24", 80, 24, false, false},
		{"40x8", 40, 8, false, false},
		{"39x8", 39, 8, false, false},
		{"ready 80x24", 80, 24, true, false},
		{"ready 40x8", 40, 8, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRoot(t, long, notAvail)
			r.Update(tea.WindowSizeMsg{Width: tc.w, Height: tc.h})
			if tc.ready {
				r.Init() // marks the target loading; the returned Cmd is not run
				counts := map[check.Status]int{}
				for _, s := range countOrder {
					counts[s] = 123456
				}
				r.Update(computedMsg{idx: 0, gen: r.states[0].gen, dash: dashboard.Dashboard{Counts: counts}})
			}

			r.showHelp = tc.help
			out := r.render()
			if tc.help {
				for _, want := range []string{helpTitle, sidebarTitle} {
					// At 40x8 the clamped popup covers the whole body.
					if want == sidebarTitle && tc.w < 80 {
						continue
					}
					if !strings.Contains(out, want) {
						t.Errorf("render with help open lacks %q:\n%s", want, out)
					}
				}
			}
			for i, line := range strings.Split(out, "\n") {
				if got := lipgloss.Width(line); got > tc.w {
					t.Errorf("line %d width = %d, want <= %d: %q", i, got, tc.w, line)
				}
			}
		})
	}
}
