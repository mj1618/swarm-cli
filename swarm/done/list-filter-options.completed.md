# Add filter options to `swarm list` command

## Problem

Users running many agents have no way to filter the `swarm list` output. Currently, `swarm list` shows all agents (optionally including terminated ones with `-a`), but there's no way to narrow down results by:

1. **Prompt name** - "Show me only agents running the 'coder' prompt"
2. **Model** - "Show me only agents using claude-sonnet-4"
3. **Status** - "Show me only paused agents" or "Show me only running agents"

This becomes unwieldy when managing many agents across a project. Users currently need to pipe through `grep`, losing the nice table formatting:

```bash
# Current workaround (loses formatting, error-prone)
swarm list -a | grep coder
swarm list -a | grep paused
```

## Solution

Add filter flags to `swarm list` that filter results before display. Filters are combined with AND logic (all conditions must match).

### Proposed API

```bash
# Filter by prompt name (exact match or substring)
swarm list --prompt coder
swarm list -p coder

# Filter by model (substring match)
swarm list --model sonnet
swarm list -m claude-opus

# Filter by status
swarm list --status running
swarm list --status paused
swarm list --status terminated

# Combine filters (AND logic)
swarm list --prompt coder --model sonnet
swarm list --status paused --prompt planner

# Works with existing flags
swarm list -a --prompt coder          # include terminated
swarm list -g --model sonnet          # global scope
swarm list -q --status running        # quiet mode (IDs only)
swarm list --format json --prompt x   # JSON output
```

### Filter behavior

- **--prompt**: Substring match (case-insensitive). `--prompt cod` matches "coder", "decoder", etc.
- **--model**: Substring match (case-insensitive). `--model sonnet` matches "claude-sonnet-4-20250514".
- **--status**: Exact match. Values: `running`, `paused`, `terminated`. Note: `paused` is a subset of `running` (agent is running but paused between iterations).

## Files to change

- `cmd/list.go` - Add filter flags and filtering logic

## Implementation details

### cmd/list.go changes

```go
package cmd

// Add new filter variables
var (
    listAll     bool
    listQuiet   bool
    listFormat  string
    listPrompt  string  // NEW
    listModel   string  // NEW
    listStatus  string  // NEW
)

var listCmd = &cobra.Command{
    Use:     "list",
    Aliases: []string{"ps", "ls"},
    Short:   "List running agents",
    Long: `List running agents with their status and configuration.

By default, only shows running agents started in the current directory.
Use --all to include terminated agents.
Use --global to show agents from all directories.

Filter options:
  --prompt, -p    Filter by prompt name (substring match)
  --model, -m     Filter by model name (substring match)
  --status        Filter by status (running, paused, terminated)`,
    Example: `  # List running agents in current project
  swarm list

  # List all agents (including terminated) in current project
  swarm list -a

  # Filter by prompt name
  swarm list --prompt coder
  swarm list -p planner

  # Filter by model
  swarm list --model sonnet
  swarm list -m opus

  # Filter by status
  swarm list --status paused
  swarm list --status terminated -a

  # Combine filters
  swarm list --prompt coder --model sonnet
  swarm list -a --status terminated --prompt planner

  # With other flags
  swarm list -q --status running    # just IDs of running agents
  swarm list --format json -p coder # JSON output filtered by prompt`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... existing setup code ...

        agents, err := mgr.List(onlyRunning)
        if err != nil {
            return fmt.Errorf("failed to list agents: %w", err)
        }

        // Apply filters
        agents = filterAgents(agents, listPrompt, listModel, listStatus)

        // ... rest of existing code ...
    },
}

// filterAgents applies prompt, model, and status filters to the agent list.
// All non-empty filters must match (AND logic).
func filterAgents(agents []*state.AgentState, promptFilter, modelFilter, statusFilter string) []*state.AgentState {
    if promptFilter == "" && modelFilter == "" && statusFilter == "" {
        return agents
    }

    promptFilter = strings.ToLower(promptFilter)
    modelFilter = strings.ToLower(modelFilter)
    statusFilter = strings.ToLower(statusFilter)

    var filtered []*state.AgentState
    for _, agent := range agents {
        // Check prompt filter (substring, case-insensitive)
        if promptFilter != "" && !strings.Contains(strings.ToLower(agent.Prompt), promptFilter) {
            continue
        }

        // Check model filter (substring, case-insensitive)
        if modelFilter != "" && !strings.Contains(strings.ToLower(agent.Model), modelFilter) {
            continue
        }

        // Check status filter (exact match for running/terminated, special handling for paused)
        if statusFilter != "" {
            effectiveStatus := agent.Status
            if agent.Status == "running" && agent.Paused {
                effectiveStatus = "paused"
            }
            if strings.ToLower(effectiveStatus) != statusFilter {
                continue
            }
        }

        filtered = append(filtered, agent)
    }

    return filtered
}

