package ui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/dashboard"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	sidebarWidth    = 30
	minWidth        = 40
	minHeight       = 8
	panelChrome     = 2 // tab row + blank line above the tab content
	maxWarningLines = 4
	sidebarTitle    = "Targets"
	helpTitle       = "Help"
	computingText   = "Computing..."
	notAvailable    = "Not available"
	tooSmallText    = "Terminal too small"
	noTargetsText   = "No targets"
)

type phase int

const (
	idle phase = iota
	loading
	ready
)

type focus int

const (
	focusSidebar focus = iota
	focusPanel
)

type targetState struct {
	phase  phase
	gen    int // drops a result that arrives after a newer recompute was started
	loaded bool
	dash   dashboard.Dashboard // the last accepted result, kept on screen while a refresh runs
	cancel context.CancelFunc  // stops the running computation when a newer one supersedes it
}

type computedMsg struct {
	idx, gen int
	dash     dashboard.Dashboard
}

// Root is the only tea.Model; it never blocks, long work runs in a tea.Cmd.
type Root struct {
	targets []target.Target
	svc     *services

	warnings []string
	notice   string // one-line refusal shown above the keybar until the next key

	ctx    context.Context
	cancel context.CancelFunc

	help help.Model

	states   []targetState
	tabs     [][]tab // per target; none for an unavailable target
	cursor   int
	focus    focus
	tab      int
	wizard   *wizard
	run      *run // set from the moment a plan is accepted until the output pane is dismissed
	showHelp bool
	w, h     int
}

// NewRoot's warnings stay visible above the keybar for the whole session.
func NewRoot(targets []target.Target, deps Deps, warnings []string) *Root {
	ctx, cancel := context.WithCancel(context.Background())
	svc := &services{Deps: deps, ctx: ctx, th: newTheme(defaultDark), keys: newKeyMap()}
	r := &Root{
		targets:  targets,
		svc:      svc,
		warnings: warnings,
		ctx:      ctx,
		cancel:   cancel,
		help:     help.New(),
		states:   make([]targetState, len(targets)),
		tabs:     make([][]tab, len(targets)),
	}
	for i, t := range targets {
		if available(t) {
			st := &r.states[i]
			r.tabs[i] = []tab{newDashboardTab(svc, st), newUpdateTab(svc, st, t), newPullTab(svc, t), newBackupsTab(svc, t)}
		}
	}
	return r
}

func (r *Root) Init() tea.Cmd { return tea.Batch(r.compute(r.cursor, false), r.activate()) }

// compute starts a computation for targets[idx] unless it is unavailable. Without force it also
// skips a ready (cached) or running one; force supersedes the running one, whose result is dropped (D18).
func (r *Root) compute(idx int, force bool) tea.Cmd {
	if idx >= len(r.targets) || !available(r.targets[idx]) {
		return nil
	}
	st := &r.states[idx]
	if !force && (st.phase == loading || st.phase == ready) {
		return nil
	}
	if st.cancel != nil {
		st.cancel()
	}
	ctx, cancel := context.WithCancel(r.ctx)
	st.cancel = cancel
	st.phase = loading
	st.gen++
	gen, env, t := st.gen, r.svc.Env, r.targets[idx]
	return func() tea.Msg {
		return computedMsg{idx: idx, gen: gen, dash: dashboard.Compute(ctx, env, t)}
	}
}

// activate tells the focused tab that its target or the tab itself was entered.
func (r *Root) activate() tea.Cmd {
	cur := r.activeTab()
	if cur == nil {
		return nil
	}
	return cur.Activate(r.targets[r.cursor])
}

// activeTab is nil for an unavailable target or when there are no targets.
func (r *Root) activeTab() tab {
	if r.cursor >= len(r.tabs) || len(r.tabs[r.cursor]) == 0 {
		return nil
	}
	r.tab = min(r.tab, len(r.tabs[r.cursor])-1)
	return r.tabs[r.cursor][r.tab]
}

// broadcast offers a non-key message to every tab and the wizard; each ignores what is not its own.
func (r *Root) broadcast(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for _, tabs := range r.tabs {
		for i, t := range tabs {
			var cmd tea.Cmd
			tabs[i], cmd = t.Update(msg)
			cmds = append(cmds, cmd)
		}
	}
	if r.wizard != nil {
		cmds = append(cmds, r.wizard.Update(msg))
	}
	return tea.Batch(cmds...)
}

