# Add `swarm attach` command for interactive agent monitoring

## Problem

When users start agents in detached mode (`-d`), the only way to monitor them is via `swarm logs -f`. While this works for basic log tailing, it lacks:

1. **Status awareness**: No visibility into agent status (running/paused), current iteration, or when the agent terminates
2. **Interactive controls**: No way to pause, resume, or update the agent without opening another terminal
3. **Context**: No header showing agent info (name, model, iterations) while viewing output

Users who want to "check in" on a running agent have to juggle multiple commands:

```bash
# Currently: need multiple terminals or commands
swarm inspect my-agent    # See status
swarm logs -f my-agent    # Watch output
swarm update my-agent ... # Control it
```

## Solution

Add a `swarm attach` command that provides an interactive monitoring experience for detached agents, similar to `docker attach` or `screen -r`.

### Proposed API

```bash
# Attach to agent by ID or name
swarm attach abc123
swarm attach my-agent

# Attach without keyboard shortcuts (just enhanced log following)
swarm attach my-agent --no-interactive

# Show more/fewer context lines when attaching
swarm attach my-agent --tail 100
```

### Interactive Mode (default)

When attached, the terminal shows:

```
╭─ Agent: my-agent (abc123) ──────────────────────────╮
│ Status: running  │  Iteration: 3/10  │  Model: claude-opus-4-20250514 │
╰─────────────────────────────────────────────────────╯
Press: [p]ause  [r]esume  [+]iter  [-]iter  [k]ill  [q]uit

... agent log output follows here ...
```

The status bar updates in real-time as the agent progresses through iterations.

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| `p` | Pause agent (stop after current iteration) |
| `r` | Resume paused agent |
| `+` | Increase iteration count by 1 |
| `-` | Decrease iteration count by 1 (minimum: current iteration) |
| `k` | Kill agent (with confirmation) |
| `q` | Detach (quit viewing, agent continues running) |
| `Ctrl+C` | Same as `q` - detach without killing |

### Non-interactive mode

With `--no-interactive`, acts like enhanced `logs -f`:
- Shows status header (updates periodically)
- Follows log output
- No keyboard controls
- `Ctrl+C` detaches

## Files to create/change

- Create `cmd/attach.go` - new command implementation

## Implementation details

### cmd/attach.go

```go
package cmd

import (
    "bufio"
    "fmt"
    "io"
    "os"
    "time"

    "github.com/eiannone/keyboard"
    "github.com/fatih/color"
    "github.com/matt/swarm-cli/internal/state"
    "github.com/spf13/cobra"
)

var (
    attachNoInteractive bool
    attachTail          int
)

var attachCmd = &cobra.Command{
    Use:   "attach [agent-id-or-name]",
    Short: "Attach to a running agent",
    Long: `Attach to a running detached agent for interactive monitoring.

Shows a status header with agent info that updates in real-time,
follows log output, and provides keyboard shortcuts for control.

