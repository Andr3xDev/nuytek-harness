package ui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const dashboardTitle = "Dashboard"

// dashboardTab shows the cached dashboard of its target; the previous one stays while a refresh runs.
type dashboardTab struct {
	svc     *services
	st      *targetState
	specter func() *specterState
	scroll  scroller
}

func newDashboardTab(svc *services, st *targetState, specter func() *specterState) *dashboardTab {
	return &dashboardTab{svc: svc, st: st, specter: specter}
}

func (d *dashboardTab) Title() string { return dashboardTitle }

func (d *dashboardTab) Activate(target.Target) tea.Cmd {
	d.scroll.reset()
	return nil
}

func (d *dashboardTab) Update(msg tea.Msg) (tab, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		d.scroll.by(d.svc.keys.step(k))
	}
	return d, nil
}

func (d *dashboardTab) View(w, h int) string {
	th := d.svc.th
	head := []string{specterLine(th, d.specter(), w), separatorLine(th, w)}
	if !d.st.loaded {
		return strings.Join(append(head, th.muted.Render(computingText)), "\n")
	}
	body := d.scroll.window(dashboardLines(th, d.st.dash, w), max(h-len(head), 0))
	return strings.Join(append(head, body...), "\n")
}

func (d *dashboardTab) Keys() []key.Binding { return []key.Binding{d.svc.keys.Move} }
