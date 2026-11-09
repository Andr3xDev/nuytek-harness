package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

func available(t target.Target) bool { return t.Err == nil && t.Spec.Checks != nil }

func sidebarLines(th theme, targets []target.Target, cursor, width int) []string {
	lines := make([]string, len(targets))
	for i, t := range targets {
		label := t.Spec.Name
		switch {
		case t.Err != nil:
			label += " (error)"
		case t.Spec.Checks == nil:
			label += " (not available)"
		}
		label = truncate("  "+label, width)
		switch {
		case i == cursor:
			plain := "> " + strings.TrimPrefix(label, "  ")
			label = th.selected.Render(plain + strings.Repeat(" ", max(width-lipgloss.Width(plain), 0)))
		case !available(t):
			label = th.muted.Render(label)
		}
		lines[i] = label
	}
	return lines
}
