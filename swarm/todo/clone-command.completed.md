# Add `swarm clone` command to duplicate agent configuration

## Completion Note (cd59a862)

Implemented the `swarm clone` command with all specified functionality:

**Files created:**
- `cmd/clone.go` - Full implementation of the clone command
- `cmd/clone_test.go` - Tests for command flags, usage, and args validation

**Features implemented:**
- Clone agent by ID or name (also supports `@last` and `_` identifiers)
- Override iterations (`-n`), model (`-m`), name (`-N`)
- `--dry-run` flag to show equivalent run command
- `--same-dir` flag to run in source agent's directory
- `--foreground` flag to override detached mode
- `--forever` flag for unlimited iterations
- `--env` flag to set environment variables
- `--on-complete` hook support
- Auto-generates name from source (e.g., "my-agent" -> "my-agent-clone")
- Works for both running and terminated agents
- Proper state management and cleanup

All tests pass. Command is registered in root.go.

## Problem

When users want to run another agent with the same or similar configuration as an existing agent, they must manually remember and re-specify all the options:

```bash
# First agent
swarm run -p complex-task -n 20 -m claude-opus-4-20250514 -d --name worker-1

# To run another one, user must remember all options:
swarm run -p complex-task -n 20 -m claude-opus-4-20250514 -d --name worker-2
```

This is error-prone and tedious, especially for:
- Re-running a completed agent with the same configuration
- Running multiple agents with the same prompt in parallel
- Running the same task with a tweaked setting (e.g., different model)

## Solution

Add a `swarm clone` command that creates a new agent based on an existing agent's configuration.

### Proposed API

```bash
# Clone an agent by ID (starts immediately with same config)
swarm clone abc123

# Clone an agent by name
swarm clone my-agent

# Clone with overrides
swarm clone abc123 --name worker-2
swarm clone abc123 -n 30              # Different iteration count
swarm clone abc123 -m claude-sonnet-4-20250514  # Different model

# Clone in foreground (even if original was detached)
swarm clone abc123 --foreground

# Clone but don't start yet (just show the equivalent run command)
swarm clone abc123 --dry-run
```

### Behavior

1. Looks up the source agent by ID or name
2. Creates a new agent with the same:
   - Prompt
   - Model
   - Iterations
   - Working directory (current dir, or original dir with `--same-dir`)
3. Allows overrides via flags
4. Starts the new agent (detached by default if source was detached)
5. Returns the new agent's ID

### Flags

| Flag | Description |
|------|-------------|
| `--name, -N` | Name for the cloned agent (default: auto-generated from source) |
| `-n, --iterations` | Override iteration count |
| `-m, --model` | Override model |
| `-d, --detach` | Run detached (default: matches source) |
| `--foreground` | Run in foreground (opposite of -d) |
| `--same-dir` | Run in the same directory as the source agent |
| `--dry-run` | Print the equivalent `swarm run` command instead of running |

## Files to create/change

- Create `cmd/clone.go` - new command implementation

## Implementation

### cmd/clone.go

```go
package cmd

import (
    "fmt"
    "os"
    "strings"

    "github.com/mj1618/swarm-cli/internal/state"
    "github.com/spf13/cobra"
)

var (
    cloneName       string
    cloneIterations int
    cloneModel      string
    cloneDetach     bool
    cloneForeground bool
    cloneSameDir    bool
    cloneDryRun     bool
)

var cloneCmd = &cobra.Command{
    Use:   "clone [process-id-or-name]",
    Short: "Clone an agent's configuration to start a new agent",
    Long: `Clone an existing agent's configuration to start a new agent.

This is useful for:
- Re-running a completed agent
- Running multiple agents with the same prompt
- Running with slight configuration changes

