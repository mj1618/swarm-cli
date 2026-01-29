# Add `@last` special identifier for quick agent access

## Completion Notes

**Completed by:** cd59a862
**Date:** 2026-01-28

### Implementation Summary

1. Added `GetLast()` method to `internal/state/manager.go` that returns the most recently started agent respecting scope settings.

2. Added helper functions in `cmd/root.go`:
   - `IsLastIdentifier()` - checks if identifier is `@last` or `_`
   - `ResolveAgentIdentifier()` - resolves identifiers including special ones

3. Updated all commands to use `ResolveAgentIdentifier()`:
   - logs, inspect, stop, start, kill, restart, update, rm

4. Updated help text for all commands to document the `@last` and `_` special identifiers.

5. Added tests for `GetLast()` in `internal/state/manager_test.go`.

All acceptance criteria met. Code compiles and tests pass.

---

## Problem

When working with swarm, users frequently want to interact with the most recently started agent. Currently, this requires either:
1. Running `swarm list` to find the agent ID, then copying it
2. Remembering the name they assigned to the agent

This is friction-heavy for common workflows:

```bash
# Current workflow - tedious
swarm run -p coder -n 10 -d
# Agent started: f8e2a4b6

# ... some time later ...
swarm list                    # Find the ID
swarm logs f8e2a4b6 -f        # Copy-paste ID
swarm inspect f8e2a4b6        # Copy-paste again
swarm stop f8e2a4b6           # And again
```

The docker CLI has a similar gap, but `docker ps -lq` combined with shell substitution helps. Swarm could do better with a built-in shorthand.

## Solution

Add a special identifier `@last` (and the shorter alias `_`) that can be used anywhere a process ID or name is accepted. It resolves to the most recently started agent.

### Proposed Usage

```bash
# Start an agent
swarm run -p coder -n 10 -d

# Now interact with it easily
swarm logs @last -f          # Follow logs of most recent agent
swarm logs _ -f              # Same thing with shorter alias

swarm inspect @last          # Inspect most recent agent
swarm stop _                 # Stop most recent agent
swarm kill @last             # Kill most recent agent
swarm restart _              # Restart most recent agent

# Works with all commands that accept agent identifiers
swarm update @last --iterations 20
swarm rm _
```

### Scope awareness

The `@last` identifier respects the scope flags:
```bash
# Most recent agent in current project (default)
swarm logs @last

# Most recent agent globally
swarm logs @last -g
```

### Status filtering

By default, `@last` finds the most recently **started** agent regardless of status:
```bash
# Most recent agent (could be running or terminated)
swarm logs @last

# For commands that require running agents (stop, pause, etc.),
# the command itself will error if the agent isn't running
swarm stop @last  # Error: agent is not running (status: terminated)
```

## Files to change

- `internal/state/manager.go` - add `GetLast()` method
- `cmd/root.go` - add helper function `ResolveAgentIdentifier()`
- All commands that use `mgr.GetByNameOrID()`:
  - `cmd/logs.go`
  - `cmd/inspect.go`
  - `cmd/stop.go`
  - `cmd/start.go`
  - `cmd/kill.go`
  - `cmd/restart.go`
  - `cmd/update.go`
  - `cmd/rm.go`

## Implementation details

### internal/state/manager.go

Add a new method to get the most recently started agent:

```go
// GetLast returns the most recently started agent.
// Returns nil if no agents exist.
func (m *Manager) GetLast() (*AgentState, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    state, err := m.load()
    if err != nil {
        if os.IsNotExist(err) {
            return nil, fmt.Errorf("no agents found")
        }
        return nil, err
    }

    var latest *AgentState
    for _, agent := range state.Agents {
        // Filter by scope
        if m.scope == scope.ScopeProject && agent.WorkingDir != m.workingDir {
            continue
        }
        if latest == nil || agent.StartedAt.After(latest.StartedAt) {
            latest = agent
        }
    }

    if latest == nil {
        return nil, fmt.Errorf("no agents found")
    }

    return latest, nil
}
```

### cmd/root.go

Add a helper function used by all commands:

```go
// IsLastIdentifier returns true if the identifier refers to the most recent agent.
func IsLastIdentifier(identifier string) bool {
    return identifier == "@last" || identifier == "_"
}

// ResolveAgentIdentifier resolves an agent identifier to an AgentState.
// Handles special identifiers like "@last" and "_".
func ResolveAgentIdentifier(mgr *state.Manager, identifier string) (*state.AgentState, error) {
    if IsLastIdentifier(identifier) {
        agent, err := mgr.GetLast()
        if err != nil {
            return nil, fmt.Errorf("no recent agent found: %w", err)
        }
        return agent, nil
    }
    return mgr.GetByNameOrID(identifier)
}
```

### Command changes (example: cmd/logs.go)

Replace direct `GetByNameOrID` calls with `ResolveAgentIdentifier`:

```go
// Before:
agent, err := mgr.GetByNameOrID(agentIdentifier)
if err != nil {
    return fmt.Errorf("agent not found: %w", err)
}

// After:
agent, err := ResolveAgentIdentifier(mgr, agentIdentifier)
if err != nil {
    return fmt.Errorf("agent not found: %w", err)
}
```

### Update command documentation

Each command's help text should mention the special identifiers:

```go
Long: `View the log output of a detached agent.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent`,
```

## Edge cases

1. **No agents exist**: Return clear error "no agents found" rather than cryptic nil/empty error.

2. **All agents pruned**: Same as no agents - clear error message.

3. **Most recent is terminated**: `@last` still resolves to it. Commands that require running agents (like `stop`) will give their normal "agent is not running" error.

4. **Conflicting agent named "@last" or "_"**: Explicit ID/name lookup takes precedence. Users probably shouldn't name agents these, but if they do, use the full agent ID to access them.

5. **Multiple agents started at exact same time**: Take whichever one comes last in iteration order (effectively arbitrary, but deterministic).

6. **Global vs project scope**: `@last` respects the `-g` flag, finding the most recent agent in the appropriate scope.

7. **Used in commands that take multiple identifiers**: Each `@last` resolves independently to the same agent. For example, if a future command supports `swarm diff _ abc123`, both would work correctly.

## Examples

### Rapid iteration workflow
```bash
# Start working on a feature
swarm run -p coder -d -n 5

# Check how it's going
swarm logs _ -f

# Hmm, need more iterations
swarm update _ -n 10

# Actually, let's use a different model
swarm update _ -m claude-sonnet-4-20250514

# Done for now, stop it
swarm stop _
```

### Quick inspection
```bash
swarm run -p planner -d
# Started agent: abc12345

# Quickly check on it
swarm inspect _

# Agent Details
# ─────────────────────────────────
# ID:            abc12345
# ...
```

### Error handling
```bash
# No agents running
swarm logs _
# Error: no recent agent found: no agents found

# Agent is terminated
swarm stop @last
# Error: agent is not running (status: terminated)
```

## Acceptance criteria

- `@last` and `_` resolve to the most recently started agent
- Works with all commands that accept agent identifiers: logs, inspect, stop, start, kill, restart, update, rm
- Respects `-g` (global) flag for scope
- Clear error message when no agents exist
- Commands that require running agents still validate status after resolving `@last`
- Help text updated to document the special identifiers
- If an agent is literally named "@last" or "_", the explicit name match takes precedence (documented edge case)
