package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/matt/swarm-cli/internal/process"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
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

The dashboard updates automatically and allows quick actions on agents
like pausing, resuming, or killing them.

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

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)
)

type tickMsg time.Time

type topModel struct {
	mgr      *state.Manager
	agents   []*state.AgentState
	cursor   int
	width    int
	height   int
	showAll  bool
	global   bool
	interval time.Duration
	err      error
}

func initialTopModel() topModel {
	s := GetScope()
	global := s == scope.ScopeGlobal

	mgr, err := state.NewManagerWithScope(s, "")

	return topModel{
		mgr:      mgr,
		cursor:   0,
		showAll:  topAll,
		global:   global,
		interval: topInterval,
		err:      err,
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

func (m topModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.agents)-1 {
				m.cursor++
			}
		case "p":
			return m, m.pauseSelected()
		case "r":
			return m, m.resumeSelected()
		case "+":
			return m, m.increaseIterations()
		case "-":
			return m, m.decreaseIterations()
		case "K":
			return m, m.killSelected()
		case "l":
			return m, m.viewLogs()
		case "enter":
			return m, m.attachSelected()
		case "a":
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
			return m, m.refreshAgentsCmd()
		}

	case []*state.AgentState:
		m.agents = msg
		if m.cursor >= len(m.agents) && len(m.agents) > 0 {
			m.cursor = len(m.agents) - 1
		}

	case tickMsg:
		return m, tea.Batch(m.refreshAgentsCmd(), m.tickCmd())

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case error:
		m.err = msg
	}

	return m, nil
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

	// Detail panel for selected agent
	if len(m.agents) > 0 && m.cursor < len(m.agents) {
		b.WriteString(m.renderDetail(m.agents[m.cursor]))
		b.WriteString("\n")
	}

	// Help line
	b.WriteString(m.renderHelp())

	return b.String()
}

func (m topModel) renderHeader() string {
	running, paused, terminated := 0, 0, 0
	for _, a := range m.agents {
		switch {
		case a.Status == "terminated":
			terminated++
		case a.Paused:
			paused++
		default:
			running++
		}
	}

	scopeStr := "project"
	if m.global {
		scopeStr = "global"
	}

	allIndicator := ""
	if m.showAll {
		allIndicator = " +all"
	}

	header := fmt.Sprintf(
		"╭─ Swarm Top (%s%s) ─────────────────────────────────────────────────────╮\n"+
			"│  Running: %s   Paused: %s   Terminated: %s   │   Refresh: %s   │   [q]uit  │\n"+
			"╰────────────────────────────────────────────────────────────────────────────────╯",
		scopeStr, allIndicator,
		runningStyle.Render(fmt.Sprintf("%d", running)),
		pausedStyle.Render(fmt.Sprintf("%d", paused)),
		terminatedStyle.Render(fmt.Sprintf("%d", terminated)),
		m.interval,
	)

	return headerStyle.Render(header)
}

func (m topModel) renderTable() string {
	if len(m.agents) == 0 {
		return dimStyle.Render("  No agents found. Start one with: swarm run -p <prompt>")
	}

	var b strings.Builder

	// Header
	b.WriteString(dimStyle.Render("  ID        NAME             PROMPT           MODEL              ITER    STATUS      RUNTIME\n"))
	b.WriteString(dimStyle.Render("  ─────────────────────────────────────────────────────────────────────────────────────────────\n"))

	for i, a := range m.agents {
		prefix := "  "
		if i == m.cursor {
			prefix = "▸ "
		}

		name := a.Name
		if name == "" {
			name = "-"
		}

		statusStr, statusStyle := getStatusDisplay(a)

		runtime := time.Since(a.StartedAt).Round(time.Second)

		iterStr := fmt.Sprintf("%2d/%-3d", a.CurrentIter, a.Iterations)
		if a.Iterations == 0 {
			iterStr = fmt.Sprintf("%2d/∞  ", a.CurrentIter)
		}

		line := fmt.Sprintf("%s%-8s  %-15s  %-15s  %-17s  %s  %s  %s\n",
			prefix,
			truncateTop(a.ID, 8),
			truncateTop(name, 15),
			truncateTop(a.Prompt, 15),
			truncateTop(a.Model, 17),
			iterStr,
			statusStyle.Render(fmt.Sprintf("%-10s", statusStr)),
			formatTopDuration(runtime),
		)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(line)
		}
	}

	return b.String()
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

func (m topModel) renderDetail(a *state.AgentState) string {
	name := a.Name
	if name == "" {
		name = a.ID
	}

	workDir := a.WorkingDir
	if len(workDir) > 60 {
		workDir = "..." + workDir[len(workDir)-57:]
	}

	logFile := a.LogFile
	if logFile == "" {
		logFile = "-"
	} else if len(logFile) > 60 {
		logFile = "..." + logFile[len(logFile)-57:]
	}

	detail := fmt.Sprintf(
		"╭─ Selected: %s (%s) ─────────────────────────────────────────────────╮\n"+
			"│  Started: %-62s │\n"+
			"│  Directory: %-60s │\n"+
			"│  Log: %-66s │\n"+
			"╰────────────────────────────────────────────────────────────────────────────────╯",
		a.ID, name,
		a.StartedAt.Format("2006-01-02 15:04:05"),
		workDir,
		logFile,
	)

	return dimStyle.Render(detail)
}

func (m topModel) renderHelp() string {
	return dimStyle.Render("Keys: [↑/↓] select  [p]ause  [r]esume  [+/-] iterations  [K]ill  [l]ogs  [Enter] attach  [a]ll  [g]lobal  [q]uit")
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
		agent.Paused = true
		now := time.Now()
		agent.PausedAt = &now
		m.mgr.Update(agent)
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
		agent.Paused = false
		agent.PausedAt = nil
		m.mgr.Update(agent)
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
		agent.Iterations++
		m.mgr.Update(agent)
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
			agent.Iterations--
			m.mgr.Update(agent)
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
		agent.TerminateMode = "immediate"
		m.mgr.Update(agent)
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
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
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
