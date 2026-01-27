# Add `swarm summary` command for quick agent run overview

## Problem

When agents complete or terminate, users often want a quick overview of what happened without reading through potentially thousands of lines of logs. Currently, users must:

1. **Read full logs**: `swarm logs my-agent --tail 500` and manually scan for relevant info
2. **Use inspect**: `swarm inspect my-agent` only shows metadata, not what the agent actually did
3. **Grep manually**: `cat ~/.swarm/logs/abc123.log | grep -i error` to find specific patterns

This is time-consuming for users who manage multiple agents or need quick status checks.

**Current workflow:**
```bash
# Agent finished - what did it do?
swarm inspect my-agent      # Just shows config, not results
swarm logs my-agent         # 5000 lines of raw output...
swarm logs my-agent | grep -i "error\|warning\|completed"  # Manual pattern matching
```

**Desired workflow:**
```bash
swarm summary my-agent
# Shows: files changed, errors encountered, key milestones, duration per iteration
```

## Solution

Add a `swarm summary` command that parses agent logs and extracts key information into a concise, actionable overview.

### Proposed API

```bash
# Get summary of a specific agent
swarm summary abc123
swarm summary my-agent

# Summary of most recent agent
swarm summary @last
swarm summary _

# Output as JSON for scripting
swarm summary my-agent --format json

# Include more detail levels
swarm summary my-agent --verbose
```

### Summary output

```
Agent Summary: my-agent (abc123)
───────────────────────────────────────────────────────────────

Status:       terminated
Duration:     45m 23s
Iterations:   20/20 completed

Iteration Breakdown:
  Avg duration:  2m 16s
  Fastest:       1m 02s (iteration 3)
  Slowest:       4m 51s (iteration 12)

Activity Summary:
  Files created:    8
  Files modified:   23
  Files deleted:    2
  Tool calls:       156
  Errors:           2

Errors Encountered:
  [iter 7]  Failed to read file: permission denied
  [iter 15] API rate limit exceeded (retried successfully)

Key Events:
  [iter 1]  Started task: "Implement user authentication"
  [iter 5]  Completed: Created auth middleware
  [iter 10] Completed: Added login/logout endpoints
  [iter 15] Warning: Rate limit hit, backing off
  [iter 20] Completed: All tests passing

Final State:
  Last action: "Committed changes to feature branch"
```

### JSON output

```json
{
  "agent_id": "abc123",
  "agent_name": "my-agent",
  "status": "terminated",
  "duration_seconds": 2723,
  "iterations": {
    "completed": 20,
    "total": 20,
    "avg_duration_seconds": 136,
    "fastest_seconds": 62,
    "slowest_seconds": 291
  },
  "activity": {
    "files_created": 8,
    "files_modified": 23,
    "files_deleted": 2,
    "tool_calls": 156,
    "errors": 2
  },
  "errors": [
    {"iteration": 7, "message": "Failed to read file: permission denied"},
    {"iteration": 15, "message": "API rate limit exceeded (retried successfully)"}
  ],
  "events": [
    {"iteration": 1, "type": "start", "message": "Started task: \"Implement user authentication\""},
    {"iteration": 5, "type": "complete", "message": "Created auth middleware"}
  ]
}
```

## Files to create/change

- Create `cmd/summary.go` - new command implementation
- Create `internal/logsummary/parser.go` - log parsing and summarization logic
- Create `internal/logsummary/parser_test.go` - tests for parser

## Implementation details

### cmd/summary.go

```go
package cmd

import (
    "encoding/json"
    "fmt"

    "github.com/fatih/color"
    "github.com/matt/swarm-cli/internal/logsummary"
    "github.com/matt/swarm-cli/internal/state"
    "github.com/spf13/cobra"
)

var (
    summaryFormat  string
    summaryVerbose bool
)

var summaryCmd = &cobra.Command{
    Use:   "summary [agent-id-or-name]",
    Short: "Show a summary of an agent's run",
    Long: `Display a concise summary of what an agent accomplished.