The source agent can be running or terminated.`,
    Example: `  # Clone a completed agent to run it again
  swarm clone abc123

  # Clone with a new name
  swarm clone my-agent --name my-agent-v2

  # Clone with different iterations
  swarm clone abc123 -n 50

  # Clone with different model
  swarm clone abc123 -m claude-sonnet-4-20250514

  # See what command would be run without executing
  swarm clone abc123 --dry-run`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        sourceIdentifier := args[0]

        // Create state manager with scope
        mgr, err := state.NewManagerWithScope(GetScope(), "")
        if err != nil {
            return fmt.Errorf("failed to initialize state manager: %w", err)
        }

        source, err := mgr.GetByNameOrID(sourceIdentifier)
        if err != nil {
            return fmt.Errorf("source agent not found: %w", err)
        }

        // Determine configuration (source values with overrides)
        prompt := source.Prompt
        model := source.Model
        iterations := source.Iterations
        name := ""
        detached := source.LogFile != "" // Was originally detached if it has a log file

        // Apply overrides
        if cmd.Flags().Changed("iterations") {
            iterations = cloneIterations
        }
        if cmd.Flags().Changed("model") {
            model = cloneModel
        }
        if cmd.Flags().Changed("name") {
            name = cloneName
        } else if source.Name != "" {
            // Auto-generate name from source: "name" -> "name-clone"
            name = source.Name + "-clone"
        }
        if cloneForeground {
            detached = false
        } else if cmd.Flags().Changed("detach") {
            detached = cloneDetach
        }

        // Determine working directory
        workDir := ""
        if cloneSameDir && source.WorkingDir != "" {
            workDir = source.WorkingDir
        }

        // Build the equivalent run command
        var cmdParts []string
        cmdParts = append(cmdParts, "swarm", "run", "-p", prompt)
        cmdParts = append(cmdParts, "-n", fmt.Sprintf("%d", iterations))
        cmdParts = append(cmdParts, "-m", model)
        if name != "" {
            cmdParts = append(cmdParts, "--name", name)
        }
        if detached {
            cmdParts = append(cmdParts, "-d")
        }

        if cloneDryRun {
            fmt.Println(strings.Join(cmdParts, " "))
            return nil
        }

        // Set values for runCmd to use
        runPrompt = prompt
        runModel = model
        runIterations = iterations
        runDetach = detached
        runName = name

        // Change to source directory if --same-dir
        if workDir != "" {
            if err := os.Chdir(workDir); err != nil {
                return fmt.Errorf("failed to change to source directory: %w", err)
            }
        }

        // Execute the run command
        fmt.Printf("Cloning agent %s...\n", source.ID)
        return runCmd.RunE(cmd, []string{})
    },
}

func init() {
    cloneCmd.Flags().StringVarP(&cloneName, "name", "N", "", "Name for the cloned agent")
    cloneCmd.Flags().IntVarP(&cloneIterations, "iterations", "n", 0, "Override iteration count")
    cloneCmd.Flags().StringVarP(&cloneModel, "model", "m", "", "Override model")
    cloneCmd.Flags().BoolVarP(&cloneDetach, "detach", "d", false, "Run in detached mode")
    cloneCmd.Flags().BoolVar(&cloneForeground, "foreground", false, "Run in foreground mode")
    cloneCmd.Flags().BoolVar(&cloneSameDir, "same-dir", false, "Run in same directory as source agent")
    cloneCmd.Flags().BoolVar(&cloneDryRun, "dry-run", false, "Print equivalent run command without executing")
    rootCmd.AddCommand(cloneCmd)
}
```

## Use cases

### Re-run a completed agent

```bash
# Agent finished but you want to run it again
swarm list -a
# Shows: abc123  worker  my-prompt  terminated  20/20

swarm clone abc123
# Starts new agent with same config
```

### Run multiple agents in parallel

```bash
swarm run -p batch-task -n 10 -d --name batch-1
swarm clone batch-1 --name batch-2
swarm clone batch-1 --name batch-3
# Now 3 agents running the same task
```

### Iterate on a task with different models

```bash
# First try with opus
swarm run -p complex-task -n 5 -d --name attempt-opus -m claude-opus-4-20250514

# Try again with sonnet
swarm clone attempt-opus --name attempt-sonnet -m claude-sonnet-4-20250514
```

### Quick re-run with more iterations

```bash
swarm clone my-agent -n 100  # Same everything, but 100 iterations
```

## Edge cases

1. **Source agent not found**: Error with helpful message
2. **Prompt file deleted**: Error when run command tries to load prompt
3. **Name conflict**: State manager auto-appends number suffix (existing behavior)
4. **Source in different directory**: By default runs in current dir, use `--same-dir` to match source
5. **Circular cloning**: No issue - just copies config, doesn't track lineage

## Acceptance criteria

- `swarm clone abc123` creates new agent with same prompt, model, iterations
- `--name` sets custom name for cloned agent
- `-n` overrides iteration count
- `-m` overrides model
- `--dry-run` prints equivalent `swarm run` command
- `--foreground` forces foreground mode
- `--same-dir` runs in source agent's original directory
- Clone works for both running and terminated agents
- New agent ID is printed on success
