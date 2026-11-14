package ui

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

const (
	jumpRows = 5
	jumpHelp = "5 rows"
)

type keyMap struct {
	Up, Down, JumpUp, JumpDown  key.Binding
	Move, Focus, Back           key.Binding
	PrevTab, NextTab            key.Binding
	Refresh, Help, Quit         key.Binding
	Install, Mode               key.Binding
	Toggle, All, Accept, Cancel key.Binding
	Remove, Restore, Delete     key.Binding
	Backup                      key.Binding
	Validate, FastInit          key.Binding
	Maintenance                 key.Binding
	Edit, Reset, UpdateTUI      key.Binding
	Yes, CancelRun              key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		// A text keypress stringifies as "J"; the kitty keyboard protocol may report "shift+j" without text.
		JumpUp:   key.NewBinding(key.WithKeys("K", "shift+k"), key.WithHelp("K", "up "+jumpHelp)),
		JumpDown: key.NewBinding(key.WithKeys("J", "shift+j"), key.WithHelp("J", "down "+jumpHelp)),
		Move:     key.NewBinding(key.WithKeys("up", "down", "k", "j", "K", "J", "shift+k", "shift+j"), key.WithHelp("↑/↓", "move")),
		Focus:    key.NewBinding(key.WithKeys("tab", "enter", "right", "l"), key.WithHelp("tab/enter", "open panel")),
		Back:     key.NewBinding(key.WithKeys("esc", "tab"), key.WithHelp("esc/tab", "sidebar")),
		PrevTab:  key.NewBinding(key.WithKeys("[", "left", "h"), key.WithHelp("[/h", "prev tab")),
		NextTab:  key.NewBinding(key.WithKeys("]", "right", "l"), key.WithHelp("]/l", "next tab")),
		Refresh:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Install:  key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "install")),
		Mode:     key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "mode")),
		Toggle:   key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
		All:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "all / none")),
		Accept:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept")),
		Cancel:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Remove:   key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "remove")),
		Restore:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restore")),
		Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Backup:   key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "backup now")),
		Yes:      key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "confirm")),

		Validate:    key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "validate")),
		FastInit:    key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "fast init")),
		Maintenance: key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "maintenance")),
		Edit:        key.NewBinding(key.WithKeys("e", "enter"), key.WithHelp("e/enter", "edit")),
		Reset:       key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reset to default")),
		UpdateTUI:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update TUI")),
		// Outside a run ctrl+c quits through Quit; CancelRun only describes it while one is active.
		CancelRun: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "cancel run")),
	}
}

// step is the cursor or scroll delta a movement key asks for, or 0 for any other key.
func (k keyMap) step(msg tea.KeyPressMsg) int {
	switch {
	case key.Matches(msg, k.Up):
		return -1
	case key.Matches(msg, k.Down):
		return 1
	case key.Matches(msg, k.JumpUp):
		return -jumpRows
	case key.Matches(msg, k.JumpDown):
		return jumpRows
	}
	return 0
}

// described returns b with another help text, for keys whose meaning depends on the stage.
func described(b key.Binding, desc string) key.Binding {
	b.SetHelp(b.Help().Key, desc)
	return b
}

// claims reports whether one of the bindings answers to the key.
func claims(bindings []key.Binding, k string) bool {
	return slices.ContainsFunc(bindings, func(b key.Binding) bool { return slices.Contains(b.Keys(), k) })
}

// pinnedKeys stay visible at the right of the keybar however many keys the focus has.
func (k keyMap) pinnedKeys() []key.Binding { return []key.Binding{k.Help, k.Quit} }

// sidebarKeys and panelKeys are the keys valid in each focus; the keybar and help derive from them.
func (k keyMap) sidebarKeys() []key.Binding {
	return []key.Binding{k.Focus, k.Install, k.Move}
}

// panelKeys puts the focused tab's keys first. r stays "refresh" unless the tab claims it
// (Backups: restore); esc goes back to the sidebar unless the tab claims it (a flow).
func (k keyMap) panelKeys(tabKeys []key.Binding) []key.Binding {
	keys := slices.Clone(tabKeys)
	if !claims(tabKeys, "r") {
		keys = append(keys, k.Refresh)
	}
	keys = append(keys, k.Install)
	if !claims(tabKeys, "esc") {
		keys = append(keys, k.Back)
	}
	return keys
}

// helpGroups lists every key in three columns; r shows what it does in the focused tab.
// m means maintenance on the Specter screen and the install mode elsewhere, so each context lists only its own.
func (k keyMap) helpGroups(tabKeys []key.Binding, onSpecter bool) [][]key.Binding {
	r := k.Refresh
	if claims(tabKeys, "r") {
		r = k.Restore
	}
	tabs := key.NewBinding(key.WithKeys("[", "]", "h", "l", "left", "right"), key.WithHelp("[/] h/l", "tabs"))
	middle := []key.Binding{tabs, r, k.Install, k.Mode}
	if onSpecter {
		middle = []key.Binding{tabs, r, k.Validate, k.FastInit, k.Maintenance}
	}
	return [][]key.Binding{
		{k.Up, k.Down, k.JumpUp, k.JumpDown, k.Focus, k.Back, k.Help, k.Quit, k.CancelRun},
		middle,
		{k.Toggle, k.All, k.Accept, k.Yes, k.Cancel, k.Remove, k.Delete, k.Backup},
	}
}

// helpIntro names the tabs of the focused target, which the key columns cannot.
func helpIntro(titles []string) string {
	if len(titles) == 0 {
		return ""
	}
	return "Tabs: " + strings.Join(titles, ", ")
}
