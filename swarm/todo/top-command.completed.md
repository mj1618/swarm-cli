# Add `swarm top` command for real-time agent monitoring dashboard

## Problem

Currently, users who want to monitor multiple running agents have limited options:

1. **`swarm list`** - Shows a snapshot but doesn't update in real-time. Users must repeatedly run it to see changes.
2. **`swarm attach`** (pending) - Focuses on a single agent, can't monitor multiple agents at once.
3. **`swarm logs -f`** - Follows a single agent's logs, no overview of all agents.

For users running several agents simultaneously (e.g., via `swarm up` with a compose file), there's no way to get a real-time dashboard view of all agents' progress.

Common workflow pain points:
```bash
# Currently: repeatedly run list to check on agents
watch -n 5 swarm list   # External workaround, clunky output

# Or check each agent individually
swarm inspect agent1
swarm inspect agent2
swarm inspect agent3
```

## Solution

Add a `swarm top` command that provides a real-time TUI dashboard showing all running agents with live updates, similar to `htop` for processes or `docker stats` for containers.

### Proposed API

```bash
# Show all running agents in current project
swarm top

# Show all agents globally
swarm top --global

# Include terminated agents
swarm top --all

# Custom refresh interval (default: 2s)
swarm top --interval 1s
```

### Dashboard layout

```
╭─ Swarm Top ──────────────────────────────────────────────────────────────────╮
│  Running: 3   Paused: 1   Terminated: 2   │   Refresh: 2s   │   [q]uit       │
╰──────────────────────────────────────────────────────────────────────────────╯

  ID        NAME             PROMPT           MODEL              ITER    STATUS    RUNTIME
  ─────────────────────────────────────────────────────────────────────────────────────────
▸ abc123    frontend-work    frontend-task    claude-sonnet-4    8/20    running   12m 34s
  def456    backend-api      backend-task     claude-opus-4      3/10    running    5m 12s
  ghi789    test-runner      run-tests        claude-sonnet-4    5/5     paused     8m 45s
  jkl012    cleanup          cleanup-task     claude-sonnet-4    1/1     terminated 2m 10s

╭─ Selected: abc123 (frontend-work) ───────────────────────────────────────────╮
│  Started: 2026-01-28 14:32:05                                                │
│  Directory: /Users/matt/projects/frontend                                    │
│  Log: ~/.swarm/logs/abc123.log                                               │
╰──────────────────────────────────────────────────────────────────────────────╯

Keys: [↑/↓] select  [p]ause  [r]esume  [k]ill  [l]ogs  [Enter] attach  [q]uit
```

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| `↑`/`k` | Move selection up |
| `↓`/`j` | Move selection down |
| `p` | Pause selected agent |
| `r` | Resume selected agent |
| `+` | Increase iterations for selected agent |
| `-` | Decrease iterations for selected agent |
| `K` | Kill selected agent (with confirmation) |
| `l` | View logs for selected agent (opens in pager) |
| `Enter` | Attach to selected agent (opens `swarm attach`) |
| `a` | Toggle showing all agents (including terminated) |
| `g` | Toggle global mode |
| `q` / `Ctrl+C` | Quit |

### Features

1. **Live updates**: Agent status, iteration counts, and runtime update automatically
2. **Selection**: Navigate between agents with arrow keys to see details
3. **Quick actions**: Pause, resume, or kill agents without leaving the dashboard
4. **Detail panel**: Shows additional info for the selected agent
5. **Sorting**: Agents sorted by status (running > paused > terminated), then by start time

## Files to create/change

- Create `cmd/top.go` - new command implementation
- May need to add a TUI library dependency (e.g., `github.com/charmbracelet/bubbletea`)

## Implementation details

### Dependencies

