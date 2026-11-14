package ui

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

const dryRunLabel = "DRY-RUN"

// keybar renders the bindings valid in the current focus, with the dry-run badge on the right.
func keybar(t theme, h help.Model, bindings []key.Binding, dryRun bool, width int) string {
	badge := ""
	if dryRun {
		badge = t.badge.Render(dryRunLabel)
	}
	h.Styles = t.help
	h.SetWidth(max(width-lipgloss.Width(badge)-1, 0))
	left := h.ShortHelpView(bindings)
	gap := max(width-lipgloss.Width(left)-lipgloss.Width(badge), 0)
	return left + strings.Repeat(" ", gap) + badge
}
