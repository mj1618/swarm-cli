# Add environment variable support for agents

## Problem

When running agents, users sometimes need to pass custom environment variables to configure agent behavior, provide secrets, or integrate with other tools. Currently, agents only inherit the parent process's environment with no way to add or override variables.

Common use cases that aren't supported:

1. **Passing API keys or secrets**: Users may want to provide credentials to agents without exposing them in prompts
2. **Customizing behavior**: Set flags like `DEBUG=true` or `VERBOSE=1` for troubleshooting
3. **Integration with external tools**: Pass configuration like `DATABASE_URL` or `OUTPUT_DIR`
4. **CI/CD pipelines**: Inject build-specific variables into agent runs

Currently, users would need to:
- Export variables globally (pollutes shell environment)
- Modify their shell profile (not practical for one-off runs)
- Include sensitive values in prompts (security risk)

## Solution

Add `-e/--env` flag to `swarm run` and `swarm restart` commands to pass environment variables to agents.

### Proposed API

```bash
# Single variable
swarm run -p coder -e DEBUG=true

# Multiple variables
swarm run -p coder -e DEBUG=true -e API_KEY=secret123

# Variable from shell environment (like docker)
swarm run -p coder -e API_KEY

# Combined with other flags
swarm run -p coder -n 10 -d -e TIMEOUT=300 -e LOG_LEVEL=debug
```

### Restart support

Environment variables should also work with restart:

```bash
# Override env vars on restart
swarm restart my-agent -e DEBUG=true

# Original env vars are NOT preserved (too complex to track)
# User must re-specify if needed
```

### Inspect output

Show environment variables in inspect output:

```
$ swarm inspect abc123
Agent Details
─────────────────────────────────
ID:            abc123
Name:          my-agent
...
Environment:
  DEBUG=true
  LOG_LEVEL=debug
```

## Files to change

- `internal/agent/config.go` - add Env field to Config struct
- `internal/agent/runner.go` - apply env vars when starting process
- `internal/state/manager.go` - add Env field to AgentState for persistence
- `cmd/run.go` - add -e/--env flag, pass to agent config
- `cmd/restart.go` - add -e/--env flag support
- `cmd/inspect.go` - display environment variables

## Implementation details

### internal/agent/config.go

Add environment field to Config:

```go
type Config struct {
    Model   string
    Prompt  string
    Command CommandConfig
    Env     []string  // Environment variables in KEY=VALUE format
}
```

### internal/agent/runner.go

Apply environment variables when starting the process:

```go
func (r *Runner) Run(out io.Writer) error {
    args := r.config.Command.ExpandArgs(r.config.Model, r.config.Prompt)
    r.cmd = exec.Command(r.config.Command.Executable, args...)

    // Inherit parent environment and add custom vars
    if len(r.config.Env) > 0 {
        r.cmd.Env = append(os.Environ(), r.config.Env...)
    }

    // ... rest of existing code
}
```

### internal/state/manager.go

Add Env field to AgentState for persistence:

```go
type AgentState struct {
    ID            string     `json:"id"`
    Name          string     `json:"name,omitempty"`
    // ... existing fields ...
    Env           []string   `json:"env,omitempty"` // Environment variables (values may be redacted)
}
```

