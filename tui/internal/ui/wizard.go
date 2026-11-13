package ui

import (
	"context"
	"errors"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	wizardTitle   = "Install"
	modeInstall   = "Install"
	modeHard      = "Hard install"
	hardSummary   = "Hard install resets every directory component to the repo, deleting custom files inside it, and prunes MCP servers and plugins the repo does not declare. Components whose tool is missing are skipped and left untouched."
	installPrompt = "Select components to install"
)

// wizard is the install entry: a mode (Install or Hard install), the dependency screen and one flow per mode.
type wizard struct {
	svc     *services
	missing []string // one line per missing tool, with the components it blocks
	hard    bool
	install *flow
	hardRun *flow
	closed  bool
}

func newWizard(svc *services, t target.Target) *wizard {
	w := &wizard{svc: svc}
	var blocked []string
	missingTools := map[string][]string{} // blocked component -> tools it lacks
	for _, m := range action.MissingTools(svc.Env.Exec, t.Manifest) {
		text := m.Tool
		if len(m.Dependents) > 0 {
			text += " (blocks " + strings.Join(m.Dependents, ", ") + ")"
		}
		w.missing = append(w.missing, text)
		for _, dependent := range m.Dependents {
			if !slices.Contains(blocked, dependent) {
				blocked = append(blocked, dependent)
			}
			missingTools[dependent] = append(missingTools[dependent], m.Tool)
		}
	}

	w.install = newSelectFlow(svc, modeInstall, installPrompt, nothingToDoText, func(ctx context.Context, names []string) (plan.Plan, error) {
		// Blocked components are not selectable but go to the planner so the summary lists them as skipped.
		return action.PlanInstall(ctx, svc.Env, t, append(slices.Clone(names), blocked...))
	})
	choices := make([]choice, 0, len(t.Manifest.Components))
	for _, c := range t.Manifest.Components {
		ch := choice{name: c.Name, selected: true}
		if slices.Contains(blocked, c.Name) {
			ch.disabled, ch.note = true, "blocked, missing "+strings.Join(missingTools[c.Name], ", ")
		}
		choices = append(choices, ch)
	}
	w.install.setChoices(choices)

	w.hardRun = newDirectFlow(svc, modeHard, nothingToDoText, func(ctx context.Context, _ []string) (plan.Plan, error) {
		p, err := action.PlanHardInstall(ctx, svc.Env, t)
		var refused action.ErrMissingTools
		if errors.As(err, &refused) {
			tools := make([]string, len(refused.Tools))
			for i, m := range refused.Tools {
				tools[i] = m.Tool
			}
			return p, errors.New("hard install refused, missing: " + strings.Join(tools, ", "))
		}
		return p, err
	})
	return w
}

func (w *wizard) current() *flow {
	if w.hard {
		return w.hardRun
	}
	return w.install
}

func (w *wizard) Update(msg tea.Msg) tea.Cmd {
	keyMsg, isKey := msg.(tea.KeyPressMsg)
	if !isKey {
		return tea.Batch(w.install.Update(msg), w.hardRun.Update(msg))
	}
	k, f := w.svc.keys, w.current()
	switch {
	case f.atEntry() && key.Matches(keyMsg, k.Cancel):
		w.closed = true
	case f.atEntry() && key.Matches(keyMsg, k.Mode):
		w.hard = !w.hard
	case w.hard && f.atEntry() && key.Matches(keyMsg, k.Accept):
		return f.start()
	default:
		return f.Update(msg)
	}
	return nil
}

func (w *wizard) Keys() []key.Binding {
	k, f := w.svc.keys, w.current()
	if !f.atEntry() {
		return f.Keys()
	}
	keys := []key.Binding{k.Mode}
	if w.hard {
		keys = append(keys, described(k.Accept, "preview"))
	} else {
		keys = append(keys, f.Keys()...)
	}
	return append(keys, described(k.Cancel, "close"))
}

func (w *wizard) View(width, h int) string {
	f := w.current()
	if !f.atEntry() {
		return f.View(width, h)
	}
	header := w.header(width)
	if w.hard {
		body := styleAll(w.svc.th.warning, wrap(hardSummary, width))
		return strings.Join(append(header, body...), "\n")
	}
	return strings.Join(append(header, f.View(width, h-len(header))), "\n")
}

func (w *wizard) header(width int) []string {
	th := w.svc.th
	modes := []string{modeInstall, modeHard}
	for i, name := range modes {
		style := th.tabIdle
		if (i == 1) == w.hard {
			style = th.tabActive
		}
		modes[i] = style.Render(name)
	}
	lines := []string{"Mode: " + strings.Join(modes, "  ")}
	if len(w.missing) == 0 {
		return append(lines, th.ok.Render(statusGlyph(check.StatusOK)+" All dependencies present"), "")
	}
	for _, m := range w.missing {
		lines = append(lines, styleAll(th.err, wrap(statusGlyph(check.StatusError)+" missing "+m, width))...)
	}
	return append(lines, "")
}