func init() {
    listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "Show all agents including terminated")
    listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "Only display agent IDs")
    listCmd.Flags().StringVar(&listFormat, "format", "", "Output format: json or table (default)")
    
    // NEW filter flags
    listCmd.Flags().StringVarP(&listPrompt, "prompt", "p", "", "Filter by prompt name (substring match)")
    listCmd.Flags().StringVarP(&listModel, "model", "m", "", "Filter by model name (substring match)")
    listCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status: running, paused, or terminated")
}
```

## Edge cases

1. **No matches**: Display "No agents found matching filters." instead of the generic "No agents found" message.

2. **--status without -a**: If user filters `--status terminated` without `-a`, they'll get no results since `List(true)` excludes terminated. Show a hint: "No agents found. Use -a to include terminated agents."

3. **Invalid status value**: Return error: "invalid status filter: must be one of 'running', 'paused', or 'terminated'"

4. **Empty filter value**: `--prompt ""` should be treated same as no filter (not filter for empty prompt).

5. **Combining with -q (quiet mode)**: Filters should apply before outputting IDs. `swarm list -q --status running` outputs only IDs of running agents.

6. **JSON output**: Filters apply to the JSON output as well - only matching agents are included.

## Use cases

### Managing agents by task type

```bash
# See all coding agents
swarm list -a --prompt coder

# Check status of all planner agents
swarm list --prompt planner

# Find which agents are using the expensive model
swarm list --model opus
```

### Scripting and automation

```bash
# Get IDs of all paused agents for batch resume
for id in $(swarm list -q --status paused); do
    swarm start $id
done

# Kill all agents using a specific prompt
swarm list -q --prompt buggy-prompt | xargs -n1 swarm kill
```

### Quick status check

```bash
# Are any agents paused?
swarm list --status paused

# How many terminated agents do I have?
swarm list -aq --status terminated | wc -l
```

## Testing considerations

- Test each filter individually
- Test filter combinations (AND logic)
- Test filters with -a, -q, --format json, -g flags
- Test case insensitivity
- Test substring matching
- Test invalid status value
- Test when no agents match filters
- Test --status paused correctly matches running agents that are paused

## Acceptance criteria

- `swarm list --prompt coder` shows only agents whose prompt contains "coder"
- `swarm list --model sonnet` shows only agents whose model contains "sonnet"
- `swarm list --status running` shows only non-paused running agents
- `swarm list --status paused` shows only paused agents
- `swarm list --status terminated -a` shows only terminated agents
- Filters are case-insensitive
- Multiple filters combine with AND logic
- Filters work with -q (quiet), --format json, -a (all), -g (global)
- Helpful error message for invalid status value
- Helpful hint when filtering for terminated without -a flag

---

## Completion Notes (Agent 1a025fb7)

**Completed:** 2026-01-28

**Changes made:**
- Modified `cmd/list.go` to add three new filter flags:
  - `--prompt, -p` for filtering by prompt name (substring, case-insensitive)
  - `--model, -m` for filtering by model name (substring, case-insensitive)
  - `--status` for filtering by status (running, paused, terminated)
- Added `filterAgents()` helper function implementing AND logic for filters
- Added validation for `--status` flag with helpful error message for invalid values
- Added hint message when filtering for `--status terminated` without `-a` flag
- Updated command help text and examples to document the new filters

**Testing performed:**
- All existing tests pass (`go test ./...`)
- Manually tested all filter flags individually and in combination
- Verified filters work with `-q` (quiet), `--format json`, `-a` (all), `-g` (global)
- Verified case-insensitivity of filters
- Verified invalid status value error handling
- Verified helpful hint when filtering terminated without -a
