package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	planningText    = "Planning..."
	confirmTitle    = "Confirm"
	confirmBoxWidth = 50
)

func (f *flow) View(w, h int) string {
	if f.stage == stageConfirm {
		return f.confirmView(w, h)
	}
	var lines []string
	switch f.stage {
	case stageSelect:
		lines = f.selectLines(w, h)
	case stagePlanning:
		lines = []string{f.svc.th.muted.Render(planningText)}
	case stagePreview:
		lines = f.previewLines(w, h)
	}
	return strings.Join(lines, "\n")
}

func (f *flow) selectLines(w, h int) []string {
	th := f.svc.th
	rows := make([]listRow, len(f.choices))
	for i, c := range f.choices {
		mark := "[ ]"
		switch {
		case c.disabled:
			mark = "[-]"
		case f.picked[c.name]:
			mark = "[x]"
		}
		text := mark + " " + c.name
		if c.note != "" {
			text += " - " + sanitizeLine(c.note)
		}
		rows[i] = listRow{text: text, muted: c.disabled}
	}
	list, at := listLines(th, rows, f.cursor, w)
	listH := h - 1 // title line
	if f.notice != "" {
		listH--
	}
	lines := append([]string{th.section.Render(f.prompt)}, windowAround(list, at, listH)...)
	if f.notice != "" {
		lines = append(lines, th.warning.Render(f.notice))
	}
	return lines
}

func (f *flow) header(w int) []string {
	th := f.svc.th
	lines := styleAll(th.section, wrap(f.title, w))
	if f.svc.DryRun {
		lines = append(lines, styleAll(th.warning, wrap(dryRunNotice, w))...)
	}
	return lines
}

func (f *flow) previewLines(w, h int) []string {
	th := f.svc.th
	if f.err != nil {
		lines := styleAll(th.section, wrap(f.title+": cannot continue", w))
		return append(lines, styleAll(th.err, wrap(f.err.Error(), w))...)
	}
	header := f.header(w)
	body := previewBody(th, f.p, f.emptyText, w)
	return append(header, f.body.window(body, h-len(header))...)
}

// confirmView draws the question as a compact box centered in the w x h panel.
func (f *flow) confirmView(w, h int) string {
	th := f.svc.th
	boxW := min(confirmBoxWidth, w)
	inner := contentWidth(boxW)
	header := f.header(inner)
	body := styleAll(th.warning, wrap(f.p.Confirm, inner))
	lines := append(header, f.body.window(body, max(h-borderCells-len(header), 0))...)
	lines = lines[:min(len(lines), max(h-borderCells, 0))]
	box := renderBox(th, confirmTitle, lines, boxW, len(lines)+borderCells, true)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}
