# Add `--older-than` flag to `swarm prune` command

## Problem

Users accumulate many terminated agents and log files over time. Currently, `swarm prune` removes ALL terminated agents, which is an all-or-nothing approach. Users may want to:

1. Keep recent terminated agents for debugging/reference
2. Only clean up agents older than a certain time period
3. Implement a retention policy (e.g., "keep last 7 days")

The `doctor-command.pending.md` already suggests `swarm prune --logs --older-than 7d` but this flag doesn't exist yet.

## Solution

Add an `--older-than` flag to `swarm prune` that filters which terminated agents are pruned based on their termination time (or start time if termination time isn't tracked).

### Proposed API

```bash
# Prune agents terminated more than 7 days ago
swarm prune --older-than 7d

# Prune agents and logs older than 24 hours
swarm prune --logs --older-than 24h

# Prune agents older than 30 minutes
swarm prune --older-than 30m

# Combine with force flag
swarm prune --older-than 7d --force

# Prune ALL terminated agents (current behavior, no change)
swarm prune
```

### Supported duration formats

Use the same duration parsing as `swarm logs --since`:
- `30s` - 30 seconds
- `5m` - 5 minutes
- `2h` - 2 hours
- `1d` - 1 day (24 hours)
- `7d` - 7 days

## Files to change

- `cmd/prune.go` - add `--older-than` flag and filtering logic

## Implementation details

### prune.go changes

```go
var (
    pruneForce    bool
    pruneLogs     bool
    pruneOlderThan string  // NEW
)

// Add helper function (reuse from logs.go or extract to shared location)
func parseDurationWithDays(s string) (time.Duration, error) {
    if strings.HasSuffix(s, "d") {
        days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
        if err != nil {
            return 0, err
        }
        return time.Duration(days) * 24 * time.Hour, nil
    }
    return time.ParseDuration(s)
}

// In RunE, filter terminated agents by age:
func(cmd *cobra.Command, args []string) error {
    // ... existing code to get all terminated agents ...

    // NEW: Parse --older-than if specified
    var cutoffTime time.Time
    if pruneOlderThan != "" {
        duration, err := parseDurationWithDays(pruneOlderThan)
        if err != nil {
            return fmt.Errorf("invalid --older-than format: %w (use 30m, 2h, 7d, etc.)", err)
        }
        cutoffTime = time.Now().Add(-duration)
    }

    // Filter to only terminated agents (and optionally by age)
    var terminated []*state.AgentState
    for _, agent := range agents {
        if agent.Status != "terminated" {
            continue
        }
        
        // NEW: Filter by age if --older-than specified
        if !cutoffTime.IsZero() {
            // Use StartedAt as the reference time
            // (TerminatedAt would be better if we tracked it)
            if agent.StartedAt.After(cutoffTime) {
                continue  // Skip agents newer than cutoff
            }
        }
        
        terminated = append(terminated, agent)
    }

    // ... rest of existing prune logic ...
}

// In init():
pruneCmd.Flags().StringVar(&pruneOlderThan, "older-than", "", "Only prune agents older than duration (e.g., 7d, 24h, 30m)")
```

### Updated confirmation message

```go
if pruneOlderThan != "" {
    if pruneLogs {
        fmt.Printf("This will remove %d terminated agent(s) older than %s and their log files. Are you sure? [y/N] ", 
            len(terminated), pruneOlderThan)
    } else {
        fmt.Printf("This will remove %d terminated agent(s) older than %s. Are you sure? [y/N] ", 
            len(terminated), pruneOlderThan)
    }
} else {
    // existing messages
}
```

### Output examples

With `--older-than`:
```
$ swarm prune --older-than 7d
This will remove 5 terminated agent(s) older than 7d. Are you sure? [y/N] y
abc12345
def67890
ghi11111
jkl22222
mno33333
Removed 5 agent(s).
```

With `--older-than` and `--logs`:
```
$ swarm prune --older-than 24h --logs --force
abc12345
def67890
Removed 2 agent(s) and 2 log file(s).
```

No matches:
```
$ swarm prune --older-than 1h
No terminated agents older than 1h to remove.
```

## Edge cases

1. **No terminated agents match**: Print "No terminated agents older than X to remove." and exit cleanly.

2. **Invalid duration format**: Return error with helpful message showing valid formats.

3. **Zero duration**: `--older-than 0s` should prune all terminated agents (same as no flag).

4. **Missing TerminatedAt field**: Use `StartedAt` as the reference time since `AgentState` doesn't currently track termination time. This is slightly imprecise but good enough for cleanup purposes.

5. **Combined with scope**: `--older-than` should respect `--global` flag, only showing/pruning agents in the relevant scope.

## Future enhancement

Consider adding a `TerminatedAt` field to `AgentState` to track when agents were terminated. This would make `--older-than` more accurate (based on termination time rather than start time). This is out of scope for this feature but noted for future improvement.

## Acceptance criteria

- `swarm prune --older-than 7d` only removes agents started more than 7 days ago
- Duration parsing supports: seconds (s), minutes (m), hours (h), days (d)
- Invalid duration format returns helpful error message
- Confirmation message includes the duration when `--older-than` is specified
- "No agents to remove" message includes the duration
- Works correctly with `--logs` flag
- Works correctly with `--force` flag
- Works correctly with `--global` flag
- Existing behavior unchanged when `--older-than` is not specified

---

## Completion Notes (Agent cd59a862)

**Completed on:** 2026-01-28

**Files modified:**
- `cmd/prune.go` - Added `--older-than` flag with duration parsing and filtering logic

**Implementation summary:**
- Added `pruneOlderThan` variable and `--older-than` flag
- Added `pruneParseDurationWithDays()` helper function for parsing durations with day support
- Updated RunE to filter terminated agents by age based on `StartedAt` field
- Updated confirmation messages to include the duration when specified
- Updated "no agents" message to include the duration

**All acceptance criteria met:**
- `swarm prune --older-than 7d` filters agents by start time
- Duration parsing supports: seconds (s), minutes (m), hours (h), days (d)
- Invalid duration format returns helpful error message
- Confirmation message includes duration when `--older-than` specified
- "No agents to remove" message includes duration
- Works correctly with `--logs`, `--force`, and `--global` flags
- Existing behavior unchanged when `--older-than` is not specified

**Testing performed:**
- `swarm prune --help` - Shows new flag with documentation
- `swarm prune --older-than 7d` - Returns "No terminated agents older than 7d to remove."
- `swarm prune --older-than invalid` - Returns helpful error message
- `swarm prune --older-than 30m` - Correctly finds older agents and prompts for confirmation
