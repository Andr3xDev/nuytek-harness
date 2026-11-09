package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/dashboard"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
)

const (
	backupTimeLayout = "2006-01-02 15:04"
	stateLabelWidth  = len("not installed") // longest drift.State label, keeps component names aligned
)

var countOrder = []check.Status{check.StatusOK, check.StatusWarning, check.StatusError, check.StatusUnknown}

func driftLabel(s drift.State) string {
	return strings.ReplaceAll(string(s), "_", " ")
}

// countRows joins the status counts on one line, starting a new line only when the next does not fit.
func countRows(t theme, d dashboard.Dashboard, width int) []string {
	const separator = "  "
	var rows, line []string
	lineW := 0
	flush := func() {
		if len(line) > 0 {
			rows = append(rows, strings.Join(line, separator))
			line, lineW = nil, 0
		}
	}
	for _, s := range countOrder {
		plain := fmt.Sprintf("%s %s %d", statusGlyph(s), s, d.Counts[s])
		w := lipgloss.Width(plain)
		if len(line) > 0 && lineW+len(separator)+w > width {
			flush()
		}
		if w > width {
			rows = append(rows, styleAll(t.forStatus(s), wrap(plain, width))...)
			continue
		}
		if len(line) > 0 {
			lineW += len(separator)
		}
		line, lineW = append(line, t.forStatus(s).Render(plain)), lineW+w
	}
	flush()
	return rows
}

// dashboardLines renders d as plain wrapped lines, one section after another.
func dashboardLines(t theme, d dashboard.Dashboard, width int) []string {
	var rows []string
	add := func(style lipgloss.Style, plain string) {
		// Wrap the plain text first so ANSI sequences are never split, then style each row.
		for _, l := range wrap(plain, width) {
			rows = append(rows, style.Render(strings.TrimRight(l, " ")))
		}
	}
	section := func(name string) {
		if len(rows) > 0 {
			rows = append(rows, "")
		}
		rows = append(rows, t.section.Render(name))
	}

	section("Checks")
	rows = append(rows, countRows(t, d, width)...)

	section("Issues")
	issues := d.Issues()
	if len(issues) == 0 {
		rows = append(rows, t.ok.Render(statusGlyph(check.StatusOK)+" No issues"))
	}
	for _, r := range issues {
		plain := fmt.Sprintf("%s %s: %s", statusGlyph(r.Status), r.Subject, r.Message)
		add(t.forStatus(r.Status), plain)
	}

	section("Components")
	for _, c := range d.Components {
		plain := fmt.Sprintf("%s %-*s  %s", driftGlyph(c.State), stateLabelWidth, driftLabel(c.State), c.Name)
		if c.Reason != "" {
			plain += " - " + c.Reason
		}
		add(t.forDrift(c.State), plain)
	}

	section("Missing dependencies")
	if len(d.MissingDeps) == 0 {
		rows = append(rows, t.ok.Render(statusGlyph(check.StatusOK)+" None missing"))
	}
	for _, m := range d.MissingDeps {
		plain := statusGlyph(check.StatusError) + " " + m.Tool
		if len(m.Dependents) > 0 {
			plain += " (needed by " + strings.Join(m.Dependents, ", ") + ")"
		}
		add(t.err, plain)
	}

	section("Last backup")
	if d.LastBackup == nil {
		rows = append(rows, t.muted.Render("no backups"))
	} else {
		add(t.plain, fmt.Sprintf("%s (%s)", d.LastBackup.Time.Format(backupTimeLayout), d.LastBackup.Label))
	}
	return rows
}
