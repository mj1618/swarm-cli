package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mj1618/swarm-cli/internal/config"
	"github.com/mj1618/swarm-cli/internal/logparser"
	"github.com/mj1618/swarm-cli/internal/process"
	"github.com/mj1618/swarm-cli/internal/scope"
	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	topInterval time.Duration
	topAll      bool
)

var topCmd = &cobra.Command{
	Use:   "top",
	Short: "Real-time agent monitoring dashboard",
	Long: `Display a real-time TUI dashboard showing all running agents.

The dashboard shows agent status, iterations, token usage, costs, current task,
and optionally streaming logs for the selected agent.

Use arrow keys or j/k to navigate between agents. Press Enter to attach
to the selected agent, or use keyboard shortcuts for quick actions.`,
	Example: `  # Monitor agents in current project
  swarm top

  # Monitor all agents globally
  swarm top --global

  # Include terminated agents
  swarm top --all

  # Faster refresh rate
  swarm top --interval 1s`,
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(initialTopModel(), tea.WithAltScreen())
		_, err := p.Run()
		return err
	},
}

// Styles for the TUI
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212"))

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229"))

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	pausedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	terminatedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	logPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63"))

	logHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))

	costStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("220"))

	tokenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	taskStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("147"))
)

type tickMsg time.Time
type logLineMsg string
type logLinesMsg []string

type topModel struct {
	mgr           *state.Manager
	cfg           *config.Config
	agents        []*state.AgentState
	cursor        int
	width         int
	height        int
	showAll       bool
	global        bool
	interval      time.Duration
	err           error
	showLogs      bool
	logLines      []string
	maxLogLines   int
	logWatcherID  string // ID of agent whose logs we're watching
	logFile       *os.File
	logFileReader *bufio.Reader
}

func initialTopModel() topModel {
	s := GetScope()
	global := s == scope.ScopeGlobal

	mgr, err := state.NewManagerWithScope(s, "")
	cfg, _ := config.Load()

	return topModel{
		mgr:         mgr,
		cfg:         cfg,
		cursor:      0,
		showAll:     topAll,
		global:      global,
		interval:    topInterval,
		err:         err,
		showLogs:    true,
		logLines:    make([]string, 0),
		maxLogLines: 15,
	}
}

func (m topModel) Init() tea.Cmd {
	return tea.Batch(
		m.refreshAgentsCmd(),
		m.tickCmd(),
	)
}

func (m topModel) tickCmd() tea.Cmd {
	return tea.Tick(m.interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m topModel) refreshAgentsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.mgr == nil {
			return nil
		}

		agents, err := m.mgr.List(!m.showAll)
		if err != nil {
			return err
		}

		// Sort: running > paused > terminated, then by start time (newest first within category)
		sort.Slice(agents, func(i, j int) bool {
			orderI := getStatusOrder(agents[i])
			orderJ := getStatusOrder(agents[j])
			if orderI != orderJ {
				return orderI < orderJ
			}
			return agents[i].StartedAt.After(agents[j].StartedAt)
		})

		return agents
	}
}

func getStatusOrder(a *state.AgentState) int {
	if a.Status == "terminated" {
		return 2
	}
	if a.Paused {
		return 1
	}
	return 0
}

// readNewLogLines reads any new lines from the log file
func (m *topModel) readNewLogLines() tea.Cmd {
	return func() tea.Msg {
		if m.logFile == nil || m.logFileReader == nil {
			return nil
		}

		var newLines []string
		for {
			line, err := m.logFileReader.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			// Parse and format the line
			formatted := formatLogLine(line)
			if formatted != "" {
				newLines = append(newLines, formatted)
			}
		}

		if len(newLines) > 0 {
			return logLinesMsg(newLines)
		}
		return nil
	}
}

