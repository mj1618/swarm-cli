# Add `swarm replay` command to re-run previous agents

## Problem

When working with agents, users frequently want to re-run a previous agent with the same configuration. Common scenarios include:

1. **Retrying failed runs**: An agent failed due to a transient error (network, rate limit) and users want to try again
2. **Reproducing results**: Users want to run the same task again to verify behavior or compare outputs
3. **Iterating on code changes**: After modifying the codebase, users want to run the same agent task again
4. **Tweaking parameters**: Run the same prompt but with different iterations or model

Currently, users must manually reconstruct the command:
```bash
# Find the original run details
swarm inspect my-agent
# Note: prompt=coder, model=claude-opus-4-20250514, iterations=10

# Manually reconstruct and run
swarm run -p coder -m claude-opus-4-20250514 -n 10 -d
```

This is tedious and error-prone, especially for complex configurations.

## Solution

Add a `swarm replay` command that re-runs an agent using its saved configuration.

### Proposed API

```bash
# Replay an agent by ID or name
swarm replay abc123
swarm replay my-agent

# Replay the most recent agent
swarm replay @last
swarm replay _

# Override specific parameters
swarm replay my-agent -n 20              # More iterations
swarm replay my-agent -m claude-sonnet-4-20250514  # Different model
swarm replay my-agent -N retry-1         # New name

# Replay in detached mode (even if original wasn't)
swarm replay my-agent -d

# Replay in foreground (even if original was detached)
swarm replay my-agent --no-detach

# Show what would be run without executing (dry run)
swarm replay my-agent --dry-run
```

### Default behavior

```
$ swarm replay my-agent

Replaying agent: my-agent (abc123)
Original configuration:
  Prompt:     coder
  Model:      claude-opus-4-20250514
  Iterations: 10
  Detached:   yes

Started detached agent: def456 (PID: 12345)
Name: my-agent-replay-1
Iterations: 10
Log file: ~/swarm/logs/def456.log
```

### Dry run output

```
$ swarm replay my-agent --dry-run

Would replay agent: my-agent (abc123)
Command: swarm run -p coder -m claude-opus-4-20250514 -n 10 -d -N my-agent-replay-1
```

## Files to create/change

- Create `cmd/replay.go` - new command implementation

## Implementation details

### cmd/replay.go

```go
package cmd

import (
    "fmt"
    "strconv"

    "github.com/mj1618/swarm-cli/internal/state"
    "github.com/spf13/cobra"
)

var (
    replayIterations int
    replayModel      string
    replayName       string
    replayDetach     bool
    replayNoDetach   bool
    replayDryRun     bool
)

var replayCmd = &cobra.Command{
    Use:   "replay [process-id-or-name]",
    Short: "Re-run a previous agent with the same configuration",
    Long: `Re-run a previous agent using its saved configuration.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent

By default, the replay inherits the original agent's:
  - Prompt
  - Model
  - Iteration count
  - Detached mode (if log file exists)

