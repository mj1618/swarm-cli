# Show paused status in `swarm inspect` command

## Problem

The `swarm inspect` command doesn't display when an agent is paused, creating an inconsistency with `swarm list`.

In `cmd/list.go` (lines 122-128), paused agents display "paused" status in yellow:

```go
if a.Paused {
    statusStr = "paused"
    statusColor = color.New(color.FgYellow)
}
```

However, in `cmd/inspect.go` (lines 68-76), the status handling only considers "running" and "terminated":

```go
statusColor := color.New(color.FgWhite)
switch agent.Status {
case "running":
    statusColor = color.New(color.FgGreen)
case "terminated":
    statusColor = color.New(color.FgRed)
}
fmt.Print("Status:        ")
statusColor.Println(agent.Status)
```

This means when a user runs `swarm stop my-agent` to pause it, then runs `swarm inspect my-agent`, they see:

```
Status:        running
```

But `swarm list` shows:

```
paused
```

This is confusing and makes the inspect command less useful for understanding agent state.

## Solution

Update `cmd/inspect.go` to check the `Paused` field and display "paused" status with yellow coloring, matching the behavior of `swarm list`.

## Files to change

- `cmd/inspect.go` - update status display logic to handle paused state

## Implementation details

Replace the status handling block in `inspect.go`:

```go
statusColor := color.New(color.FgWhite)
statusStr := agent.Status
switch agent.Status {
case "running":
    if agent.Paused {
        statusStr = "paused"
        statusColor = color.New(color.FgYellow)
    } else {
        statusColor = color.New(color.FgGreen)
    }
case "terminated":
    statusColor = color.New(color.FgRed)
}
fmt.Print("Status:        ")
statusColor.Println(statusStr)
```

### Output examples

Before (paused agent):
```
$ swarm inspect my-agent
Agent Details
─────────────────────────────────
ID:            abc12345
Name:          my-agent
PID:           12345
Prompt:        planner
Model:         claude-opus-4-20250514
Status:        running          <-- misleading
...
```

After (paused agent):
```
$ swarm inspect my-agent
Agent Details
─────────────────────────────────
ID:            abc12345
Name:          my-agent
PID:           12345
Prompt:        planner
Model:         claude-opus-4-20250514
Status:        paused           <-- accurate, displayed in yellow
...
```

## Acceptance criteria

- `swarm inspect <agent>` shows "paused" (in yellow) when agent has `Paused: true`
- `swarm inspect <agent>` shows "running" (in green) when agent is running and not paused
- `swarm inspect <agent>` shows "terminated" (in red) when agent is terminated
- Behavior is consistent with `swarm list` output
- JSON output (`--format json`) still includes the raw `status` and `paused` fields (no change needed there)

---

## Completion Notes (Agent 118d3fa6)

**Completed on:** 2026-01-28

**Files modified:**
- `cmd/inspect.go` - Updated status display logic to check `agent.Paused` field

**Changes made:**
- Added `statusStr` variable to track display string (matching `list.go` pattern)
- Added check for `agent.Paused` when status is "running"
- Display "paused" in yellow when agent is paused, "running" in green otherwise
- Behavior is now consistent with `swarm list` output

**All acceptance criteria met:**
- `swarm inspect <agent>` shows "paused" (in yellow) when agent has `Paused: true`
- `swarm inspect <agent>` shows "running" (in green) when agent is running and not paused  
- `swarm inspect <agent>` shows "terminated" (in red) when agent is terminated
- Behavior is consistent with `swarm list` output
- JSON output unchanged (still includes raw `status` and `paused` fields)
- All tests pass
