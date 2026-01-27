# Add ability to rename agents via `swarm update --name`

## Completion Notes

**Completed by agent 1a025fb7 on 2026-01-28**

Implemented the `--name` / `-N` flag in `cmd/update.go`:

- Added `updateName` variable and flag registration
- Name update is handled before the status check to allow renaming terminated agents
- Checks for name conflicts with other running agents
- Skips silently if renaming to the same name
- Empty names are rejected with an error
- Name changes are persisted even if other operations fail due to agent status
- All existing tests pass
- Build succeeds

## Problem

Users can update various properties of a running agent via `swarm update`:
- `--iterations` to change the iteration count
- `--model` to change the model
- `--terminate` / `--terminate-after` for termination control

However, there's no way to rename an agent after it's been started. Users might want to:

1. Give a running agent a more descriptive name after seeing what it's actually doing
2. Fix a typo in the agent name
3. Rename an agent to match a new naming convention they've adopted
4. Distinguish between multiple agents that were auto-named from the same prompt

Currently, agents auto-named from prompts get names like `planner` or `coder`, and if multiple agents run the same prompt, they get suffixes like `planner-2`, `planner-3`. Users have no way to give these more meaningful names like `refactor-auth-module` or `fix-login-bug`.

## Solution

Add a `--name` / `-N` flag to the `swarm update` command that allows renaming an agent.

### Proposed API

```bash
# Rename by agent ID
swarm update abc123 --name new-name

# Rename by current name (using short flag)
swarm update my-agent -N better-name

# Combine with other updates
swarm update abc123 --name "refactoring-task" --iterations 30
```

### Name uniqueness

The same uniqueness rules from `state.Manager.Register()` should apply:
- If the new name conflicts with another running agent, reject with a clear error
- Only consider running agents for conflicts (terminated agents can share names)

## Files to change

- `cmd/update.go` - add `--name` flag and implement rename logic

## Implementation details

### update.go changes

```go
var (
    updateIterations     int
    updateModel          string
    updateName           string  // NEW
    updateTerminate      bool
    updateTerminateAfter bool
)

// In RunE, after fetching the agent, add name update logic:
if cmd.Flags().Changed("name") {
    // Check for name conflicts with other running agents
    allAgents, err := mgr.List(true) // true = only running
    if err != nil {
        return fmt.Errorf("failed to check name availability: %w", err)
    }
    
    for _, other := range allAgents {
        if other.ID != agent.ID && other.Name == updateName {
            return fmt.Errorf("name '%s' is already in use by agent %s", updateName, other.ID)
        }
    }
    
    oldName := agent.Name
    agent.Name = updateName
    updated = true
    fmt.Printf("Renamed agent from '%s' to '%s'\n", oldName, updateName)
}

// In init():
updateCmd.Flags().StringVarP(&updateName, "name", "N", "", "Set new name for the agent")
```

### Output examples

Success:
```
$ swarm update abc123 --name refactor-auth
Renamed agent from 'planner' to 'refactor-auth'
```

Name conflict:
```
$ swarm update abc123 --name my-agent
Error: name 'my-agent' is already in use by agent def456
```

Combined update:
```
$ swarm update abc123 --name "long-running-task" --iterations 100
Renamed agent from 'coder' to 'long-running-task'
Updated iterations to 100
```

## Edge cases

1. **Empty name**: Reject with an error - use case is unclear.

2. **Same name**: If user renames to the same name it already has, skip silently (no error, no message).

3. **Terminated agents**: Allow renaming terminated agents - it doesn't hurt and might help with organization before pruning. Note: the current check `if agent.Status != "running"` should be moved to after the name handling, or name updates should bypass this check.

4. **Name with special characters**: No additional validation needed beyond what the state manager already handles.

## Acceptance criteria

- `swarm update abc123 --name new-name` renames the agent
- `swarm update abc123 -N new-name` works with short flag
- Renaming to a name already used by a running agent fails with clear error
- Renaming to the same name is a no-op (no error)
- Renaming can be combined with other `--iterations` and `--model` flags
- Works with both agent ID and current name as identifier
- Renaming a terminated agent works (not blocked by status check)
- `swarm list` shows the new name after rename
- `swarm inspect new-name` works after rename
