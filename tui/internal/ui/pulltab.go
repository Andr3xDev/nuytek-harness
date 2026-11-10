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
	t    target.Target
	pick *flow
}

func newPullTab(svc *services, t target.Target) *pullTab {
	p := &pullTab{svc: svc, t: t}
	p.pick = newSelectFlow(svc, pullTitle, pullPrompt, nothingToPull, func(ctx context.Context, names []string) (plan.Plan, error) {
		return action.PlanPull(ctx, svc.Env, t, names)
	})
	p.setChoices()
	return p
}

func (p *pullTab) setChoices() {
	var choices []choice
	for _, c := range p.t.Manifest.Components {
		if c.Pull {
			choices = append(choices, choice{name: c.Name})
		}
	}
	p.pick.setChoices(choices)
}

func (p *pullTab) Title() string { return pullTitle }

func (p *pullTab) Activate(target.Target) tea.Cmd {
	p.setChoices()
	return nil
}

func (p *pullTab) Update(msg tea.Msg) (tab, tea.Cmd) {
	switch msg.(type) {
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
	if len(p.pick.choices) == 0 {
		return p.svc.th.muted.Render(noPullableText)
	}
	return p.pick.View(w, h)
}

var _ tab = (*pullTab)(nil)
