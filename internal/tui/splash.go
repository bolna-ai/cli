package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/bolna-ai/cli/internal/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/lipgloss"
)

const splashFPS = 60

var banner = []string{
	`██████╗  ██████╗ ██╗     ███╗   ██╗ █████╗ `,
	`██╔══██╗██╔═══██╗██║     ████╗  ██║██╔══██╗`,
	`██████╔╝██║   ██║██║     ██╔██╗ ██║███████║`,
	`██╔══██╗██║   ██║██║     ██║╚██╗██║██╔══██║`,
	`██████╔╝╚██████╔╝███████╗██║ ╚████║██║  ██║`,
	`╚═════╝  ╚═════╝ ╚══════╝╚═╝  ╚═══╝╚═╝  ╚═╝`,
}

type splashTickMsg time.Time

type splashDoneMsg struct{}

// splashModel plays a short spring-physics slide-in of the wordmark: the
// banner starts offset well below its resting position and a harmonica
// spring pulls it up into place, settling naturally rather than linearly.
type splashModel struct {
	theme    styles.Theme
	spring   harmonica.Spring
	pos      float64
	velocity float64
	done     bool
}

func newSplashModel(theme styles.Theme) splashModel {
	return splashModel{
		theme:  theme,
		spring: harmonica.NewSpring(harmonica.FPS(splashFPS), 6.0, 0.65),
		pos:    18, // starts 18 lines "below" its resting position
	}
}

func splashTick() tea.Cmd {
	return tea.Tick(time.Second/splashFPS, func(t time.Time) tea.Msg {
		return splashTickMsg(t)
	})
}

func (m splashModel) Init() tea.Cmd {
	return splashTick()
}

func (m splashModel) Update(msg tea.Msg) (splashModel, tea.Cmd) {
	switch msg.(type) {
	case splashTickMsg:
		m.pos, m.velocity = m.spring.Update(m.pos, m.velocity, 0)
		if math.Abs(m.pos) < 0.05 && math.Abs(m.velocity) < 0.05 {
			m.pos = 0
			m.done = true
			return m, func() tea.Msg { return splashDoneMsg{} }
		}
		return m, splashTick()
	}
	return m, nil
}

// bannerGradient renders the wordmark with a per-line gradient from a deep
// steel blue to a light periwinkle — the same two shades Bolna's own ASCII
// wordmark at mcp.bolna.ai fades between, rather than a flat single color.
func bannerGradient() string {
	colors := gradient("#3F5C8C", "#A9BCDD", len(banner))
	lines := make([]string, len(banner))
	for i, line := range banner {
		lines[i] = lipgloss.NewStyle().Bold(true).Foreground(colors[i]).Render(line)
	}
	return strings.Join(lines, "\n")
}

// gradient linearly interpolates n colors between two "#RRGGBB" hex strings.
func gradient(fromHex, toHex string, n int) []lipgloss.Color {
	fr, fg, fb := hexRGB(fromHex)
	tr, tg, tb := hexRGB(toHex)
	colors := make([]lipgloss.Color, n)
	for i := 0; i < n; i++ {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		r := lerp(fr, tr, t)
		g := lerp(fg, tg, t)
		b := lerp(fb, tb, t)
		colors[i] = lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b))
	}
	return colors
}

func hexRGB(hex string) (r, g, b int) {
	hex = strings.TrimPrefix(hex, "#")
	r64, _ := strconv.ParseInt(hex[0:2], 16, 0)
	g64, _ := strconv.ParseInt(hex[2:4], 16, 0)
	b64, _ := strconv.ParseInt(hex[4:6], 16, 0)
	return int(r64), int(g64), int(b64)
}

func lerp(from, to int, t float64) int {
	return int(math.Round(float64(from) + (float64(to)-float64(from))*t))
}

func (m splashModel) View(width, height int) string {
	padLines := int(math.Round(m.pos))
	if padLines < 0 {
		// The spring is underdamped and legitimately overshoots below 0
		// before settling; clamp rather than pass a negative count to
		// strings.Repeat, which panics.
		padLines = 0
	}
	pad := strings.Repeat("\n", padLines)
	block := lipgloss.JoinVertical(lipgloss.Center,
		bannerGradient(),
		"",
		m.theme.Muted.Render("voice AI, from the terminal"),
	)
	content := pad + block
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