// formatLogLine formats a JSON log line for display
func formatLogLine(line string) string {
	event := logparser.ParseEvent(line)
	if event == nil {
		// Not JSON, return as-is (trimmed)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return ""
		}
		return trimmed
	}

	// Format based on event type
	switch event.Type {
	case "assistant":
		if event.Message != nil {
			var texts []string
			for _, c := range event.Message.Content {
				if c.Text != "" {
					texts = append(texts, c.Text)
				}
			}
			text := strings.Join(texts, " ")
			if len(text) > 100 {
				text = text[:97] + "..."
			}
			if text != "" {
				return "[assistant] " + text
			}
		}
		return ""
	case "thinking":
		text := event.Text
		if len(text) > 100 {
			text = text[:97] + "..."
		}
		if text != "" {
			return "[thinking] " + text
		}
		return ""
	case "tool_call":
		return "[tool] " + summarizeToolCallShort(event)
	case "result":
		result := event.Result
		if len(result) > 80 {
			result = result[:77] + "..."
		}
		return "[result] " + result
	case "system":
		if event.Subtype == "init" {
			return "[system] Initialized"
		}
		return "[system] " + event.Subtype
	default:
		if event.Type != "" {
			return "[" + event.Type + "]"
		}
	}
	return ""
}

// summarizeToolCallShort creates a short summary of a tool call
func summarizeToolCallShort(event *logparser.LogEvent) string {
	if event.ToolCall == nil {
		return "tool call"
	}

	var toolName string
	var inner map[string]interface{}
	for k, v := range event.ToolCall {
		toolName = k
		if m, ok := v.(map[string]interface{}); ok {
			inner = m
		}
		break
	}

	if toolName == "" {
		return "tool call"
	}

	var args map[string]interface{}
	if inner != nil {
		if a, ok := inner["args"].(map[string]interface{}); ok {
			args = a
		}
	}

	getArg := func(keys ...string) string {
		if args == nil {
			return ""
		}
		for _, key := range keys {
			if v, ok := args[key]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
		}
		return ""
	}

	switch toolName {
	case "shellToolCall":
		cmd := getArg("command", "simpleCommand")
		if len(cmd) > 40 {
			cmd = cmd[:37] + "..."
		}
		return "Shell: " + cmd
	case "readToolCall":
		path := getArg("file_path", "path")
		if len(path) > 40 {
			parts := strings.Split(path, "/")
			path = parts[len(parts)-1]
		}
		return "Read: " + path
	case "writeToolCall":
		path := getArg("file_path", "path")
		if len(path) > 40 {
			parts := strings.Split(path, "/")
			path = parts[len(parts)-1]
		}
		return "Write: " + path
	case "lsToolCall":
		path := getArg("path")
		return "List: " + path
	default:
		name := strings.TrimSuffix(toolName, "ToolCall")
		return name
	}
}

