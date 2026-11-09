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
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	sidebarWidth    = 30
	minWidth        = 40
	minHeight       = 8
	panelChrome     = 2 // tab row + blank line above the scrollable content
	maxWarningLines = 4
	sidebarTitle    = "Targets"
	helpTitle       = "Help"
	computingText   = "Computing..."
	notAvailable    = "Not available"
	tooSmallText    = "Terminal too small"
	noTargetsText   = "No targets"
	dashboardTitle  = "Dashboard"
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
	phase phase
	gen   int // drops a result that arrives after a newer recompute was started
	dash  dashboard.Dashboard
}

type computedMsg struct {
	idx, gen int
	dash     dashboard.Dashboard
}

// Root is the only tea.Model; it never blocks, long work runs in a tea.Cmd.
type Root struct {
	targets []target.Target
	env     check.Env
	dryRun  bool

	warnings []string

	ctx    context.Context
	cancel context.CancelFunc

	keys keyMap
	th   theme
	help help.Model

	states   []targetState
	cursor   int
	focus    focus
	tabs     []string
	tab      int
	scroll   int
	showHelp bool
	w, h     int
}

// NewRoot's warnings stay visible above the keybar for the whole session.
func NewRoot(targets []target.Target, env check.Env, dryRun bool, warnings []string) *Root {
	ctx, cancel := context.WithCancel(context.Background())
	return &Root{
		targets:  targets,
		env:      env,
		dryRun:   dryRun,
		warnings: warnings,
		ctx:      ctx,
		cancel:   cancel,
		keys:     newKeyMap(),
		th:       newTheme(defaultDark),
		help:     help.New(),
		states:   make([]targetState, len(targets)),
		tabs:     []string{dashboardTitle},
	}
}

func (r *Root) Init() tea.Cmd { return r.compute(r.cursor, false) }

// compute starts a computation for targets[idx] unless it is unavailable, already
// ready (cached) or already running. force recomputes a ready one.
func (r *Root) compute(idx int, force bool) tea.Cmd {
	if idx >= len(r.targets) || !available(r.targets[idx]) {
		return nil
	}
	st := &r.states[idx]
	if st.phase == loading || (st.phase == ready && !force) {
		return nil
	}
	st.phase = loading
	st.gen++
	gen, ctx, env, t := st.gen, r.ctx, r.env, r.targets[idx]
	return func() tea.Msg {
		return computedMsg{idx: idx, gen: gen, dash: dashboard.Compute(ctx, env, t)}
	}
}

func (r *Root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.w, r.h = msg.Width, msg.Height
		r.clampScroll()
	case computedMsg:
		if msg.idx < len(r.states) && r.states[msg.idx].phase == loading && r.states[msg.idx].gen == msg.gen {
			st := &r.states[msg.idx]
			st.phase, st.dash = ready, msg.dash
			r.clampScroll()
		}
	case tea.KeyPressMsg:
		return r, r.handleKey(msg)
	}
	return r, nil
}

func (r *Root) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	if key.Matches(msg, r.keys.Quit) {
		r.cancel()
		return tea.Quit
	}
	if r.showHelp {
		if key.Matches(msg, r.keys.Help) || msg.String() == "esc" {
			r.showHelp = false
		}
		return nil
	}
	switch {
	case key.Matches(msg, r.keys.Help):
		r.showHelp = true
	case len(r.targets) == 0:
	case r.focus == focusSidebar:
		return r.handleSidebarKey(msg)
	default:
		return r.handlePanelKey(msg)
	}
	return nil
}

func (r *Root) handleSidebarKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, r.keys.Up):
		return r.selectTarget(r.cursor - 1)
	case key.Matches(msg, r.keys.Down):
		return r.selectTarget(r.cursor + 1)
	case key.Matches(msg, r.keys.Focus):
		if available(r.targets[r.cursor]) {
			r.focus = focusPanel
		}
	}
	return nil
}

