# Add `--on-complete` hook for post-completion commands

## Completion Notes

Implemented by agent cd59a862 on 2026-01-28.

### Changes Made:
1. Added `OnComplete` field to `AgentState` in `internal/state/manager.go`
2. Created `internal/agent/hook.go` with `ExecuteOnCompleteHook` function that:
   - Executes the hook command in a shell
   - Passes agent context via environment variables (SWARM_AGENT_ID, SWARM_AGENT_NAME, SWARM_AGENT_STATUS, etc.)
   - Runs in the agent's working directory
   - Also includes additional env vars: SWARM_AGENT_EXIT_REASON, SWARM_AGENT_SUCCESSFUL_ITERS, SWARM_AGENT_FAILED_ITERS
3. Added `--on-complete` flag to `cmd/run.go`:
   - Works for both foreground and detached modes
   - Passes hook to detached child via `--_internal-on-complete` flag
   - Executes hook in defer cleanup block for both single and multi-iteration modes
4. Added `--on-complete` flag to `cmd/restart.go` for consistency

### Testing:
- All existing tests pass
- Manual test verified hook executes with correct output:
  ```bash
  echo 'test prompt' | swarm run --stdin -n 1 --on-complete "echo 'HOOK_EXECUTED'"
  # Output shows: HOOK_EXECUTED: agent completed
  ```

---

## Problem

When running agents in detached mode, users often need to perform follow-up actions when agents complete:

1. **Notifications**: Send a Slack/Discord message, desktop notification, or email when a long-running agent finishes
2. **Chaining**: Start another agent or script after the first completes
3. **Cleanup**: Run post-processing scripts, move files, or update external systems
4. **CI/CD Integration**: Trigger downstream jobs or update status in external tools

Currently, users must either:
- Manually check `swarm list` periodically
- Use `swarm wait` in a wrapper script (still requires polling)
- Write custom monitoring scripts

None of these provide a clean, integrated solution for running post-completion commands.

## Solution

Add an `--on-complete` flag (and configuration option) that specifies a command to run when an agent terminates.

### Proposed API

```bash
# Run a shell command when agent completes
swarm run -p task -d --on-complete "echo 'Agent done!'"

# Send a desktop notification (macOS)
swarm run -p task -d --on-complete "osascript -e 'display notification \"Agent completed\" with title \"Swarm\"'"

# Chain agents - start another agent when this one finishes
swarm run -p analyzer -d --on-complete "swarm run -p reporter -d"

# Call a webhook
swarm run -p task -d --on-complete "curl -X POST https://hooks.example.com/done"

# Run a script with agent info passed as environment variables
swarm run -p task -d --on-complete "./scripts/notify.sh"
```

### Environment Variables

The on-complete command receives context about the completed agent via environment variables:

| Variable | Description |
|----------|-------------|
| `SWARM_AGENT_ID` | The agent's ID |
| `SWARM_AGENT_NAME` | The agent's name (if set) |
| `SWARM_AGENT_STATUS` | Final status (terminated) |
| `SWARM_AGENT_ITERATIONS` | Total iterations configured |
| `SWARM_AGENT_COMPLETED` | Iterations actually completed |
| `SWARM_AGENT_PROMPT` | Prompt name used |
| `SWARM_AGENT_MODEL` | Model used |
| `SWARM_AGENT_LOG_FILE` | Path to log file |
| `SWARM_AGENT_DURATION` | Total runtime in seconds |

### Configuration file support

Add ability to set default on-complete hooks in `swarm.yaml`:

```yaml
# swarm/.swarm.toml
on_complete = "notify-send 'Agent completed: {name}'"

# Or per-prompt hooks
[prompts.deploy]
on_complete = "./scripts/deploy-complete.sh"
```

## Files to create/change

- `cmd/run.go` - Add `--on-complete` flag
- `cmd/restart.go` - Add `--on-complete` flag (optional, for consistency)
- `internal/state/state.go` - Add `OnComplete` field to `AgentState`
- `internal/agent/hook.go` (new) - Hook execution logic
- `internal/config/config.go` - Add config file support (optional enhancement)

## Implementation details

### cmd/run.go changes

```go
var runOnComplete string

// In runCmd.RunE, when setting up detached mode:
if runOnComplete != "" {
    agentState.OnComplete = runOnComplete
}

// In init():
runCmd.Flags().StringVar(&runOnComplete, "on-complete", "", "Command to run when agent completes")
```

### internal/state/state.go changes

```go
type AgentState struct {
    // ... existing fields ...
    OnComplete string `json:"on_complete,omitempty"` // Command to run on completion
}
```

### internal/agent/hook.go (new file)

