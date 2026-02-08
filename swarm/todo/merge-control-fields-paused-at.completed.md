# Fix: `mergeControlFields` should preserve `PausedAt`

## Problem

The `mergeControlFields()` function in `internal/state/manager.go` preserves control signal fields from disk state when the runner calls `MergeUpdate()`. It correctly preserves `Paused`, `TerminateMode`, `Iterations`, and `Model`, but does NOT preserve `PausedAt`.

This causes a race condition:

1. User runs `swarm stop <agent>` which calls `SetPaused(id, true)` — sets both `Paused=true` and `PausedAt=<timestamp>` on disk
2. The runner's usage callback fires during the current iteration and calls `MergeUpdate(agentState)` 
3. `mergeControlFields` copies `Paused=true` from disk (correct), but the in-memory `agentState.PausedAt` is `nil`
4. `MergeUpdate` writes the full agent state to disk with `Paused=true` but `PausedAt=nil`
5. The `stop` command's wait loop polls `agent.PausedAt != nil` to detect actual pause — it never sees `PausedAt` set because it was wiped

In practice, the runner loop sets its own `PausedAt` when it detects `Paused=true` between iterations (loop.go ~line 178), so pause still works. But `PausedAt` gets wiped between `SetPaused()` and the runner detecting the flag, making the `stop` command's wait loop unreliable.

## Solution

Add `PausedAt` to the `mergeControlFields` function:

```go
func mergeControlFields(existing, agent *AgentState) {
    agent.Iterations = existing.Iterations
    agent.Model = existing.Model
    agent.TerminateMode = existing.TerminateMode
    agent.Paused = existing.Paused
    agent.PausedAt = existing.PausedAt  // ADD THIS
}
```

## Relevant Files

- `internal/state/manager.go` — `mergeControlFields()` (~line 287)
- `internal/runner/loop.go` — pause detection (~line 174), `MergeUpdate` calls throughout
- `cmd/stop.go` — wait loop checking `PausedAt` (~line 151)
