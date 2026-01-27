# Add timeout support for agent runs

## Problem

When running agents, especially in detached mode or CI/CD pipelines, there's no way to automatically stop an agent that takes too long. This can lead to:

1. **Runaway agents**: An agent might hang indefinitely due to a bug or infinite loop
2. **CI/CD failures**: Pipelines have time limits and need agents to respect them
3. **Resource waste**: Long-running agents consume compute credits even when stuck
4. **Manual intervention required**: Users must manually monitor and kill stuck agents

Currently the only options are:
- Manually watch and kill agents (`swarm kill`)
- Set a system-level timeout externally (e.g., `timeout` command on Linux)

## Solution

Add timeout flags to `swarm run` that automatically terminate agents when time limits are exceeded.

### Proposed API

```bash
# Set a total timeout for the entire run (all iterations)
swarm run -p my-task -n 20 --timeout 2h

# Set a timeout per iteration
swarm run -p my-task -n 20 --iter-timeout 10m

# Combine both: each iteration max 10 minutes, total max 1 hour
swarm run -p my-task -n 20 --iter-timeout 10m --timeout 1h

# Timeout in detached mode
swarm run -p my-task -n 20 -d --timeout 30m

# Different time units supported
swarm run -p my-task --timeout 30s     # seconds
swarm run -p my-task --timeout 5m      # minutes
swarm run -p my-task --timeout 2h      # hours
swarm run -p my-task --timeout 1h30m   # combined
```

### Behavior

1. **Total timeout (`--timeout`)**: Stops the entire run (all iterations) after the specified duration
   - Counts from when the agent starts
   - Terminates gracefully after current iteration if possible
   - Force kills if iteration doesn't complete within grace period (30s)

2. **Per-iteration timeout (`--iter-timeout`)**: Stops individual iterations that exceed the limit
   - Kills the current iteration's agent subprocess
   - Continues to next iteration (unless `--timeout` also exceeded)
   - Logged as an iteration failure, same as any other agent error

3. **Exit codes**:
   - Exit code 0: Completed all iterations successfully
   - Exit code 1: General error
   - Exit code 124: Timed out (matches GNU timeout convention)

4. **State tracking**: Timeout status is recorded in agent state
   - `timeout_at`: When the timeout will/did trigger
   - `timeout_reason`: "total" or "iteration" when terminated by timeout

### Configuration

Timeouts can also be set in config files:

```toml
# .swarm.toml or ~/.config/swarm/config.toml

# Default total timeout (0 = no timeout)
timeout = "2h"

# Default per-iteration timeout (0 = no timeout)
iter_timeout = "15m"
```

CLI flags override config values.

## Files to create/change

- Modify `cmd/run.go` - add timeout flags and logic
- Modify `internal/agent/runner.go` - add timeout to agent subprocess
- Modify `internal/state/state.go` - add timeout fields to AgentState
- Modify `internal/config/config.go` - add timeout config options

## Implementation details

### cmd/run.go changes

```go
var (
    runTimeout     string
    runIterTimeout string
)

// In init():
runCmd.Flags().StringVar(&runTimeout, "timeout", "", "Total timeout for run (e.g., 30m, 2h)")
runCmd.Flags().StringVar(&runIterTimeout, "iter-timeout", "", "Timeout per iteration (e.g., 10m)")

// In RunE:
// Parse timeout durations
var totalTimeout, iterTimeout time.Duration
if runTimeout != "" {
    var err error
    totalTimeout, err = time.ParseDuration(runTimeout)
    if err != nil {
        return fmt.Errorf("invalid timeout format: %w", err)
    }
}
if runIterTimeout != "" {
    var err error
    iterTimeout, err = time.ParseDuration(runIterTimeout)
    if err != nil {
        return fmt.Errorf("invalid iter-timeout format: %w", err)
    }
}

// Set up total timeout
var timeoutCtx context.Context
var timeoutCancel context.CancelFunc
if totalTimeout > 0 {
    timeoutCtx, timeoutCancel = context.WithTimeout(context.Background(), totalTimeout)
    defer timeoutCancel()
} else {
    timeoutCtx = context.Background()
}

// In iteration loop, check for total timeout
select {
case <-timeoutCtx.Done():
    fmt.Println("\n[swarm] Total timeout reached, stopping")
    agentState.Status = "terminated"
    agentState.TimeoutReason = "total"
    _ = mgr.Update(agentState)
    os.Exit(124)
default:
    // Continue
}

// Run agent with per-iteration timeout
cfg := agent.Config{
    Model:   agentState.Model,
    Prompt:  promptContent,
    Command: appConfig.Command,
    Timeout: iterTimeout,
}
```

### internal/agent/runner.go changes

```go
type Config struct {
    Model   string
    Prompt  string
    Command CommandConfig
    Timeout time.Duration // Per-iteration timeout
}

func (r *Runner) Run(w io.Writer) error {
    ctx := context.Background()
    if r.config.Timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
        defer cancel()
    }
    
    cmd := exec.CommandContext(ctx, r.config.Command.Executable, args...)
    // ... existing code ...
    
    err := cmd.Wait()
    if ctx.Err() == context.DeadlineExceeded {
        return fmt.Errorf("iteration timed out after %v", r.config.Timeout)
    }
    return err
}
```

### internal/state/state.go changes

```go
type AgentState struct {
    // ... existing fields ...
    TimeoutAt     *time.Time `json:"timeout_at,omitempty"`      // When timeout will trigger
    TimeoutReason string     `json:"timeout_reason,omitempty"`  // "total" or "iteration"
}
```

### internal/config/config.go changes

```go
type Config struct {
    // ... existing fields ...
    Timeout     string `toml:"timeout"`
    IterTimeout string `toml:"iter_timeout"`
}
```

## Use cases

### CI/CD pipeline with time limit

```bash
# In CI script - fail if agent doesn't complete in 30 minutes
swarm run -p ci-task -n 5 --timeout 30m
if [ $? -eq 124 ]; then
    echo "Agent timed out!"
    exit 1
fi
```

### Prevent runaway iterations

```bash
# Each iteration should complete in ~5 minutes, kill if it takes 15
swarm run -p complex-task -n 50 --iter-timeout 15m
```

### Long overnight run with safeguard

```bash
# Run overnight but stop after 8 hours max
swarm run -p refactor-task -n 100 -d --timeout 8h
```

### Quick exploration with strict limits

```bash
# Try something quickly, don't let it run away
swarm run -s "Explore the codebase" --timeout 5m
```

## Edge cases

1. **Timeout during pause**: If agent is paused and timeout triggers, resume and terminate gracefully
2. **Graceful shutdown**: On total timeout, wait up to 30s for current iteration to complete before force kill
3. **Zero timeout**: Value of 0 or empty string means no timeout (current behavior)
4. **Timeout display**: Show remaining time in `swarm inspect` output
5. **Negative values**: Reject with error
6. **Config + flag**: CLI flags override config file values

## Acceptance criteria

- `--timeout` flag sets total run timeout
- `--iter-timeout` flag sets per-iteration timeout
- Time formats accepted: `30s`, `5m`, `2h`, `1h30m`
- Exit code 124 when terminated by timeout
- Timeout reason recorded in agent state
- `swarm inspect` shows timeout info when set
- Works in both foreground and detached modes
- Config file supports `timeout` and `iter_timeout` options
- Graceful termination attempted before force kill
