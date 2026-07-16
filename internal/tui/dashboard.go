package tui

import (
	"fmt"
	"strings"

	"github.com/bolna-ai/bolna-cli/internal/api"
	"github.com/bolna-ai/bolna-cli/internal/config"
	"github.com/bolna-ai/bolna-cli/internal/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	scrAgents screen = iota
	scrAgentDetail
	scrCalls
	scrCallDetail
	scrNumbers
	scrBatches
	scrAccount
)

// Loaded-data messages. Each API call runs in its own tea.Cmd (real HTTP
// calls, never blocking Update) and reports back via one of these.
type agentsLoadedMsg struct {
	agents []api.AgentSummary
	err    error
}
type agentLoadedMsg struct {
	agent api.Agent
	err   error
}
type executionsLoadedMsg struct {
	page *api.ExecutionsPage
	err  error
}
type executionLoadedMsg struct {
	execution api.Execution
	err       error
}
type numbersLoadedMsg struct {
	numbers []api.PhoneNumber
	err     error
}
type batchesLoadedMsg struct {
	batches []api.Batch
	err     error
}
type accountLoadedMsg struct {
	info api.UserInfo
	err  error
}

func fetchAgents(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		agents, err := client.ListAgents(0, 0)
		return agentsLoadedMsg{agents: agents, err: err}
	}
}
func fetchAgent(client *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		agent, err := client.GetAgent(id)
		return agentLoadedMsg{agent: agent, err: err}
	}
}
func fetchExecutions(client *api.Client, agentID string) tea.Cmd {
	return func() tea.Msg {
		page, err := client.ListAgentExecutions(api.ListExecutionsInput{AgentID: agentID, PageSize: 50})
		return executionsLoadedMsg{page: page, err: err}
	}
}
func fetchExecution(client *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		execution, err := client.GetExecution(id)
		return executionLoadedMsg{execution: execution, err: err}
	}
}
func fetchNumbers(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		numbers, err := client.ListPhoneNumbers()
		return numbersLoadedMsg{numbers: numbers, err: err}
	}
}
func fetchBatches(client *api.Client, agentID string) tea.Cmd {
	return func() tea.Msg {
		batches, err := client.ListBatches(agentID, 0, 0)
		return batchesLoadedMsg{batches: batches, err: err}
	}
}
func fetchAccount(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		info, err := client.GetUserInfo()
		return accountLoadedMsg{info: info, err: err}
	}
}

type dashboard struct {
	client *api.Client
	theme  styles.Theme

	width, height int

	splash        splashModel
	showingSplash bool

	screen     screen
	backStack  []screen
	loading    bool
	loadingMsg string
	errMsg     string
	spinner    spinner.Model

	agents          []api.AgentSummary
	selectedAgent   api.Agent
	selectedAgentID string
	executions      *api.ExecutionsPage
	execution       api.Execution
	numbers         []api.PhoneNumber
	batches         []api.Batch
	account         api.UserInfo

	agentsTable table.Model
	callsTable  table.Model
	numberTable table.Model
	batchTable  table.Model
	detailView  viewport.Model

	paletteOpen bool
	palette     list.Model

	quitting bool
}

