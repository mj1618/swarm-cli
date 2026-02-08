# Add `swarm cost` command for token usage and cost reporting

## Problem

The `AgentState` already tracks `InputTokens`, `OutputTokens`, and `TotalCost` fields, but there is no way for users to view this information:

1. `swarm inspect` does not display token/cost fields at all (lines 62-176 of `cmd/inspect.go` show every field except tokens and cost)
2. `swarm stats` shows iteration counts and runtime but not cost/token information
3. `swarm list` has no cost column

Users running multiple agents need to understand:
- How much individual agents cost
- Total spending across agents (per project or globally)
- Cost breakdown by model (e.g., opus vs sonnet usage patterns)
- Cost over time periods (today, this week, etc.)
- Which prompts/tasks are most expensive

Without this, users have no visibility into their API spend, which can be significant when running many parallel agents with expensive models.

## Solution

Add a `swarm cost` command that displays token usage and cost information, and also fix `swarm inspect` to show cost fields when available.

### Proposed API

```bash
# Show cost summary for current project
swarm cost

# Show cost for all projects
swarm cost --global

# Show cost for a specific agent
swarm cost abc123

# Show cost for the last 24 hours
swarm cost --since 24h

# Show cost for the last 7 days
swarm cost --since 7d

# Break down cost by model
swarm cost --by model

# Break down cost by prompt
swarm cost --by prompt

# Output as JSON (for dashboards/scripting)
swarm cost --format json
```

### Default output (no args)

```
Cost Report
-----------

Summary
  Total cost:      $12.47
  Input tokens:    2,340,000
  Output tokens:   890,000
  Agents:          15

By Model
  opus              $10.20   (81.8%)   1,800,000 in / 600,000 out   8 agents
  sonnet            $2.27    (18.2%)   540,000 in / 290,000 out     7 agents

By Prompt (top 5)
  planner           $5.40    8 runs
  coder             $4.10    5 runs
  reviewer          $2.97    2 runs

Recent (last 5 agents)
  abc123  planner  opus    $1.20   2h ago
  def456  coder    sonnet  $0.35   3h ago
  ...
```

### Single agent output

```
Cost Report: my-agent (abc123)
------------------------------

  Input tokens:    156,000
  Output tokens:   42,000
  Total cost:      $5.49
  Cost per iter:   $0.55 avg (10 iterations)
  Model:           opus
```

## Files to create/change

- Create `cmd/cost.go` — new cost command implementation
- Edit `cmd/inspect.go` — add token/cost display to inspect output

## Implementation details

### cmd/cost.go

The command should:

1. Accept an optional agent identifier argument (shows single-agent cost if provided)
2. Support `--global` flag via existing `GetScope()` infrastructure
3. Support `--since` time filter using existing `ParseTimeFlag()` from `cmd/logs.go`
4. Support `--by` flag for grouping (values: `model`, `prompt`)
5. Support `--format json` for machine-readable output
6. Use existing `state.NewManagerWithScope()` and `mgr.List()` to get agents
7. Calculate costs from `AgentState.InputTokens`, `OutputTokens`, `TotalCost`
8. Format currency with `$X.XX` and token counts with comma separators

Key data structures:

```go
type CostReport struct {
    TotalCost      float64        `json:"total_cost_usd"`
    InputTokens    int64          `json:"input_tokens"`
    OutputTokens   int64          `json:"output_tokens"`
    AgentCount     int            `json:"agent_count"`
    ModelBreakdown []ModelCost    `json:"model_breakdown"`
    PromptBreakdown []PromptCost  `json:"prompt_breakdown"`
}

type ModelCost struct {
    Model        string  `json:"model"`
    Cost         float64 `json:"cost_usd"`
    Percentage   float64 `json:"percentage"`
    InputTokens  int64   `json:"input_tokens"`
    OutputTokens int64   `json:"output_tokens"`
    AgentCount   int     `json:"agent_count"`
}

type PromptCost struct {
    Prompt     string  `json:"prompt"`
    Cost       float64 `json:"cost_usd"`
    RunCount   int     `json:"run_count"`
}
```

### Changes to cmd/inspect.go

Add a "Token Usage" section after the iteration breakdown, displayed when any token/cost data is non-zero:

```go
// After iteration breakdown section (around line 126)
if agent.InputTokens > 0 || agent.OutputTokens > 0 || agent.TotalCost > 0 {
    fmt.Println()
    bold.Println("Token Usage")
    fmt.Println("---")
    fmt.Printf("Input tokens:  %s\n", formatTokenCount(agent.InputTokens))
    fmt.Printf("Output tokens: %s\n", formatTokenCount(agent.OutputTokens))
    if agent.TotalCost > 0 {
        fmt.Printf("Total cost:    $%.2f\n", agent.TotalCost)
    }
}
```

## Edge cases

1. **No cost data**: Many agents may have zero tokens/cost (e.g., if the backend doesn't report them). Show "No cost data available" or skip cost sections gracefully.

2. **Mixed data**: Some agents have cost data, others don't. Only include agents with non-zero data in calculations.

3. **No agents**: Show all zeros with a helpful message, same pattern as `swarm stats`.

4. **Single agent with no cost data**: Show the agent info but note that no cost data was recorded.

5. **Very large numbers**: Format tokens with commas (e.g., "2,340,000") and costs with 2 decimal places.

6. **Time filter with `--since`**: Reuse `ParseTimeFlag` from logs.go. Filter agents by `StartedAt` time.

## Acceptance criteria

- `swarm cost` shows aggregate cost report for agents in current project
- `swarm cost --global` shows costs across all projects
- `swarm cost <agent-id>` shows cost for a specific agent
- `swarm cost --since 7d` filters to agents started in the last 7 days
- `swarm cost --by model` groups costs by model
- `swarm cost --by prompt` groups costs by prompt
- `swarm cost --format json` outputs machine-readable JSON
- `swarm inspect` now shows token/cost fields when available
- Handles agents with no cost data gracefully (no errors, clear messaging)
- Currency formatted as `$X.XX`, tokens formatted with comma separators
- All existing tests continue to pass
- New command has unit tests
