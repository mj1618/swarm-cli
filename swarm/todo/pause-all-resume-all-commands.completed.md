# Add `swarm pause-all` and `swarm resume-all` commands

## Problem

Users can pause and resume individual agents with `swarm stop` and `swarm start`, and can terminate all agents with `swarm kill-all`. However, there's no way to pause or resume all running agents at once.

This creates friction in common workflows:

1. **Taking a break**: User wants to pause all agents to free up resources while reviewing code or attending a meeting
2. **System resource management**: Need to temporarily reduce CPU/memory usage without losing agent progress
3. **Code review checkpoint**: Pause all agents to review accumulated changes before continuing
4. **Debugging**: Pause all agents to investigate an issue without interference
5. **Machine sleep/hibernate**: Pause agents before putting laptop to sleep

Current workaround is tedious:
```bash
# Manually pause each agent
swarm list -q | xargs -I{} swarm stop {}

# Or use a loop
for id in $(swarm list -q); do swarm stop $id; done
```

## Solution

Add `swarm pause-all` and `swarm resume-all` commands that batch pause/resume all running agents with a single command.

### Proposed API

```bash
# Pause all running agents
swarm pause-all

# Resume all paused agents
swarm resume-all

# Pause with filters (only pause matching agents)
swarm pause-all --name coder
swarm pause-all --prompt planner
swarm pause-all --model opus

# Dry run - show what would be paused/resumed
swarm pause-all --dry-run
swarm resume-all --dry-run

# Force without confirmation (for scripting)
swarm pause-all -y
swarm resume-all -y
```

### Output examples

**Pause all:**
```
$ swarm pause-all
Pausing 3 running agents...
  abc123 (frontend-task)  paused
  def456 (backend-api)    paused
  ghi789 (coder)          paused

All 3 agents paused. Use 'swarm resume-all' to continue.
```

**Resume all:**
```
$ swarm resume-all
Resuming 3 paused agents...
  abc123 (frontend-task)  resumed
  def456 (backend-api)    resumed
  ghi789 (coder)          resumed

All 3 agents resumed.
```

**With filters:**
```
$ swarm pause-all --name coder
Pausing 1 running agent matching --name "coder"...
  ghi789 (coder)  paused

1 agent paused.
```

**Dry run:**
```
$ swarm pause-all --dry-run
Would pause 3 agents:
  abc123 (frontend-task)
  def456 (backend-api)
  ghi789 (coder)

Run without --dry-run to pause.
```

**No matching agents:**
```
$ swarm pause-all
No running agents to pause.

$ swarm resume-all
No paused agents to resume.
```

## Files to create/change

- Create `cmd/pause_all.go` - new pause-all command
- Create `cmd/resume_all.go` - new resume-all command

## Implementation details

### cmd/pause_all.go

```go
package cmd

import (
    "fmt"
    "strings"

    "github.com/matt/swarm-cli/internal/state"
    "github.com/spf13/cobra"
)

var (
    pauseAllName   string
    pauseAllPrompt string
    pauseAllModel  string
    pauseAllDryRun bool
    pauseAllYes    bool
)

var pauseAllCmd = &cobra.Command{
    Use:   "pause-all",
    Short: "Pause all running agents",
    Long: `Pause all running agents in the current project.

Paused agents will stop after completing their current iteration.
Use 'swarm resume-all' to continue paused agents.

