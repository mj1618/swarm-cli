# Add `--continue` flag to restart command

## Problem

When users restart a terminated agent using `swarm restart`, the agent always begins from iteration 1 regardless of how far the original agent progressed. This creates friction for users who want to pick up where a terminated agent left off.

**Current behavior:**
```bash
# Original agent ran 15/20 iterations before being terminated
swarm restart my-agent
# Restarts at iteration 1/20 - loses progress
```

**Desired behavior:**
```bash
swarm restart my-agent --continue
# Restarts at iteration 16/20 - continues from where it stopped
```

### Use cases

1. **Network/system interruptions**: Agent terminates unexpectedly, user wants to continue
2. **Manual pause for review**: User terminated to review logs, now wants to continue
3. **Resource management**: User terminated overnight, resuming in the morning
4. **Iteration adjustment**: User wants to continue but with more iterations added

## Solution

Add a `--continue` / `-c` flag to `swarm restart` that starts the new agent from the iteration after the terminated agent's last completed iteration.

### Proposed API

```bash
# Continue from last iteration (16/20 if original was at 15/20)
swarm restart my-agent --continue

# Continue with short flag
swarm restart my-agent -c

# Continue but add more iterations
swarm restart my-agent -c --iterations 30
# Starts at 16/30

# Continue with different model
swarm restart my-agent -c --model claude-sonnet-4-20250514
```

## Files to change

- `cmd/restart.go` - add `--continue` flag and implement continuation logic

## Implementation details

### restart.go changes

```go
var (
    restartModel      string
    restartIterations int
    restartName       string
    restartDetach     bool
    restartContinue   bool  // NEW
)

// In RunE, after loading oldAgent:
startingIteration := 1
if restartContinue {
    startingIteration = oldAgent.CurrentIter + 1
    
    // Validate there are iterations remaining
    effectiveIterations := oldAgent.Iterations
    if cmd.Flags().Changed("iterations") {
        effectiveIterations = restartIterations
    }
    
    if startingIteration > effectiveIterations {
        return fmt.Errorf("agent already completed all %d iterations; use --iterations to add more", oldAgent.Iterations)
    }
    
    fmt.Printf("Continuing from iteration %d\n", startingIteration)
}

// When creating agentState:
agentState := &state.AgentState{
    ID:          state.GenerateID(),
    Name:        effectiveName,
    PID:         os.Getpid(),
    Prompt:      promptName,
    Model:       effectiveModel,
    StartedAt:   time.Now(),
    Iterations:  effectiveIterations,
    CurrentIter: startingIteration - 1,  // Will be incremented to startingIteration in first loop
    Status:      "running",
    WorkingDir:  effectiveWorkingDir,
}

// Modify the loop to start from startingIteration:
for i := startingIteration; i <= agentState.Iterations; i++ {
    // ... existing loop body
}

// In init():
restartCmd.Flags().BoolVarP(&restartContinue, "continue", "c", false, "Continue from last iteration")
```

### Output examples

Normal restart (unchanged):
```
$ swarm restart my-agent
Restarting agent 'my-agent' with prompt: planner, model: claude-opus-4-20250514, iterations: 20

[swarm] === Iteration 1/20 ===
```

Continue restart:
```
$ swarm restart my-agent --continue
Continuing from iteration 16
Restarting agent 'my-agent' with prompt: planner, model: claude-opus-4-20250514, iterations: 20

[swarm] === Iteration 16/20 ===
```

Continue with more iterations:
```
$ swarm restart my-agent -c -n 30
Continuing from iteration 16
Restarting agent 'my-agent' with prompt: planner, model: claude-opus-4-20250514, iterations: 30

[swarm] === Iteration 16/30 ===
```

Already completed all iterations:
```
$ swarm restart my-agent --continue
Error: agent already completed all 20 iterations; use --iterations to add more
```

## Edge cases

1. **Agent completed all iterations**: If `CurrentIter >= Iterations`, return an error suggesting to use `--iterations` to add more.

2. **CurrentIter is 0**: Agent was terminated before first iteration completed. Start from iteration 1 (same as without `--continue`).

3. **Combined with --iterations**: If user specifies fewer iterations than already completed, return an error. If more, continue from where it left off toward the new total.

4. **Detached mode**: The `--continue` flag should work with `-d` for background restart. Pass the starting iteration to the detached child via internal flag.

5. **String prompts**: String prompts (`-s`) cannot be restarted, so `--continue` is not applicable. This is already handled by existing validation.

## Acceptance criteria

- `swarm restart my-agent` without `--continue` works exactly as before (starts at iteration 1)
- `swarm restart my-agent --continue` starts from `CurrentIter + 1`
- `swarm restart my-agent -c` works with short flag
- Error shown if agent already completed all iterations
- `--continue` can be combined with `--iterations` to add more iterations
- `--continue` can be combined with `--model` to change model
- `--continue` works with `-d` (detached mode)
- Iteration counter in output shows correct starting point (e.g., "Iteration 16/20")
- New agent's `CurrentIter` is set correctly so `swarm list` shows accurate progress

## Completion Notes (Agent cd59a862)

### Changes Made

1. **cmd/restart.go**:
   - Added `restartContinue` and `restartInternalStart` flag variables
   - Added `--continue` / `-c` flag registration
   - Added `--_internal-start-iter` hidden flag for passing start iteration to detached child
   - Added logic to calculate `startingIteration` based on `oldAgent.CurrentIter + 1`
   - Added validation that iterations remain when using `--continue`
   - Updated loop to start from `startingIteration`
   - Updated agentState.CurrentIter initialization to `startingIteration - 1`
   - Updated detached mode to pass start iteration via internal flag
   - Updated help text and examples

2. **cmd/run.go**:
   - Added `runInternalStartIter` flag variable
   - Added `--_internal-start-iter` hidden flag registration
   - Added logic to use `startingIteration` from internal flag
   - Updated loop to start from `startingIteration`

### Testing

- All existing tests pass
- Project builds successfully
- Help output shows the new `--continue` / `-c` flag correctly