Parses agent logs to extract key information including:
- Iteration timing and performance
- Files created, modified, and deleted
- Errors and warnings encountered
- Key milestones and events

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent`,
    Example: `  # Summary of agent by ID
  swarm summary abc123

  # Summary of agent by name
  swarm summary my-agent

  # Summary of most recent agent
  swarm summary @last

  # Output as JSON
  swarm summary my-agent --format json

  # More detailed output
  swarm summary my-agent --verbose`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        agentIdentifier := args[0]

        mgr, err := state.NewManagerWithScope(GetScope(), "")
        if err != nil {
            return fmt.Errorf("failed to initialize state manager: %w", err)
        }

        agent, err := ResolveAgentIdentifier(mgr, agentIdentifier)
        if err != nil {
            return fmt.Errorf("agent not found: %w", err)
        }

        if agent.LogFile == "" {
            return fmt.Errorf("agent %s has no log file (was not started in detached mode)", agentIdentifier)
        }

        // Parse logs and generate summary
        summary, err := logsummary.Parse(agent)
        if err != nil {
            return fmt.Errorf("failed to parse logs: %w", err)
        }

        if summaryFormat == "json" {
            output, err := json.MarshalIndent(summary, "", "  ")
            if err != nil {
                return fmt.Errorf("failed to marshal summary: %w", err)
            }
            fmt.Println(string(output))
            return nil
        }

        printSummary(agent, summary, summaryVerbose)
        return nil
    },
}

func printSummary(agent *state.AgentState, s *logsummary.Summary, verbose bool) {
    bold := color.New(color.Bold)
    
    // Header
    name := agent.Name
    if name == "" {
        name = agent.ID
    }
    bold.Printf("Agent Summary: %s (%s)\n", name, agent.ID)
    fmt.Println("───────────────────────────────────────────────────────────────")
    fmt.Println()

    // Status section
    statusColor := color.New(color.FgGreen)
    if agent.Status == "terminated" {
        statusColor = color.New(color.FgRed)
    }
    fmt.Print("Status:       ")
    statusColor.Println(agent.Status)
    fmt.Printf("Duration:     %s\n", s.FormatDuration())
    fmt.Printf("Iterations:   %d/%d completed\n", s.IterationsCompleted, agent.Iterations)
    fmt.Println()

    // Iteration breakdown
    if s.IterationsCompleted > 0 {
        bold.Println("Iteration Breakdown:")
        fmt.Printf("  Avg duration:  %s\n", s.FormatAvgIteration())
        if s.FastestIteration > 0 {
            fmt.Printf("  Fastest:       %s (iteration %d)\n", s.FormatFastestIteration(), s.FastestIterationNum)
        }
        if s.SlowestIteration > 0 {
            fmt.Printf("  Slowest:       %s (iteration %d)\n", s.FormatSlowestIteration(), s.SlowestIterationNum)
        }
        fmt.Println()
    }

    // Activity summary
    bold.Println("Activity Summary:")
    fmt.Printf("  Files created:    %d\n", s.FilesCreated)
    fmt.Printf("  Files modified:   %d\n", s.FilesModified)
    fmt.Printf("  Files deleted:    %d\n", s.FilesDeleted)
    fmt.Printf("  Tool calls:       %d\n", s.ToolCalls)
    fmt.Printf("  Errors:           %d\n", len(s.Errors))
    fmt.Println()

    // Errors
    if len(s.Errors) > 0 {
        errColor := color.New(color.FgRed)
        bold.Println("Errors Encountered:")
        limit := 5
        if verbose {
            limit = len(s.Errors)
        }
        for i, e := range s.Errors {
            if i >= limit {
                fmt.Printf("  ... and %d more errors\n", len(s.Errors)-limit)
                break
            }
            errColor.Printf("  [iter %d]  ", e.Iteration)
            fmt.Println(e.Message)
        }
        fmt.Println()
    }

    // Key events (in verbose mode or if few events)
    if verbose && len(s.Events) > 0 {
        bold.Println("Key Events:")
        for _, e := range s.Events {
            fmt.Printf("  [iter %d]  %s\n", e.Iteration, e.Message)
        }
        fmt.Println()
    }

    // Final state
    if s.LastAction != "" {
        bold.Println("Final State:")
        fmt.Printf("  Last action: %q\n", s.LastAction)
    }
}

func init() {
    summaryCmd.Flags().StringVar(&summaryFormat, "format", "", "Output format: json or table (default)")
    summaryCmd.Flags().BoolVarP(&summaryVerbose, "verbose", "v", false, "Show more detailed output")
    rootCmd.AddCommand(summaryCmd)
}
```

### internal/logsummary/parser.go