Use --global to pause agents across all projects.`,
    Example: `  # Pause all running agents
  swarm pause-all

  # Pause only agents with matching name
  swarm pause-all --name coder

  # Preview what would be paused
  swarm pause-all --dry-run

  # Skip confirmation
  swarm pause-all -y`,
    RunE: func(cmd *cobra.Command, args []string) error {
        mgr, err := state.NewManagerWithScope(GetScope(), "")
        if err != nil {
            return fmt.Errorf("failed to initialize state manager: %w", err)
        }

        // Get all running agents (not paused)
        agents, err := mgr.List(true) // only running
        if err != nil {
            return fmt.Errorf("failed to list agents: %w", err)
        }

        // Filter to only actually running (not already paused)
        var toPause []*state.AgentState
        for _, agent := range agents {
            if agent.Status != "running" || agent.Paused {
                continue
            }

            // Apply filters
            if pauseAllName != "" && !strings.Contains(strings.ToLower(agent.Name), strings.ToLower(pauseAllName)) {
                continue
            }
            if pauseAllPrompt != "" && !strings.Contains(strings.ToLower(agent.Prompt), strings.ToLower(pauseAllPrompt)) {
                continue
            }
            if pauseAllModel != "" && !strings.Contains(strings.ToLower(agent.Model), strings.ToLower(pauseAllModel)) {
                continue
            }

            toPause = append(toPause, agent)
        }

        if len(toPause) == 0 {
            fmt.Println("No running agents to pause.")
            return nil
        }

        // Dry run mode
        if pauseAllDryRun {
            fmt.Printf("Would pause %d agent(s):\n", len(toPause))
            for _, agent := range toPause {
                name := agent.Name
                if name == "" {
                    name = "-"
                }
                fmt.Printf("  %s (%s)\n", agent.ID, name)
            }
            fmt.Println("\nRun without --dry-run to pause.")
            return nil
        }

        // Confirmation (unless -y)
        if !pauseAllYes {
            fmt.Printf("Pause %d running agent(s)? [y/N]: ", len(toPause))
            var response string
            fmt.Scanln(&response)
            if strings.ToLower(response) != "y" {
                fmt.Println("Cancelled.")
                return nil
            }
        }

        // Pause agents
        fmt.Printf("Pausing %d running agent(s)...\n", len(toPause))
        var paused int
        for _, agent := range toPause {
            agent.Paused = true
            if err := mgr.Update(agent); err != nil {
                name := agent.Name
                if name == "" {
                    name = "-"
                }
                fmt.Printf("  %s (%s)  failed: %v\n", agent.ID, name, err)
                continue
            }
            
            name := agent.Name
            if name == "" {
                name = "-"
            }
            fmt.Printf("  %s (%s)  paused\n", agent.ID, name)
            paused++
        }

        if paused > 0 {
            fmt.Printf("\n%d agent(s) paused. Use 'swarm resume-all' to continue.\n", paused)
        }
        return nil
    },
}

func init() {
    pauseAllCmd.Flags().StringVarP(&pauseAllName, "name", "N", "", "Filter by agent name (substring match)")
    pauseAllCmd.Flags().StringVarP(&pauseAllPrompt, "prompt", "p", "", "Filter by prompt name (substring match)")
    pauseAllCmd.Flags().StringVarP(&pauseAllModel, "model", "m", "", "Filter by model name (substring match)")
    pauseAllCmd.Flags().BoolVar(&pauseAllDryRun, "dry-run", false, "Show what would be paused without pausing")
    pauseAllCmd.Flags().BoolVarP(&pauseAllYes, "yes", "y", false, "Skip confirmation prompt")
    rootCmd.AddCommand(pauseAllCmd)
}
```

### cmd/resume_all.go

```go
package cmd

import (
    "fmt"
    "strings"
    "time"

    "github.com/matt/swarm-cli/internal/state"
    "github.com/spf13/cobra"
)

var (
    resumeAllName   string
    resumeAllPrompt string
    resumeAllModel  string
    resumeAllDryRun bool
    resumeAllYes    bool
)

var resumeAllCmd = &cobra.Command{
    Use:   "resume-all",
    Short: "Resume all paused agents",
    Long: `Resume all paused agents in the current project.

Paused agents will continue from their next iteration.
Use --global to resume agents across all projects.`,
    Example: `  # Resume all paused agents
  swarm resume-all

  # Resume only agents with matching name
  swarm resume-all --name coder

  # Preview what would be resumed
  swarm resume-all --dry-run

  # Skip confirmation
  swarm resume-all -y`,
    RunE: func(cmd *cobra.Command, args []string) error {
        mgr, err := state.NewManagerWithScope(GetScope(), "")
        if err != nil {
            return fmt.Errorf("failed to initialize state manager: %w", err)
        }

        // Get all running agents
        agents, err := mgr.List(true) // only running
        if err != nil {
            return fmt.Errorf("failed to list agents: %w", err)
        }

        // Filter to only paused agents
        var toResume []*state.AgentState
        for _, agent := range agents {
            if agent.Status != "running" || !agent.Paused {
                continue
            }

            // Apply filters
            if resumeAllName != "" && !strings.Contains(strings.ToLower(agent.Name), strings.ToLower(resumeAllName)) {
                continue
            }
            if resumeAllPrompt != "" && !strings.Contains(strings.ToLower(agent.Prompt), strings.ToLower(resumeAllPrompt)) {
                continue
            }
            if resumeAllModel != "" && !strings.Contains(strings.ToLower(agent.Model), strings.ToLower(resumeAllModel)) {
                continue
            }

            toResume = append(toResume, agent)
        }

        if len(toResume) == 0 {
            fmt.Println("No paused agents to resume.")
            return nil
        }

        // Dry run mode
        if resumeAllDryRun {
            fmt.Printf("Would resume %d agent(s):\n", len(toResume))
            for _, agent := range toResume {
                name := agent.Name
                if name == "" {
                    name = "-"
                }
                pausedDur := ""
                if agent.PausedAt != nil {
                    pausedDur = fmt.Sprintf(" (paused %s)", time.Since(*agent.PausedAt).Round(time.Second))
                }
                fmt.Printf("  %s (%s)%s\n", agent.ID, name, pausedDur)
            }
            fmt.Println("\nRun without --dry-run to resume.")
            return nil
        }

        // Confirmation (unless -y)
        if !resumeAllYes {
            fmt.Printf("Resume %d paused agent(s)? [y/N]: ", len(toResume))
            var response string
            fmt.Scanln(&response)
            if strings.ToLower(response) != "y" {
                fmt.Println("Cancelled.")
                return nil
            }
        }

        // Resume agents
        fmt.Printf("Resuming %d paused agent(s)...\n", len(toResume))
        var resumed int
        for _, agent := range toResume {
            agent.Paused = false
            agent.PausedAt = nil
            if err := mgr.Update(agent); err != nil {
                name := agent.Name
                if name == "" {
                    name = "-"
                }
                fmt.Printf("  %s (%s)  failed: %v\n", agent.ID, name, err)
                continue
            }
            
            name := agent.Name
            if name == "" {
                name = "-"
            }
            fmt.Printf("  %s (%s)  resumed\n", agent.ID, name)
            resumed++
        }

        if resumed > 0 {
            fmt.Printf("\n%d agent(s) resumed.\n", resumed)
        }
        return nil
    },
}

func init() {
    resumeAllCmd.Flags().StringVarP(&resumeAllName, "name", "N", "", "Filter by agent name (substring match)")
    resumeAllCmd.Flags().StringVarP(&resumeAllPrompt, "prompt", "p", "", "Filter by prompt name (substring match)")
    resumeAllCmd.Flags().StringVarP(&resumeAllModel, "model", "m", "", "Filter by model name (substring match)")
    resumeAllCmd.Flags().BoolVar(&resumeAllDryRun, "dry-run", false, "Show what would be resumed without resuming")
    resumeAllCmd.Flags().BoolVarP(&resumeAllYes, "yes", "y", false, "Skip confirmation prompt")
    rootCmd.AddCommand(resumeAllCmd)
}
```

