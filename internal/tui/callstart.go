package tui

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/bolna-ai/bolna-cli/internal/api"
	"github.com/bolna-ai/bolna-cli/internal/tui/styles"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var e164Pattern = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

type callStep int

const (
	stepPhone callStep = iota
	stepConfirm
	stepDialing
	stepLive
	stepDone
)

// callAction tells the parent dashboard what to do once callStartModel has
// handled a message: keep going, pop back to the agent detail screen
// (cancelled or finished), or nothing special.
type callAction int

const (
	callNone callAction = iota
	callExit
)

const (
	waveFPS         = 12
	callPollEvery   = 1500 * time.Millisecond
	waveBars        = 14
	waveTargetEvery = 6 // wave ticks between picking new random bar targets
)

type waveTickMsg time.Time
type callPollTickMsg time.Time
type callStartedMsg struct {
	executionID string
	err         error
}
type callPolledMsg struct {
	execution api.Execution
	err       error
}

// callStartModel drives `s` (start call) from an agent's detail screen: a
// phone-number prompt, a cost/balance confirmation (placing a call spends
// real account balance — this step is never skipped), a live animated
// "dialing" waveform that polls the execution's status, and finally the
// transcript once the call ends.
type callStartModel struct {
	theme     styles.Theme
	client    *api.Client
	agentID   string
	agentName string
	balance   float64
	hasBal    bool

	step  callStep
	phone textinput.Model
	err   string

	spinner spinner.Model
	bars    []float64
	targets []float64
	waveN   int

	executionID string
	execution   api.Execution
}

func newCallStartModel(theme styles.Theme, client *api.Client, agentID, agentName string, balance float64, hasBal bool) callStartModel {
	ti := textinput.New()
	ti.Placeholder = "+14155552671"
	ti.Prompt = "Recipient phone number (E.164): "
	ti.PromptStyle = theme.Prompt
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = theme.Subtitle

	bars := make([]float64, waveBars)
	targets := make([]float64, waveBars)
	for i := range targets {
		targets[i] = rand.Float64()
	}

	return callStartModel{
		theme:     theme,
		client:    client,
		agentID:   agentID,
		agentName: agentName,
		balance:   balance,
		hasBal:    hasBal,
		step:      stepPhone,
		phone:     ti,
		spinner:   sp,
		bars:      bars,
		targets:   targets,
	}
}

func (m callStartModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m callStartModel) Update(msg tea.Msg) (callStartModel, tea.Cmd, callAction) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.step {
		case stepPhone:
			switch msg.String() {
			case "esc":
				return m, nil, callExit
			case "enter":
				phone := strings.TrimSpace(m.phone.Value())
				if !e164Pattern.MatchString(phone) {
					m.err = "enter a valid E.164 number, e.g. +14155552671"
					return m, nil, callNone
				}
				m.err = ""
				m.step = stepConfirm
				return m, nil, callNone
			}
			var cmd tea.Cmd
			m.phone, cmd = m.phone.Update(msg)
			return m, cmd, callNone

		case stepConfirm:
			switch msg.String() {
			case "y", "Y", "enter":
				m.step = stepDialing
				return m, tea.Batch(m.spinner.Tick, startCallCmd(m.client, m.agentID, strings.TrimSpace(m.phone.Value()))), callNone
			case "n", "N", "esc":
				return m, nil, callExit
			}
			return m, nil, callNone

		case stepDialing:
			if msg.String() == "esc" {
				return m, nil, callExit
			}
			return m, nil, callNone

		case stepLive:
			if msg.String() == "esc" {
				return m, nil, callExit
			}
			return m, nil, callNone

		case stepDone:
			return m, nil, callExit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd, callNone

	case callStartedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.step = stepDone
			return m, nil, callNone
		}
		m.executionID = msg.executionID
		m.step = stepLive
		m.waveN = 0
		return m, tea.Batch(waveTick(), pollTick(), fetchExecutionForCall(m.client, m.executionID)), callNone

	case waveTickMsg:
		if m.step != stepLive {
			return m, nil, callNone
		}
		m.waveN++
		if m.waveN%waveTargetEvery == 0 {
			for i := range m.targets {
				m.targets[i] = rand.Float64()
			}
		}
		for i := range m.bars {
			m.bars[i] += (m.targets[i] - m.bars[i]) * 0.3
		}
		return m, waveTick(), callNone

	case callPollTickMsg:
		if m.step != stepLive {
			return m, nil, callNone
		}
		return m, tea.Batch(pollTick(), fetchExecutionForCall(m.client, m.executionID)), callNone

	case callPolledMsg:
		if msg.err != nil {
			return m, nil, callNone
		}
		m.execution = msg.execution
		if isTerminalCallStatus(msg.execution.Status()) {
			m.step = stepDone
		}
		return m, nil, callNone
	}
	return m, nil, callNone
}