// RunDashboard launches the full-screen mission-control TUI: a splash
// slide-in, then a sidebar-navigated view over agents, calls, numbers,
// batches, and account — all read-only (writes go through the dedicated
// `bolna agents create/update/delete` and `bolna call start` commands, which
// have their own confirmation flows).
func RunDashboard(client *api.Client, theme styles.Theme) error {
	m := newDashboard(client, theme)
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func newDashboard(client *api.Client, theme styles.Theme) dashboard {
	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = theme.Subtitle

	mkTable := func(cols []table.Column) table.Model {
		t := table.New(table.WithColumns(cols), table.WithFocused(true))
		st := table.DefaultStyles()
		st.Header = st.Header.Foreground(theme.Palette.Primary).Bold(true)
		st.Selected = st.Selected.Foreground(theme.Palette.Text).Background(theme.Palette.Primary).Bold(true)
		t.SetStyles(st)
		return t
	}

	return dashboard{
		client:        client,
		theme:         theme,
		splash:        newSplashModel(theme),
		showingSplash: true,
		screen:        scrAgents,
		spinner:       sp,
		agentsTable:   mkTable([]table.Column{{Title: "NAME", Width: 24}, {Title: "STATUS", Width: 12}, {Title: "ID", Width: 24}}),
		callsTable:    mkTable([]table.Column{{Title: "STATUS", Width: 12}, {Title: "DURATION", Width: 10}, {Title: "TO", Width: 16}, {Title: "CREATED", Width: 22}}),
		numberTable:   mkTable([]table.Column{{Title: "NUMBER", Width: 16}, {Title: "PROVIDER", Width: 12}, {Title: "AGENT ID", Width: 24}}),
		batchTable:    mkTable([]table.Column{{Title: "BATCH ID", Width: 24}, {Title: "STATUS", Width: 12}, {Title: "SCHEDULED", Width: 22}}),
		detailView:    viewport.New(80, 20),
	}
}

func (m dashboard) Init() tea.Cmd {
	return tea.Batch(m.splash.Init(), fetchAccount(m.client), fetchAgents(m.client), tea.EnterAltScreen)
}

func (m dashboard) push(s screen) dashboard {
	m.backStack = append(m.backStack, m.screen)
	m.screen = s
	return m
}

func (m dashboard) pop() dashboard {
	if len(m.backStack) == 0 {
		return m
	}
	m.screen = m.backStack[len(m.backStack)-1]
	m.backStack = m.backStack[:len(m.backStack)-1]
	return m
}

func (m dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// -2/-2 for the outer border, -1/-1 for the header/footer rows.
		contentH := m.height - 8
		if contentH < 3 {
			contentH = 3
		}
		m.agentsTable.SetHeight(contentH)
		m.callsTable.SetHeight(contentH)
		m.numberTable.SetHeight(contentH)
		m.batchTable.SetHeight(contentH)
		m.detailView.Width = m.width - 6
		m.detailView.Height = contentH
		return m, nil

	case splashTickMsg:
		if m.showingSplash {
			var cmd tea.Cmd
			m.splash, cmd = m.splash.Update(msg)
			return m, cmd
		}
		return m, nil

	case splashDoneMsg:
		m.showingSplash = false
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case agentsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.agents = msg.agents
		rows := make([]table.Row, len(msg.agents))
		for i, ag := range msg.agents {
			rows[i] = table.Row{ag.AgentName, ag.AgentStatus, ag.ID}
		}
		m.agentsTable.SetRows(rows)
		return m, nil

	case agentLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.selectedAgent = msg.agent
		m.detailView.SetContent(renderAgentCard(m.theme, msg.agent))
		return m, nil

	case executionsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.executions = msg.page
		rows := make([]table.Row, len(msg.page.Data))
		for i, e := range msg.page.Data {
			to := ""
			if e.TelephonyData != nil {
				to = e.TelephonyData.ToNumber
			}
			rows[i] = table.Row{e.Status, fmtDuration(e.ConversationDuration), orDash(to), e.CreatedAt}
		}
		m.callsTable.SetRows(rows)
		return m, nil

	case executionLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.execution = msg.execution
		m.detailView.SetContent(renderExecutionCard(m.theme, msg.execution))
		return m, nil

	case numbersLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.numbers = msg.numbers
		rows := make([]table.Row, len(msg.numbers))
		for i, n := range msg.numbers {
			rows[i] = table.Row{n.PhoneNumber, n.TelephonyProvider, orDash(n.AgentID)}
		}
		m.numberTable.SetRows(rows)
		return m, nil

	case batchesLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.batches = msg.batches
		rows := make([]table.Row, len(msg.batches))
		for i, b := range msg.batches {
			scheduled := "—"
			if b.ScheduledAt != nil {
				scheduled = *b.ScheduledAt
			}
			rows[i] = table.Row{b.BatchID, b.Status, scheduled}
		}
		m.batchTable.SetRows(rows)
		return m, nil

	case accountLoadedMsg:
		if msg.err == nil {
			m.account = msg.info
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m dashboard) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showingSplash {
		m.showingSplash = false
		return m, nil
	}

	if m.paletteOpen {
		switch msg.String() {
		case "esc":
			m.paletteOpen = false
			return m, nil
		case "enter":
			m.paletteOpen = false
			if item, ok := m.palette.SelectedItem().(paletteItem); ok {
				return m.applyPaletteSelection(item)
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.palette, cmd = m.palette.Update(msg)
		return m, cmd
	}

	switch {
	case key.Matches(msg, keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, keys.Palette):
		items := buildPaletteItems(m.agents)
		m.palette = newPaletteList(m.theme, items, m.width-8, m.height-8)
		m.paletteOpen = true
		return m, nil
	case key.Matches(msg, keys.Theme):
		return m.cycleTheme()
	case key.Matches(msg, keys.Back):
		m = m.pop()
		return m, nil
	case key.Matches(msg, keys.Agents):
		m.screen = scrAgents
		return m, nil
	case key.Matches(msg, keys.Numbers):
		m.screen = scrNumbers
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, fetchNumbers(m.client))
	case key.Matches(msg, keys.Account):
		m.screen = scrAccount
		return m, nil
	case key.Matches(msg, keys.Refresh):
		return m.refreshCurrent()
	}

	switch m.screen {
	case scrAgents:
		if key.Matches(msg, keys.Select) {
			if row := m.agentsTable.SelectedRow(); len(row) > 0 {
				m.selectedAgentID = row[2]
				m.loading = true
				m = m.push(scrAgentDetail)
				return m, tea.Batch(m.spinner.Tick, fetchAgent(m.client, m.selectedAgentID))
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.agentsTable, cmd = m.agentsTable.Update(msg)
		return m, cmd

	case scrAgentDetail:
		switch {
		case key.Matches(msg, keys.Calls):
			m.loading = true
			m = m.push(scrCalls)
			return m, tea.Batch(m.spinner.Tick, fetchExecutions(m.client, m.selectedAgentID))
		case key.Matches(msg, keys.Batches):
			m.loading = true
			m = m.push(scrBatches)
			return m, tea.Batch(m.spinner.Tick, fetchBatches(m.client, m.selectedAgentID))
		}
		var cmd tea.Cmd
		m.detailView, cmd = m.detailView.Update(msg)
		return m, cmd

	case scrCalls:
		if key.Matches(msg, keys.Select) {
			if m.executions != nil {
				idx := m.callsTable.Cursor()
				if idx >= 0 && idx < len(m.executions.Data) {
					id := m.executions.Data[idx].ID
					m.loading = true
					m = m.push(scrCallDetail)
					return m, tea.Batch(m.spinner.Tick, fetchExecution(m.client, id))
				}
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.callsTable, cmd = m.callsTable.Update(msg)
		return m, cmd

	case scrCallDetail:
		var cmd tea.Cmd
		m.detailView, cmd = m.detailView.Update(msg)
		return m, cmd

	case scrNumbers:
		var cmd tea.Cmd
		m.numberTable, cmd = m.numberTable.Update(msg)
		return m, cmd

	case scrBatches:
		var cmd tea.Cmd
		m.batchTable, cmd = m.batchTable.Update(msg)
		return m, cmd

	case scrAccount:
		var cmd tea.Cmd
		m.detailView, cmd = m.detailView.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m dashboard) applyPaletteSelection(item paletteItem) (tea.Model, tea.Cmd) {
	switch item.action {
	case actionGoAgents:
		m.screen = scrAgents
		return m, nil
	case actionGoNumbers:
		m.screen = scrNumbers
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, fetchNumbers(m.client))
	case actionGoAccount:
		m.screen = scrAccount
		return m, nil
	case actionGoAgentDetail:
		m.selectedAgentID = item.agentID
		m.screen = scrAgentDetail
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, fetchAgent(m.client, item.agentID))
	}
	return m, nil
}

func (m dashboard) cycleTheme() (dashboard, tea.Cmd) {
	current := m.theme.Palette.Name
	idx := 0
	for i, p := range styles.Themes {
		if p.Name == current {
			idx = i
			break
		}
	}
	next := styles.Themes[(idx+1)%len(styles.Themes)]
	m.theme = styles.New(next)
	cfg, _ := config.Load()
	cfg.Theme = next.Name
	_ = config.Save(cfg)
	return m, nil
}

func (m dashboard) refreshCurrent() (dashboard, tea.Cmd) {
	m.loading = true
	switch m.screen {
	case scrAgents:
		return m, tea.Batch(m.spinner.Tick, fetchAgents(m.client))
	case scrAgentDetail:
		return m, tea.Batch(m.spinner.Tick, fetchAgent(m.client, m.selectedAgentID))
	case scrCalls:
		return m, tea.Batch(m.spinner.Tick, fetchExecutions(m.client, m.selectedAgentID))
	case scrNumbers:
		return m, tea.Batch(m.spinner.Tick, fetchNumbers(m.client))
	case scrBatches:
		return m, tea.Batch(m.spinner.Tick, fetchBatches(m.client, m.selectedAgentID))
	}
	m.loading = false
	return m, nil
}

func (m dashboard) View() string {
	if m.quitting {
		return ""
	}
	if m.showingSplash {
		return m.splash.View(m.width, m.height)
	}
	if m.width == 0 {
		return "loading…"
	}

	innerWidth := m.width - 2
	innerHeight := m.height - 2

	header := m.renderHeader(innerWidth)
	footer := m.renderFooter()

	var body string
	if m.paletteOpen {
		body = m.palette.View()
	} else {
		body = m.renderScreen()
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	// The outer frame is bolna-cli's one signature visual: a border in the
	// same steel-blue → periwinkle brand color as mcp.bolna.ai's wordmark,
	// with a small badge chip standing in for a logo mark in the header.
	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Palette.Primary).
		Width(innerWidth).
		Height(innerHeight)
	return frame.Render(inner)
}

func (m dashboard) renderHeader(width int) string {
	badge := m.theme.Badge.Render(" bolna ")
	title := badge + " " + m.theme.Muted.Render(breadcrumb(m.screen))
	var right string
	if bal, ok := m.account.Balance(); ok {
		right = m.theme.Success.Render(fmt.Sprintf("$%.2f", bal))
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, title, strings.Repeat(" ", max(1, width-lipgloss.Width(title)-lipgloss.Width(right)-2)), right)
	return m.theme.StatusBar.Width(width).Render(bar)
}

func breadcrumb(s screen) string {
	switch s {
	case scrAgents:
		return "· agents"
	case scrAgentDetail:
		return "· agents · detail"
	case scrCalls:
		return "· agents · calls"
	case scrCallDetail:
		return "· agents · calls · transcript"
	case scrNumbers:
		return "· numbers"
	case scrBatches:
		return "· agents · batches"
	case scrAccount:
		return "· account"
	}
	return ""
}

func (m dashboard) renderFooter() string {
	if m.errMsg != "" {
		return m.theme.Danger.Render(" " + m.errMsg)
	}
	if m.loading {
		return " " + m.spinner.View() + m.theme.Muted.Render(" loading…")
	}
	return " " + m.theme.HelpDesc.Render(strings.Join(helpLine(screenHint(m.screen)), "  •  "))
}

func screenHint(s screen) string {
	switch s {
	case scrAgents:
		return "enter: view agent"
	case scrAgentDetail:
		return "c: calls  •  b: batches"
	case scrCalls:
		return "enter: transcript"
	default:
		return ""
	}
}

func (m dashboard) renderScreen() string {
	switch m.screen {
	case scrAgents:
		if len(m.agents) == 0 && !m.loading {
			return m.theme.Muted.Render("No agents yet. Create one with `bolna agents create`.")
		}
		return m.agentsTable.View()
	case scrAgentDetail:
		return m.detailView.View()
	case scrCalls:
		return m.callsTable.View()
	case scrCallDetail:
		return m.detailView.View()
	case scrNumbers:
		return m.numberTable.View()
	case scrBatches:
		return m.batchTable.View()
	case scrAccount:
		return m.theme.Card.Render(renderAccountCard(m.theme, m.account))
	}
	return ""
}

func renderAgentCard(theme styles.Theme, agent api.Agent) string {
	header := theme.Title.Render(orDash(agent.Name())) + "  " + theme.StatusColor(agent.Status()).Render(orDash(agent.Status()))
	body := header + "\n" + theme.Muted.Render("ID: "+agent.ID())
	if wm := agent.WelcomeMessage(); wm != "" {
		body += "\n\n" + theme.Subtitle.Render("Welcome message") + "\n" + wm
	}
	if sp := agent.SystemPrompt(); sp != "" {
		if rendered, err := glamour.Render("**System prompt**\n\n"+sp, "auto"); err == nil {
			body += "\n" + rendered
		} else {
			body += "\n\n" + theme.Subtitle.Render("System prompt") + "\n" + sp
		}
	}
	return body
}

func renderExecutionCard(theme styles.Theme, execution api.Execution) string {
	header := theme.Title.Render(execution.ID()) + "  " + theme.StatusColor(execution.Status()).Render(orDash(execution.Status()))
	body := header
	if transcript := execution.Transcript(); transcript != "" {
		if rendered, err := glamour.Render("**Transcript**\n\n"+transcript, "auto"); err == nil {
			body += "\n" + rendered
		} else {
			body += "\n\n" + theme.Subtitle.Render("Transcript") + "\n" + transcript
		}
	} else {
		body += "\n\n" + theme.Muted.Render("(no transcript on this execution)")
	}
	return body
}

func renderAccountCard(theme styles.Theme, info api.UserInfo) string {
	if info == nil {
		return theme.Muted.Render("Loading account info…")
	}
	var lines []string
	if name := info.Name(); name != "" {
		lines = append(lines, theme.Bold.Render(name))
	}
	if email := info.Email(); email != "" {
		lines = append(lines, theme.Muted.Render(email))
	}
	if bal, ok := info.Balance(); ok {
		lines = append(lines, fmt.Sprintf("%s %s", theme.Subtitle.Render("Wallet balance:"), theme.Success.Render(fmt.Sprintf("$%.2f", bal))))
	}
	if current, max, ok := info.Concurrency(); ok {
		lines = append(lines, fmt.Sprintf("%s %d/%d", theme.Subtitle.Render("Concurrency:"), current, max))
	}
	if len(lines) == 0 {
		lines = append(lines, theme.Muted.Render("(no displayable fields)"))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func fmtDuration(seconds *float64) string {
	if seconds == nil {
		return "—"
	}
	total := int(*seconds)
	return fmt.Sprintf("%dm%02ds", total/60, total%60)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