func (m topModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.closeLogFile()
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.switchLogFile()
			}
		case "down", "j":
			if m.cursor < len(m.agents)-1 {
				m.cursor++
				m.switchLogFile()
			}
		case "p":
			return m, m.pauseSelected()
		case "r":
			return m, m.resumeSelected()
		case "+", "=":
			return m, m.increaseIterations()
		case "-":
			return m, m.decreaseIterations()
		case "K", "shift+k":
			return m, m.killSelected()
		case "L", "shift+l":
			return m, m.viewLogs()
		case "l":
			m.showLogs = !m.showLogs
			if m.showLogs {
				m.switchLogFile()
			} else {
				m.closeLogFile()
			}
		case "enter", "a":
			return m, m.attachSelected()
		case "A", "shift+a":
			m.showAll = !m.showAll
			return m, m.refreshAgentsCmd()
		case "g":
			m.global = !m.global
			// Recreate manager with new scope
			s := scope.ScopeProject
			if m.global {
				s = scope.ScopeGlobal
			}
			mgr, _ := state.NewManagerWithScope(s, "")
			m.mgr = mgr
			m.cursor = 0
			m.closeLogFile()
			return m, m.refreshAgentsCmd()
		}

	case []*state.AgentState:
		m.agents = msg
		if m.cursor >= len(m.agents) && len(m.agents) > 0 {
			m.cursor = len(m.agents) - 1
		}
		// Update log file if selected agent changed
		if m.showLogs && len(m.agents) > 0 && m.cursor < len(m.agents) {
			if m.logWatcherID != m.agents[m.cursor].ID {
				m.switchLogFile()
			}
		}

	case tickMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, m.refreshAgentsCmd(), m.tickCmd())
		if m.showLogs && m.logFile != nil {
			cmds = append(cmds, m.readNewLogLines())
		}
		return m, tea.Batch(cmds...)

	case logLinesMsg:
		for _, line := range msg {
			m.logLines = append(m.logLines, line)
		}
		// Trim to max lines
		if len(m.logLines) > m.maxLogLines*2 {
			m.logLines = m.logLines[len(m.logLines)-m.maxLogLines:]
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust max log lines based on height
		if m.height > 30 {
			m.maxLogLines = (m.height - 20) / 2
		} else {
			m.maxLogLines = 8
		}

	case error:
		m.err = msg
	}

	return m, nil
}

func (m *topModel) closeLogFile() {
	if m.logFile != nil {
		m.logFile.Close()
		m.logFile = nil
		m.logFileReader = nil
		m.logWatcherID = ""
	}
}

func (m *topModel) switchLogFile() {
	m.closeLogFile()
	m.logLines = nil

	if !m.showLogs || len(m.agents) == 0 || m.cursor >= len(m.agents) {
		return
	}

	agent := m.agents[m.cursor]
	if agent.LogFile == "" {
		return
	}

	file, err := os.Open(agent.LogFile)
	if err != nil {
		return
	}

	// Seek to near end of file to show recent logs
	stat, err := file.Stat()
	if err == nil && stat.Size() > 8192 {
		file.Seek(-8192, io.SeekEnd)
		// Skip partial line
		reader := bufio.NewReader(file)
		reader.ReadString('\n')
		m.logFileReader = reader
	} else {
		m.logFileReader = bufio.NewReader(file)
	}

	m.logFile = file
	m.logWatcherID = agent.ID

	// Read initial lines
	for i := 0; i < m.maxLogLines*2; i++ {
		line, err := m.logFileReader.ReadString('\n')
		if err != nil {
			break
		}
		formatted := formatLogLine(line)
		if formatted != "" {
			m.logLines = append(m.logLines, formatted)
		}
	}

	// Keep only recent lines
	if len(m.logLines) > m.maxLogLines {
		m.logLines = m.logLines[len(m.logLines)-m.maxLogLines:]
	}
}