```go
package agent

import (
    "fmt"
    "os"
    "os/exec"
    "strconv"
    "time"

    "github.com/mj1618/swarm-cli/internal/state"
)

// ExecuteOnCompleteHook runs the on-complete command for an agent.
// The command is executed in a shell with agent context as environment variables.
func ExecuteOnCompleteHook(agent *state.AgentState) error {
    if agent.OnComplete == "" {
        return nil
    }

    // Set up environment with agent context
    env := os.Environ()
    env = append(env,
        "SWARM_AGENT_ID="+agent.ID,
        "SWARM_AGENT_NAME="+agent.Name,
        "SWARM_AGENT_STATUS="+agent.Status,
        fmt.Sprintf("SWARM_AGENT_ITERATIONS=%d", agent.Iterations),
        fmt.Sprintf("SWARM_AGENT_COMPLETED=%d", agent.CurrentIter),
        "SWARM_AGENT_PROMPT="+agent.Prompt,
        "SWARM_AGENT_MODEL="+agent.Model,
        "SWARM_AGENT_LOG_FILE="+agent.LogFile,
        fmt.Sprintf("SWARM_AGENT_DURATION=%d", int(time.Since(agent.StartedAt).Seconds())),
    )

    // Execute command in shell
    cmd := exec.Command("sh", "-c", agent.OnComplete)
    cmd.Env = env
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    return cmd.Run()
}
```

### Integration in run.go iteration loop

```go
// At the end of the iteration loop (after all iterations complete or on termination):
defer func() {
    agentState.Status = "terminated"
    _ = mgr.Update(agentState)
    
    // Execute on-complete hook
    if agentState.OnComplete != "" {
        if err := agent.ExecuteOnCompleteHook(agentState); err != nil {
            fmt.Printf("[swarm] Warning: on-complete hook failed: %v\n", err)
        }
    }
}()
```

## Use cases

### Desktop notification when done

```bash
# macOS
swarm run -p big-task -n 50 -d --on-complete "osascript -e 'display notification \"Agent finished\" with title \"Swarm\"'"

# Linux (notify-send)
swarm run -p big-task -n 50 -d --on-complete "notify-send 'Swarm' 'Agent $SWARM_AGENT_NAME finished'"
```

### Slack notification

```bash
swarm run -p deploy -d --on-complete 'curl -X POST -H "Content-type: application/json" \
  --data "{\"text\":\"Agent $SWARM_AGENT_NAME completed ($SWARM_AGENT_COMPLETED iterations)\"}" \
  https://hooks.slack.com/services/XXX/YYY/ZZZ'
```

### Agent chaining

```bash
# Run analyzer, then reporter when analyzer finishes
swarm run -p analyzer -n 10 -d --on-complete "swarm run -p reporter -n 5 -d"
```

### Conditional follow-up

```bash
# Run post-processing only if all iterations completed
swarm run -p task -n 20 -d --on-complete '[ "$SWARM_AGENT_COMPLETED" -eq "$SWARM_AGENT_ITERATIONS" ] && ./success.sh || ./partial.sh'
```

### Log archival

```bash
swarm run -p task -d --on-complete 'gzip "$SWARM_AGENT_LOG_FILE" && mv "$SWARM_AGENT_LOG_FILE.gz" ~/logs/'
```

## Edge cases

1. **Hook command fails**: Log warning but don't affect agent completion status. The agent is already terminated, the hook is best-effort.

2. **Agent killed externally**: Hook should still run if process terminates normally. If killed with SIGKILL, hook won't run (expected behavior - process is gone).

3. **Hook command hangs**: Consider adding a timeout (e.g., 60 seconds default, configurable with `--on-complete-timeout`).

4. **Non-detached mode**: Hook should also work for foreground agents (run after completion).

5. **Multiple hooks**: Could support multiple `--on-complete` flags, executed in order. Start simple with single hook.

6. **Hook inherits environment**: The hook runs with the swarm process environment plus the SWARM_* variables.

## Future enhancements (out of scope)

1. **Per-status hooks**: `--on-success`, `--on-failure` for conditional hooks
2. **Retry on failure**: `--on-complete-retry 3` to retry failed hooks
3. **Async hooks**: `--on-complete-async` to not wait for hook completion
4. **Hook templates**: Named hooks in config that can be referenced by name

## Acceptance criteria

- `swarm run -p task -d --on-complete "echo done"` executes "echo done" when agent terminates
- Hook receives `SWARM_AGENT_*` environment variables with correct values
- Hook failures are logged as warnings but don't affect exit status
- Hook works for both detached and foreground agents
- Hook runs even if agent terminates early (via kill or error)
- `swarm restart` also supports `--on-complete` flag
