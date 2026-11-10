package ui

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
)

type keyMap struct {
	Up, Down, Move, Focus, Back key.Binding
	PrevTab, NextTab            key.Binding
	Refresh, Help, Quit         key.Binding
	Install, Mode               key.Binding
	Toggle, All, Accept, Cancel key.Binding
	Remove, Restore, Delete     key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Move:    key.NewBinding(key.WithKeys("up", "down", "k", "j"), key.WithHelp("↑/↓", "move")),
		Focus:   key.NewBinding(key.WithKeys("tab", "enter", "right", "l"), key.WithHelp("tab/enter", "open panel")),
		Back:    key.NewBinding(key.WithKeys("esc", "tab", "left", "h"), key.WithHelp("esc/tab", "sidebar")),
		PrevTab: key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev tab")),
		NextTab: key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next tab")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Install: key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "install")),
		Mode:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "mode")),
		Toggle:  key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
		All:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "all / none")),
		Accept:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept")),
		Cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Remove:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "remove")),
		Restore: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restore")),
		Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	}
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
func (k keyMap) helpGroups(tabKeys []key.Binding) [][]key.Binding {
	r := k.Refresh
	if claims(tabKeys, "r") {
		r = k.Restore
	}
	tabs := key.NewBinding(key.WithKeys("[", "]"), key.WithHelp("[/]", "tabs"))
	return [][]key.Binding{
		{k.Up, k.Down, k.Focus, k.Back, k.Help, k.Quit},
		{tabs, r, k.Install, k.Mode},
		{k.Toggle, k.All, k.Accept, k.Cancel, k.Remove, k.Delete},
	}
}

// helpIntro names the tabs of the focused target, which the key columns cannot.
func helpIntro(titles []string) string {
	if len(titles) == 0 {
		return ""
	}
	return "Tabs: " + strings.Join(titles, ", ")
}