Press 'q' or Ctrl+C to detach without killing the agent.`,
    Example: `  # Attach to agent by ID
  swarm attach abc123

  # Attach to agent by name
  swarm attach my-agent

  # Attach without keyboard controls
  swarm attach my-agent --no-interactive

  # Show last 100 lines when attaching
  swarm attach my-agent --tail 100`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        agentIdentifier := args[0]

        mgr, err := state.NewManagerWithScope(GetScope(), "")
        if err != nil {
            return fmt.Errorf("failed to initialize state manager: %w", err)
        }

        agent, err := mgr.GetByNameOrID(agentIdentifier)
        if err != nil {
            return fmt.Errorf("agent not found: %w", err)
        }

        if agent.Status != "running" {
            return fmt.Errorf("agent is not running (status: %s)", agent.Status)
        }

        if agent.LogFile == "" {
            return fmt.Errorf("agent was not started in detached mode (no log file)")
        }

        if attachNoInteractive {
            return attachNonInteractive(mgr, agent)
        }

        return attachInteractive(mgr, agent)
    },
}

func attachInteractive(mgr *state.Manager, agent *state.AgentState) error {
    // Initialize keyboard
    if err := keyboard.Open(); err != nil {
        // Fall back to non-interactive if keyboard fails
        fmt.Println("Warning: keyboard input unavailable, running in non-interactive mode")
        return attachNonInteractive(mgr, agent)
    }
    defer keyboard.Close()

    // Print initial header
    printStatusHeader(agent)
    printHelpLine()

    // Open log file
    file, err := os.Open(agent.LogFile)
    if err != nil {
        return fmt.Errorf("failed to open log file: %w", err)
    }
    defer file.Close()

    // Show last N lines
    if err := showLastLines(file, attachTail); err != nil {
        return err
    }

    // Seek to end for following
    file.Seek(0, io.SeekEnd)

    // Channels for coordination
    done := make(chan struct{})
    logLines := make(chan string, 100)

    // Goroutine: read log file
    go func() {
        reader := bufio.NewReader(file)
        for {
            select {
            case <-done:
                return
            default:
                line, err := reader.ReadString('\n')
                if err == io.EOF {
                    time.Sleep(100 * time.Millisecond)
                    continue
                }
                if err != nil {
                    return
                }
                logLines <- line
            }
        }
    }()

    // Goroutine: update status periodically
    statusTicker := time.NewTicker(2 * time.Second)
    defer statusTicker.Stop()

    // Main loop
    for {
        select {
        case line := <-logLines:
            fmt.Print(line)

        case <-statusTicker.C:
            // Refresh agent state
            updated, err := mgr.Get(agent.ID)
            if err == nil {
                agent = updated
                // Move cursor up, clear line, print header, move back
                refreshStatusHeader(agent)
            }
            // Check if terminated
            if agent.Status == "terminated" {
                close(done)
                fmt.Println("\n[swarm] Agent terminated")
                return nil
            }

        default:
            // Check for keyboard input (non-blocking)
            char, key, err := keyboard.GetKey()
            if err != nil {
                continue
            }

            if key == keyboard.KeyCtrlC || char == 'q' {
                close(done)
                fmt.Println("\n[swarm] Detached from agent (agent still running)")
                return nil
            }

            switch char {
            case 'p':
                agent.Paused = true
                mgr.Update(agent)
                fmt.Println("\n[swarm] Agent paused")
            case 'r':
                agent.Paused = false
                mgr.Update(agent)
                fmt.Println("\n[swarm] Agent resumed")
            case '+':
                agent.Iterations++
                mgr.Update(agent)
                fmt.Printf("\n[swarm] Iterations increased to %d\n", agent.Iterations)
            case '-':
                if agent.Iterations > agent.CurrentIter {
                    agent.Iterations--
                    mgr.Update(agent)
                    fmt.Printf("\n[swarm] Iterations decreased to %d\n", agent.Iterations)
                }
            case 'k':
                fmt.Print("\n[swarm] Kill agent? (y/n): ")
                char2, _, _ := keyboard.GetKey()
                if char2 == 'y' {
                    agent.TerminateMode = "immediate"
                    mgr.Update(agent)
                    close(done)
                    fmt.Println("\n[swarm] Agent killed")
                    return nil
                }
                fmt.Println("Cancelled")
            }
        }
    }
}

func attachNonInteractive(mgr *state.Manager, agent *state.AgentState) error {
    printStatusHeader(agent)
    fmt.Println("\nPress Ctrl+C to detach\n")

    // Similar to logs -f but with periodic status refresh
    // ... implementation similar to current followFile()
    return nil
}

func printStatusHeader(agent *state.AgentState) {
    bold := color.New(color.Bold)
    
    // Clear screen from cursor and move to top
    fmt.Print("\033[2J\033[H")
    
    bold.Printf("╭─ Agent: %s (%s) ", agent.Name, agent.ID)
    fmt.Println("─────────────────────────────────╮")
    
    statusStr := agent.Status
    if agent.Paused {
        statusStr = "paused"
    }
    
    fmt.Printf("│ Status: %-8s │ Iteration: %d/%d │ Model: %s │\n",
        statusStr, agent.CurrentIter, agent.Iterations, agent.Model)
    fmt.Println("╰─────────────────────────────────────────────────────────╯")
}

func printHelpLine() {
    dim := color.New(color.Faint)
    dim.Println("Press: [p]ause  [r]esume  [+]iter  [-]iter  [k]ill  [q]uit")
    fmt.Println()
}

func refreshStatusHeader(agent *state.AgentState) {
    // Save cursor, move to line 2, update, restore cursor
    fmt.Print("\033[s")      // Save cursor
    fmt.Print("\033[2;0H")   // Move to line 2
    
    statusStr := agent.Status
    if agent.Paused {
        statusStr = "paused"
    }
    
    fmt.Printf("│ Status: %-8s │ Iteration: %d/%d │ Model: %s │",
        statusStr, agent.CurrentIter, agent.Iterations, agent.Model)
    fmt.Print("\033[K")      // Clear to end of line
    fmt.Print("\033[u")      // Restore cursor
}

func init() {
    attachCmd.Flags().BoolVar(&attachNoInteractive, "no-interactive", false, "Disable keyboard controls")
    attachCmd.Flags().IntVar(&attachTail, "tail", 50, "Number of lines to show from the end")
    rootCmd.AddCommand(attachCmd)
}
```

## Dependencies

- `github.com/eiannone/keyboard` - for cross-platform keyboard input (lightweight, no CGO)

Alternative: Could use raw terminal mode without external dependency, but keyboard library handles edge cases better.

## Use cases

### Monitoring a long-running agent

```bash
# Start agent
swarm run -p big-task -n 50 -d --name worker

# Later, check on it
swarm attach worker
# See status, watch output, press 'q' to detach
```

### Quick iteration adjustment

```bash
swarm attach my-agent
# See it's on iteration 8/10, press '+' a few times to extend to 15
# Press 'q' to detach and let it continue
```

### Pausing to review

```bash
swarm attach my-agent
# Press 'p' to pause between iterations
# Review the output
# Press 'r' to resume
# Press 'q' to detach
```

## Edge cases

1. **Agent terminates while attached**: Display termination message and exit cleanly
2. **Log file deleted**: Show error and detach gracefully
3. **Keyboard unavailable** (piped input, etc.): Fall back to non-interactive mode
4. **Terminal too narrow**: Truncate status header gracefully
5. **Agent not detached**: Error "agent was not started in detached mode"

## Acceptance criteria

- `swarm attach abc123` connects to agent abc123
- Status header shows and updates: name, ID, status, iteration, model
- Log output follows in real-time
- Keyboard shortcuts work: p (pause), r (resume), +/- (iterations), k (kill), q (quit)
- `Ctrl+C` detaches without killing agent
- `--no-interactive` disables keyboard controls
- `--tail N` controls initial log lines shown
- Clean exit when agent terminates
- Falls back to non-interactive if keyboard unavailable