func startCallCmd(client *api.Client, agentID, recipient string) tea.Cmd {
	return func() tea.Msg {
		result, err := client.StartCall(api.StartCallInput{AgentID: agentID, RecipientPhoneNumber: recipient})
		if err != nil {
			return callStartedMsg{err: err}
		}
		id, _ := result["execution_id"].(string)
		if id == "" {
			id, _ = result["id"].(string)
		}
		return callStartedMsg{executionID: id}
	}
}

func fetchExecutionForCall(client *api.Client, executionID string) tea.Cmd {
	return func() tea.Msg {
		execution, err := client.GetExecution(executionID)
		return callPolledMsg{execution: execution, err: err}
	}
}

func waveTick() tea.Cmd {
	return tea.Tick(time.Second/waveFPS, func(t time.Time) tea.Msg { return waveTickMsg(t) })
}

func pollTick() tea.Cmd {
	return tea.Tick(callPollEvery, func(t time.Time) tea.Msg { return callPollTickMsg(t) })
}

func isTerminalCallStatus(status string) bool {
	switch status {
	case "completed", "call-disconnected", "failed", "error", "busy", "no-answer", "no_answer", "cancelled":
		return true
	}
	return false
}

func (m callStartModel) View(width int) string {
	theme := m.theme
	var body string

	switch m.step {
	case stepPhone:
		body = theme.Title.Render("Start a call") + "\n" +
			theme.Muted.Render("Agent: "+m.agentName) + "\n\n" +
			m.phone.View()
		if m.err != "" {
			body += "\n\n" + theme.Danger.Render(m.err)
		}
		body += "\n\n" + theme.HelpDesc.Render("enter: continue  •  esc: cancel")

	case stepConfirm:
		body = theme.Warning.Render("⚠ This places a real phone call and spends account balance.") + "\n\n"
		body += fmt.Sprintf("%s %s\n", theme.Subtitle.Render("Agent:"), m.agentName)
		body += fmt.Sprintf("%s %s\n", theme.Subtitle.Render("Recipient:"), strings.TrimSpace(m.phone.Value()))
		if m.hasBal {
			body += fmt.Sprintf("%s %s\n", theme.Subtitle.Render("Wallet:"), theme.Success.Render(fmt.Sprintf("$%.2f", m.balance)))
		}
		body += "\n" + theme.Bold.Render("Place this call?") + "  " + theme.HelpDesc.Render("y: yes  •  n/esc: cancel")

	case stepDialing:
		body = theme.Title.Render("Start a call") + "\n\n" +
			m.spinner.View() + " " + theme.Muted.Render("Placing call…")

	case stepLive:
		status := m.execution.Status()
		body = theme.Title.Render("Call in progress") + "  " + theme.StatusColor(status).Render(orDash(status)) + "\n"
		body += theme.Muted.Render("Execution ID: "+m.executionID) + "\n\n"
		body += renderWaveform(theme, m.bars) + "\n\n"
		body += theme.HelpDesc.Render("esc: back (call keeps running)")

	case stepDone:
		if m.err != "" {
			body = theme.Danger.Render("✗ "+m.err) + "\n\n" + theme.HelpDesc.Render("esc: back")
		} else {
			body = renderExecutionCard(theme, m.execution) + "\n\n" + theme.HelpDesc.Render("esc: back")
		}
	}

	return theme.Card.Width(width - 6).Render(body)
}

func renderWaveform(theme styles.Theme, bars []float64) string {
	const maxHeight = 6
	rows := make([][]rune, maxHeight)
	for r := range rows {
		rows[r] = make([]rune, len(bars))
	}
	for col, v := range bars {
		filled := int(v * float64(maxHeight))
		for r := 0; r < maxHeight; r++ {
			if maxHeight-r <= filled {
				rows[r][col] = '█'
			} else {
				rows[r][col] = ' '
			}
		}
	}
	lines := make([]string, maxHeight)
	for r, row := range rows {
		lines[r] = theme.Subtitle.Render(string(row))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
