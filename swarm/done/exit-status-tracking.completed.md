# Track agent exit status and termination details

## Completion Notes

**Completed by agent cd59a862 on 2026-01-28**

Implemented all acceptance criteria:

1. Added new fields to `AgentState` in `internal/state/manager.go`:
   - `TerminatedAt` - When agent stopped
   - `ExitReason` - completed, killed, signal, or crashed
   - `LastError` - Last error message if any
   - `SuccessfulIters` - Count of iterations that completed without error
   - `FailedIters` - Count of iterations that errored

2. Updated `cmd/run.go` to:
   - Track iteration success/failure after each iteration
   - Set `TerminatedAt` and `ExitReason` on termination
   - Set `ExitReason = "killed"` when terminated via TerminateMode
   - Set `ExitReason = "signal"` when receiving SIGINT/SIGTERM

3. Updated `cmd/restart.go` with the same tracking changes

4. Updated `cmd/inspect.go` to display:
   - Terminated timestamp and runtime for terminated agents
   - Exit reason
   - Success/failure counts and success rate
   - Last error message (truncated if very long)

5. Updated `cmd/stats.go` to:
   - Track aggregate successful/failed iterations
   - Calculate and display success rate
   - Use actual `TerminatedAt` for runtime calculations when available

6. Updated `Manager.cleanup()` to set `ExitReason = "crashed"` when detecting crashed processes

All tests pass.

---

## Problem

When agents terminate, valuable information is lost:

1. **No termination timestamp**: `AgentState` tracks `StartedAt` but not when the agent ended. Users can't see how long an agent actually ran (especially for terminated agents).

2. **No exit status**: When an agent is killed, times out, or errors out, there's no record of how it terminated. Looking at `cmd/run.go`:

```go
// Run agent - errors should NOT stop the run
runner := agent.NewRunner(cfg)
if err := runner.Run(os.Stdout); err != nil {
    fmt.Printf("\n[swarm] Agent error (continuing): %v\n", err)
}
```

Errors are printed but not tracked. A user looking at a terminated agent can't tell if it completed successfully or failed.

3. **No failure count**: In multi-iteration runs, we don't track how many iterations succeeded vs failed. This is useful for:
   - Identifying flaky prompts
   - Measuring reliability
   - Debugging intermittent issues

Current `swarm inspect` output:
```
Status:        terminated
Iteration:     10/10
```

There's no indication whether those 10 iterations succeeded or if 8 of them errored out.

## Solution

Extend `AgentState` to track termination details and iteration outcomes.

### State changes

Add these fields to `AgentState` in `internal/state/manager.go`:

```go
type AgentState struct {
    // ... existing fields ...
    
    // Termination tracking
    TerminatedAt     *time.Time `json:"terminated_at,omitempty"`     // When agent stopped
    ExitReason       string     `json:"exit_reason,omitempty"`       // completed, killed, error, signal
    LastError        string     `json:"last_error,omitempty"`        // Last error message if any
    
    // Iteration outcomes
    SuccessfulIters  int        `json:"successful_iterations"`       // Iterations that completed without error
    FailedIters      int        `json:"failed_iterations"`           // Iterations that errored
}
```

### Exit reasons

| Reason | Description |
|--------|-------------|
| `completed` | All iterations finished (success or failure) |
| `killed` | User killed via `swarm kill` |
| `signal` | Process received SIGINT/SIGTERM |
| `error` | Fatal error prevented continuation |

### Updated inspect output

```
Agent Details
─────────────────────────────────
ID:            abc123
Name:          my-agent
Status:        terminated
Started:       2026-01-28T10:00:00Z
Terminated:    2026-01-28T10:15:32Z
Runtime:       15m 32s
Exit reason:   completed

Iterations
─────────────────────────────────
Completed:     10/10
Successful:    8
Failed:        2
Success rate:  80%
```

### Updated list output

Add an optional `--show-exit` flag to show exit info in list:

```
ID        NAME        STATUS       ITERATION  EXIT      RUNTIME
abc123    my-agent    terminated   10/10      completed 15m 32s
def456    worker      terminated   5/10       killed    8m 12s
ghi789    test        terminated   3/10       error     2m 45s
```

## Files to change

- `internal/state/manager.go` - Add new fields to `AgentState`
- `cmd/run.go` - Track iteration outcomes and set exit reason on termination
- `cmd/restart.go` - Same changes for restart command's iteration loop
- `cmd/inspect.go` - Display new fields
- `cmd/list.go` - Add `--show-exit` flag
- `cmd/stats.go` - Include success/failure rates in stats

## Implementation details

### internal/state/manager.go

Add fields to AgentState:

```go
type AgentState struct {
    ID            string     `json:"id"`
    Name          string     `json:"name,omitempty"`
    PID           int        `json:"pid"`
    Prompt        string     `json:"prompt"`
    Model         string     `json:"model"`
    StartedAt     time.Time  `json:"started_at"`
    Iterations    int        `json:"iterations"`
    CurrentIter   int        `json:"current_iteration"`
    Status        string     `json:"status"`
    TerminateMode string     `json:"terminate_mode"`
    Paused        bool       `json:"paused"`
    PausedAt      *time.Time `json:"paused_at,omitempty"`
    LogFile       string     `json:"log_file"`
    WorkingDir    string     `json:"working_dir"`
    EnvNames      []string   `json:"env_names,omitempty"`
    
    // New fields for exit tracking
    TerminatedAt    *time.Time `json:"terminated_at,omitempty"`
    ExitReason      string     `json:"exit_reason,omitempty"`
    LastError       string     `json:"last_error,omitempty"`
    SuccessfulIters int        `json:"successful_iterations"`
    FailedIters     int        `json:"failed_iterations"`
}
```