func (m topModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Agent table
	b.WriteString(m.renderTable())
	b.WriteString("\n")

	// Log panel (if enabled)
	if m.showLogs && len(m.agents) > 0 && m.cursor < len(m.agents) {
		b.WriteString(m.renderLogPanel())
		b.WriteString("\n")
	}

	// Help line
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m topModel) renderHeader() string {
	running, paused, terminated := 0, 0, 0
	var totalTokens int64
	var totalCost float64

	for _, a := range m.agents {
		switch {
		case a.Status == "terminated":
			terminated++
		case a.Paused:
			paused++
		default:
			running++
		}
		totalTokens += a.InputTokens + a.OutputTokens
		totalCost += a.TotalCost
	}

	scopeStr := "project"
	if m.global {
		scopeStr = "global"
	}

	allIndicator := ""
	if m.showAll {
		allIndicator = " +all"
	}

	tokensStr := formatTokenCount(totalTokens)
	costStr := fmt.Sprintf("$%.2f", totalCost)

	// Build content line without box characters first
	title := fmt.Sprintf(" Swarm Dashboard (%s%s) ", scopeStr, allIndicator)
	stats := fmt.Sprintf("  Running: %s   Paused: %s   Terminated: %s   Tokens: %s   Cost: %s  ",
		runningStyle.Render(fmt.Sprintf("%d", running)),
		pausedStyle.Render(fmt.Sprintf("%d", paused)),
		terminatedStyle.Render(fmt.Sprintf("%d", terminated)),
		tokenStyle.Render(tokensStr),
		costStyle.Render(costStr),
	)

	// Calculate visual width (accounting for ANSI codes)
	statsVisualWidth := lipgloss.Width(stats)
	titleVisualWidth := lipgloss.Width(title)

	// Use the wider of the two, with a minimum
	boxWidth := statsVisualWidth
	if titleVisualWidth > boxWidth {
		boxWidth = titleVisualWidth
	}
	if boxWidth < 60 {
		boxWidth = 60
	}

	// Build the header with proper alignment
	var b strings.Builder

	// Top border with title
	b.WriteString("╭─")
	b.WriteString(title)
	remaining := boxWidth - titleVisualWidth
	if remaining > 0 {
		b.WriteString(strings.Repeat("─", remaining))
	}
	b.WriteString("─╮\n")

	// Stats line
	b.WriteString("│")
	b.WriteString(stats)
	padding := boxWidth - statsVisualWidth + 1
	if padding > 0 {
		b.WriteString(strings.Repeat(" ", padding))
	}
	b.WriteString("│\n")

	// Bottom border
	b.WriteString("╰")
	b.WriteString(strings.Repeat("─", boxWidth+2))
	b.WriteString("╯")

	return headerStyle.Render(b.String())
}

func (m topModel) renderTable() string {
	if len(m.agents) == 0 {
		return dimStyle.Render("  No agents found. Start one with: swarm run -p <prompt>")
	}

	var b strings.Builder

	// Column widths
	const (
		colID     = 8
		colName   = 14
		colParent = 10
		colStatus = 10
		colIter   = 7
		colTokens = 8
		colCost   = 7
		colTask   = 30
	)

	// Header - build with exact spacing
	header := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s %-*s %-*s %s",
		colID, "ID",
		colName, "NAME",
		colParent, "PARENT",
		colStatus, "STATUS",
		colIter, "ITER",
		colTokens, "TOKENS",
		colCost, "COST",
		"TASK",
	)
	b.WriteString(dimStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  " + strings.Repeat("─", colID+colName+colParent+colStatus+colIter+colTokens+colCost+colTask+12)))
	b.WriteString("\n")

	for i, a := range m.agents {
		prefix := "  "
		if i == m.cursor {
			prefix = "▸ "
		}

		name := a.Name
		if name == "" {
			name = "-"
		}

		parent := a.ParentID
		if parent == "" {
			parent = "-"
		}

		statusStr, statusSty := getStatusDisplay(a)

		iterStr := fmt.Sprintf("%d/%d", a.CurrentIter, a.Iterations)
		if a.Iterations == 0 {
			iterStr = fmt.Sprintf("%d/∞", a.CurrentIter)
		}

		tokens := a.InputTokens + a.OutputTokens
		tokensStr := formatTokenCount(tokens)

		costStr := fmt.Sprintf("$%.2f", a.TotalCost)

		task := a.CurrentTask
		if task == "" {
			task = "-"
		}
		if len(task) > colTask {
			task = task[:colTask-3] + "..."
		}

		// Build line with proper padding for each column
		// Apply style to content, then pad to column width
		var line strings.Builder
		line.WriteString(prefix)
		line.WriteString(padRight(truncateTop(a.ID, colID-1), colID))
		line.WriteString(" ")
		line.WriteString(padRight(truncateTop(name, colName-1), colName))
		line.WriteString(" ")
		line.WriteString(padRight(truncateTop(parent, colParent-1), colParent))
		line.WriteString(" ")
		line.WriteString(statusSty.Render(padRight(statusStr, colStatus)))
		line.WriteString(" ")
		line.WriteString(padRight(iterStr, colIter))
		line.WriteString(" ")
		line.WriteString(tokenStyle.Render(padLeft(tokensStr, colTokens)))
		line.WriteString(" ")
		line.WriteString(costStyle.Render(padLeft(costStr, colCost)))
		line.WriteString(" ")
		line.WriteString(taskStyle.Render(task))

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(line.String()))
		} else {
			b.WriteString(line.String())
		}
		b.WriteString("\n")
	}

	return b.String()
}