## Use cases

### Taking a break

```bash
# Pause everything before lunch
swarm pause-all -y

# After lunch, resume
swarm resume-all -y
```

### Selective pause for resource management

```bash
# Pause only the resource-heavy opus agents
swarm pause-all --model opus

# Keep sonnet agents running
swarm list  # Shows sonnet still running, opus paused
```

### Code review checkpoint

```bash
# Pause all to review changes
swarm pause-all

# Review accumulated changes
git diff

# Look good, resume
swarm resume-all
```

### Before system sleep

```bash
# Pause before closing laptop
swarm pause-all -y

# Resume after waking
swarm resume-all -y
```

## Edge cases

1. **No running agents**: Show "No running agents to pause." (not an error)
2. **No paused agents**: Show "No paused agents to resume." (not an error)
3. **Mixed state (some paused, some running)**: `pause-all` only pauses running ones, `resume-all` only resumes paused ones
4. **Some agents fail to update**: Continue with others, report failures individually
5. **Filters match nothing**: Show "No running agents to pause." or "No paused agents to resume."
6. **Global scope**: Works with `--global` flag to affect agents across all projects
7. **Confirmation cancelled**: No changes made, clean exit

## Testing scenarios

```bash
# Start multiple agents
swarm run -p coder -n 20 -d --name coder-1
swarm run -p coder -n 20 -d --name coder-2
swarm run -p planner -n 10 -d --name planner-1

# Pause all
swarm pause-all
swarm list  # All show "paused" status

# Resume all
swarm resume-all
swarm list  # All show "running" status

# Pause with filter
swarm pause-all --name coder
swarm list  # Only coder agents paused

# Resume with filter
swarm resume-all --prompt coder
swarm list  # Coder agents resumed

# Dry run
swarm pause-all --dry-run  # Shows what would be paused
swarm list  # Nothing changed

# Force mode (for scripting)
swarm pause-all -y && sleep 60 && swarm resume-all -y
```

## Relationship to existing commands

- `swarm stop <agent>` - Pause a single agent (existing)
- `swarm start <agent>` - Resume a single agent (existing)
- `swarm stop-all` - Stop all compose services (different - for compose mode)
- `swarm start-all` - Start all compose services (different - for compose mode)
- `swarm kill-all` - Terminate all agents (destructive)
- `swarm pause-all` - Pause all agents (new - non-destructive)
- `swarm resume-all` - Resume all agents (new)

## Acceptance criteria

- `swarm pause-all` pauses all running agents in the current project
- `swarm resume-all` resumes all paused agents in the current project
- `--global` flag affects agents across all projects
- `--name`, `--prompt`, `--model` filters work correctly
- `--dry-run` shows what would be affected without making changes
- `-y` flag skips confirmation prompt
- Confirmation prompt shows count of agents to be affected
- Output shows each agent's status after the operation
- No error when there are no agents to pause/resume
- Failed updates are reported but don't stop other operations
- `swarm list` shows correct status after pause-all/resume-all

---

## Completion Notes (Agent cd59a862)

**Completed:** 2026-01-28

**Implementation:**
- Created `cmd/pause_all.go` with all requested features
- Created `cmd/resume_all.go` with all requested features
- Registered both commands in `cmd/root.go`

**Features implemented:**
- Filter flags: `--name/-N`, `--prompt/-p`, `--model/-m` (all case-insensitive substring matches)
- Dry-run mode: `--dry-run` shows what would be affected without making changes
- Confirmation prompts with `-y` to skip (handles non-interactive mode gracefully)
- Proper output showing each agent's status after operation
- Works with `--global` flag for cross-project operations

**Note:** The existing `swarm stop-all` and `swarm start-all` commands already provide basic pause/resume all functionality. The new `pause-all` and `resume-all` commands add enhanced features (filters, dry-run, confirmation). Both command sets coexist and can be used interchangeably.
