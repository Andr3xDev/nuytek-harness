package ui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

// defaultDark is fixed: querying the terminal background is not worth a startup round trip.
const defaultDark = true

type theme struct {
	focusedBorder, blurredBorder lipgloss.Style
	title, selected, muted       lipgloss.Style
	ok, warning, err, unknown    lipgloss.Style
	badge, tabActive, tabIdle    lipgloss.Style
	section, plain               lipgloss.Style
	help                         help.Styles
}

func newTheme(isDark bool) theme {
	pick := lipgloss.LightDark(isDark)
	accent := pick(lipgloss.Color("#0B7A3B"), lipgloss.Color("#5FD787"))
	warn := pick(lipgloss.Color("#9A6700"), lipgloss.Color("#FFD75F"))
	dim := pick(lipgloss.Color("#6E6E6E"), lipgloss.Color("#8A8A8A"))
	fg := pick(lipgloss.Color("#1C1C1C"), lipgloss.Color("#E4E4E4"))
	selectedBg := pick(lipgloss.Color("#CFE8D8"), lipgloss.Color("#2A3B31"))

	helpStyles := help.DefaultStyles(isDark)
	helpStyles.ShortKey = lipgloss.NewStyle().Bold(true).Foreground(accent)
	helpStyles.FullKey = helpStyles.ShortKey
	helpStyles.ShortDesc = lipgloss.NewStyle().Foreground(dim)
	helpStyles.FullDesc = helpStyles.ShortDesc
	helpStyles.ShortSeparator = lipgloss.NewStyle().Foreground(dim)
	helpStyles.FullSeparator = helpStyles.ShortSeparator

	return theme{
		focusedBorder: lipgloss.NewStyle().Foreground(accent),
		blurredBorder: lipgloss.NewStyle().Foreground(dim),
		title:         lipgloss.NewStyle().Bold(true),
		selected:      lipgloss.NewStyle().Bold(true).Foreground(fg).Background(selectedBg),
		muted:         lipgloss.NewStyle().Foreground(dim),
		ok:            lipgloss.NewStyle().Foreground(accent),
		warning:       lipgloss.NewStyle().Foreground(warn),
		err:           lipgloss.NewStyle().Foreground(pick(lipgloss.Color("#B3261E"), lipgloss.Color("#FF5F5F"))),
		unknown:       lipgloss.NewStyle().Foreground(dim),
		badge:         lipgloss.NewStyle().Bold(true).Foreground(pick(lipgloss.Color("#FFFFFF"), lipgloss.Color("#000000"))).Background(warn).Padding(0, 1),
		tabActive:     lipgloss.NewStyle().Bold(true).Foreground(accent).Underline(true),
		tabIdle:       lipgloss.NewStyle().Foreground(dim),
		section:       lipgloss.NewStyle().Bold(true).Foreground(accent),
		plain:         lipgloss.NewStyle(),
		help:          helpStyles,
	}
}

func statusGlyph(s check.Status) string {
	switch s {
	case check.StatusOK:
		return "✓"
	case check.StatusWarning:
		return "!"
	case check.StatusError:
		return "✗"
	default:
		return "?"
	}
}

func driftGlyph(s drift.State) string {
	switch s {
	case drift.StateInstalled:
		return "●"
	case drift.StateDrift:
		return "◐"
	case drift.StateNotInstalled:
		return "○"
	default:
		return "?"
	}
}

func (t theme) forStatus(s check.Status) lipgloss.Style {
	switch s {
	case check.StatusOK:
		return t.ok
	case check.StatusWarning:
		return t.warning
	case check.StatusError:
		return t.err
	default:
		return t.unknown
	}
}

func (t theme) forDrift(s drift.State) lipgloss.Style {
	switch s {
	case drift.StateInstalled:
		return t.ok
	case drift.StateDrift:
		return t.warning
	case drift.StateNotInstalled:
		return t.err
	default:
		return t.unknown
	}
}

func (t theme) forEffect(e plan.Effect) lipgloss.Style {
	switch e {
	case plan.EffectCreate:
		return t.ok
	case plan.EffectOverwrite:
		return t.warning
	case plan.EffectDelete:
		return t.err
	case plan.EffectRun:
		return t.plain
	default:
		return t.muted
	}
}

func (t theme) border(focused bool) lipgloss.Style {
	if focused {
		return t.focusedBorder
	}
	return t.blurredBorder
}
