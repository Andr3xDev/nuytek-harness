package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

func available(t target.Target) bool { return t.Err == nil && t.Spec.Checks != nil }

// sidebarHeader is the rows above the CLI targets: the Specter entry and a separator.
const sidebarHeader = 2

func selectableRow(th theme, label string, selected bool, width int) string {
	label = truncate("  "+label, width)
	if !selected {
		return label
	}
	plain := "> " + strings.TrimPrefix(label, "  ")
	return th.selected.Render(plain + strings.Repeat(" ", max(width-lipgloss.Width(plain), 0)))
}

func sidebarLines(th theme, targets []target.Target, cursor int, specterSelected bool, width int) []string {
	lines := make([]string, sidebarHeader+len(targets))
	lines[0] = selectableRow(th, specterName, specterSelected, width)
	lines[1] = separatorLine(th, width)
	for i, t := range targets {
		label := t.Spec.Name
		switch {
		case t.Err != nil:
			label += " (error)"
		case t.Spec.Checks == nil:
			label += " (not available)"
		}
		selected := i == cursor && !specterSelected
		label = selectableRow(th, label, selected, width)
		if !selected && !available(t) {
			label = th.muted.Render(label)
		}
		lines[sidebarHeader+i] = label
	}
	return lines
}