func padRight(s string, width int) string {
	visualWidth := lipgloss.Width(s)
	if visualWidth >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visualWidth)
}

func padLeft(s string, width int) string {
	visualWidth := lipgloss.Width(s)
	if visualWidth >= width {
		return s
	}
	return strings.Repeat(" ", width-visualWidth) + s
}

func getStatusDisplay(a *state.AgentState) (string, lipgloss.Style) {
	switch {
	case a.Status == "terminated":
		return "terminated", terminatedStyle
	case a.Paused && a.PausedAt != nil:
		return "paused", pausedStyle
	case a.Paused:
		return "pausing", pausedStyle
	default:
		return "running", runningStyle
	}
}

func (m topModel) renderLogPanel() string {
	agent := m.agents[m.cursor]
	name := agent.Name
	if name == "" {
		name = agent.ID
	}

	// Determine panel width
	width := 90
	if m.width > 0 && m.width < 100 {
		width = m.width - 4
	}
	if width < 50 {
		width = 50
	}
	innerWidth := width - 4 // Account for "│ " and " │"

	var b strings.Builder

	// Header line with title
	title := fmt.Sprintf(" Logs: %s (%s) ", truncateTop(name, 20), truncateTop(agent.ID, 8))
	titleLen := len(title)
	b.WriteString("╭─")
	b.WriteString(logHeaderStyle.Render(title))
	remaining := width - titleLen - 3
	if remaining > 0 {
		b.WriteString(strings.Repeat("─", remaining))
	}
	b.WriteString("╮\n")

	// Log content
	if len(m.logLines) == 0 {
		var msg string
		if agent.LogFile == "" {
			msg = "No log file (agent not detached)"
		} else {
			msg = "Waiting for log output..."
		}
		b.WriteString("│ ")
		b.WriteString(dimStyle.Render(msg))
		padding := innerWidth - len(msg)
		if padding > 0 {
			b.WriteString(strings.Repeat(" ", padding))
		}
		b.WriteString(" │\n")
	} else {
		displayLines := m.logLines
		if len(displayLines) > m.maxLogLines {
			displayLines = displayLines[len(displayLines)-m.maxLogLines:]
		}
		for _, line := range displayLines {
			// Truncate line to fit inner width
			displayLine := line
			if len(displayLine) > innerWidth {
				displayLine = displayLine[:innerWidth-3] + "..."
			}
			b.WriteString("│ ")
			b.WriteString(displayLine)
			padding := innerWidth - len(displayLine)
			if padding > 0 {
				b.WriteString(strings.Repeat(" ", padding))
			}
			b.WriteString(" │\n")
		}
	}

	// Bottom border
	b.WriteString("╰")
	b.WriteString(strings.Repeat("─", width-2))
	b.WriteString("╯")

	return b.String()
}

func (m topModel) renderHelp() string {
	logsToggle := "[l] show logs"
	if m.showLogs {
		logsToggle = "[l] hide logs"
	}
	return dimStyle.Render(fmt.Sprintf("Keys: [↑/↓] select  [p]ause  [r]esume  [=/-] iter  [K]ill  [a]ttach  %s  [A]ll  [g]lobal  [q]uit", logsToggle))
}

// Action commands

