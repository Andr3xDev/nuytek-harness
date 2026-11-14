package ui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	manageTitle       = "Manage"
	manageLoadingText = "Loading sections..."
)

// manageLoaded carries the loaded sections; epoch drops one that a newer load superseded.
type manageLoaded struct {
	m        *manageTab
	epoch    int
	sections []target.ManageSection
}

// manageTab is a read-only view over one target: its Doctor results come from the cached dashboard,
// the other sections from Spec.Manage. It runs no plan.
type manageTab struct {
	svc      *services
	st       *targetState
	sections []target.ManageSection // loaded by Spec.Manage; Doctor is not among them
	loaded   bool
	epoch    int
	cursor   int // flat over the rows of every section, in display order
}

func newManageTab(svc *services, st *targetState) *manageTab {
	return &manageTab{svc: svc, st: st}
}

func (m *manageTab) Title() string { return manageTitle }

func (m *manageTab) Activate(t target.Target) tea.Cmd {
	m.epoch++
	epoch, env, ctx := m.epoch, m.svc.Env, m.svc.ctx
	return func() tea.Msg {
		return manageLoaded{m: m, epoch: epoch, sections: t.Spec.Manage(ctx, env, t.Manifest)}
	}
}

// all puts Doctor first; it follows the dashboard cache, so it is never probed a second time.
func (m *manageTab) all() []target.ManageSection {
	doctor := target.DoctorSection(m.st.dash.Results)
	if !m.st.loaded {
		doctor.Empty = computingText
	}
	return append([]target.ManageSection{doctor}, m.sections...)
}

func rowCount(sections []target.ManageSection) int {
	n := 0
	for _, s := range sections {
		n += len(s.Rows)
	}
	return n
}

func (m *manageTab) clampCursor() {
	m.cursor = max(min(m.cursor, rowCount(m.all())-1), 0)
}

func (m *manageTab) Update(msg tea.Msg) (tab, tea.Cmd) {
	switch msg := msg.(type) {
	case manageLoaded:
		if msg.m == m && msg.epoch == m.epoch {
			m.sections, m.loaded = msg.sections, true
			m.clampCursor()
		}
	case dashboardReady:
		if msg.st == m.st {
			m.clampCursor()
		}
	case tea.KeyPressMsg:
		k := m.svc.keys
		switch {
		case key.Matches(msg, k.Up):
			m.cursor = max(m.cursor-1, 0)
		case key.Matches(msg, k.Down):
			m.cursor = max(min(m.cursor+1, rowCount(m.all())-1), 0)
		}
	}
	return m, nil
}

func (m *manageTab) Keys() []key.Binding { return []key.Binding{m.svc.keys.Move} }

func (m *manageTab) View(w, h int) string {
	th := m.svc.th
	var lines []string
	var selected target.ManageRow
	at, seen := 0, 0
	for _, s := range m.all() {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, styleAll(th.section, wrap(s.Title, w))...)
		switch {
		case s.Unknown != "":
			lines = append(lines, styleAll(th.unknown, wrap(unknownText+": "+s.Unknown, w))...)
		case len(s.Rows) == 0:
			lines = append(lines, styleAll(th.muted, wrap(s.Empty, w))...)
		default:
			rows := make([]listRow, len(s.Rows))
			for i, r := range s.Rows {
				rows[i] = listRow{text: manageRowText(r)}
			}
			local := m.cursor - seen
			sectionLines, sectionAt := listLines(th, rows, local, w)
			if local >= 0 && local < len(rows) {
				at, selected = len(lines)+sectionAt, s.Rows[local]
			}
			lines = append(lines, sectionLines...)
			seen += len(rows)
		}
	}
	if !m.loaded {
		lines = append(lines, "", th.muted.Render(manageLoadingText))
	}
	preview := m.previewLines(selected, w, h)
	return strings.Join(append(windowAround(lines, at, h-len(preview)), preview...), "\n")
}

// previewLines is the selected row's long text under a separator, at most a third of the height.
func (m *manageTab) previewLines(r target.ManageRow, w, h int) []string {
	text := r.Preview
	if text == "" {
		text = r.Detail
	}
	if text == "" {
		return nil
	}
	body := wrap(text, w)
	body = body[:min(len(body), max(h/3, 1))]
	return append([]string{separatorLine(m.svc.th, w)}, body...)
}

func manageRowText(r target.ManageRow) string {
	text := r.Name
	if r.Status != "" {
		text = statusGlyph(r.Status) + " " + text
	}
	if first, _, _ := strings.Cut(r.Detail, "\n"); first != "" {
		text += "  " + first
	}
	return sanitizeLine(text)
}

var _ tab = (*manageTab)(nil)
