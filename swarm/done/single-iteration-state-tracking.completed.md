# Single-iteration agents should be tracked in state

## Problem

In `cmd/run.go` (lines 216-228), when `effectiveIterations == 1`, the agent runs directly without any state management:

```go
if effectiveIterations == 1 {
    fmt.Printf("Running agent with prompt: %s, model: %s\n", promptName, effectiveModel)
    cfg := agent.Config{
        Model:   effectiveModel,
        Prompt:  promptContent,
        Command: appConfig.Command,
    }
    runner := agent.NewRunner(cfg)
    return runner.Run(os.Stdout)
}
```

This means single-iteration agents:
- Do not appear in `swarm list`
- Cannot be killed via `swarm kill`
- Have no state record at all (no ID, no name, no start time)
- Are invisible to the management system

Multi-iteration agents (even `-n 2`) get full state tracking with registration, cleanup, and visibility. There's no reason single-iteration agents should be treated differently.

## Solution

Register single-iteration agents in the state manager the same way multi-iteration agents are registered. The single-iteration code path should:

1. Create a `state.NewManagerWithScope` and register an `AgentState` with `Iterations: 1`
2. Set up the same `defer` cleanup to mark as `"terminated"` on exit
3. Update `CurrentIter` to 1 before running
4. Still keep the simpler flow (no loop, no signal handling, no pause polling needed)

This is a small change -- roughly wrapping the existing single-iteration block with register/defer/update calls mirroring what the multi-iteration path does (lines 230-261), but without the loop or control polling.

## Files to change

- `cmd/run.go` -- modify the `effectiveIterations == 1` block (lines 216-228)

## Acceptance criteria

- Running `swarm run -p example` (single iteration) registers the agent in state
- `swarm list` shows the running single-iteration agent while it executes
- After completion, the agent state is marked `"terminated"`
- `swarm kill <id>` works on a running single-iteration agent
- No regressions in multi-iteration behavior
- Existing tests continue to pass

## Completion Notes (2026-01-28)

**Agent ID:** cd59a862

**Changes made:**
- Modified `cmd/run.go` to add full state management for single-iteration agents
- The single-iteration code path now:
  1. Creates a state manager with proper scope
  2. Registers an `AgentState` with `Iterations: 1` and `CurrentIter: 1`
  3. Sets up a `defer` to mark agent as "terminated" with proper `TerminatedAt` timestamp
  4. Tracks success/failure via `SuccessfulIters` and `FailedIters` fields
  5. Handles timeout scenarios with proper exit codes
  6. Keeps the simpler flow (no loop, no signal handling, no pause polling)

**Testing:**
- All existing tests pass (`go test ./...`)
- Code compiles successfully

**Behavior change:**
- Single-iteration agents (default `swarm run -p example`) now appear in `swarm list`
- They can be seen while running and show as terminated after completion
- `swarm kill` can target them if they're still running
