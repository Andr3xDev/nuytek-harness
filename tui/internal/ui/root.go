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
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
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

	dashboardTabIndex = 0
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

// specterState mirrors targetState for the global Specter entry.
type specterState struct {
	phase  phase
	gen    int
	loaded bool
	facts  specter.Facts
	cancel context.CancelFunc

	validation map[string]*storeValidation // by store id; survives reloads, gone with the session
	scroll     scroller
}

type specterLoadedMsg struct {
	gen   int
	facts specter.Facts
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

	states        []targetState
	specter       specterState
	entry         entry // what the sidebar selects; cursor itself stays on a target
	configPanel   configState
	configFlow    *flow // the self-update, from preview to the accepted run
	specterFlow   *flow // fast init or maintenance, from selection to the accepted run
	specterReport bool  // the open flow shows the drift report above its preview
	validateSlots chan struct{}
	tabs          [][]tab // per target; none for an unavailable target
	cursor        int
	focus         focus
	tab           int
	wizard        *wizard
	run           *run // set from the moment a plan is accepted until the output pane is dismissed
	showHelp      bool
	w, h          int
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

		configPanel: newConfigState(),

		validateSlots: make(chan struct{}, specter.MaxConcurrentOpenspec),
	}
	for i, t := range targets {
		if available(t) {
			st := &r.states[i]
			r.tabs[i] = []tab{newDashboardTab(svc, st, func() *specterState { return &r.specter }), newUpdateTab(svc, st, t), newPullTab(svc, st, t), newBackupsTab(svc, t)}
			if t.Spec.Manage != nil {
				r.tabs[i] = append(r.tabs[i], newManageTab(svc, st))
			}
		}
	}
	return r
}

func (r *Root) Init() tea.Cmd {
	return tea.Batch(r.compute(r.cursor, false), r.activate(), r.loadSpecter())
}

// loadSpecter always supersedes a running load, whose result is dropped. It only reads: validation never starts here.
func (r *Root) loadSpecter() tea.Cmd {
	load := r.svc.SpecterLoad
	if load == nil {
		return nil
	}
	env := r.svc.Env
	st := &r.specter
	if st.cancel != nil {
		st.cancel()
	}
	ctx, cancel := context.WithCancel(r.ctx)
	st.cancel = cancel
	st.phase = loading
	st.gen++
	gen := st.gen
	return func() tea.Msg { return specterLoadedMsg{gen: gen, facts: load(ctx, env)} }
}

// currentTarget is false while the Specter or Config entry is selected or there are no targets.
func (r *Root) currentTarget() (target.Target, bool) {
	if r.entry != entryTarget || r.cursor >= len(r.targets) {
		return target.Target{}, false
	}
	return r.targets[r.cursor], true
}

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
	t, ok := r.currentTarget()
	if cur == nil || !ok {
		return nil
	}
	return cur.Activate(t)
}

// reloadManage re-runs the Manage sections of the focused target, if it has that tab.
func (r *Root) reloadManage() tea.Cmd {
	for _, tb := range r.tabs[r.cursor] {
		if m, ok := tb.(*manageTab); ok {
			return m.Activate(r.targets[r.cursor])
		}
	}
	return nil
}

