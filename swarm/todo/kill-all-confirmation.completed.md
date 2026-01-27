# Add confirmation prompt to `swarm kill-all` command

## Completed by agent 5a26f1d6

### Changes made:
- Modified `cmd/kill_all.go` to add confirmation prompt before killing agents
- Added `--force` / `-f` flag to skip confirmation (for scripting)
- Confirmation shows scope context (project vs global)
- Confirmation lists agent names/IDs when count <= 5
- Non-interactive stdin without `--force` aborts safely
- Updated command help and examples
- Added tests in `cmd/kill_all_test.go`

All acceptance criteria have been met.

## Problem

The `swarm kill-all` command immediately terminates all running and paused agents without any confirmation prompt. This is dangerous because:

1. Users might accidentally run `kill-all` when they meant `kill <id>`
2. Running `swarm kill-all --global` could terminate agents across many projects
3. There's no way to see what agents will be affected before they're killed
4. This is inconsistent with `swarm prune` which requires confirmation and has a `--force` flag

Currently, `kill-all` behavior:

```bash
$ swarm kill-all --global
Killed 15 agent(s)
# Gone - no way to undo, no warning
```

Compare to `swarm prune` which has confirmation:

```bash
$ swarm prune
This will remove 3 terminated agent(s). Are you sure? [y/N]
```

## Solution

Add a confirmation prompt to `swarm kill-all` that shows how many agents will be affected, and add a `--force` / `-f` flag to skip confirmation (for scripting).

### Proposed API

```bash
# Show confirmation (new default)
swarm kill-all
# > This will terminate 3 running agent(s). Are you sure? [y/N]

# Skip confirmation
swarm kill-all --force

# Combine with existing flags
swarm kill-all --global --force
swarm kill-all --graceful --force
```

### Confirmation message should include

- Number of agents that will be affected
- Scope context (project vs global)
- List of agent names/IDs when count is small (e.g., <= 5)

## Files to change

- `cmd/kill_all.go` - add confirmation prompt and `--force` flag

## Implementation details

### kill_all.go changes

```go
var (
    killAllGraceful bool
    killAllForce    bool  // NEW
)

// In RunE, after counting agents but before killing:

if len(agents) == 0 {
    fmt.Println("No running or paused agents found")
    return nil
}

// Show confirmation unless --force is used
if !killAllForce {
    scopeStr := "in this project"
    if GetScope() == scope.ScopeGlobal {
        scopeStr = "globally (all projects)"
    }
    
    fmt.Printf("This will terminate %d agent(s) %s:\n", len(agents), scopeStr)
    
    // List agents if small number
    if len(agents) <= 5 {
        for _, agent := range agents {
            name := agent.ID
            if agent.Name != "" {
                name = fmt.Sprintf("%s (%s)", agent.Name, agent.ID)
            }
            fmt.Printf("  - %s\n", name)
        }
    }
    
    fmt.Print("Are you sure? [y/N] ")
    var response string
    fmt.Scanln(&response)
    
    if response != "y" && response != "Y" {
        fmt.Println("Aborted")
        return nil
    }
}

// ... existing kill logic ...

// In init():
killAllCmd.Flags().BoolVarP(&killAllForce, "force", "f", false, "Skip confirmation prompt")
```

### Output examples

Small number of agents:

```
$ swarm kill-all
This will terminate 2 agent(s) in this project:
  - planner (abc12345)
  - coder (def67890)
Are you sure? [y/N] y
Killed 2 agent(s)
```

Many agents (global mode):

```
$ swarm kill-all --global
This will terminate 15 agent(s) globally (all projects).
Are you sure? [y/N] n
Aborted
```

With force flag (for scripts):

```
$ swarm kill-all --force
Killed 3 agent(s)
```

Graceful mode with confirmation:

```
$ swarm kill-all --graceful
This will terminate 3 agent(s) in this project:
  - planner (abc12345)
  - coder (def67890)
  - tester (ghi11111)
Are you sure? [y/N] y
3 agent(s) will terminate after current iteration
```

## Edge cases

1. **Zero agents**: Don't show confirmation, just report "No running or paused agents found" (unchanged behavior).

2. **Single agent**: Still show confirmation - the user should use `swarm kill <id>` if they know which specific agent to kill.

3. **Mixed paused/running**: Confirmation message says "running and paused agents" if there are both types.

4. **Terminal not interactive**: If stdin is not a terminal (piped input), treat as "N" unless `--force` is used. This prevents accidental kills in scripts that forgot `--force`.

## Acceptance criteria

- `swarm kill-all` shows confirmation prompt with agent count before killing
- `swarm kill-all --force` skips confirmation and kills immediately
- `swarm kill-all -f` works with short flag
- Confirmation shows scope context (project vs global)
- Confirmation lists agent names/IDs when count <= 5
- Typing "n" or anything other than "y"/"Y" aborts the operation
- Non-interactive stdin without `--force` aborts safely
- `--force` can be combined with `--graceful` and `--global`
- Zero agents case doesn't show confirmation prompt