Recommend using [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI, with [Lip Gloss](https://github.com/charmbracelet/lipgloss) for styling. These are modern, well-maintained Go TUI libraries.

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
```

### cmd/top.go

```go
package cmd

import (
    "fmt"
    "os"
    "os/exec"
    "sort"
    "strings"
    "time"

    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/bubbles/table"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
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
    Long: `Display a real-time dashboard showing all running agents.

The dashboard updates automatically and allows quick actions on agents
like pausing, resuming, or killing them.`,
    Example: `  # Monitor agents in current project
  swarm top

  # Monitor all agents globally
  swarm top --global

  # Include terminated agents
  swarm top --all

  # Faster refresh rate
  swarm top --interval 1s`,
    RunE: func(cmd *cobra.Command, args []string) error {
        p := tea.NewProgram(initialModel(), tea.WithAltScreen())
        _, err := p.Run()
        return err
    },
}

type tickMsg time.Time

type model struct {
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

func initialModel() model {
    s := GetScope()
    global := s == scope.ScopeGlobal
    
    mgr, err := state.NewManagerWithScope(s, "")
    
    return model{
        mgr:      mgr,
        cursor:   0,
        showAll:  topAll,
        global:   global,
        interval: topInterval,
        err:      err,
    }
}

func (m model) Init() tea.Cmd {
    return tea.Batch(
        m.refreshAgents,
        m.tick(),
    )
}

func (m model) tick() tea.Cmd {
    return tea.Tick(m.interval, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

func (m model) refreshAgents() tea.Msg {
    if m.mgr == nil {
        return nil
    }
    
    agents, err := m.mgr.List(!m.showAll)
    if err != nil {
        return err
    }
    
    // Sort: running > paused > terminated, then by start time
    sort.Slice(agents, func(i, j int) bool {
        if agents[i].Status != agents[j].Status {
            order := map[string]int{"running": 0, "terminated": 1}
            oi, oj := order[agents[i].Status], order[agents[j].Status]
            if agents[i].Paused {
                oi = 0 // paused sorts with running
            }
            if agents[j].Paused {
                oj = 0
            }
            return oi < oj
        }
        return agents[i].StartedAt.Before(agents[j].StartedAt)
    })
    
    return agents
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
            return m, m.refreshAgentsCmd
        case "g":
            m.global = !m.global
            // Recreate manager with new scope
            s := scope.ScopeProject
            if m.global {
                s = scope.ScopeGlobal
            }
            mgr, _ := state.NewManagerWithScope(s, "")
            m.mgr = mgr
            return m, m.refreshAgentsCmd
        }

    case []*state.AgentState:
        m.agents = msg
        if m.cursor >= len(m.agents) {
            m.cursor = max(0, len(m.agents)-1)
        }

    case tickMsg:
        return m, tea.Batch(m.refreshAgentsCmd, m.tick())

    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height

    case error:
        m.err = msg
    }

    return m, nil
}

func (m model) View() string {
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

func (m model) renderHeader() string {
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
    
    return fmt.Sprintf(
        "╭─ Swarm Top (%s) ────────────────────────────────────────────────────────╮\n"+
        "│  Running: %d   Paused: %d   Terminated: %d   │   Refresh: %s   │   [q]uit  │\n"+
        "╰────────────────────────────────────────────────────────────────────────────╯",
        scopeStr, running, paused, terminated, m.interval,
    )
}

func (m model) renderTable() string {
    if len(m.agents) == 0 {
        return "  No agents found. Start one with: swarm run -p <prompt>"
    }
    
    var b strings.Builder
    
    // Header
    b.WriteString("  ID        NAME             PROMPT           MODEL              ITER    STATUS      RUNTIME\n")
    b.WriteString("  ─────────────────────────────────────────────────────────────────────────────────────────────\n")
    
    for i, a := range m.agents {
        prefix := "  "
        if i == m.cursor {
            prefix = "▸ "
        }
        
        name := a.Name
        if name == "" {
            name = "-"
        }
        
        status := a.Status
        if a.Paused {
            status = "paused"
        }
        
        runtime := time.Since(a.StartedAt).Round(time.Second)
        
        b.WriteString(fmt.Sprintf("%s%-8s  %-15s  %-15s  %-17s  %2d/%-3d  %-10s  %s\n",
            prefix,
            truncate(a.ID, 8),
            truncate(name, 15),
            truncate(a.Prompt, 15),
            truncate(a.Model, 17),
            a.CurrentIter, a.Iterations,
            status,
            formatDuration(runtime),
        ))
    }
    
    return b.String()
}

func (m model) renderDetail(a *state.AgentState) string {
    name := a.Name
    if name == "" {
        name = a.ID
    }
    
    return fmt.Sprintf(
        "╭─ Selected: %s (%s) ─────────────────────────────────────────────────╮\n"+
        "│  Started: %s\n"+
        "│  Directory: %s\n"+
        "│  Log: %s\n"+
        "╰────────────────────────────────────────────────────────────────────────────╯",
        a.ID, name,
        a.StartedAt.Format("2006-01-02 15:04:05"),
        truncate(a.WorkingDir, 60),
        truncate(a.LogFile, 60),
    )
}

func (m model) renderHelp() string {
    return "Keys: [↑/↓] select  [p]ause  [r]esume  [+/-] iterations  [K]ill  [l]ogs  [Enter] attach  [a]ll  [g]lobal  [q]uit"
}

// Action commands

func (m model) pauseSelected() tea.Cmd {
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
        return m.refreshAgents()
    }
}

func (m model) resumeSelected() tea.Cmd {
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
        return m.refreshAgents()
    }
}

func (m model) increaseIterations() tea.Cmd {
    return func() tea.Msg {
        if m.cursor >= len(m.agents) {
            return nil
        }
        agent := m.agents[m.cursor]
        agent.Iterations++
        m.mgr.Update(agent)
        return m.refreshAgents()
    }
}

func (m model) decreaseIterations() tea.Cmd {
    return func() tea.Msg {
        if m.cursor >= len(m.agents) {
            return nil
        }
        agent := m.agents[m.cursor]
        if agent.Iterations > agent.CurrentIter {
            agent.Iterations--
            m.mgr.Update(agent)
        }
        return m.refreshAgents()
    }
}

func (m model) killSelected() tea.Cmd {
    return func() tea.Msg {
        if m.cursor >= len(m.agents) {
            return nil
        }
        agent := m.agents[m.cursor]
        agent.TerminateMode = "immediate"
        m.mgr.Update(agent)
        return m.refreshAgents()
    }
}

func (m model) viewLogs() tea.Cmd {
    if m.cursor >= len(m.agents) {
        return nil
    }
    agent := m.agents[m.cursor]
    if agent.LogFile == "" {
        return nil
    }
    
    // Open logs in less
    c := exec.Command("less", "+F", agent.LogFile)
    return tea.ExecProcess(c, func(err error) tea.Msg {
        return m.refreshAgents()
    })
}

func (m model) attachSelected() tea.Cmd {
    if m.cursor >= len(m.agents) {
        return nil
    }
    agent := m.agents[m.cursor]
    
    c := exec.Command("swarm", "attach", agent.ID)
    return tea.ExecProcess(c, func(err error) tea.Msg {
        return m.refreshAgents()
    })
}

func (m model) refreshAgentsCmd() tea.Msg {
    return m.refreshAgents()
}

// Helpers

func truncate(s string, max int) string {
    if len(s) <= max {
        return s
    }
    return s[:max-3] + "..."
}

func formatDuration(d time.Duration) string {
    h := int(d.Hours())
    m := int(d.Minutes()) % 60
    s := int(d.Seconds()) % 60
    
    if h > 0 {
        return fmt.Sprintf("%dh %dm", h, m)
    }
    if m > 0 {
        return fmt.Sprintf("%dm %ds", m, s)
    }
    return fmt.Sprintf("%ds", s)
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func init() {
    topCmd.Flags().DurationVar(&topInterval, "interval", 2*time.Second, "Refresh interval")
    topCmd.Flags().BoolVarP(&topAll, "all", "a", false, "Show all agents including terminated")
    rootCmd.AddCommand(topCmd)
}
```

## Use cases

### Monitoring a swarm up session

```bash
# Start multiple agents
swarm up -d

# Monitor them all in real-time
swarm top
```

### Quick multi-agent management

```bash
swarm top
# Use arrow keys to select an agent
# Press 'p' to pause it
# Press '+' to add iterations to another
# Press 'K' to kill one that's stuck
# Press 'q' to exit
```

### CI/CD monitoring

```bash
# Start agents
swarm run -p tests -n 10 -d --name unit-tests
swarm run -p lint -n 5 -d --name linting

# Monitor until all complete
swarm top --all
```

## Edge cases

1. **No agents**: Show helpful message with how to start an agent.

2. **Terminal too small**: Truncate columns gracefully, don't crash.

3. **Agent terminates while viewing**: List updates automatically, cursor adjusts if needed.

4. **Rapid state changes**: Updates are batched on refresh interval, won't overwhelm the display.

5. **`swarm attach` not available**: If attach command fails, return to dashboard gracefully.

6. **Log file missing**: Show "-" or empty for log file path, don't error.

## Comparison with related commands

| Command | Purpose | Multi-agent | Real-time | Interactive |
|---------|---------|-------------|-----------|-------------|
| `swarm list` | Snapshot of agents | Yes | No | No |
| `swarm inspect` | Details of one agent | No | No | No |
| `swarm attach` | Monitor/control one agent | No | Yes | Yes |
| `swarm top` | Dashboard for all agents | **Yes** | **Yes** | **Yes** |

## Acceptance criteria

- `swarm top` shows all running agents in a TUI dashboard
- Dashboard updates in real-time (default: every 2 seconds)
- Arrow keys navigate between agents
- Status bar shows running/paused/terminated counts
- Detail panel shows selected agent's full info
- `p` pauses the selected agent
- `r` resumes the selected paused agent
- `+`/`-` adjusts iterations
- `K` kills the selected agent
- `l` opens logs in a pager
- `Enter` opens `swarm attach` for selected agent
- `a` toggles showing terminated agents
- `g` toggles global mode
- `--interval` flag controls refresh rate
- `--all` flag includes terminated agents
- `--global` flag shows agents from all projects
- Clean exit with `q` or `Ctrl+C`
- Handles empty agent list gracefully
- Works when terminal is resized

---

## Completion Notes (Agent cd59a862)

**Completed:** 2026-01-28

### Implementation Summary

1. **Added dependencies:**
   - `github.com/charmbracelet/bubbletea` - TUI framework
   - `github.com/charmbracelet/lipgloss` - Styling library

2. **Created `cmd/top.go`:**
   - Real-time TUI dashboard with Bubble Tea framework
   - Shows running/paused/terminated agent counts in styled header
   - Table view with ID, name, prompt, model, iteration, status, and runtime
   - Detail panel showing selected agent's started time, directory, and log file
   - Keyboard navigation with arrow keys/j/k
   - All action shortcuts: [p]ause, [r]esume, [+/-] iterations, [K]ill, [l]ogs, [Enter] attach
   - Toggle terminated agents with [a], global scope with [g]
   - Auto-refresh at configurable interval (default 2s)
   - Color-coded status (green=running, yellow=paused, red=terminated)

3. **Registered command in `cmd/root.go`**

4. **All tests pass, `go vet` and `gofmt` clean**
