package ui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	updateTitle    = "Update"
	upToDateText   = "Everything is up to date."
	noChangesText  = "No file changes."
	updatePrompt   = "Components with drift"
	notRemovedText = " is not removed by the harness"
)

// detailMsg carries the rows an update of one component would change.
type detailMsg struct {
	u     *updateTab
	epoch int
	name  string
	p     plan.Plan
	err   error
}

type updateDetail struct {
	p   plan.Plan
	err error
}

// updateTab lists the drifted components of the cached dashboard; its select flow doubles as the list.
type updateTab struct {
	svc     *services
	st      *targetState
	t       target.Target
	pick    *flow
	removal *flow // set while a remove flow runs
	epoch   int   // drops a detail computed for an older dashboard
	detail  map[string]updateDetail
	asked   map[string]bool
	notice  string
}

func newUpdateTab(svc *services, st *targetState, t target.Target) *updateTab {
	u := &updateTab{svc: svc, st: st, t: t, detail: map[string]updateDetail{}, asked: map[string]bool{}}
	u.pick = newSelectFlow(svc, updateTitle, updatePrompt, nothingToDoText, func(ctx context.Context, names []string) (plan.Plan, error) {
		return action.PlanInstall(ctx, svc.Env, t, names)
	})
	return u
}

func (u *updateTab) Title() string { return updateTitle }

func (u *updateTab) Activate(target.Target) tea.Cmd {
	return u.sync()
}

// sync rebuilds the list from the dashboard; plugins is listed but never preselected.
func (u *updateTab) sync() tea.Cmd {
	u.epoch++
	u.detail, u.asked = map[string]updateDetail{}, map[string]bool{}
	if !u.st.loaded {
		return nil
	}
	inUpdateAll := map[string]bool{}
	if comps, err := action.Select(u.t, []string{action.AllNames}, action.ScopeUpdate); err == nil {
		for _, c := range comps {
			inUpdateAll[c.Name] = true
		}
	}
	var choices []choice
	for _, s := range u.st.dash.Components {
		if s.State == drift.StateDrift {
			choices = append(choices, choice{name: s.Name, note: s.Reason, selected: inUpdateAll[s.Name]})
		}
	}
	u.pick.setChoices(choices)
	return u.loadDetail()
}

func (u *updateTab) loadDetail() tea.Cmd {
	c, ok := u.pick.highlighted()
	if !ok || u.asked[c.name] {
		return nil
	}
	u.asked[c.name] = true
	epoch, ctx, env, t := u.epoch, u.svc.ctx, u.svc.Env, u.t
	return func() tea.Msg {
		p, err := action.PlanInstall(ctx, env, t, []string{c.name})
		return detailMsg{u: u, epoch: epoch, name: c.name, p: p, err: err}
	}
}

func (u *updateTab) Update(msg tea.Msg) (tab, tea.Cmd) {
	switch msg := msg.(type) {
	case dashboardReady:
		if msg.st == u.st {
			return u, u.sync()
		}
	case detailMsg:
		if msg.u == u && msg.epoch == u.epoch {
			u.detail[msg.name] = updateDetail{p: msg.p, err: msg.err}
		}
	case runDismissed:
		u.removal = nil
	case plannedMsg:
		cmds := []tea.Cmd{u.pick.Update(msg)}
		if u.removal != nil {
			cmds = append(cmds, u.removal.Update(msg))
		}
		return u, tea.Batch(cmds...)
	case tea.KeyPressMsg:
		u.notice = ""
		return u, u.handleKey(msg)
	}
	return u, nil
}

func (u *updateTab) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	if u.removal != nil {
		cmd := u.removal.Update(msg)
		if u.removal.finished() {
			u.removal = nil
		}
		return cmd
	}
	if key.Matches(msg, u.svc.keys.Remove) && u.pick.atEntry() {
		return u.startRemoval()
	}
	before, _ := u.pick.highlighted()
	cmd := u.pick.Update(msg)
	if after, _ := u.pick.highlighted(); after.name != before.name {
		cmd = tea.Batch(cmd, u.loadDetail())
	}
	return cmd
}

func (u *updateTab) removable(name string) bool {
	comps, err := action.Select(u.t, []string{action.AllNames}, action.ScopeRemove)
	if err != nil {
		return false
	}
	for _, c := range comps {
		if c.Name == name {
			return true
		}
	}
	return false
}

func (u *updateTab) startRemoval() tea.Cmd {
	c, ok := u.pick.highlighted()
	if !ok {
		return nil
	}
	if !u.removable(c.name) {
		u.notice = c.name + notRemovedText
		return nil
	}
	svc, t := u.svc, u.t
	u.removal = newDirectFlow(svc, "Remove "+c.name, "Nothing to remove.", func(ctx context.Context, _ []string) (plan.Plan, error) {
		return action.PlanRemove(ctx, svc.Env, t, []string{c.name})
	})
	return u.removal.start()
}

func (u *updateTab) Keys() []key.Binding {
	if u.removal != nil {
		return u.removal.Keys()
	}
	if len(u.pick.choices) == 0 {
		return nil
	}
	return append(u.pick.Keys(), u.svc.keys.Remove)
}

func (u *updateTab) View(w, h int) string {
	th := u.svc.th
	if u.removal != nil {
		return u.removal.View(w, h)
	}
	switch {
	case !u.st.loaded:
		return th.muted.Render(computingText)
	case len(u.pick.choices) == 0:
		return th.ok.Render(upToDateText)
	case !u.pick.atEntry():
		return u.pick.View(w, h)
	}
	var notice []string
	if u.notice != "" {
		notice = styleAll(th.warning, wrap(u.notice, w))
	}
	h -= len(notice)
	listH := min(len(u.pick.choices)+1, max(h/2, 2))
	lines := []string{u.pick.View(w, listH)}
	lines = append(lines, u.detailLines(w, h-listH)...)
	return strings.Join(append(lines, notice...), "\n")
}

// detailLines shows the rows an update of the highlighted component changes.
func (u *updateTab) detailLines(w, h int) []string {
	th := u.svc.th
	c, ok := u.pick.highlighted()
	if !ok || h < 2 {
		return nil
	}
	lines := append([]string{""}, styleAll(th.section, wrap("Changes: "+c.name, w))...)
	d, loaded := u.detail[c.name]
	switch {
	case !loaded:
		lines = append(lines, th.muted.Render(planningText))
	case d.err != nil:
		lines = append(lines, styleAll(th.err, wrap(d.err.Error(), w))...)
	default:
		lines = append(lines, previewBody(th, d.p, noChangesText, w)...)
	}
	return lines[:min(len(lines), h)]
}

var _ tab = (*updateTab)(nil)
