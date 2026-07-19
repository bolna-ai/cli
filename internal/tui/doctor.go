// Package tui holds every Bubble Tea program bolna-cli runs: the doctor
// checklist, the full-screen dashboard, and the shared components/styles
// they're built from.
package tui

import (
	"fmt"
	"strings"

	"github.com/bolna-ai/cli/internal/tui/styles"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Check is one doctor check: a name and a function that runs it. Run should
// itself apply any timeout it needs — the doctor program doesn't impose one.
type Check struct {
	Name string
	Run  func() (ok bool, detail string)
}

type checkState int

const (
	pending checkState = iota
	running
	passed
	failed
)

type doctorResultMsg struct {
	index  int
	ok     bool
	detail string
}

type doctorModel struct {
	checks  []Check
	states  []checkState
	details []string
	index   int
	spinner spinner.Model
	theme   styles.Theme
	allOK   bool
}

func newDoctorModel(checks []Check, theme styles.Theme) doctorModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = theme.Subtitle
	return doctorModel{
		checks:  checks,
		states:  make([]checkState, len(checks)),
		details: make([]string, len(checks)),
		spinner: sp,
		theme:   theme,
	}
}

func runCheck(checks []Check, index int) tea.Cmd {
	return func() tea.Msg {
		ok, detail := checks[index].Run()
		return doctorResultMsg{index: index, ok: ok, detail: detail}
	}
}

func (m doctorModel) Init() tea.Cmd {
	if len(m.checks) == 0 {
		return tea.Quit
	}
	m.states[0] = running
	return tea.Batch(m.spinner.Tick, runCheck(m.checks, 0))
}

func (m doctorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	case doctorResultMsg:
		if msg.ok {
			m.states[msg.index] = passed
		} else {
			m.states[msg.index] = failed
		}
		m.details[msg.index] = msg.detail
		next := msg.index + 1
		m.index = next
		if next >= len(m.checks) {
			m.allOK = true
			for _, s := range m.states {
				if s != passed {
					m.allOK = false
				}
			}
			return m, tea.Quit
		}
		m.states[next] = running
		return m, runCheck(m.checks, next)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m doctorModel) View() string {
	var b strings.Builder
	b.WriteString(m.theme.Title.Render("bolna doctor") + "\n\n")
	for i, check := range m.checks {
		switch m.states[i] {
		case passed:
			b.WriteString(m.theme.Success.Render("✓ ") + check.Name)
		case failed:
			b.WriteString(m.theme.Danger.Render("✗ ") + check.Name)
		case running:
			b.WriteString(m.spinner.View() + " " + check.Name)
		default:
			b.WriteString(m.theme.Muted.Render("  " + check.Name))
		}
		if d := m.details[i]; d != "" && m.states[i] != running {
			b.WriteString(m.theme.Muted.Render("  — " + d))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// RunDoctor runs every check with an animated spinner (on a TTY) and returns
// whether all checks passed.
func RunDoctor(checks []Check, theme styles.Theme) (bool, error) {
	m := newDoctorModel(checks, theme)
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return false, err
	}
	return final.(doctorModel).allOK, nil
}

// RunDoctorPlain runs every check sequentially with plain ✓/✗ output, for
// non-TTY sessions (CI, piping) where a Bubble Tea program can't render.
func RunDoctorPlain(checks []Check, theme styles.Theme) bool {
	allOK := true
	fmt.Println(theme.Title.Render("bolna doctor"))
	for _, check := range checks {
		ok, detail := check.Run()
		mark := theme.Success.Render("✓")
		if !ok {
			mark = theme.Danger.Render("✗")
			allOK = false
		}
		line := mark + " " + check.Name
		if detail != "" {
			line += theme.Muted.Render("  — " + detail)
		}
		fmt.Println(line)
	}
	return allOK
}