// activeTab is nil for the Specter or Config entry, an unavailable target, or when there are no targets.
func (r *Root) activeTab() tab {
	if r.entry != entryTarget || r.cursor >= len(r.tabs) || len(r.tabs[r.cursor]) == 0 {
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
			cmds := []tea.Cmd{r.broadcast(dashboardReady{st: st})}
			if p, ok := r.activeTab().(*pullTab); ok && p.dirty {
				cmds = append(cmds, p.load())
			}
			return r, tea.Batch(cmds...)
		}
	case specterLoadedMsg:
		if st := &r.specter; st.phase == loading && st.gen == msg.gen {
			st.phase, st.facts, st.loaded = ready, msg.facts, true
			st.cancel()
			st.cancel = nil
		}
	case validatedMsg:
		r.specter.finishValidation(msg)
	case configSaved:
		return r, r.reloadFromConfig()
	case plannedMsg:
		cmds := []tea.Cmd{r.broadcast(msg)}
		if r.specterFlow != nil {
			cmds = append(cmds, r.specterFlow.Update(msg))
		}
		if r.configFlow != nil {
			cmds = append(cmds, r.configFlow.Update(msg))
		}
		return r, tea.Batch(cmds...)
	case runRequest:
		// An accepted select flow rests on its list again; the Specter flow must not reappear after the run.
		r.specterFlow, r.configFlow = nil, nil
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
	case tea.PasteMsg:
		if r.run != nil || !r.inputActive() {
			return r, nil
		}
		var cmd tea.Cmd
		r.configPanel.input, cmd = r.configPanel.input.Update(msg)
		r.configPanel.err = ""
		return r, cmd
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
	if r.inputActive() && msg.String() != "ctrl+c" {
		return r.handleConfigKey(msg)
	}
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
	case key.Matches(msg, k.Install) && r.installable():
		t, _ := r.currentTarget()
		r.wizard = newWizard(r.svc, t)
	case r.focus == focusSidebar:
		return r.handleSidebarKey(msg)
	default:
		return r.handlePanelKey(msg)
	}
	return nil
}

// handleRunKey keeps the output pane modal: while the run is active everything but scrolling and cancel is refused.
// ctrl+c cancels the run; q stays refused so a quit cannot be mistaken for a cancel.
func (r *Root) handleRunKey(msg tea.KeyPressMsg) tea.Cmd {
	k := r.svc.keys
	switch {
	case k.step(msg) != 0:
		r.run.scrollBy(k.step(msg))
	case !r.run.done && key.Matches(msg, k.CancelRun):
		r.run.requestCancel()
	case !r.run.done:
		r.notice = runInProgressText
	case key.Matches(msg, k.Quit):
		r.cancel()
		return tea.Quit
	case key.Matches(msg, k.Cancel):
		r.run = nil
		return tea.Batch(r.refreshSelected(), r.broadcast(runDismissed{}))
	}
	return nil
}

func (r *Root) handleSidebarKey(msg tea.KeyPressMsg) tea.Cmd {
	k := r.svc.keys
	switch {
	case k.step(msg) != 0:
		return r.moveSidebar(k.step(msg))
	case key.Matches(msg, k.Focus):
		if t, ok := r.currentTarget(); r.entry != entryTarget || ok && available(t) {
			r.focus = focusPanel
		}
	}
	return nil
}

// refreshSelected recomputes what the sidebar selects. The target dashboard is recomputed even under Specter,
// because a Specter run changes the `specter` component status shown there.
func (r *Root) refreshSelected() tea.Cmd {
	switch r.entry {
	case entrySpecter:
		return tea.Batch(r.loadSpecter(), r.compute(r.cursor, true))
	case entryConfig:
		return nil
	}
	return tea.Batch(r.compute(r.cursor, true), r.reloadManage())
}

func (r *Root) installable() bool {
	t, ok := r.currentTarget()
	return ok && available(t)
}

// moveSidebar steps the target cursor, clamped; the Specter and Config entries sit above the first target.
// A jump that starts on or reaches past the entries moves one entry only.
func (r *Root) moveSidebar(delta int) tea.Cmd {
	switch {
	case delta < 0 && (r.entry != entryTarget || r.cursor == 0):
		return r.selectUp()
	case delta > 0 && r.entry != entryTarget:
		return r.selectDown()
	}
	next := max(min(r.cursor+delta, len(r.targets)-1), 0)
	if next == r.cursor {
		return nil
	}
	return r.selectTarget(next)
}

