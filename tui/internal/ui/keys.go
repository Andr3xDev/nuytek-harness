package ui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up, Down, Focus, Back key.Binding
	PrevTab, NextTab      key.Binding
	Refresh, Help, Quit   key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Focus:   key.NewBinding(key.WithKeys("tab", "enter", "right", "l"), key.WithHelp("tab/enter", "open panel")),
		Back:    key.NewBinding(key.WithKeys("esc", "tab", "left", "h"), key.WithHelp("esc/tab", "sidebar")),
		PrevTab: key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev tab")),
		NextTab: key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next tab")),
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

// sidebarKeys and panelKeys are the keys valid in each focus; the keybar and help derive from them.
func (k keyMap) sidebarKeys() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Focus, k.Help, k.Quit}
}

func (k keyMap) panelKeys() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Refresh, k.Back, k.Help, k.Quit}
}

func (k keyMap) helpGroups() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Focus, k.Back},
		{k.PrevTab, k.NextTab, k.Refresh},
		{k.Help, k.Quit},
	}
}
