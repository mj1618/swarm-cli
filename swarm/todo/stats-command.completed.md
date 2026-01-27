# Add `swarm stats` command for usage statistics

## Completion Notes (Agent cd59a862 - 2026-01-28)

Task completed successfully. Implemented the `swarm stats` command with:

**Files created:**
- `cmd/stats.go` - Main command implementation
- `cmd/stats_test.go` - Comprehensive unit tests

**Features implemented:**
- Status summary (running/paused/terminated counts)
- Iteration counts (completed/total)
- Top prompts by run count (limited to top 5)
- Models used with agent counts
- Runtime statistics (total and average)
- `--format json` for machine-readable output
- `--global` flag support via existing scope infrastructure
- Handles empty prompt/model names gracefully (shown as "(none)"/"(unknown)")

**All acceptance criteria met:**
- ✅ `swarm stats` shows aggregate statistics for agents in current project
- ✅ `swarm stats --global` shows statistics across all projects
- ✅ `swarm stats --format json` outputs machine-readable JSON
- ✅ Status counts accurate
- ✅ Iteration counts accurate
- ✅ Prompt stats sorted by run count
- ✅ Model stats show distribution
- ✅ Works with no agents (shows zeros)
- ✅ All tests pass

## Problem

Users who run many agents have no way to get a high-level overview of their agent usage patterns. Currently, `swarm list` shows individual agents, but there's no aggregate view that answers questions like:

1. How many agents have I run total?
2. How many iterations have been completed?
3. Which prompts do I use most frequently?
4. Which models am I using?
5. How long do my agents typically run?

This information would be useful for:
- Understanding usage patterns to optimize workflows
- Identifying which prompts are most productive
- Tracking resource usage over time
- Getting a quick health check on the swarm ("3 running, 2 paused, 15 terminated")

## Solution

Add a `swarm stats` command that displays aggregate statistics about agents.

### Proposed API

```bash
# Show stats for current project
swarm stats

# Show stats for all projects
swarm stats --global

# Output as JSON (for scripting/dashboards)
swarm stats --format json
```

### Default output

```
Agent Statistics
─────────────────────────────────

Status Summary
  Running:    3
  Paused:     1
  Terminated: 12
  Total:      16

Iterations
  Completed:  247
  In Progress: 8/10 (running agent abc123)
  Total:      255

Top Prompts (by run count)
  planner      8 runs  (142 iterations)
  coder        5 runs  (87 iterations)
  reviewer     3 runs  (26 iterations)

Models Used
  claude-opus-4-20250514     10 agents
  claude-sonnet-4-20250514    6 agents

Runtime
  Total:   14h 32m
  Average: 54m per agent
```

## Files to create/change

- Create `cmd/stats.go` - new command implementation

## Implementation details

### cmd/stats.go

```go
package cmd

import (
    "encoding/json"
    "fmt"
    "sort"
    "time"

    "github.com/fatih/color"
    "github.com/matt/swarm-cli/internal/state"
    "github.com/spf13/cobra"
)

var statsFormat string

type Stats struct {
    Running     int `json:"running"`
    Paused      int `json:"paused"`
    Terminated  int `json:"terminated"`
    Total       int `json:"total"`
    
    IterationsCompleted int `json:"iterations_completed"`
    IterationsTotal     int `json:"iterations_total"`
    
    PromptStats []PromptStat `json:"prompt_stats"`
    ModelStats  []ModelStat  `json:"model_stats"`
    
    TotalRuntime   time.Duration `json:"total_runtime_seconds"`
    AverageRuntime time.Duration `json:"average_runtime_seconds"`
}

type PromptStat struct {
    Name       string `json:"name"`
    RunCount   int    `json:"run_count"`
    Iterations int    `json:"iterations"`
}

type ModelStat struct {
    Name  string `json:"name"`
    Count int    `json:"count"`
}

var statsCmd = &cobra.Command{
    Use:   "stats",
    Short: "Show agent usage statistics",
    Long: `Display aggregate statistics about agent usage.

