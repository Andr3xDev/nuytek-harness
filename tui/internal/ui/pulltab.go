package ui

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	pullTitle      = "Pull"
	pullPrompt     = "Select components to pull into the repo"
	nothingToPull  = "Nothing to pull."
	noPullableText = "No component can be pulled."
)

// pullTab lists the components with changes to pull; its select flow doubles as the list.
type pullTab struct {
	svc  *services
	st   *targetState
	t    target.Target
	pick *flow
	// listed is false until the first listing arrives, so an untouched tab does not claim nothing can be pulled.
	listed bool
	epoch  int
	// dirty marks a dashboard change not yet reflected in the listing; Root reloads it when the tab is active.
	dirty  bool
	cancel context.CancelFunc
}

func newPullTab(svc *services, st *targetState, t target.Target) *pullTab {
	p := &pullTab{svc: svc, st: st, t: t}
	p.pick = newSelectFlow(svc, pullTitle, pullPrompt, nothingToPull, func(ctx context.Context, names []string) (plan.Plan, error) {
		return action.PlanPull(ctx, svc.Env, t, names)
	})
	return p
}

// pullListed carries the components with pending changes; epoch drops a result a newer listing superseded.
type pullListed struct {
	p     *pullTab
	epoch int
	names []string
}

// load diffs every component off the UI loop; a newer load cancels the one still running.
func (p *pullTab) load() tea.Cmd {
	if p.cancel != nil {
		p.cancel()
	}
	p.epoch++
	p.dirty = false
	ctx, cancel := context.WithCancel(p.svc.ctx)
	p.cancel = cancel
	epoch, env, t := p.epoch, p.svc.Env, p.t
	return func() tea.Msg {
		return pullListed{p: p, epoch: epoch, names: action.PullChanges(ctx, env, t)}
	}
}

func (p *pullTab) Title() string { return pullTitle }

func (p *pullTab) Activate(target.Target) tea.Cmd { return p.load() }

func (p *pullTab) Update(msg tea.Msg) (tab, tea.Cmd) {
	switch msg := msg.(type) {
	case dashboardReady:
		if msg.st == p.st {
			p.dirty = true
		}
	case pullListed:
		if msg.p == p && msg.epoch == p.epoch {
			p.listed = true
			var choices []choice
			for _, name := range msg.names {
				choices = append(choices, choice{name: name})
			}
			p.pick.setChoices(choices)
		}
	case plannedMsg, tea.KeyPressMsg:
		return p, p.pick.Update(msg)
	}
	return p, nil
}

func (p *pullTab) Keys() []key.Binding {
	if len(p.pick.choices) == 0 {
		return nil
	}
	return p.pick.Keys()
}

func (p *pullTab) View(w, h int) string {
	if !p.listed {
		return p.svc.th.muted.Render(computingText)
	}
	if len(p.pick.choices) == 0 {
		return p.svc.th.muted.Render(noPullableText)
	}
	return p.pick.View(w, h)
}

var _ tab = (*pullTab)(nil)
