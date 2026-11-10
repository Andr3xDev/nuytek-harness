package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const (
	dryRunNotice    = "Dry-run: nothing will be written or executed."
	nothingToDoText = "Nothing to do."
	effectLabelCols = len("overwrite")
)

func effectGlyph(e plan.Effect) string {
	switch e {
	case plan.EffectCreate:
		return "+"
	case plan.EffectOverwrite:
		return "~"
	case plan.EffectDelete:
		return "-"
	case plan.EffectRun:
		return "$"
	default:
		return "·"
	}
}

// changeRow is the plain text of one preview row; the caller wraps and styles it.
func changeRow(c plan.Change) string {
	row := fmt.Sprintf("%s %-*s %s", effectGlyph(c.Effect), effectLabelCols, c.Effect, c.Path)
	if c.Note != "" {
		row += " (" + c.Note + ")"
	}
	if c.Unrestorable {
		row += " [cannot be restored]"
	}
	return sanitizeLine(row)
}

func changeLines(th theme, rows []plan.Change, width int) []string {
	var lines []string
	for _, c := range rows {
		lines = append(lines, styleAll(th.forEffect(c.Effect), wrap(changeRow(c), width))...)
	}
	return lines
}

// stepLines groups the rows under the step that makes them: a Copy row is relative to its target,
// so without the step header the same name under two components would be ambiguous.
func stepLines(th theme, steps []plan.Step, width int) []string {
	var lines []string
	for _, s := range steps {
		if rows := s.Changes(); len(rows) > 0 {
			lines = append(lines, styleAll(th.section, wrap(s.Describe(), width))...)
			lines = append(lines, changeLines(th, rows, width)...)
		}
	}
	return lines
}

// previewBody lists the rows, then the skipped components and notes. A plan with no row but
// Skip rows is not "nothing to do": the Skip rows are the answer.
func previewBody(th theme, p plan.Plan, emptyText string, width int) []string {
	rows := p.Rows()
	var lines []string
	if p.Empty() && !slices.ContainsFunc(rows, func(c plan.Change) bool { return c.Effect == plan.EffectSkip }) {
		lines = append(lines, th.muted.Render(emptyText))
	}
	if len(p.Steps) > 0 {
		if snap, ok := p.Steps[0].(plan.Snapshot); ok {
			lines = append(lines, styleAll(th.muted, wrap(snap.Describe()+" before the first change", width))...)
		}
	}
	lines = append(lines, stepLines(th, p.Steps, width)...)
	if len(p.Skipped) > 0 {
		lines = append(lines, "", th.section.Render("Skipped components"))
		for _, s := range p.Skipped {
			lines = append(lines, styleAll(th.muted, wrap(s.Name+" - "+s.Reason, width))...)
		}
	}
	if len(p.Notes) > 0 {
		lines = append(lines, "", th.section.Render("Notes"))
		lines = append(lines, styleAll(th.muted, wrap(strings.Join(p.Notes, "\n"), width))...)
	}
	return lines
}