func (r *Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.w, r.h = msg.Width, msg.Height
	case computedMsg:
		if msg.idx < len(r.states) && r.states[msg.idx].phase == loading && r.states[msg.idx].gen == msg.gen {
			st := &r.states[msg.idx]
			st.phase, st.dash, st.loaded = ready, msg.dash, true
			st.cancel()
			st.cancel = nil
			return r, r.broadcast(dashboardReady{st: st})
		}
	case runRequest:
		return r, r.startRun(msg.plan)
	case applyStarted:
		if r.run != nil {
			r.run.events = msg.events
			return r, waitEvent(msg.events)
		}
	case applyEvent:
		if r.run != nil {
			r.run.add(msg.event)
			return r, waitEvent(r.run.events)
		}
	case applyDone:
		if r.run != nil {
			r.run.finish(msg.result)
			if r.run.cancel != nil {
				r.run.cancel()
			}
		}
	case tea.KeyPressMsg:
		return r, r.handleKey(msg)
	default:
		return r, r.broadcast(msg)
	}
	return r, nil
}

func (r *Root) startRun(p plan.Plan) tea.Cmd {
	if r.run != nil {
		return nil
	}
	r.wizard = nil
	runCtx, cancel := context.WithCancel(r.ctx)
	r.run = newRun(p.Title, r.svc.DryRun)
	r.run.cancel = cancel
	return startApply(runCtx, r.svc.Runner, p)
}

func (r *Root) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	r.notice = ""
	if r.run != nil {
		return r.handleRunKey(msg)
	}
	k := r.svc.keys
	if key.Matches(msg, k.Quit) {
		r.cancel()
		return tea.Quit
	}
	if r.showHelp {
		if key.Matches(msg, k.Help) || msg.String() == "esc" {
			r.showHelp = false
		}
		return nil
	}
	switch {
	case key.Matches(msg, k.Help):
		r.showHelp = true
	case r.wizard != nil:
		cmd := r.wizard.Update(msg)
		if r.wizard.closed {
			r.wizard = nil
		}
		return cmd
	case len(r.targets) == 0:
	case key.Matches(msg, k.Install) && available(r.targets[r.cursor]):
		r.wizard = newWizard(r.svc, r.targets[r.cursor])
	case r.focus == focusSidebar:
		return r.handleSidebarKey(msg)
	default:
		return r.handlePanelKey(msg)
	}
	return nil
}

// cancelRunKey stops an active run; q stays refused so a quit cannot be mistaken for a cancel.
const cancelRunKey = "ctrl+c"

// handleRunKey keeps the output pane modal: while the run is active everything but scrolling and cancel is refused.
func (r *Root) handleRunKey(msg tea.KeyPressMsg) tea.Cmd {
	k := r.svc.keys
	switch {
	case key.Matches(msg, k.Up):
		r.run.scrollBy(-1)
	case key.Matches(msg, k.Down):
		r.run.scrollBy(1)
	case !r.run.done && msg.String() == cancelRunKey:
		r.run.requestCancel()
	case !r.run.done:
		r.notice = runInProgressText
	case key.Matches(msg, k.Quit):
		r.cancel()
		return tea.Quit
	case key.Matches(msg, k.Cancel):
		r.run = nil
		return tea.Batch(r.compute(r.cursor, true), r.broadcast(runDismissed{}))
	}
	return nil
}

func (r *Root) handleSidebarKey(msg tea.KeyPressMsg) tea.Cmd {
	k := r.svc.keys
	switch {
	case key.Matches(msg, k.Up):
		return r.selectTarget(r.cursor - 1)
	case key.Matches(msg, k.Down):
		return r.selectTarget(r.cursor + 1)
	case key.Matches(msg, k.Focus):
		if available(r.targets[r.cursor]) {
			r.focus = focusPanel
		}
	}
	return nil
}

func (r *Root) handlePanelKey(msg tea.KeyPressMsg) tea.Cmd {
	k := r.svc.keys
	if cur := r.activeTab(); cur != nil && key.Matches(msg, cur.Keys()...) {
		var cmd tea.Cmd
		r.tabs[r.cursor][r.tab], cmd = cur.Update(msg)
		return cmd
	}
	switch {
	case key.Matches(msg, k.Back):
		r.focus = focusSidebar
	case key.Matches(msg, k.PrevTab):
		r.tab = (r.tab + len(r.tabs[r.cursor]) - 1) % len(r.tabs[r.cursor])
		return r.activate()
	case key.Matches(msg, k.NextTab):
		r.tab = (r.tab + 1) % len(r.tabs[r.cursor])
		return r.activate()
	case key.Matches(msg, k.Refresh):
		return r.compute(r.cursor, true)
	}
	return nil
}

func (r *Root) selectTarget(idx int) tea.Cmd {
	if idx < 0 || idx >= len(r.targets) {
		return nil
	}
	r.cursor = idx
	return tea.Batch(r.compute(idx, false), r.activate())
}

type layout struct{ sidebarW, panelW, bodyH int }

func (r *Root) layout() layout {
	sidebarW := min(sidebarWidth, r.w*2/5)
	return layout{sidebarW: sidebarW, panelW: r.w - sidebarW, bodyH: r.h - 1 - len(r.warningLines())}
}

