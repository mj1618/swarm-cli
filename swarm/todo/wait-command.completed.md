# Add `swarm wait` command for scripting

## Problem

Currently, when running agents in detached mode (`-d`), there's no way to programmatically wait for them to complete. Users who want to:

1. Start an agent in the background and wait for completion before continuing a script
2. Orchestrate multiple agents and wait for specific ones to finish
3. Build CI/CD pipelines that wait for agent tasks to complete

...have no clean way to do this. They would need to poll `swarm list` or `swarm inspect` in a loop, which is cumbersome and error-prone.

For example, a user wanting to run multiple agents and wait for all to finish would need to write:

```bash
# Currently: awkward polling loop
swarm run -p task1 -n 10 -d
swarm run -p task2 -n 10 -d

while swarm list -q | grep -q .; do
    sleep 5
done
echo "All agents finished"
```

## Solution

Add a `swarm wait` command that blocks until the specified agent(s) terminate. Similar to `docker wait` or shell's `wait` builtin.

### Proposed API

```bash
# Wait for a single agent by ID or name
swarm wait abc123
swarm wait my-agent

# Wait for multiple agents
swarm wait abc123 def456 ghi789

# Wait with timeout (exit with error if timeout exceeded)
swarm wait abc123 --timeout 30m

# Wait for any agent (first to terminate wins)
swarm wait --any abc123 def456

# Poll interval (default: 1s)
swarm wait abc123 --interval 2s
```

### Exit codes

- `0` - Agent(s) terminated normally
- `1` - Agent not found or other error
- `2` - Timeout exceeded (when `--timeout` is used)

### Output

By default, minimal output. With `--verbose`, print status updates:

```
$ swarm wait abc123 --verbose
Waiting for agent abc123...
Agent abc123 terminated (was running for 5m23s)
```

## Files to create/change

- Create `cmd/wait.go` - new command implementation
- Update README.md to document the new command (optional, could be separate PR)

## Implementation details

```go
package cmd

var (
    waitTimeout  time.Duration
    waitInterval time.Duration
    waitAny      bool
    waitVerbose  bool
)

var waitCmd = &cobra.Command{
    Use:   "wait [agent-id-or-name...]",
    Short: "Wait for agent(s) to terminate",
    Long: `Wait for one or more agents to terminate.

Blocks until all specified agents have terminated, or until the timeout
is reached (if specified). Useful for scripting and orchestration.`,
    Example: `  # Wait for a single agent
  swarm wait abc123

  # Wait for agent by name
  swarm wait my-agent

  # Wait for multiple agents
  swarm wait abc123 def456

  # Wait with 30 minute timeout
  swarm wait abc123 --timeout 30m

  # Wait for any agent to finish (first wins)
  swarm wait --any abc123 def456`,
    Args: cobra.MinimumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        mgr, err := state.NewManagerWithScope(GetScope(), "")
        if err != nil {
            return fmt.Errorf("failed to initialize state manager: %w", err)
        }

        // Resolve all agent identifiers to IDs
        agentIDs := make([]string, 0, len(args))
        for _, identifier := range args {
            agent, err := mgr.GetByNameOrID(identifier)
            if err != nil {
                return fmt.Errorf("agent not found: %s", identifier)
            }
            agentIDs = append(agentIDs, agent.ID)
        }

        // Set up timeout if specified
        var deadline time.Time
        if waitTimeout > 0 {
            deadline = time.Now().Add(waitTimeout)
        }

        // Polling loop
        for {
            allTerminated := true
            anyTerminated := false

            for _, id := range agentIDs {
                agent, err := mgr.Get(id)
                if err != nil || agent == nil {
                    // Agent state removed = terminated
                    anyTerminated = true
                    continue
                }
                if agent.Status == "terminated" {
                    anyTerminated = true
                } else {
                    allTerminated = false
                }
            }

            // Check exit conditions
            if waitAny && anyTerminated {
                return nil
            }
            if !waitAny && allTerminated {
                return nil
            }

            // Check timeout
            if !deadline.IsZero() && time.Now().After(deadline) {
                return fmt.Errorf("timeout waiting for agent(s)")
            }

            time.Sleep(waitInterval)
        }
    },
}

func init() {
    waitCmd.Flags().DurationVar(&waitTimeout, "timeout", 0, "Maximum time to wait (e.g., 30m, 1h)")
    waitCmd.Flags().DurationVar(&waitInterval, "interval", time.Second, "Polling interval")
    waitCmd.Flags().BoolVar(&waitAny, "any", false, "Return when any agent terminates")
    waitCmd.Flags().BoolVarP(&waitVerbose, "verbose", "v", false, "Print status updates")
    rootCmd.AddCommand(waitCmd)
}
```

## Use cases

### CI/CD pipeline

```bash
#!/bin/bash
# Start agent in background
swarm run -p deploy-task -n 1 -d --name deploy

# Wait for completion with 1 hour timeout
if ! swarm wait deploy --timeout 1h; then
    echo "Deploy agent failed or timed out"
    swarm kill deploy
    exit 1
fi

echo "Deploy completed successfully"
```

### Parallel agent orchestration

```bash
# Start multiple agents
swarm run -p test-frontend -n 5 -d --name frontend-tests
swarm run -p test-backend -n 5 -d --name backend-tests
swarm run -p test-e2e -n 3 -d --name e2e-tests

# Wait for all to complete
swarm wait frontend-tests backend-tests e2e-tests --timeout 2h

echo "All tests completed"
```

### Quick one-off with wait

```bash
# Run in background and immediately wait (like foreground but with log file)
swarm run -p my-task -d && swarm wait $(swarm list -q | head -1)
```

## Acceptance criteria

- `swarm wait abc123` blocks until agent abc123 terminates
- `swarm wait abc123 def456` blocks until both agents terminate
- `swarm wait abc123 --timeout 5m` returns error after 5 minutes if not terminated
- `swarm wait --any abc123 def456` returns when first agent terminates
- Exit code 0 on success, 1 on error, 2 on timeout
- Works with both agent IDs and names
- `--verbose` flag shows progress updates

## Completion Notes

**Completed by agent cd59a862 on 2026-01-28**

Implementation:
- Created `cmd/wait.go` with the `swarm wait` command
- Created `cmd/wait_test.go` with unit tests for the command
- Supports all specified features:
  - Wait for single or multiple agents by ID or name
  - `--timeout` flag for maximum wait time (exits with code 2 on timeout)
  - `--interval` flag for custom polling interval (default 1s)
  - `--any` flag to return when any agent terminates
  - `--verbose` flag for progress updates
  - Support for special identifiers `@last` and `_`
  - Works with both project and global scope (`-g` flag)
- All tests pass
