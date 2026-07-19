package tui

import (
	"github.com/bolna-ai/cli/internal/api"
	"github.com/bolna-ai/cli/internal/tui/styles"
	"github.com/charmbracelet/bubbles/list"
)

// paletteAction identifies what a palette selection should do; the
// dashboard interprets it after list.Model returns the chosen item.
type paletteAction int

const (
	actionGoAgents paletteAction = iota
	actionGoNumbers
	actionGoAccount
	actionGoAgentDetail
)

type paletteItem struct {
	title, desc string
	action      paletteAction
	agentID     string
}

func (p paletteItem) Title() string       { return p.title }
func (p paletteItem) Description() string { return p.desc }
func (p paletteItem) FilterValue() string { return p.title }

// buildPaletteItems lists the fixed sections plus every known agent, so
// typing ":" and a few letters of an agent's name jumps straight to it.
func buildPaletteItems(agents []api.AgentSummary) []list.Item {
	items := []list.Item{
		paletteItem{title: "Agents", desc: "list every agent on the account", action: actionGoAgents},
		paletteItem{title: "Numbers", desc: "phone numbers on the account", action: actionGoNumbers},
		paletteItem{title: "Account", desc: "wallet balance and concurrency", action: actionGoAccount},
	}
	for _, ag := range agents {
		items = append(items, paletteItem{
			title:   ag.AgentName,
			desc:    "agent · " + ag.AgentStatus + " · " + ag.ID,
			action:  actionGoAgentDetail,
			agentID: ag.ID,
		})
	}
	return items
}

func newPaletteList(theme styles.Theme, items []list.Item, width, height int) list.Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(theme.Palette.Primary).BorderLeftForeground(theme.Palette.Primary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(theme.Palette.Muted).BorderLeftForeground(theme.Palette.Primary)

	l := list.New(items, delegate, width, height)
	l.Title = "Jump to…"
	l.Styles.Title = theme.Title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	return l
}