func (r *Root) handlePanelKey(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, r.keys.Back):
		r.focus = focusSidebar
	case key.Matches(msg, r.keys.Up):
		r.scrollBy(-1)
	case key.Matches(msg, r.keys.Down):
		r.scrollBy(1)
	case key.Matches(msg, r.keys.PrevTab):
		r.tab = (r.tab + len(r.tabs) - 1) % len(r.tabs)
	case key.Matches(msg, r.keys.NextTab):
		r.tab = (r.tab + 1) % len(r.tabs)
	case key.Matches(msg, r.keys.Refresh):
		return r.compute(r.cursor, true)
	}
	return nil
}

func (r *Root) selectTarget(idx int) tea.Cmd {
	if idx < 0 || idx >= len(r.targets) {
		return nil
	}
	r.cursor, r.scroll = idx, 0
	return r.compute(idx, false)
}

func (r *Root) scrollBy(delta int) {
	r.scroll += delta
	r.clampScroll()
}

func (r *Root) clampScroll() {
	r.scroll = max(min(r.scroll, len(r.panelContent())-r.viewportHeight()), 0)
}

type layout struct{ sidebarW, panelW, bodyH int }

func (r *Root) layout() layout {
	sidebarW := min(sidebarWidth, r.w*2/5)
	return layout{sidebarW: sidebarW, panelW: r.w - sidebarW, bodyH: r.h - 1 - len(r.warningLines())}
}

// warningLines wraps the warnings to the terminal width, capped so they cannot crowd out the body.
func (r *Root) warningLines() []string {
	if len(r.warnings) == 0 {
		return nil
	}
	lines := wrap(strings.Join(r.warnings, "  "), r.w)
	lines = lines[:min(len(lines), maxWarningLines)]
	return styleAll(r.th.warning, lines)
}

func (r *Root) viewportHeight() int {
	return max(r.layout().bodyH-borderCells-panelChrome, 0)
}

func (r *Root) panelContent() []string {
	if len(r.targets) == 0 {
		return nil
	}
	width := contentWidth(r.layout().panelW)
	t := r.targets[r.cursor]
	switch st := r.states[r.cursor]; {
	case t.Err != nil:
		return styleAll(r.th.err, wrap(statusGlyph(check.StatusError)+" "+t.Err.Error(), width))
	case !available(t):
		return []string{r.th.muted.Render(notAvailable)}
	case st.phase == ready:
		return dashboardLines(r.th, st.dash, width)
	default:
		return []string{r.th.muted.Render(computingText)}
	}
}

func styleAll(s lipgloss.Style, lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = s.Render(strings.TrimRight(l, " "))
	}
	return out
}

func (r *Root) tabRow() string {
	titles := make([]string, len(r.tabs))
	for i, name := range r.tabs {
		style := r.th.tabIdle
		if i == r.tab {
			style = r.th.tabActive
		}
		titles[i] = style.Render(name)
	}
	return strings.Join(titles, "  ")
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
	l := r.layout()
	sidebar := renderBox(r.th, sidebarTitle, sidebarLines(r.th, r.targets, r.cursor, contentWidth(l.sidebarW)), l.sidebarW, l.bodyH, r.focus == focusSidebar)
	panel := renderBox(r.th, r.targets[r.cursor].Spec.Name, r.panelBody(), l.panelW, l.bodyH, r.focus == focusPanel)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, panel)
	bindings := r.keys.sidebarKeys()
	if r.focus == focusPanel {
		bindings = r.keys.panelKeys()
	}
	if r.showHelp {
		body = overlayCentered(body, helpPopup(r.th, r.help, r.keys, r.w, l.bodyH), r.w, l.bodyH)
		bindings = []key.Binding{r.keys.Help, r.keys.Quit}
	}
	for _, line := range r.warningLines() {
		body += "\n" + line
	}
	return body + "\n" + keybar(r.th, r.help, bindings, r.dryRun, r.w)
}

func (r *Root) panelBody() []string {
	if !available(r.targets[r.cursor]) {
		return r.panelContent()
	}
	content := r.panelContent()
	end := min(r.scroll+r.viewportHeight(), len(content))
	return append([]string{r.tabRow(), ""}, content[min(r.scroll, end):end]...)
}
