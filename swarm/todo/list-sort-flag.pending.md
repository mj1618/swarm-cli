# Add `--sort` flag to `swarm list` command

## Problem

The `swarm list` command always displays agents sorted by `StartedAt` (oldest first). When managing many agents, users often want to sort by different criteria to quickly find what they're looking for:

1. **By status** - Group running, paused, and terminated agents together
2. **By cost** - Find the most expensive agents
3. **By name** - Alphabetical ordering for easier scanning
4. **By iterations** - See which agents have run the most iterations

Currently users must resort to piping through `sort`, which breaks table formatting and requires knowledge of column positions:

```bash
# Current workaround (fragile, loses formatting)
swarm list -a | sort -k6    # sort by status column (position depends on flags)
```

## Solution

Add a `--sort` / `-S` flag to `swarm list` that controls the sort field and direction. Default remains `started` (oldest first) for backward compatibility.

### Proposed API

```bash
# Sort by start time (default, oldest first)
swarm list --sort started
swarm list -S started

# Sort by status (running first, then paused, then terminated)
swarm list --sort status

# Sort by name (alphabetical)
swarm list --sort name

# Sort by model (alphabetical)
swarm list --sort model

# Sort by cost (highest first)
swarm list --sort cost

# Sort by iterations (highest first)
swarm list --sort iterations

# Reverse sort direction
swarm list --sort started --reverse
swarm list -S cost --reverse

# Works with existing flags
swarm list -a --sort cost              # all agents sorted by cost
swarm list --sort status --format json # JSON output sorted by status
swarm list --sort name -q              # IDs sorted by agent name
swarm list --sort cost --last 5        # top 5 most expensive agents
```

### Sort fields

| Field | Sort order (default) | Description |
|---|---|---|
| `started` | Oldest first | Sort by agent start time (current default behavior) |
| `status` | running > pausing > paused > terminated | Group by effective status |
| `name` | A-Z (case-insensitive) | Sort alphabetically by agent name |
| `model` | A-Z (case-insensitive) | Sort alphabetically by model name |
| `cost` | Highest first | Sort by total cost (USD) descending |
| `iterations` | Highest first | Sort by current iteration count descending |

The `--reverse` flag inverts the default sort direction for any field.

## Files to change

- `cmd/list.go` - Add `--sort` and `--reverse` flags, implement sorting logic

## Implementation details

### cmd/list.go changes

```go
// Add new variables
var listSort string
var listReverse bool

// Add sorting function
func sortAgents(agents []*state.AgentState, sortField string, reverse bool) {
    sort.SliceStable(agents, func(i, j int) bool {
        var less bool
        switch sortField {
        case "started":
            less = agents[i].StartedAt.Before(agents[j].StartedAt)
        case "status":
            less = statusOrder(agents[i]) < statusOrder(agents[j])
        case "name":
            less = strings.ToLower(agents[i].Name) < strings.ToLower(agents[j].Name)
        case "model":
            less = strings.ToLower(agents[i].Model) < strings.ToLower(agents[j].Model)
        case "cost":
            less = agents[i].TotalCost > agents[j].TotalCost // Descending by default
        case "iterations":
            less = agents[i].CurrentIter > agents[j].CurrentIter // Descending by default
        default:
            less = agents[i].StartedAt.Before(agents[j].StartedAt)
        }
        if reverse {
            return !less
        }
        return less
    })
}

// statusOrder returns a numeric rank for effective status for sorting purposes.
// Lower values sort first: running=0, pausing=1, paused=2, terminated=3
func statusOrder(a *state.AgentState) int {
    if a.Status == "running" {
        if a.Paused {
            if a.PausedAt != nil {
                return 2 // paused
            }
            return 1 // pausing
        }
        return 0 // running
    }
    return 3 // terminated
}
```

Insert the sort call after filtering and before the `--last` limit:

```go
// Apply filters
agents = filterAgents(agents, listName, listPrompt, listModel, listStatus, labelFilters)

// Apply sorting
sortAgents(agents, listSort, listReverse)

// Apply --last limit (agents are now sorted, so we want last N)
if listLast > 0 && len(agents) > listLast {
    agents = agents[len(agents)-listLast:]
}
```

Add flag registration in `init()`:

```go
// Sort flags
listCmd.Flags().StringVarP(&listSort, "sort", "S", "started", "Sort by: started, status, name, model, cost, iterations")
listCmd.Flags().BoolVar(&listReverse, "reverse", false, "Reverse sort order")
```

Add validation in `RunE`:

```go
// Validate sort field
validSorts := []string{"started", "status", "name", "model", "cost", "iterations"}
isValidSort := false
for _, s := range validSorts {
    if listSort == s {
        isValidSort = true
        break
    }
}
if !isValidSort {
    return fmt.Errorf("invalid sort field %q: must be one of 'started', 'status', 'name', 'model', 'cost', or 'iterations'", listSort)
}
```

Update Long help text and examples to document the new flags.

## Edge cases

1. **Invalid sort field**: Return error with list of valid options.
2. **Sort with --last**: Sort applies first, then `--last` takes the last N from the sorted list. This means `--sort cost --last 5` gives the 5 most expensive agents.
3. **Sort with --latest**: Same as `--last 1` - returns the single top result after sorting.
4. **Sort with --count**: Sort has no effect on count output (still just returns count).
5. **Sort with --quiet**: Sort applies before emitting IDs, so IDs come out in sorted order.
6. **Sort with --format json**: JSON array is ordered by the specified sort.
7. **Empty name sort**: Agents without names (empty string) sort before named agents in alphabetical order.
8. **Equal values**: `sort.SliceStable` preserves original order for equal elements, so agents with the same cost/status/etc. remain in StartedAt order.
9. **Default behavior unchanged**: When `--sort` is not specified, it defaults to "started" which is the current behavior.
10. **--reverse with --last**: The `--last` logic should still take from the tail after sorting. The existing reversal for `--last` display should be removed when an explicit `--sort` is used, since the user is controlling order.

## Use cases

### Finding expensive agents

```bash
# Which agents cost the most?
swarm list -a --sort cost

# Top 3 most expensive
swarm list -a --sort cost --last 3
```

### Grouping by status

```bash
# See all agents grouped by status
swarm list -a --sort status
```

### Alphabetical browsing

```bash
# Browse agents by name
swarm list --sort name

# Browse by model to see what's running on each model
swarm list --sort model
```

### Scripting

```bash
# Get ID of the most expensive running agent
swarm list -q --sort cost --latest

# Kill the 3 agents with the most iterations
swarm list -q --sort iterations --last 3 | xargs -n1 swarm kill
```

## Testing considerations

- Test each sort field produces correct ordering
- Test `--reverse` inverts direction for each field
- Test sort combined with `--last` and `--latest`
- Test sort combined with filters (`--status`, `--model`, etc.)
- Test sort with `-q`, `--format json`, `--count`
- Test invalid sort field error
- Test default sort (no flag) matches current behavior
- Test stability: equal values preserve original StartedAt ordering

## Acceptance criteria

- `swarm list --sort started` sorts by start time (oldest first, same as default)
- `swarm list --sort status` groups by status: running, pausing, paused, terminated
- `swarm list --sort name` sorts alphabetically by agent name
- `swarm list --sort model` sorts alphabetically by model
- `swarm list --sort cost` sorts by total cost (highest first)
- `swarm list --sort iterations` sorts by current iteration (highest first)
- `--reverse` inverts the sort direction for any field
- Invalid sort field returns helpful error with valid options
- Default behavior (no `--sort` flag) is unchanged
- Sort works correctly with all existing flags (`-a`, `-q`, `--format json`, `--last`, `--latest`, filters)
- `sort.SliceStable` is used to preserve sub-ordering for equal values