### cmd/run.go

Update iteration loop to track outcomes:

```go
// Run agent - errors should NOT stop the run
runner := agent.NewRunner(cfg)
if err := runner.Run(os.Stdout); err != nil {
    fmt.Printf("\n[swarm] Agent error (continuing): %v\n", err)
    agentState.FailedIters++
    agentState.LastError = err.Error()
} else {
    agentState.SuccessfulIters++
}
_ = mgr.Update(agentState)
```

Update cleanup defer to set termination time and reason:

```go
defer func() {
    agentState.Status = "terminated"
    now := time.Now()
    agentState.TerminatedAt = &now
    if agentState.ExitReason == "" {
        agentState.ExitReason = "completed"
    }
    _ = mgr.Update(agentState)
}()
```

Update signal handling:

```go
case sig := <-sigChan:
    fmt.Printf("\n[swarm] Received signal %v, stopping\n", sig)
    agentState.ExitReason = "signal"
    return nil
```

Update termination mode handling:

```go
if currentState.TerminateMode == "immediate" {
    fmt.Println("\n[swarm] Received immediate termination signal")
    agentState.ExitReason = "killed"
    return nil
}
```

### cmd/inspect.go

Display new fields:

```go
// After showing status
fmt.Printf("Started:       %s\n", agent.StartedAt.Format(time.RFC3339))
if agent.TerminatedAt != nil {
    fmt.Printf("Terminated:    %s\n", agent.TerminatedAt.Format(time.RFC3339))
    duration := agent.TerminatedAt.Sub(agent.StartedAt).Round(time.Second)
    fmt.Printf("Runtime:       %s\n", duration)
} else {
    fmt.Printf("Running for:   %s\n", time.Since(agent.StartedAt).Round(time.Second))
}

if agent.ExitReason != "" {
    fmt.Printf("Exit reason:   %s\n", agent.ExitReason)
}

fmt.Printf("Iteration:     %d/%d\n", agent.CurrentIter, agent.Iterations)

// Show iteration breakdown if there were any iterations
if agent.SuccessfulIters > 0 || agent.FailedIters > 0 {
    fmt.Printf("Successful:    %d\n", agent.SuccessfulIters)
    fmt.Printf("Failed:        %d\n", agent.FailedIters)
    total := agent.SuccessfulIters + agent.FailedIters
    if total > 0 {
        rate := float64(agent.SuccessfulIters) / float64(total) * 100
        fmt.Printf("Success rate:  %.0f%%\n", rate)
    }
}

if agent.LastError != "" {
    fmt.Println()
    bold.Println("Last Error")
    fmt.Println("─────────────────────────────────")
    fmt.Println(agent.LastError)
}
```

### cmd/stats.go

Update stats to include success rates:

```go
type Stats struct {
    // ... existing fields ...
    
    IterationsSuccessful int `json:"iterations_successful"`
    IterationsFailed     int `json:"iterations_failed"`
    SuccessRate          float64 `json:"success_rate"`
}

// In calculateStats
stats.IterationsSuccessful += agent.SuccessfulIters
stats.IterationsFailed += agent.FailedIters

// Calculate success rate
total := stats.IterationsSuccessful + stats.IterationsFailed
if total > 0 {
    stats.SuccessRate = float64(stats.IterationsSuccessful) / float64(total) * 100
}
```

## Use cases

### Debugging a failed agent

```bash
$ swarm inspect my-agent
Agent Details
─────────────────────────────────
ID:            abc123
Status:        terminated
Exit reason:   error
Successful:    7
Failed:        3
Success rate:  70%

Last Error
─────────────────────────────────
agent process exited with code 1: API rate limit exceeded
```

### Checking agent reliability

```bash
$ swarm stats
...
Iterations
  Completed:  150
  Successful: 142
  Failed:     8
  Success rate: 94.7%
```

### Finding problematic runs

```bash
# List agents that were killed or errored
$ swarm list -a --format json | jq '.[] | select(.exit_reason != "completed")'
```

## Edge cases

1. **Existing state without new fields**: Old state entries will have zero values for new fields. This is safe - empty `TerminatedAt` is handled, zero counts display correctly.

2. **Agent killed before any iteration**: `SuccessfulIters` and `FailedIters` both remain 0, `ExitReason` is "killed".

3. **Single-iteration agents**: Currently not tracked in state. When that's fixed, they should also set these fields.

4. **Crash without cleanup**: If the process crashes hard (SIGKILL, OOM), the defer won't run. The cleanup routine in `Manager` already marks these as terminated, but `ExitReason` would be empty. Could add a `"crashed"` reason detected during cleanup.

5. **Detached agents**: Same tracking applies - the iteration loop in the detached process updates state normally.

## Acceptance criteria

- `AgentState` includes `TerminatedAt`, `ExitReason`, `LastError`, `SuccessfulIters`, `FailedIters`
- Running agents track iteration success/failure in real-time
- On termination, `TerminatedAt` and `ExitReason` are set
- `swarm inspect` displays termination details and iteration breakdown
- `swarm stats` shows aggregate success rates
- Existing agents without new fields display gracefully (no errors)
- JSON output includes all new fields
- `cmd/restart.go` has same tracking as `cmd/run.go`