Shows counts of running/paused/terminated agents, iteration totals,
prompt usage frequency, and model distribution.`,
    Example: `  # Show stats for current project
  swarm stats

  # Show stats across all projects
  swarm stats --global

  # Output as JSON
  swarm stats --format json`,
    RunE: func(cmd *cobra.Command, args []string) error {
        mgr, err := state.NewManagerWithScope(GetScope(), "")
        if err != nil {
            return fmt.Errorf("failed to initialize state manager: %w", err)
        }

        // Get all agents (not just running)
        agents, err := mgr.List(false)
        if err != nil {
            return fmt.Errorf("failed to list agents: %w", err)
        }

        stats := calculateStats(agents)

        if statsFormat == "json" {
            output, err := json.MarshalIndent(stats, "", "  ")
            if err != nil {
                return fmt.Errorf("failed to marshal stats: %w", err)
            }
            fmt.Println(string(output))
            return nil
        }

        printStats(stats)
        return nil
    },
}

func calculateStats(agents []*state.AgentState) Stats {
    stats := Stats{}
    promptMap := make(map[string]*PromptStat)
    modelMap := make(map[string]int)

    now := time.Now()

    for _, agent := range agents {
        stats.Total++

        // Status counts
        switch agent.Status {
        case "running":
            if agent.Paused {
                stats.Paused++
            } else {
                stats.Running++
            }
        case "terminated":
            stats.Terminated++
        }

        // Iteration counts
        stats.IterationsCompleted += agent.CurrentIter
        stats.IterationsTotal += agent.Iterations

        // Prompt stats
        if ps, ok := promptMap[agent.Prompt]; ok {
            ps.RunCount++
            ps.Iterations += agent.CurrentIter
        } else {
            promptMap[agent.Prompt] = &PromptStat{
                Name:       agent.Prompt,
                RunCount:   1,
                Iterations: agent.CurrentIter,
            }
        }

        // Model stats
        modelMap[agent.Model]++

        // Runtime calculation
        var endTime time.Time
        if agent.Status == "terminated" {
            // For terminated agents, estimate end time from start + reasonable duration
            // In practice, we'd want to track actual end time in AgentState
            endTime = agent.StartedAt.Add(time.Duration(agent.CurrentIter) * 5 * time.Minute)
        } else {
            endTime = now
        }
        stats.TotalRuntime += endTime.Sub(agent.StartedAt)
    }

    // Convert maps to sorted slices
    for _, ps := range promptMap {
        stats.PromptStats = append(stats.PromptStats, *ps)
    }
    sort.Slice(stats.PromptStats, func(i, j int) bool {
        return stats.PromptStats[i].RunCount > stats.PromptStats[j].RunCount
    })

    for model, count := range modelMap {
        stats.ModelStats = append(stats.ModelStats, ModelStat{Name: model, Count: count})
    }
    sort.Slice(stats.ModelStats, func(i, j int) bool {
        return stats.ModelStats[i].Count > stats.ModelStats[j].Count
    })

    // Calculate average
    if stats.Total > 0 {
        stats.AverageRuntime = stats.TotalRuntime / time.Duration(stats.Total)
    }

    return stats
}

func printStats(stats Stats) {
    bold := color.New(color.Bold)

    bold.Println("Agent Statistics")
    fmt.Println("─────────────────────────────────")
    fmt.Println()

    bold.Println("Status Summary")
    green := color.New(color.FgGreen)
    yellow := color.New(color.FgYellow)
    red := color.New(color.FgRed)

    fmt.Print("  Running:    ")
    green.Println(stats.Running)
    fmt.Print("  Paused:     ")
    yellow.Println(stats.Paused)
    fmt.Print("  Terminated: ")
    red.Println(stats.Terminated)
    fmt.Printf("  Total:      %d\n", stats.Total)
    fmt.Println()

    bold.Println("Iterations")
    fmt.Printf("  Completed: %d\n", stats.IterationsCompleted)
    fmt.Printf("  Total:     %d\n", stats.IterationsTotal)
    fmt.Println()

    if len(stats.PromptStats) > 0 {
        bold.Println("Top Prompts (by run count)")
        limit := 5
        if len(stats.PromptStats) < limit {
            limit = len(stats.PromptStats)
        }
        for i := 0; i < limit; i++ {
            ps := stats.PromptStats[i]
            fmt.Printf("  %-16s %d runs  (%d iterations)\n", ps.Name, ps.RunCount, ps.Iterations)
        }
        fmt.Println()
    }

    if len(stats.ModelStats) > 0 {
        bold.Println("Models Used")
        for _, ms := range stats.ModelStats {
            fmt.Printf("  %-30s %d agents\n", ms.Name, ms.Count)
        }
        fmt.Println()
    }

    bold.Println("Runtime")
    fmt.Printf("  Total:   %s\n", formatDuration(stats.TotalRuntime))
    fmt.Printf("  Average: %s per agent\n", formatDuration(stats.AverageRuntime))
}

func formatDuration(d time.Duration) string {
    hours := int(d.Hours())
    minutes := int(d.Minutes()) % 60
    
    if hours > 0 {
        return fmt.Sprintf("%dh %dm", hours, minutes)
    }
    return fmt.Sprintf("%dm", minutes)
}

func init() {
    statsCmd.Flags().StringVar(&statsFormat, "format", "", "Output format: json or table (default)")
    rootCmd.AddCommand(statsCmd)
}
```

## Edge cases

1. **No agents**: Show all zeros with a message "No agents found. Run `swarm run` to start an agent."

2. **All agents pruned**: Same as no agents - all stats are zero.

3. **Single agent**: Stats still display correctly, even if some sections look sparse.

4. **Very long prompt names**: Truncate prompt names to fit in the fixed-width column (16 chars suggested).

5. **Unknown/empty model**: Group under "(unknown)" or skip from model stats.

6. **Runtime calculation without end time**: Currently `AgentState` doesn't track when an agent terminated. For terminated agents, we can estimate based on start time + iterations, or track actual end time (separate improvement).

## Future enhancements (out of scope for this feature)

1. **Time-based filtering**: `swarm stats --since 7d` to show stats for last 7 days
2. **Track actual end time**: Add `EndedAt` field to `AgentState` for accurate runtime stats
3. **Success/failure tracking**: Track whether iterations completed successfully
4. **Disk usage**: Show total size of log files

## Acceptance criteria

- `swarm stats` shows aggregate statistics for agents in current project
- `swarm stats --global` shows statistics across all projects
- `swarm stats --format json` outputs machine-readable JSON
- Status counts (running/paused/terminated) are accurate
- Iteration counts (completed/total) are accurate
- Prompt stats show usage frequency sorted by run count
- Model stats show distribution across agents
- Works correctly when no agents exist (shows zeros, no error)
- Works correctly with only running/only terminated agents
