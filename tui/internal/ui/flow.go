package ui

import (
	"context"
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const selectOneText = "Select at least one component."

type flowStage int

const (
	stageIdle flowStage = iota // a flow without selection that has not started
	stageSelect
	stagePlanning
	stagePreview
	stageConfirm
)

type choice struct {
	name     string
	note     string
	disabled bool
	selected bool // initial state, applied the first time the name is seen
}

type planFunc func(ctx context.Context, names []string) (plan.Plan, error)

type plannedMsg struct {
	f   *flow
	gen int
	p   plan.Plan
	err error
}

// runRequest asks Root to apply a plan the user accepted.
type runRequest struct{ plan plan.Plan }

// flow is the one write interaction: select (optional) -> plan -> preview -> confirm (optional) -> run.
// It is parameterized by data: a title, the choices and the planning function.
type flow struct {
	svc       *services
	title     string // names the action in the preview and confirmation
	prompt    string // heads the selection list
	emptyText string
	planFn    planFunc
	selects   bool

	choices []choice
	picked  map[string]bool
	seen    map[string]bool
	cursor  int
	notice  string

	stage flowStage
	gen   int // drops a plan that arrives after the flow moved on
	p     plan.Plan
	err   error
	body  scroller
}

// newSelectFlow starts in the selection stage, so it can rest on screen as a list.
func newSelectFlow(svc *services, title, prompt, emptyText string, planFn planFunc) *flow {
	f := newDirectFlow(svc, title, emptyText, planFn)
	f.selects, f.prompt, f.stage = true, prompt, stageSelect
	return f
}

// newDirectFlow has no selection: start plans at once.
func newDirectFlow(svc *services, title, emptyText string, planFn planFunc) *flow {
	return &flow{svc: svc, title: title, emptyText: emptyText, planFn: planFn, picked: map[string]bool{}, seen: map[string]bool{}}
}

// setChoices replaces the list, keeping the picks of names that stay.
func (f *flow) setChoices(choices []choice) {
	f.choices = choices
	present := map[string]bool{}
	for _, c := range choices {
		present[c.name] = true
		if !f.seen[c.name] {
			f.seen[c.name] = true
			f.picked[c.name] = c.selected && !c.disabled
		}
	}
	for name := range f.picked {
		if !present[name] {
			delete(f.picked, name)
			delete(f.seen, name)
		}
	}
	f.cursor = max(min(f.cursor, len(choices)-1), 0)
}

func (f *flow) highlighted() (choice, bool) {
	if f.cursor >= len(f.choices) {
		return choice{}, false
	}
	return f.choices[f.cursor], true
}

func (f *flow) names() []string {
	var names []string
	for _, c := range f.choices {
		if f.picked[c.name] && !c.disabled {
			names = append(names, c.name)
		}
	}
	return names
}

// atEntry is true before the user asked for a plan.
func (f *flow) atEntry() bool { return f.stage == stageSelect || f.stage == stageIdle }

// start opens the selection, or plans at once when there is none.
func (f *flow) start() tea.Cmd {
	if f.selects {
		f.stage = stageSelect
		return nil
	}
	return f.startPlanning(nil)
}

func (f *flow) startPlanning(names []string) tea.Cmd {
	f.stage, f.err, f.p = stagePlanning, nil, plan.Plan{}
	f.gen++
	gen, ctx, planFn := f.gen, f.svc.ctx, f.planFn
	return func() tea.Msg {
		p, err := planFn(ctx, names)
		return plannedMsg{f: f, gen: gen, p: p, err: err}
	}
}

// reset returns to the entry stage and drops any plan still being computed.
func (f *flow) reset() {
	f.gen++
	f.err, f.p, f.notice = nil, plan.Plan{}, ""
	f.body.reset()
	f.stage = stageIdle
	if f.selects {
		f.stage = stageSelect
	}
}

func (f *flow) back() {
	if f.stage == stageConfirm {
		f.stage = stagePreview
		f.body.reset()
		return
	}
	f.reset()
}

// finished is true once the flow went back to its entry stage and can be dropped.
func (f *flow) finished() bool { return f.stage == stageIdle }

func (f *flow) canApply() bool { return f.err == nil && !f.p.Empty() }

func (f *flow) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case plannedMsg:
		if msg.f == f && msg.gen == f.gen && f.stage == stagePlanning {
			f.p, f.err, f.stage = msg.p, msg.err, stagePreview
			f.body.reset()
		}
	case tea.KeyPressMsg:
		return f.handleKey(msg)
	}
	return nil
}

func (f *flow) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	k := f.svc.keys
	f.notice = ""
	switch f.stage {
	case stageSelect:
		return f.selectKey(msg)
	case stagePlanning:
		if key.Matches(msg, k.Cancel) {
			f.back()
		}
	case stagePreview, stageConfirm:
		switch {
		case key.Matches(msg, k.Cancel):
			f.back()
		case key.Matches(msg, k.Up):
			f.body.by(-1)
		case key.Matches(msg, k.Down):
			f.body.by(1)
		case key.Matches(msg, k.Accept):
			return f.accept()
		}
	}
	return nil
}

func (f *flow) selectKey(msg tea.KeyPressMsg) tea.Cmd {
	k := f.svc.keys
	switch {
	case key.Matches(msg, k.Up):
		f.cursor = max(f.cursor-1, 0)
	case key.Matches(msg, k.Down):
		f.cursor = max(min(f.cursor+1, len(f.choices)-1), 0)
	case key.Matches(msg, k.Toggle):
		if c, ok := f.highlighted(); ok && !c.disabled {
			f.picked[c.name] = !f.picked[c.name]
		}
	case key.Matches(msg, k.All):
		f.toggleAll()
	case key.Matches(msg, k.Accept):
		names := f.names()
		if len(names) == 0 {
			f.notice = selectOneText
			return nil
		}
		return f.startPlanning(names)
	}
	return nil
}

// toggleAll picks every enabled choice, or clears the picks when all of them are already picked.
func (f *flow) toggleAll() {
	all := len(f.names()) == f.enabledCount()
	for _, c := range f.choices {
		f.picked[c.name] = !c.disabled && !all
	}
}

func (f *flow) enabledCount() int {
	return len(slices.DeleteFunc(slices.Clone(f.choices), func(c choice) bool { return c.disabled }))
}

func (f *flow) accept() tea.Cmd {
	if !f.canApply() {
		return nil
	}
	if f.stage == stagePreview && f.p.Confirm != "" {
		f.stage = stageConfirm
		f.body.reset()
		return nil
	}
	p := f.p
	// The Update tab reuses the install planner, whose own title would name the run "Install".
	p.Title = f.title
	f.reset()
	return func() tea.Msg { return runRequest{plan: p} }
}

func (f *flow) Keys() []key.Binding {
	k := f.svc.keys
	switch f.stage {
	case stageSelect:
		return []key.Binding{k.Toggle, k.All, described(k.Accept, "preview"), k.Move}
	case stagePlanning:
		return []key.Binding{k.Cancel}
	case stagePreview:
		var keys []key.Binding
		if f.canApply() {
			keys = append(keys, described(k.Accept, "apply"))
		}
		return append(keys, k.Cancel, k.Move)
	case stageConfirm:
		return []key.Binding{described(k.Accept, "confirm"), described(k.Cancel, "decline"), k.Move}
	}
	return nil
}
