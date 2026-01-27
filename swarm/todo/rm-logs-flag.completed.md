# Add `--logs` flag to `swarm rm` command

## Completed by cd59a862

Implemented the `--logs` flag for the `swarm rm` command. Changes made to `cmd/rm.go`:
- Added `rmLogs` variable and `--logs` flag registration
- Updated Long description to document the new flag
- Added examples showing `--logs` and `--force --logs` usage
- Implemented log file deletion logic following the same pattern as `prune --logs`
- Prints summary when logs are removed

All acceptance criteria met:
- Build passes
- Help shows new flag
- Pattern matches existing prune --logs behavior

---

## Problem

The `swarm rm` command removes agents from the state but does not offer an option to delete their associated log files. This is inconsistent with `swarm prune` which has a `--logs` flag.

Currently, when users remove agents with `swarm rm`:

```bash
# Remove an agent
swarm rm abc123
# Log file remains at ~/.swarm/logs/abc123.log - must manually delete
```

This creates orphaned log files that accumulate over time. Users must either:

1. Manually find and delete log files: `rm ~/.swarm/logs/abc123.log`
2. Use `swarm prune --logs` to clean up, but this removes ALL terminated agents, not specific ones

The `prune` command has this capability:

```bash
swarm prune --logs   # Removes terminated agents AND their log files
```

But `rm` does not:

```bash
swarm rm abc123 --logs   # Error: unknown flag: --logs
```

## Solution

Add a `--logs` flag to `swarm rm` that also deletes the log files associated with removed agents, matching the behavior of `prune --logs`.

### Proposed API

```bash
# Remove agent and its log file
swarm rm abc123 --logs

# Remove multiple agents and their log files
swarm rm abc123 def456 --logs

# Force remove running agent and delete logs
swarm rm abc123 --force --logs
```

## Files to change

- `cmd/rm.go` - add `--logs` flag and implement log file deletion

## Implementation details

### cmd/rm.go changes

```go
var rmForce bool
var rmLogs bool  // NEW

var rmCmd = &cobra.Command{
    Use:   "rm [agent-id-or-name...]",
    Short: "Remove one or more agents",
    Long: `Remove one or more agents from the state.

By default, only terminated agents can be removed. Use --force to remove
running agents (this will also terminate them).

Use --logs to also delete the log files associated with removed agents.

The agents can be specified by their IDs or names.`,
    Example: `  # Remove a terminated agent by ID
  swarm rm abc123

  # Remove multiple agents
  swarm rm abc123 def456

  # Remove by name
  swarm rm my-agent

  # Force remove a running agent
  swarm rm abc123 --force

  # Remove agent and its log file
  swarm rm abc123 --logs

  # Force remove and delete logs
  swarm rm abc123 --force --logs`,
    Args: cobra.MinimumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... existing validation code ...

        var errors []string
        removed := 0
        logsRemoved := 0  // NEW

        for _, identifier := range args {
            agent, err := mgr.GetByNameOrID(identifier)
            if err != nil {
                errors = append(errors, fmt.Sprintf("%s: not found", identifier))
                continue
            }

            // Check if agent is running
            if agent.Status == "running" {
                if !rmForce {
                    errors = append(errors, fmt.Sprintf("%s: agent is running (use --force to remove)", identifier))
                    continue
                }

                // Force terminate the running agent
                if err := process.Kill(agent.PID); err != nil {
                    fmt.Printf("Warning: could not send signal to process %d: %v\n", agent.PID, err)
                }
            }

            // Remove the agent from state
            if err := mgr.Remove(agent.ID); err != nil {
                errors = append(errors, fmt.Sprintf("%s: failed to remove: %v", identifier, err))
                continue
            }

            // NEW: Clean up log file if requested
            if rmLogs && agent.LogFile != "" {
                if err := os.Remove(agent.LogFile); err != nil {
                    if !os.IsNotExist(err) {
                        fmt.Printf("Warning: failed to remove log file %s: %v\n", agent.LogFile, err)
                    }
                } else {
                    logsRemoved++
                }
            }

            fmt.Println(agent.ID)
            removed++
        }

        // Print errors at the end
        for _, e := range errors {
            fmt.Printf("Error: %s\n", e)
        }

        if removed == 0 && len(errors) > 0 {
            return fmt.Errorf("no agents removed")
        }

        // NEW: Summary when logs were removed
        if rmLogs && logsRemoved > 0 {
            fmt.Printf("Removed %d agent(s) and %d log file(s).\n", removed, logsRemoved)
        }

        return nil
    },
}

func init() {
    rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Force removal of running agents")
    rmCmd.Flags().BoolVar(&rmLogs, "logs", false, "Also delete log files for removed agents")  // NEW
    rootCmd.AddCommand(rmCmd)
}
```

## Example usage

### Remove agent and log file

```bash
$ swarm rm abc123 --logs
abc123
Removed 1 agent(s) and 1 log file(s).
```

### Remove multiple agents with logs

```bash
$ swarm rm abc123 def456 --logs
abc123
def456
Removed 2 agent(s) and 2 log file(s).
```

### Agent without log file (not started in detached mode)

```bash
$ swarm rm xyz789 --logs
xyz789
# No "log file" message since agent had no log file
```

### Force remove running agent with logs

```bash
$ swarm rm abc123 --force --logs
abc123
Removed 1 agent(s) and 1 log file(s).
```

## Edge cases

1. **Agent has no log file**: Agent wasn't started in detached mode, so `LogFile` is empty. Skip log deletion silently.

2. **Log file already deleted**: File doesn't exist (manually deleted or disk issue). Skip with no error (already matches prune behavior).

3. **Log file permission denied**: Print warning but continue with agent removal from state.

4. **Mixed results**: Some agents have logs, some don't. Report accurate counts.

## Acceptance criteria

- `swarm rm abc123 --logs` removes the agent and its log file
- `swarm rm abc123 def456 --logs` works with multiple agents
- `swarm rm abc123 --force --logs` works together with force flag
- Summary shows log file count when `--logs` is used and logs were deleted
- Missing log files are handled gracefully (no error)
- Permission errors show warnings but don't fail the command
- Agents without log files are removed without error when `--logs` is specified