```go
package logsummary

import (
    "bufio"
    "fmt"
    "os"
    "regexp"
    "strings"
    "time"

    "github.com/matt/swarm-cli/internal/state"
)

// Summary contains parsed summary information from agent logs.
type Summary struct {
    DurationSeconds     int64   `json:"duration_seconds"`
    IterationsCompleted int     `json:"iterations_completed"`
    
    AvgIterationSeconds   int64 `json:"avg_iteration_seconds"`
    FastestIteration      int64 `json:"fastest_iteration_seconds"`
    FastestIterationNum   int   `json:"fastest_iteration_num"`
    SlowestIteration      int64 `json:"slowest_iteration_seconds"`
    SlowestIterationNum   int   `json:"slowest_iteration_num"`
    
    FilesCreated  int `json:"files_created"`
    FilesModified int `json:"files_modified"`
    FilesDeleted  int `json:"files_deleted"`
    ToolCalls     int `json:"tool_calls"`
    
    Errors []LogError `json:"errors,omitempty"`
    Events []LogEvent `json:"events,omitempty"`
    
    LastAction string `json:"last_action,omitempty"`
}

// LogError represents an error found in logs.
type LogError struct {
    Iteration int    `json:"iteration"`
    Message   string `json:"message"`
}

// LogEvent represents a notable event from logs.
type LogEvent struct {
    Iteration int    `json:"iteration"`
    Type      string `json:"type"`
    Message   string `json:"message"`
}

// Patterns for parsing logs
var (
    iterationStartPattern = regexp.MustCompile(`\[swarm\] === Iteration (\d+)/(\d+) ===`)
    errorPattern          = regexp.MustCompile(`(?i)(error|failed|exception)`)
    toolCallPattern       = regexp.MustCompile(`(?i)tool.*call|calling.*tool|execute`)
    fileWritePattern      = regexp.MustCompile(`(?i)(created?|writ(?:e|ing|ten)|sav(?:e|ing|ed)) (?:file|to) ["`]?([^"` \n]+)`)
    fileEditPattern       = regexp.MustCompile(`(?i)(edit(?:ed|ing)?|modif(?:y|ied|ying)|updat(?:e|ed|ing)) ["`]?([^"` \n]+)`)
    fileDeletePattern     = regexp.MustCompile(`(?i)(delet(?:e|ed|ing)|remov(?:e|ed|ing)) ["`]?([^"` \n]+)`)
)

// Parse reads agent logs and generates a summary.
func Parse(agent *state.AgentState) (*Summary, error) {
    summary := &Summary{
        DurationSeconds:     int64(time.Since(agent.StartedAt).Seconds()),
        IterationsCompleted: agent.CurrentIter,
    }

    if agent.LogFile == "" {
        return summary, nil
    }

    file, err := os.Open(agent.LogFile)
    if err != nil {
        return summary, fmt.Errorf("failed to open log file: %w", err)
    }
    defer file.Close()

    currentIteration := 0
    var iterationStart time.Time
    var iterationDurations []int64
    filesCreated := make(map[string]bool)
    filesModified := make(map[string]bool)
    filesDeleted := make(map[string]bool)
    var lastLine string

    scanner := bufio.NewScanner(file)
    buf := make([]byte, 0, 64*1024)
    scanner.Buffer(buf, 1024*1024)

    for scanner.Scan() {
        line := scanner.Text()
        lastLine = line

        // Track iteration changes
        if matches := iterationStartPattern.FindStringSubmatch(line); len(matches) > 0 {
            // Record duration of previous iteration
            if currentIteration > 0 && !iterationStart.IsZero() {
                dur := extractTimestamp(line).Sub(iterationStart).Seconds()
                if dur > 0 {
                    iterationDurations = append(iterationDurations, int64(dur))
                }
            }
            currentIteration++
            iterationStart = extractTimestamp(line)
        }

        // Count tool calls
        if toolCallPattern.MatchString(line) {
            summary.ToolCalls++
        }

        // Track file operations
        if matches := fileWritePattern.FindStringSubmatch(line); len(matches) > 2 {
            filesCreated[matches[2]] = true
        }
        if matches := fileEditPattern.FindStringSubmatch(line); len(matches) > 2 {
            filesModified[matches[2]] = true
        }
        if matches := fileDeletePattern.FindStringSubmatch(line); len(matches) > 2 {
            filesDeleted[matches[2]] = true
        }

        // Track errors
        if errorPattern.MatchString(line) && currentIteration > 0 {
            // Extract a clean error message (first 100 chars)
            msg := line
            if len(msg) > 100 {
                msg = msg[:100] + "..."
            }
            summary.Errors = append(summary.Errors, LogError{
                Iteration: currentIteration,
                Message:   msg,
            })
        }
    }

    // Calculate file counts (deduplicated)
    summary.FilesCreated = len(filesCreated)
    summary.FilesModified = len(filesModified)
    summary.FilesDeleted = len(filesDeleted)

    // Calculate iteration statistics
    if len(iterationDurations) > 0 {
        var total int64
        summary.FastestIteration = iterationDurations[0]
        summary.FastestIterationNum = 1
        summary.SlowestIteration = iterationDurations[0]
        summary.SlowestIterationNum = 1

        for i, dur := range iterationDurations {
            total += dur
            if dur < summary.FastestIteration {
                summary.FastestIteration = dur
                summary.FastestIterationNum = i + 1
            }
            if dur > summary.SlowestIteration {
                summary.SlowestIteration = dur
                summary.SlowestIterationNum = i + 1
            }
        }
        summary.AvgIterationSeconds = total / int64(len(iterationDurations))
    }

    // Extract last action from final log lines
    if lastLine != "" {
        summary.LastAction = truncate(lastLine, 80)
    }

    return summary, nil
}

func extractTimestamp(line string) time.Time {
    if len(line) < 19 {
        return time.Time{}
    }
    t, _ := time.ParseInLocation("2006-01-02 15:04:05", line[:19], time.Local)
    return t
}

func truncate(s string, max int) string {
    if len(s) <= max {
        return s
    }
    return s[:max-3] + "..."
}

// Formatting helpers

func (s *Summary) FormatDuration() string {
    return formatDuration(time.Duration(s.DurationSeconds) * time.Second)
}

func (s *Summary) FormatAvgIteration() string {
    return formatDuration(time.Duration(s.AvgIterationSeconds) * time.Second)
}

func (s *Summary) FormatFastestIteration() string {
    return formatDuration(time.Duration(s.FastestIteration) * time.Second)
}

func (s *Summary) FormatSlowestIteration() string {
    return formatDuration(time.Duration(s.SlowestIteration) * time.Second)
}

func formatDuration(d time.Duration) string {
    h := int(d.Hours())
    m := int(d.Minutes()) % 60
    s := int(d.Seconds()) % 60

    if h > 0 {
        return fmt.Sprintf("%dh %dm %ds", h, m, s)
    }
    if m > 0 {
        return fmt.Sprintf("%dm %ds", m, s)
    }
    return fmt.Sprintf("%ds", s)
}
```

## Use cases

### Quick check after long-running agent

```bash
# Agent ran overnight - what happened?
swarm summary overnight-task
# See iterations completed, any errors, files changed
```

### Compare agent performance

```bash
# Which model was faster?
swarm summary agent-opus --format json | jq .iterations.avg_duration_seconds
swarm summary agent-sonnet --format json | jq .iterations.avg_duration_seconds
```

### Error triage

```bash
# Agent had issues - quick error overview
swarm summary problematic-agent
# See all errors with their iteration numbers, then dig into specific iterations
swarm logs problematic-agent --since "iteration 7"
```

### Scripting and monitoring

```bash
# Check if agent had errors
if [ $(swarm summary my-agent --format json | jq '.errors | length') -gt 0 ]; then
    echo "Agent encountered errors"
    swarm summary my-agent --verbose
fi
```

## Edge cases

1. **No log file**: Return basic summary from agent state (iterations, duration) with message that detailed log analysis unavailable.

2. **Empty log file**: Return zero counts for all metrics, note that log was empty.

3. **Agent still running**: Summary shows current progress, noting agent is still active.

4. **Very large log files**: Stream-parse without loading entire file into memory. Limit stored events/errors to prevent memory issues.

5. **Non-standard log format**: Best-effort parsing. If patterns don't match, counts may be zero but command shouldn't fail.

6. **Binary content in logs**: Skip lines that appear to be binary, continue parsing.

## Acceptance criteria

- `swarm summary my-agent` shows formatted summary with key metrics
- `swarm summary @last` works with special identifier
- `swarm summary my-agent --format json` outputs valid JSON
- `swarm summary my-agent --verbose` shows additional details (all events, all errors)
- Summary includes: duration, iterations completed, iteration timing stats
- Summary includes: file operation counts (create/modify/delete)
- Summary includes: error count and first N error messages
- Works for both running and terminated agents
- Handles missing log file gracefully
- Handles large log files without excessive memory usage
- Completes in reasonable time (<5s) for typical log files

---

## Completion Notes (Agent cd59a862)

**Completed on:** 2026-01-28

**Implementation:**

1. Created `internal/logsummary/parser.go` with:
   - `Summary` struct containing all summary metrics
   - `Parse()` function that reads logs and extracts summary data
   - Parsing for iteration markers, tool calls, file operations, errors, and events
   - Formatting helpers for durations
   - Stream parsing to handle large log files efficiently
   - Error deduplication to prevent memory issues

2. Created `internal/logsummary/parser_test.go` with comprehensive tests:
   - Empty/missing log file handling
   - Tool call counting and file operation tracking
   - Error extraction and deduplication
   - Git commit and test event tracking
   - Duration calculation for running and terminated agents
   - Format helpers

3. Created `cmd/summary.go` with:
   - Command implementation with `--format json` and `--verbose` flags
   - Human-readable formatted output
   - Support for `@last` and `_` identifiers
   - Tab completion for agent identifiers

4. Registered the command in `cmd/root.go`

**All acceptance criteria met:**
- `swarm summary my-agent` shows formatted summary
- `swarm summary @last` works with special identifier
- `--format json` outputs valid JSON
- `--verbose` shows all errors and events
- Summary includes all specified metrics
- Handles edge cases gracefully
