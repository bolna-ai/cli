// Package styles centralizes bolna-cli's Lip Gloss theming so the plain-text
// CLI output and the full-screen TUI share one visual language. Themes use
// lipgloss.AdaptiveColor throughout so they look right in both light and
// dark terminals.
package styles

import "github.com/charmbracelet/lipgloss"

// Palette is the small set of semantic colors every theme must define.
type Palette struct {
	Name    string
	Primary lipgloss.AdaptiveColor
	Accent  lipgloss.AdaptiveColor
	Success lipgloss.AdaptiveColor
	Warning lipgloss.AdaptiveColor
	Danger  lipgloss.AdaptiveColor
	Muted   lipgloss.AdaptiveColor
	Text    lipgloss.AdaptiveColor
}

var (
	// Bolna matches the steel-blue → periwinkle gradient used in Bolna's own
	// ASCII wordmark at mcp.bolna.ai (#3F5C8C → #A9BCDD), rather than an
	// arbitrary Charm-style purple.
	Bolna = Palette{
		Name:    "bolna",
		Primary: lipgloss.AdaptiveColor{Light: "#3F5C8C", Dark: "#8AA0C5"},
		Accent:  lipgloss.AdaptiveColor{Light: "#5973A0", Dark: "#A9BCDD"},
		Success: lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#5FD97A"},
		Warning: lipgloss.AdaptiveColor{Light: "#B08800", Dark: "#F5D547"},
		Danger:  lipgloss.AdaptiveColor{Light: "#C4262E", Dark: "#FF6B6B"},
		Muted:   lipgloss.AdaptiveColor{Light: "#767676", Dark: "#8A8A8A"},
		Text:    lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#E4E4E4"},
	}
	TokyoNight = Palette{
		Name:    "tokyo-night",
		Primary: lipgloss.AdaptiveColor{Light: "#34548A", Dark: "#7AA2F7"},
		Accent:  lipgloss.AdaptiveColor{Light: "#3D6899", Dark: "#7DCFFF"},
		Success: lipgloss.AdaptiveColor{Light: "#3D7A45", Dark: "#9ECE6A"},
		Warning: lipgloss.AdaptiveColor{Light: "#9A7A1A", Dark: "#E0AF68"},
		Danger:  lipgloss.AdaptiveColor{Light: "#B5406A", Dark: "#F7768E"},
		Muted:   lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#565F89"},
		Text:    lipgloss.AdaptiveColor{Light: "#1A1B26", Dark: "#C0CAF5"},
	}
	// Nord is the well-known arctic/frost palette — every accent color in it
	// is already a shade of blue or teal.
	Nord = Palette{
		Name:    "nord",
		Primary: lipgloss.AdaptiveColor{Light: "#5E81AC", Dark: "#81A1C1"},
		Accent:  lipgloss.AdaptiveColor{Light: "#3B6EA5", Dark: "#88C0D0"},
		Success: lipgloss.AdaptiveColor{Light: "#4C7A3D", Dark: "#A3BE8C"},
		Warning: lipgloss.AdaptiveColor{Light: "#A17F1A", Dark: "#EBCB8B"},
		Danger:  lipgloss.AdaptiveColor{Light: "#B5474E", Dark: "#BF616A"},
		Muted:   lipgloss.AdaptiveColor{Light: "#6B7385", Dark: "#4C566A"},
		Text:    lipgloss.AdaptiveColor{Light: "#2E3440", Dark: "#ECEFF4"},
	}
)

// Themes lists every built-in palette, in menu order, for the theme picker.
// Bolna is first and is the default — it's the CLI's own brand theme. Every
// palette here is a shade of blue by design (no violet/purple), matching
// Bolna's brand color.
var Themes = []Palette{Bolna, TokyoNight, Nord}

// ByName looks up a palette by its Name, falling back to Bolna.
func ByName(name string) Palette {
	for _, p := range Themes {
		if p.Name == name {
			return p
		}
	}
	return Bolna
}

// Theme is a fully-built set of reusable Lip Gloss styles for one palette.
type Theme struct {
	Palette Palette

	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Muted     lipgloss.Style
	Bold      lipgloss.Style
	Success   lipgloss.Style
	Warning   lipgloss.Style
	Danger    lipgloss.Style
	Badge     lipgloss.Style
	Card      lipgloss.Style
	Sidebar   lipgloss.Style
	StatusBar lipgloss.Style
	Selected  lipgloss.Style
	TableHead lipgloss.Style
	TableRow  lipgloss.Style
	HelpKey   lipgloss.Style
	HelpDesc  lipgloss.Style
	Prompt    lipgloss.Style
}

// New builds a Theme from a Palette.
func New(p Palette) Theme {
	return Theme{
		Palette: p,

		Title:    lipgloss.NewStyle().Bold(true).Foreground(p.Primary),
		Subtitle: lipgloss.NewStyle().Foreground(p.Accent),
		Muted:    lipgloss.NewStyle().Foreground(p.Muted),
		Bold:     lipgloss.NewStyle().Bold(true).Foreground(p.Text),
		Success:  lipgloss.NewStyle().Foreground(p.Success),
		Warning:  lipgloss.NewStyle().Foreground(p.Warning),
		Danger:   lipgloss.NewStyle().Bold(true).Foreground(p.Danger),

		Badge: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("#0B0B0B")).
			Background(p.Primary),

		Card: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.Primary).
			Padding(1, 2),

		Sidebar: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(p.Muted).
			Padding(0, 1),

		StatusBar: lipgloss.NewStyle().
			Foreground(p.Text).
			Background(lipgloss.AdaptiveColor{Light: "#E4E4E4", Dark: "#262626"}).
			Padding(0, 1),

		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Text).
			Background(p.Primary).
			Padding(0, 1),

		TableHead: lipgloss.NewStyle().Bold(true).Foreground(p.Primary),
		TableRow:  lipgloss.NewStyle().Foreground(p.Text),

		HelpKey:  lipgloss.NewStyle().Foreground(p.Accent).Bold(true),
		HelpDesc: lipgloss.NewStyle().Foreground(p.Muted),

		Prompt: lipgloss.NewStyle().Foreground(p.Primary).Bold(true),
	}
}

// StatusColor maps a Bolna status string (agent_status/call status) to a
// semantic color, so tables/cards render status consistently everywhere.
func (t Theme) StatusColor(status string) lipgloss.Style {
	switch status {
	case "completed", "answered", "connected", "active", "success", "processed", "call-disconnected":
		return t.Success
	case "queued", "ringing", "in-progress", "in_progress", "created", "scheduled", "processing":
		return t.Warning
	case "failed", "error", "busy", "no-answer", "no_answer", "cancelled", "inactive":
		return t.Danger
	default:
		return t.Muted
	}
}
