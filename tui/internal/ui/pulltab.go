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

// pullTab lists the pull-enabled components; its select flow doubles as the list.
type pullTab struct {
	svc  *services
	st   *targetState
	t    target.Target
	pick *flow
	// listed is false until the first setChoices, so an untouched tab does not claim nothing can be pulled.
	listed bool
}

func newPullTab(svc *services, st *targetState, t target.Target) *pullTab {
	p := &pullTab{svc: svc, st: st, t: t}
	p.pick = newSelectFlow(svc, pullTitle, pullPrompt, nothingToPull, func(ctx context.Context, names []string) (plan.Plan, error) {
		return action.PlanPull(ctx, svc.Env, t, names)
	})
	return p
}

func (p *pullTab) setChoices() {
	p.listed = true
	var choices []choice
	for _, name := range action.PullChanges(p.svc.ctx, p.svc.Env, p.t) {
		choices = append(choices, choice{name: name})
	}
	p.pick.setChoices(choices)
}

func (p *pullTab) Title() string { return pullTitle }

func (p *pullTab) Activate(target.Target) tea.Cmd {
	p.setChoices()
	return nil
}

func (p *pullTab) Update(msg tea.Msg) (tab, tea.Cmd) {
	switch msg := msg.(type) {
	case dashboardReady:
		if msg.st == p.st {
			p.setChoices()
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