Note: For security, we may want to redact sensitive values in stored state. Options:
1. Store only variable names, not values (safest)
2. Store full values (user's responsibility)
3. Detect common secret patterns and redact (complex)

Recommend option 1 for initial implementation - store names only.

### cmd/run.go

Add flag parsing and pass to config:

```go
var runEnv []string

func init() {
    runCmd.Flags().StringArrayVarP(&runEnv, "env", "e", nil, "Set environment variables (KEY=VALUE or KEY)")
}

// In RunE:
// Expand variables without values from shell environment
expandedEnv := make([]string, 0, len(runEnv))
for _, e := range runEnv {
    if strings.Contains(e, "=") {
        expandedEnv = append(expandedEnv, e)
    } else {
        // Variable without value - look up from environment
        if val, ok := os.LookupEnv(e); ok {
            expandedEnv = append(expandedEnv, fmt.Sprintf("%s=%s", e, val))
        } else {
            return fmt.Errorf("environment variable %s not set", e)
        }
    }
}

cfg := agent.Config{
    Model:   effectiveModel,
    Prompt:  promptContent,
    Command: appConfig.Command,
    Env:     expandedEnv,
}
```

### cmd/inspect.go

Display environment variables:

```go
if len(agent.Env) > 0 {
    fmt.Println()
    bold.Println("Environment")
    fmt.Println("─────────────────────────────────")
    for _, e := range agent.Env {
        fmt.Printf("  %s\n", e)
    }
}
```

## Edge cases

1. **Empty value**: `swarm run -p coder -e DEBUG=` sets DEBUG to empty string (valid)

2. **Variable not in shell**: `swarm run -p coder -e MISSING` returns error "environment variable MISSING not set"

3. **Invalid format**: `swarm run -p coder -e "has spaces"` returns error for malformed variable

4. **Duplicate variables**: Later values override earlier ones (same as docker behavior)

5. **PATH and other special vars**: Can be overridden but may break agent execution - user's responsibility

6. **Very long values**: Should work (no artificial limits)

7. **Special characters in values**: `swarm run -p coder -e 'MSG=hello world'` works with proper quoting

8. **Detached mode**: Env vars are passed to the detached process via command-line (may need special handling for very long values)

## Security considerations

1. **Sensitive values in logs**: Env vars should NOT appear in agent logs by default. The agent process receives them, but swarm shouldn't log the values.

2. **State file**: Store only variable names (not values) in state.json to avoid persisting secrets.

3. **Command line visibility**: When running detached, env vars passed via CLI are visible in `ps` output. Consider using a temp file for very sensitive values (future enhancement).

4. **Restart behavior**: Original env vars are not preserved - this is intentional to avoid accidentally persisting secrets.

## Examples

### Development debugging

```bash
# Enable debug mode for troubleshooting
swarm run -p coder -e DEBUG=true -e LOG_LEVEL=verbose

# Agent receives DEBUG=true and LOG_LEVEL=verbose in its environment
```

### CI/CD pipeline

```bash
# Pass build context to agent
swarm run -p deploy -d \
  -e BUILD_NUMBER=$BUILD_NUMBER \
  -e GIT_SHA=$GIT_SHA \
  -e ENVIRONMENT=staging
```

### Using shell variables

```bash
export API_KEY=secret123

# Pass through from shell
swarm run -p api-task -e API_KEY

# Equivalent to:
swarm run -p api-task -e API_KEY=secret123
```

### Inspect showing env vars

```bash
$ swarm inspect abc123
Agent Details
─────────────────────────────────
ID:            abc123
Name:          my-agent
Prompt:        coder
Model:         claude-opus-4-20250514
Status:        running
Started:       2024-01-28T10:15:00Z
...

Environment
─────────────────────────────────
  DEBUG=true
  LOG_LEVEL=verbose
  BUILD_NUMBER=1234
```

## Acceptance criteria

- `swarm run -e KEY=VALUE` passes environment variable to agent
- `swarm run -e KEY` expands from shell environment
- Multiple `-e` flags can be specified
- Error on `-e KEY` when KEY not in shell environment
- Environment variables shown in `swarm inspect` output
- Env vars work correctly in detached mode (`-d`)
- `swarm restart -e` accepts new env vars (does not preserve original)
- State file stores variable names only (not values) for security
- Agent process receives the variables correctly

---

## Completion Notes (Agent cd59a862)

**Date:** 2026-01-28

**Implementation completed.** All acceptance criteria have been met:

1. **internal/agent/config.go** - Added `Env []string` field to Config struct
2. **internal/agent/runner.go** - Applied env vars when starting process using `r.cmd.Env = append(os.Environ(), r.config.Env...)`
3. **internal/state/manager.go** - Added `EnvNames []string` field to AgentState (stores only variable names for security, not values)
4. **cmd/run.go** - Added `-e/--env` flag with support for:
   - `KEY=VALUE` format (used as-is)
   - `KEY` format (expanded from shell environment)
   - Env vars passed to detached child via `--_internal-env` flag
5. **cmd/restart.go** - Added `-e/--env` flag support (original env vars are not preserved, as specified)
6. **cmd/inspect.go** - Displays environment variable names under "Environment Variables" section

**Build:** ✅ Compiles successfully
**Tests:** ✅ All agent/state/cmd tests pass (pre-existing config test failure unrelated to this change)
