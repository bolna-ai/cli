package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Palette   key.Binding
	Back      key.Binding
	Quit      key.Binding
	Select    key.Binding
	Calls     key.Binding
	Batches   key.Binding
	StartCall key.Binding
	Refresh   key.Binding
	Theme     key.Binding
	Agents    key.Binding
	Numbers   key.Binding
	Account   key.Binding
}

var keys = keyMap{
	Palette:   key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command palette")),
	Back:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Select:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	Calls:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "calls")),
	Batches:   key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "batches")),
	StartCall: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start call")),
	Refresh:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Theme:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "cycle theme")),
	Agents:    key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "agents")),
	Numbers:   key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "numbers")),
	Account:   key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "account")),
}

func helpLine(extra ...string) []string {
	base := []string{
		keys.Palette.Help().Key + " " + keys.Palette.Help().Desc,
		keys.Select.Help().Key + " " + keys.Select.Help().Desc,
		keys.Back.Help().Key + " " + keys.Back.Help().Desc,
		keys.Refresh.Help().Key + " " + keys.Refresh.Help().Desc,
		keys.Theme.Help().Key + " " + keys.Theme.Help().Desc,
		keys.Quit.Help().Key + " " + keys.Quit.Help().Desc,
	}
	return append(extra, base...)
}
