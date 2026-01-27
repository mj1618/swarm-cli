# Clean up log files when pruning terminated agents

## Problem

When users run `swarm prune` to remove terminated agents, only the state entries are removed from `~/.swarm/state.json`. The associated log files in `~/.swarm/logs/` remain on disk indefinitely.

Over time, this leads to:
1. Accumulation of orphaned log files consuming disk space
2. No clear relationship between log files and their agents
3. Manual cleanup required to reclaim disk space

Currently, the `Remove()` function in `internal/state/manager.go` only deletes the agent entry from the state map - it doesn't touch the `LogFile` field pointing to the actual file on disk.

## Solution

Add a `--logs` flag to the `swarm prune` command that deletes associated log files when pruning terminated agents. By default, log files are preserved for safety (users might want to review them later).

### Proposed API

```bash
# Prune agents only (current behavior, log files preserved)
swarm prune

# Prune agents AND their log files
swarm prune --logs

# Combine with force flag
swarm prune --logs --force
```

### Alternative considered

Making `--logs` the default behavior was considered but rejected because:
- Log files might contain valuable debugging information
- Users might want to review logs before deleting them
- Explicit opt-in for destructive operations is safer

## Files to change

- `cmd/prune.go` - add `--logs` flag and implement log file deletion

## Implementation details

### prune.go changes

```go
var (
    pruneForce bool
    pruneLogs  bool  // NEW
)

// In RunE, after removing the agent state:
for _, agent := range terminated {
    if err := mgr.Remove(agent.ID); err != nil {
        fmt.Printf("Warning: failed to remove agent %s: %v\n", agent.ID, err)
        continue
    }
    
    // NEW: Clean up log file if requested
    if pruneLogs && agent.LogFile != "" {
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

// Update summary message
if pruneLogs && logsRemoved > 0 {
    fmt.Printf("Removed %d agent(s) and %d log file(s).\n", removed, logsRemoved)
} else {
    fmt.Printf("Removed %d agent(s).\n", removed)
}

// In init():
pruneCmd.Flags().BoolVar(&pruneLogs, "logs", false, "Also delete log files for pruned agents")
```

### Confirmation message update

When `--logs` is specified, the confirmation prompt should mention log file deletion:

```go
if pruneLogs {
    fmt.Printf("This will remove %d terminated agent(s) and their log files. Are you sure? [y/N] ", len(terminated))
} else {
    fmt.Printf("This will remove %d terminated agent(s). Are you sure? [y/N] ", len(terminated))
}
```

### Output examples

Without `--logs` (current behavior preserved):
```
$ swarm prune
This will remove 3 terminated agent(s). Are you sure? [y/N] y
abc12345
def67890
ghi11111
Removed 3 agent(s).
```

With `--logs`:
```
$ swarm prune --logs
This will remove 3 terminated agent(s) and their log files. Are you sure? [y/N] y
abc12345
def67890
ghi11111
Removed 3 agent(s) and 3 log file(s).
```

With `--logs` when some agents don't have log files (foreground agents):
```
$ swarm prune --logs --force
abc12345
def67890
ghi11111
Removed 3 agent(s) and 2 log file(s).
```

## Edge cases

1. **Agent has no log file**: Foreground agents (not started with `-d`) don't have log files. Skip log deletion silently for these.

2. **Log file already deleted**: If the log file doesn't exist (already manually deleted), ignore the `os.IsNotExist` error and continue.

3. **Log file permission denied**: Print a warning but continue pruning other agents. Don't fail the entire prune operation.

4. **Empty log file path**: Some agents might have an empty string for `LogFile`. Check for this before attempting deletion.

## Acceptance criteria

- `swarm prune` without `--logs` works exactly as before (no log files deleted)
- `swarm prune --logs` deletes log files for all pruned agents that have them
- Confirmation message mentions log file deletion when `--logs` is specified
- Summary message shows count of deleted log files when applicable
- Missing log files are handled gracefully (no error, just skipped)
- Permission errors on log files show a warning but don't stop the prune operation
- Works correctly with `--force` flag to skip confirmation
- Foreground agents (no log file) are handled correctly
