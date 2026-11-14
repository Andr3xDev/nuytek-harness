package ui

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

// Deps is everything the shell needs from the outside; ui never imports an OS-facing package.
type Deps struct {
	Env    check.Env
	Runner plan.Runner
	DryRun bool
}

// services is what every tab and flow shares.
type services struct {
	Deps
	ctx  context.Context
	th   theme
	keys keyMap
}

// tab is one panel of an available target. Root offers a key to the tab only when
// Keys() lists it, so a tab owns exactly the keys it advertises.
type tab interface {
	Title() string
	Activate(t target.Target) tea.Cmd // target selected or tab entered
	Update(msg tea.Msg) (tab, tea.Cmd)
	View(w, h int) string
	Keys() []key.Binding
}

// dashboardReady tells the tabs of one target that its cached dashboard changed.
type dashboardReady struct{ st *targetState }

// runDismissed tells the tabs that the output pane was closed and files may have changed.
type runDismissed struct{}