func (m topModel) pauseSelected() tea.Cmd {
	return func() tea.Msg {
		if m.cursor >= len(m.agents) {
			return nil
		}
		agent := m.agents[m.cursor]
		if agent.Status != "running" || agent.Paused {
			return nil
		}
		m.mgr.SetPaused(agent.ID, true)
		return m.refreshAgentsCmd()()
	}
}

func (m topModel) resumeSelected() tea.Cmd {
	return func() tea.Msg {
		if m.cursor >= len(m.agents) {
			return nil
		}
		agent := m.agents[m.cursor]
		if !agent.Paused {
			return nil
		}
		m.mgr.SetPaused(agent.ID, false)
		return m.refreshAgentsCmd()()
	}
}

func (m topModel) increaseIterations() tea.Cmd {
	return func() tea.Msg {
		if m.cursor >= len(m.agents) {
			return nil
		}
		agent := m.agents[m.cursor]
		if agent.Status == "terminated" {
			return nil
		}
		m.mgr.SetIterations(agent.ID, agent.Iterations+1)
		return m.refreshAgentsCmd()()
	}
}

func (m topModel) decreaseIterations() tea.Cmd {
	return func() tea.Msg {
		if m.cursor >= len(m.agents) {
			return nil
		}
		agent := m.agents[m.cursor]
		if agent.Status == "terminated" {
			return nil
		}
		if agent.Iterations > agent.CurrentIter && agent.Iterations > 0 {
			m.mgr.SetIterations(agent.ID, agent.Iterations-1)
		}
		return m.refreshAgentsCmd()()
	}
}

func (m topModel) killSelected() tea.Cmd {
	return func() tea.Msg {
		if m.cursor >= len(m.agents) {
			return nil
		}
		agent := m.agents[m.cursor]
		if agent.Status == "terminated" {
			return nil
		}
		m.mgr.SetTerminateMode(agent.ID, "immediate")
		// Send kill signal
		process.Kill(agent.PID)
		return m.refreshAgentsCmd()()
	}
}

func (m topModel) viewLogs() tea.Cmd {
	if m.cursor >= len(m.agents) {
		return nil
	}
	agent := m.agents[m.cursor]
	if agent.LogFile == "" {
		return nil
	}

	// Check if log file exists
	if _, err := os.Stat(agent.LogFile); os.IsNotExist(err) {
		return nil
	}

	// Open logs in less with follow mode
	c := exec.Command("less", "+F", agent.LogFile)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return m.refreshAgentsCmd()()
	})
}

func (m topModel) attachSelected() tea.Cmd {
	if m.cursor >= len(m.agents) {
		return nil
	}
	agent := m.agents[m.cursor]
	if agent.Status == "terminated" {
		return nil
	}

	// Use swarm attach command
	c := exec.Command("swarm", "attach", agent.ID)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return m.refreshAgentsCmd()()
	})
}

// Helpers

func truncateTop(s string, max int) string {
	visualWidth := lipgloss.Width(s)
	if visualWidth <= max {
		return s
	}
	// Truncate rune by rune until we fit
	runes := []rune(s)
	for len(runes) > 0 {
		truncated := string(runes[:len(runes)-1]) + "..."
		if lipgloss.Width(truncated) <= max {
			return truncated
		}
		runes = runes[:len(runes)-1]
	}
	return "..."
}

func formatTokenCount(tokens int64) string {
	if tokens == 0 {
		return "-"
	}
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000)
	}
	if tokens >= 1000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1000)
	}
	return fmt.Sprintf("%d", tokens)
}

func formatTopDuration(d time.Duration) string {
	h := int(d.Hours())
	min := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, min)
	}
	if min > 0 {
		return fmt.Sprintf("%dm %ds", min, s)
	}
	return fmt.Sprintf("%ds", s)
}

func init() {
	topCmd.Flags().DurationVarP(&topInterval, "interval", "i", 2*time.Second, "Refresh interval")
	topCmd.Flags().BoolVarP(&topAll, "all", "a", false, "Show all agents including terminated")
}
