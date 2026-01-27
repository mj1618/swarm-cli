# Refactor duplicated iteration loop into shared function

## Problem

The multi-iteration agent loop is duplicated between `cmd/run.go` (lines 268-354) and `cmd/restart.go` (lines 234-320). This ~80+ line block is nearly identical in both files and handles:

- Signal handling (SIGINT/SIGTERM)
- State polling and checking for remote updates
- Iterations count updates from external `swarm update` commands
- Model updates from external `swarm update` commands
- Termination mode checking (`immediate` and `after_iteration`)
- Pause/resume handling with polling loop
- Current iteration tracking and state updates
- Agent execution with error handling (continue on failure)

When a bug is fixed or feature added to this loop, it must be changed in both places. This violates DRY and increases maintenance burden.

## Solution

Extract the shared iteration loop logic into a reusable function in a new internal package (e.g., `internal/runner/loop.go`).

### Proposed API

```go
package runner

type LoopConfig struct {
    Manager     *state.Manager
    AgentState  *state.AgentState
    PromptContent string
    AppCommand  config.CommandConfig
    Output      io.Writer
}

// RunLoop executes the multi-iteration agent loop with state management,
// signal handling, pause/resume support, and graceful termination.
// Returns when all iterations complete, termination is requested, or a signal is received.
func RunLoop(cfg LoopConfig) error
```

### Usage in run.go

```go
loopCfg := runner.LoopConfig{
    Manager:       mgr,
    AgentState:    agentState,
    PromptContent: promptContent,
    AppCommand:    appConfig.Command,
    Output:        os.Stdout,
}
return runner.RunLoop(loopCfg)
```

### Usage in restart.go

```go
loopCfg := runner.LoopConfig{
    Manager:       mgr,
    AgentState:    agentState,
    PromptContent: promptContent,
    AppCommand:    appConfig.Command,
    Output:        os.Stdout,
}
return runner.RunLoop(loopCfg)
```

## Files to change

- Create `internal/runner/loop.go` with the shared `RunLoop` function
- Create `internal/runner/loop_test.go` with tests for the new function
- `cmd/run.go` — replace multi-iteration loop (lines 262-355) with call to `runner.RunLoop`
- `cmd/restart.go` — replace multi-iteration loop (lines 228-321) with call to `runner.RunLoop`

## Implementation details

The `RunLoop` function should:

1. Set up signal handling for SIGINT/SIGTERM
2. Set up defer to mark agent as "terminated" on exit
3. Loop from iteration 1 to `AgentState.Iterations`:
   - Poll state for remote updates (iterations, model, termination mode, pause)
   - Apply updates and print notification messages
   - Handle pause state with polling loop (check every 1 second)
   - Break on termination signals
   - Update `CurrentIter` in state
   - Print iteration header
   - Execute agent via `agent.NewRunner().Run()`
   - Continue on agent errors (don't break the loop)
   - Check for OS signals between iterations
4. Print completion message

The defer cleanup should always mark the agent as terminated, ensuring state consistency even on unexpected exits.

## Acceptance criteria

- `swarm run -p example -n 5` works identically to before
- `swarm restart my-agent -n 5` works identically to before
- Remote `swarm update` commands still work mid-run (iterations, model changes)
- `swarm stop`/`swarm start` pause/resume still works
- `swarm kill` (immediate and graceful) still works
- Signal handling (Ctrl+C) still works
- Agent errors don't stop the iteration loop
- No regressions in existing tests
- New tests cover the `RunLoop` function in isolation
