package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/dashboard"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
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
	r := NewRoot([]target.Target{availableTarget()}, Deps{DryRun: true}, nil)
	r.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if out := r.render(); !strings.Contains(out, dryRunLabel) {
		t.Fatalf("render lacks %q:\n%s", dryRunLabel, out)
	}
}

func TestRoot_WarningShownAndFitsTerminalWidth(t *testing.T) {
	const text = "state file /home/u/.local/state/nuytek-harness/state.toml is corrupt: fix or delete it"
	for _, tc := range []struct{ w, h int }{{80, 24}, {40, 8}} {
		r := NewRoot([]target.Target{availableTarget()}, Deps{}, []string{text})
		r.Update(tea.WindowSizeMsg{Width: tc.w, Height: tc.h})

		out := r.render()
		if tc.w == 80 && !strings.Contains(out, "state.toml is corrupt: fix or") {
			t.Errorf("render lacks the warning:\n%s", out)
		}
		lines := strings.Split(out, "\n")
		if len(lines) > tc.h {
			t.Errorf("%dx%d: %d lines overflow the terminal", tc.w, tc.h, len(lines))
		}
		for i, line := range lines {
			if got := lipgloss.Width(line); got > tc.w {
				t.Errorf("%dx%d line %d width = %d: %q", tc.w, tc.h, i, got, line)
			}
		}
	}
}

func TestWrap_StripsTerminalSequences(t *testing.T) {
	in := "a\x1b]8;;http://x\x1b\\link\x1b]8;;\x1b\\ b\x1bPq data\x1b\\ \u009b31m c"
	for _, line := range wrap(in, 80) {
		if strings.ContainsAny(line, "\x1b\u009b") {
			t.Errorf("control character survived: %q", line)
		}
	}
}

var sanitizeCases = []struct{ name, in, wantKeepNewline string }{
	{"ESC and C1", "a\x1b]8;;x\x1b\\b\u009bc", "a]8;;x\\bc"},
	{"bidi override", "a\u202eb", "ab"},
	{"bidi isolate", "a\u2066b", "ab"},
	{"newline", "a\nb", "a\nb"},
}

func TestSanitize_DropsTerminalAndBidiControls(t *testing.T) {
	for _, tc := range sanitizeCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := sanitize(tc.in); got != tc.wantKeepNewline {
				t.Fatalf("sanitize(%q) = %q, want %q", tc.in, got, tc.wantKeepNewline)
			}
		})
	}
}

func TestSanitizeLine_AlsoDropsNewline(t *testing.T) {
	for _, tc := range sanitizeCases {
		t.Run(tc.name, func(t *testing.T) {
			want := strings.ReplaceAll(tc.wantKeepNewline, "\n", "")
			if got := sanitizeLine(tc.in); got != want {
				t.Fatalf("sanitizeLine(%q) = %q, want %q", tc.in, got, want)
			}
		})
	}
}

func TestPreviewBody_StripsBidiOverrideFromPath(t *testing.T) {
	p := plan.Plan{Steps: []plan.Step{plan.Copy{Rows: []plan.Change{
		{Effect: plan.EffectCreate, Path: "evil\u202egpj.txt"},
	}}}}

	out := strings.Join(previewBody(newTheme(defaultDark), p, "nothing", 80), "\n")
	if !strings.Contains(out, "evil") || strings.ContainsRune(out, '\u202e') {
		t.Fatalf("preview must show the path without U+202E:\n%q", out)
	}
}

func TestPreviewBody_SnapshotOnlyPlanHasNoLeadLine(t *testing.T) {
	snap := plan.Snapshot{Store: backup.Dir{Label: "home"}, Rows: []plan.Change{{Effect: plan.EffectCreate, Path: "/h/a"}}}

	out := strings.Join(previewBody(newTheme(defaultDark), plan.Plan{Steps: []plan.Step{snap}}, "nothing", 80), "\n")

	if strings.Contains(out, "before the first change") || strings.Count(out, "snapshot into home") != 1 {
		t.Fatalf("snapshot-only preview must name the snapshot once, without a lead line:\n%s", out)
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