// selectUp walks target, Config, Specter; entering Specter also forces a load (screen entry).
func (r *Root) selectUp() tea.Cmd {
	switch {
	case r.entry == entrySpecter:
		return nil
	case r.entry == entryConfig:
		r.entry = entrySpecter
		return r.loadSpecter()
	case r.cursor == 0:
		r.entry = entryConfig
		return nil
	}
	return r.selectTarget(r.cursor - 1)
}

func (r *Root) selectDown() tea.Cmd {
	switch r.entry {
	case entrySpecter:
		r.entry = entryConfig
		return nil
	case entryConfig:
		r.entry = entryTarget
		return tea.Batch(r.compute(r.cursor, false), r.activate())
	}
	return r.selectTarget(r.cursor + 1)
}

func (r *Root) handlePanelKey(msg tea.KeyPressMsg) tea.Cmd {
	k := r.svc.keys
	switch r.entry {
	case entrySpecter:
		return r.handleSpecterKey(msg)
	case entryConfig:
		return r.handleConfigKey(msg)
	}
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
		cmds := []tea.Cmd{r.compute(r.cursor, true), r.reloadManage()}
		if r.tab == dashboardTabIndex {
			cmds = append(cmds, r.loadSpecter())
		}
		return tea.Batch(cmds...)
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
	// The Specter entry is reachable only through a target; this is safe because main registers Claude statically, so targets is never empty.
	if len(r.targets) == 0 {
		return noTargetsText
	}
	th := r.svc.th
	l := r.layout()
	sidebar := renderBox(th, sidebarTitle, sidebarLines(th, r.targets, r.cursor, r.entry, contentWidth(l.sidebarW)), l.sidebarW, l.bodyH, r.focus == focusSidebar)
	title, lines := r.panel()
	panel := renderBox(th, title, lines, l.panelW, l.bodyH, r.focus == focusPanel || r.wizard != nil)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, panel)
	if r.run != nil {
		body = overlayCentered(body, outputPopup(th, r.run, r.w, l.bodyH), r.w, l.bodyH)
	}
	if r.showHelp {
		body = overlayCentered(body, helpPopup(th, r.help, r.helpIntro(), r.svc.keys.helpGroups(r.tabKeys(), r.entry == entrySpecter), r.w, l.bodyH), r.w, l.bodyH)
	}
	for _, line := range r.warningLines() {
		body += "\n" + line
	}
	keys, pinned := r.keybarKeys()
	return body + "\n" + keybar(th, r.help, keys, pinned, r.svc.DryRun, r.w)
}

// panel returns the title and lines of the right-hand box: the wizard, a tab, or why there is none.
func (r *Root) panel() (string, []string) {
	contentW, h := contentWidth(r.layout().panelW), r.viewportHeight()
	th := r.svc.th
	t, isTarget := r.currentTarget()
	switch {
	case r.wizard != nil:
		return wizardTitle, clip(strings.Split(r.wizard.View(contentW, h+panelChrome), "\n"), h+panelChrome)
	case r.entry == entryConfig:
		return configName, clip(r.configLines(contentW, h+panelChrome), h+panelChrome)
	case !isTarget:
		return specterName, r.specterPanel(contentW, h+panelChrome)
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
		return []key.Binding{k.Move, k.CancelRun}, nil
	case r.run != nil:
		return []key.Binding{described(k.Cancel, "close"), k.Move}, []key.Binding{k.Quit}
	case r.showHelp:
		return nil, k.pinnedKeys()
	case r.wizard != nil:
		return r.wizard.Keys(), k.pinnedKeys()
	case r.entry == entrySpecter && r.focus == focusPanel:
		return r.specterKeys(), k.pinnedKeys()
	case r.entry == entryConfig && r.focus == focusPanel:
		return r.configKeys(), k.pinnedKeys()
	case r.entry != entryTarget:
		return []key.Binding{k.Focus, k.Move}, k.pinnedKeys()
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
	if _, ok := r.currentTarget(); ok {
		for _, t := range r.tabs[r.cursor] {
			titles = append(titles, t.Title())
		}
	}
	return helpIntro(titles)
}