// warningLines wraps the notice and the warnings to the terminal width, capped so they cannot crowd out the body.
func (r *Root) warningLines() []string {
	var lines []string
	if r.notice != "" {
		lines = styleAll(r.svc.th.warning, wrap(r.notice, r.w))
	}
	if len(r.warnings) == 0 {
		return lines
	}
	warnings := wrap(strings.Join(r.warnings, "  "), r.w)
	return append(lines, styleAll(r.svc.th.warning, warnings[:min(len(warnings), maxWarningLines)])...)
}

func (r *Root) viewportHeight() int {
	return max(r.layout().bodyH-borderCells-panelChrome, 0)
}

func styleAll(s lipgloss.Style, lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = s.Render(strings.TrimRight(l, " "))
	}
	return out
}

// tabRow lists every tab, or only the active one when they do not fit width.
func (r *Root) tabRow(width int) string {
	titles := make([]string, len(r.tabs[r.cursor]))
	for i, t := range r.tabs[r.cursor] {
		style := r.svc.th.tabIdle
		if i == r.tab {
			style = r.svc.th.tabActive
		}
		titles[i] = style.Render(t.Title())
	}
	row := strings.Join(titles, "  ")
	if lipgloss.Width(row) > width {
		row = lipgloss.NewStyle().MaxWidth(width).Render(titles[r.tab])
	}
	return row
}

func (r *Root) View() tea.View {
	v := tea.NewView(r.render())
	v.AltScreen = true
	return v
}

func (r *Root) render() string {
	if r.w == 0 {
		return ""
	}
	if r.w < minWidth || r.h < minHeight {
		return tooSmallText
	}
	if len(r.targets) == 0 {
		return noTargetsText
	}
	th := r.svc.th
	l := r.layout()
	sidebar := renderBox(th, sidebarTitle, sidebarLines(th, r.targets, r.cursor, contentWidth(l.sidebarW)), l.sidebarW, l.bodyH, r.focus == focusSidebar)
	title, lines := r.panel()
	panel := renderBox(th, title, lines, l.panelW, l.bodyH, r.focus == focusPanel || r.wizard != nil)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, panel)
	if r.run != nil {
		body = overlayCentered(body, outputPopup(th, r.run, r.w, l.bodyH), r.w, l.bodyH)
	}
	if r.showHelp {
		body = overlayCentered(body, helpPopup(th, r.help, r.helpIntro(), r.svc.keys.helpGroups(r.tabKeys()), r.w, l.bodyH), r.w, l.bodyH)
	}
	for _, line := range r.warningLines() {
		body += "\n" + line
	}
	keys, pinned := r.keybarKeys()
	return body + "\n" + keybar(th, r.help, keys, pinned, r.svc.DryRun, r.w)
}

// panel returns the title and lines of the right-hand box: the wizard, a tab, or why there is none.
func (r *Root) panel() (string, []string) {
	t := r.targets[r.cursor]
	contentW, h := contentWidth(r.layout().panelW), r.viewportHeight()
	th := r.svc.th
	switch {
	case r.wizard != nil:
		return wizardTitle, clip(strings.Split(r.wizard.View(contentW, h+panelChrome), "\n"), h+panelChrome)
	case t.Err != nil:
		return t.Spec.Name, styleAll(th.err, wrap(statusGlyph(check.StatusError)+" "+t.Err.Error(), contentW))
	case !available(t):
		return t.Spec.Name, []string{th.muted.Render(notAvailable)}
	}
	body := strings.Split(r.activeTab().View(contentW, h), "\n")
	return t.Spec.Name, append([]string{r.tabRow(contentW), ""}, clip(body, h)...)
}

func clip(lines []string, h int) []string { return lines[:max(min(len(lines), h), 0)] }

// keybarKeys follows what has the keyboard: the output pane, help, the wizard, a tab or the sidebar.
// The second result is pinned: shown whole whatever the width.
func (r *Root) keybarKeys() (keys, pinned []key.Binding) {
	k := r.svc.keys
	switch {
	case r.run != nil && !r.run.done:
		return []key.Binding{k.Move}, nil
	case r.run != nil:
		return []key.Binding{described(k.Cancel, "close"), k.Move}, []key.Binding{k.Quit}
	case r.showHelp:
		return nil, k.pinnedKeys()
	case r.wizard != nil:
		return r.wizard.Keys(), k.pinnedKeys()
	case r.focus == focusPanel:
		return k.panelKeys(r.tabKeys()), k.pinnedKeys()
	}
	return k.sidebarKeys(), k.pinnedKeys()
}

func (r *Root) tabKeys() []key.Binding {
	if cur := r.activeTab(); cur != nil {
		return cur.Keys()
	}
	return nil
}

func (r *Root) helpIntro() string {
	var titles []string
	for _, t := range r.tabs[r.cursor] {
		titles = append(titles, t.Title())
	}
	return helpIntro(titles)
}