Use flags to override any of these settings. The new agent gets
a unique ID and a name based on the original (e.g., "my-agent-replay-1").`,
    Example: `  # Replay agent by ID
  swarm replay abc123

  # Replay agent by name
  swarm replay my-agent

  # Replay most recent agent
  swarm replay @last
  swarm replay _

  # Override iterations
  swarm replay my-agent -n 20

  # Override model
  swarm replay my-agent -m claude-sonnet-4-20250514

  # Give it a custom name
  swarm replay my-agent -N retry-attempt

  # Force detached mode
  swarm replay my-agent -d

  # Force foreground mode
  swarm replay my-agent --no-detach

  # See what would run without executing
  swarm replay my-agent --dry-run`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        agentIdentifier := args[0]

        // Create state manager with scope
        mgr, err := state.NewManagerWithScope(GetScope(), "")
        if err != nil {
            return fmt.Errorf("failed to initialize state manager: %w", err)
        }

        agent, err := ResolveAgentIdentifier(mgr, agentIdentifier)
        if err != nil {
            return fmt.Errorf("agent not found: %w", err)
        }

        // Determine configuration (original values with overrides)
        prompt := agent.Prompt
        model := agent.Model
        iterations := agent.Iterations
        detached := agent.LogFile != "" // Was detached if it has a log file

        // Apply overrides
        if cmd.Flags().Changed("iterations") {
            iterations = replayIterations
        }
        if cmd.Flags().Changed("model") {
            model = replayModel
        }
        if replayDetach {
            detached = true
        }
        if replayNoDetach {
            detached = false
        }

        // Generate name for replay
        name := replayName
        if name == "" {
            baseName := agent.Name
            if baseName == "" {
                baseName = agent.ID
            }
            name = generateReplayName(mgr, baseName)
        }

        // Build the command args
        runArgs := []string{"run"}
        runArgs = append(runArgs, "-p", prompt)
        runArgs = append(runArgs, "-m", model)
        runArgs = append(runArgs, "-n", strconv.Itoa(iterations))
        runArgs = append(runArgs, "-N", name)
        if detached {
            runArgs = append(runArgs, "-d")
        }
        if globalFlag {
            runArgs = append(runArgs, "-g")
        }

        // Dry run mode
        if replayDryRun {
            fmt.Printf("Would replay agent: %s (%s)\n", agent.Name, agent.ID)
            fmt.Printf("Command: swarm %s\n", formatArgs(runArgs))
            return nil
        }

        // Show replay info
        fmt.Printf("Replaying agent: %s (%s)\n", agent.Name, agent.ID)
        fmt.Println("Original configuration:")
        fmt.Printf("  Prompt:     %s\n", agent.Prompt)
        fmt.Printf("  Model:      %s\n", agent.Model)
        fmt.Printf("  Iterations: %d\n", agent.Iterations)
        fmt.Printf("  Detached:   %v\n", agent.LogFile != "")
        if cmd.Flags().Changed("iterations") || cmd.Flags().Changed("model") || 
           replayDetach || replayNoDetach || replayName != "" {
            fmt.Println("\nOverrides applied:")
            if cmd.Flags().Changed("iterations") {
                fmt.Printf("  Iterations: %d\n", iterations)
            }
            if cmd.Flags().Changed("model") {
                fmt.Printf("  Model:      %s\n", model)
            }
            if replayDetach || replayNoDetach {
                fmt.Printf("  Detached:   %v\n", detached)
            }
            if replayName != "" {
                fmt.Printf("  Name:       %s\n", name)
            }
        }
        fmt.Println()

        // Execute the run command by setting the run flags and calling runCmd
        runPrompt = prompt
        runModel = model
        runIterations = iterations
        runName = name
        runDetach = detached
        
        return runCmd.RunE(cmd, []string{})
    },
}

// generateReplayName creates a unique replay name based on the original
func generateReplayName(mgr *state.Manager, baseName string) string {
    // Get all agents to check for name conflicts
    agents, err := mgr.List(false)
    if err != nil {
        return baseName + "-replay"
    }

    // Find existing replay names and get the highest number
    maxNum := 0
    for _, a := range agents {
        if a.Name == baseName+"-replay" {
            if maxNum == 0 {
                maxNum = 1
            }
        }
        // Check for numbered replays like "name-replay-2"
        var num int
        if _, err := fmt.Sscanf(a.Name, baseName+"-replay-%d", &num); err == nil {
            if num >= maxNum {
                maxNum = num + 1
            }
        }
    }

    if maxNum == 0 {
        return baseName + "-replay"
    }
    return fmt.Sprintf("%s-replay-%d", baseName, maxNum)
}

// formatArgs formats args for display
func formatArgs(args []string) string {
    result := ""
    for i, arg := range args {
        if i > 0 {
            result += " "
        }
        // Quote args with spaces
        if containsSpace(arg) {
            result += `"` + arg + `"`
        } else {
            result += arg
        }
    }
    return result
}

func containsSpace(s string) bool {
    for _, c := range s {
        if c == ' ' {
            return true
        }
    }
    return false
}

func init() {
    replayCmd.Flags().IntVarP(&replayIterations, "iterations", "n", 0, "Override iteration count")
    replayCmd.Flags().StringVarP(&replayModel, "model", "m", "", "Override model")
    replayCmd.Flags().StringVarP(&replayName, "name", "N", "", "Set name for the replayed agent")
    replayCmd.Flags().BoolVarP(&replayDetach, "detach", "d", false, "Run in detached mode")
    replayCmd.Flags().BoolVar(&replayNoDetach, "no-detach", false, "Run in foreground mode")
    replayCmd.Flags().BoolVar(&replayDryRun, "dry-run", false, "Show what would be run without executing")
    rootCmd.AddCommand(replayCmd)
}
```

