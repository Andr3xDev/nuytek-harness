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
	svc    *services
	st     *targetState
	scroll scroller
}

func newDashboardTab(svc *services, st *targetState) *dashboardTab {
	return &dashboardTab{svc: svc, st: st}
}

func (d *dashboardTab) Title() string { return dashboardTitle }

func (d *dashboardTab) Activate(target.Target) tea.Cmd {
	d.scroll.reset()
	return nil
}

func (d *dashboardTab) Update(msg tea.Msg) (tab, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(k, d.svc.keys.Up):
			d.scroll.by(-1)
		case key.Matches(k, d.svc.keys.Down):
			d.scroll.by(1)
		}
	}
	return d, nil
}

func (d *dashboardTab) View(w, h int) string {
	if !d.st.loaded {
		return d.svc.th.muted.Render(computingText)
	}
	return strings.Join(d.scroll.window(dashboardLines(d.svc.th, d.st.dash, w), h), "\n")
}

func (d *dashboardTab) Keys() []key.Binding { return []key.Binding{d.svc.keys.Move} }
