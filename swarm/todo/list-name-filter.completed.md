# Add `--name` filter option to `swarm list`

## Problem

The `swarm list` command has filter options for `--prompt`, `--model`, and `--status`, but there's no way to filter by agent name. This is inconsistent because:

1. Users can assign custom names to agents via `swarm run --name my-agent`
2. The list output includes a NAME column
3. Users can reference agents by name in other commands (`swarm kill my-agent`, `swarm logs my-agent`)

Users who run many agents with consistent naming conventions (e.g., `coder-frontend`, `coder-backend`, `reviewer-api`) cannot easily filter their list to find related agents.

## Solution

Add a `--name` / `-N` flag to the `list` command that filters agents by name using substring matching (case-insensitive), consistent with the existing `--prompt` and `--model` filters.

### Proposed API

```bash
# Filter by name (substring match, case-insensitive)
swarm list --name coder
swarm list -N frontend

# Combine with other filters
swarm list --name coder --status running
swarm list -N api --model sonnet -a
```

### Example output

```bash
$ swarm list --name coder
ID          NAME             PROMPT               MODEL              STATUS        ITERATION   STARTED
abc12345    coder-frontend   frontend-task        opus               running       3/10        15m ago
def67890    coder-backend    backend-task         sonnet             running       5/10        20m ago
```

## Files to change

- `cmd/list.go` - Add the `--name` flag and update filtering logic

## Implementation details

### cmd/list.go

1. Add the flag variable:

```go
var listName string
```

2. Update the command Long description to document the new filter:

```go
Long: `List running agents with their status and configuration.

...

Filter options:
  --name, -N      Filter by agent name (substring match, case-insensitive)
  --prompt, -p    Filter by prompt name (substring match, case-insensitive)
  --model, -m     Filter by model name (substring match, case-insensitive)
  --status        Filter by status (running, pausing, paused, or terminated)

Multiple filters are combined with AND logic (all conditions must match).`,
```

3. Add examples to the Example field:

```go
  # Filter by name
  swarm list --name coder
  swarm list -N frontend

  # Combine name with other filters
  swarm list --name coder --status running
```

4. Update the `filterAgents` function signature and implementation:

```go
// filterAgents applies name, prompt, model, and status filters to the agent list.
// All non-empty filters must match (AND logic).
func filterAgents(agents []*state.AgentState, nameFilter, promptFilter, modelFilter, statusFilter string) []*state.AgentState {
    if nameFilter == "" && promptFilter == "" && modelFilter == "" && statusFilter == "" {
        return agents
    }

    nameFilter = strings.ToLower(nameFilter)
    promptFilter = strings.ToLower(promptFilter)
    modelFilter = strings.ToLower(modelFilter)
    statusFilter = strings.ToLower(statusFilter)

    var filtered []*state.AgentState
    for _, agent := range agents {
        // Check name filter (substring, case-insensitive)
        if nameFilter != "" && !strings.Contains(strings.ToLower(agent.Name), nameFilter) {
            continue
        }

        // Check prompt filter (substring, case-insensitive)
        if promptFilter != "" && !strings.Contains(strings.ToLower(agent.Prompt), promptFilter) {
            continue
        }

        // ... rest of existing filters ...
    }

    return filtered
}
```

5. Update the call to `filterAgents`:

```go
agents = filterAgents(agents, listName, listPrompt, listModel, listStatus)
```

6. Update the helpful hints check:

```go
if len(agents) == 0 && (listName != "" || listPrompt != "" || listModel != "" || listStatus != "") {
```

7. Register the flag in `init()`:

```go
listCmd.Flags().StringVarP(&listName, "name", "N", "", "Filter by agent name (substring match)")
```

Note: Using `-N` (uppercase) to avoid conflict with other potential short flags and to match the `run` command's `-N` flag for naming agents.

## Edge cases

1. **Empty name field**: Agents without names (Name == "") will not match any name filter except empty string. This is correct behavior.

2. **Name vs ID confusion**: The filter only matches against the Name field, not the ID. This is intentional - users should use the ID directly if they want a specific agent.

3. **Multiple filters**: Works with AND logic like other filters. `--name coder --status running` returns agents that match BOTH conditions.

4. **Case insensitivity**: Uses `strings.ToLower` for case-insensitive matching, consistent with existing filters.

## Tests to add

Add test cases to `cmd/list_test.go`:

```go
func TestFilterAgents_ByName(t *testing.T) {
    agents := []*state.AgentState{
        {ID: "1", Name: "coder-frontend", Prompt: "task1", Model: "opus", Status: "running"},
        {ID: "2", Name: "coder-backend", Prompt: "task2", Model: "sonnet", Status: "running"},
        {ID: "3", Name: "reviewer", Prompt: "task3", Model: "opus", Status: "running"},
        {ID: "4", Name: "", Prompt: "task4", Model: "opus", Status: "running"}, // no name
    }

    // Test name filter
    filtered := filterAgents(agents, "coder", "", "", "")
    if len(filtered) != 2 {
        t.Errorf("expected 2 agents, got %d", len(filtered))
    }

    // Test case insensitivity
    filtered = filterAgents(agents, "CODER", "", "", "")
    if len(filtered) != 2 {
        t.Errorf("expected 2 agents with case-insensitive match, got %d", len(filtered))
    }

    // Test combined filters
    filtered = filterAgents(agents, "coder", "", "opus", "")
    if len(filtered) != 1 {
        t.Errorf("expected 1 agent matching name AND model, got %d", len(filtered))
    }

    // Test no match for empty names
    filtered = filterAgents(agents, "nonexistent", "", "", "")
    if len(filtered) != 0 {
        t.Errorf("expected 0 agents, got %d", len(filtered))
    }
}
```

## Acceptance criteria

- `swarm list --name coder` filters agents whose name contains "coder" (case-insensitive)
- `swarm list -N frontend` works as short form
- Name filter combines with other filters using AND logic
- Empty name fields in agents don't cause errors
- Help text documents the new filter option
- Existing tests continue to pass
- New tests cover name filtering

---

## Completion Notes (Agent cd59a862)

**Completed on:** 2025-01-28

**Files modified:**
- `cmd/list.go` - Added `listName` flag variable, updated Long description and Example sections, modified `filterAgents` function to accept and apply name filter, updated the call site and helpful hints check, registered the `--name`/`-N` flag
- `cmd/list_test.go` - Updated existing test calls to use new 4-parameter `filterAgents` signature, added comprehensive `TestFilterAgentsByName` test function covering substring matching, case insensitivity, combined filters, no-match scenarios, and empty name handling

**All acceptance criteria met:**
- `swarm list --name coder` filters by name (substring, case-insensitive) ✓
- `swarm list -N frontend` works as short form ✓
- Name filter combines with other filters using AND logic ✓
- Empty name fields don't cause errors ✓
- Help text documents the new filter option ✓
- All existing tests pass ✓
- New tests cover name filtering ✓