## Use cases

### Retry a failed run

```bash
# Agent failed due to rate limit
swarm list -a
# ID        NAME          STATUS      ITERATION
# abc123    api-refactor  terminated  3/10

# Retry with same config
swarm replay abc123
```

### Iterate on code changes

```bash
# Run a coding task
swarm run -p implement-feature -n 10 -d -N first-attempt

# Review results, make manual fixes
# ...

# Run again to continue/verify
swarm replay first-attempt -N second-attempt
```

### Compare models

```bash
# Run with Opus
swarm run -p complex-task -n 5 -d -N opus-run

# Replay with Sonnet to compare
swarm replay opus-run -m claude-sonnet-4-20250514 -N sonnet-run
```

### Quick retry of last run

```bash
# Something went wrong with the last agent
swarm replay @last

# Or with more iterations
swarm replay _ -n 30
```

### Preview before running

```bash
# Check what config would be used
swarm replay old-agent --dry-run
# Would replay agent: old-agent (abc123)
# Command: swarm run -p coder -m claude-opus-4-20250514 -n 10 -d -N old-agent-replay

# Looks good, run it
swarm replay old-agent
```

## Edge cases

1. **Agent with prompt file/string**: If original agent used `-f` or `-s` instead of `-p`, replay uses the stored prompt name which may be `<file>` or `<string>`. In this case, show an error suggesting to use `swarm run` directly.

2. **Conflicting flags**: `--detach` and `--no-detach` are mutually exclusive. If both specified, return an error.

3. **Agent from different directory**: If replaying a global agent that was started in a different directory, the new agent runs in the current directory. Add a warning about this.

4. **Terminated vs running agents**: Replay works for both. For running agents, it's effectively "clone" functionality.

5. **Name conflicts**: If the generated replay name already exists, increment the number suffix.

6. **Missing prompt**: If the original prompt file was deleted, `swarm run` will fail with its normal error message.

## Acceptance criteria

- `swarm replay <agent>` re-runs agent with same prompt, model, iterations
- `swarm replay @last` and `swarm replay _` work for most recent agent
- `-n` flag overrides iteration count
- `-m` flag overrides model
- `-N` flag sets custom name for replay
- `-d` flag forces detached mode
- `--no-detach` flag forces foreground mode
- `--dry-run` shows command without executing
- Automatic name generation: `name-replay`, `name-replay-2`, etc.
- Works with both running and terminated agents
- Inherits detached mode from original (based on log file presence)
- Clear error message when prompt source was file/string
- Warning when replaying agent from different directory

---

## Completion Notes

**Completed by:** Agent cd59a862  
**Date:** 2026-01-28

### Implementation Summary

Created `cmd/replay.go` with full implementation:

1. **Command structure**: Uses Cobra command framework consistent with other commands
2. **Agent lookup**: Supports ID, name, and special identifiers (`@last`, `_`) via `ResolveAgentIdentifier`
3. **Non-replayable prompts**: Returns clear error for `<file>`, `<string>`, `<stdin>`, and combined stdin prompts
4. **Override flags**: `-n` (iterations), `-m` (model), `-N` (name), `-d` (detach), `--no-detach` (foreground)
5. **Conflicting flags**: Returns error if both `--detach` and `--no-detach` specified
6. **Dry-run mode**: Shows equivalent `swarm run` command without executing
7. **Name generation**: Generates unique names like `name-replay`, `name-replay-2`, etc.
8. **Execution**: Delegates to `runCmd.RunE` after setting appropriate flags

### Files Changed

- Created `cmd/replay.go` - New command implementation
- Modified `cmd/root.go` - Added `replayCmd` to root command

### Testing Performed

- Build passes: `go build ./...`
- All tests pass: `go test ./...`
- Manual testing:
  - `swarm replay --help` - Shows correct usage
  - `swarm replay <id> --dry-run` - Shows correct command
  - `swarm replay <name> --dry-run` - Name lookup works
  - `swarm replay <stdin-agent> --dry-run` - Returns appropriate error
  - `swarm replay <id> --dry-run -n 25 -m <model>` - Overrides work
  - `swarm replay <id> --dry-run --no-detach` - Foreground mode works
  - `swarm replay <id> --dry-run -d --no-detach` - Conflicting flags error works
