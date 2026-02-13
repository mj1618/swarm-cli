# Add `--count`, `--last`, and `--latest` flags to list command

## Completion Note (Agent cd59a862)

Implemented all three flags as specified:

**Changes made to `cmd/list.go`:**
- Added `--count` flag that outputs only the count of matching agents
- Added `--last N` / `-n N` flag that shows N most recently started agents (newest first)
- Added `--latest` / `-l` flag as alias for `--last 1`
- Added JSON support for count mode: `{"count": N}`
- Added validation: `--count` and `--quiet` are mutually exclusive
- Added validation: `--last` must be a positive number
- Updated help text and examples

**All acceptance criteria met:**
- `swarm list --count` outputs only the count
- `swarm list --last N` shows N most recently started agents
- `swarm list --latest` or `-l` shows only the most recent agent
- `-n` is alias for `--last`
- `--last` and `--latest` reverse the sort order (newest first)
- `--count` works with all filters
- `--last` works with `--quiet`, `--format json`, and filters
- `--count` and `--quiet` are mutually exclusive with clear error
- When no agents match, `--count` outputs `0`

---

## Problem

The `swarm list` command follows docker CLI conventions (as noted in PLAN.md), but is missing some useful flags from `docker ps`:

1. **No count option**: Users who want to know "how many agents are running?" must pipe through `wc -l` or use `--quiet` and count lines. This is common in scripts and monitoring.

2. **No way to limit results**: When there are many agents, users often just want to see the most recent ones. `docker ps` has `--last N` (show N most recently created) and `--latest` (show only the most recent).

3. **Oldest-first sort only**: Currently agents are sorted by `StartedAt` (oldest first), but there's no way to reverse this to see newest first.

Common workflows affected:
```bash
# Currently awkward:
swarm list -q | wc -l                    # Count running agents
swarm list -aq | wc -l                   # Count all agents
swarm list -a | tail -5                  # See 5 most recent (unreliable with headers)

# Would be nicer:
swarm list --count                       # Just show count
swarm list --last 5                      # Show 5 most recent
swarm list --latest                      # Show the most recent agent
```

## Solution

Add three new flags to the list command:

### Proposed API

```bash
# Count flags
swarm list --count                 # Output: 3 (just the number)
swarm list -a --count              # Count including terminated
swarm list --name coder --count    # Count matching filter

# Last/Latest flags
swarm list --last 5                # Show 5 most recently started agents
swarm list -n 5                    # Short form of --last
swarm list --latest                # Show only the most recent agent (equivalent to --last 1)
swarm list -l                      # Short form of --latest

# Combined with other flags
swarm list --last 10 -a            # Last 10 including terminated
swarm list --latest --format json  # Most recent as JSON
swarm list --name coder --last 3   # Last 3 matching filter
swarm list -l -q                   # Just ID of most recent agent
```

### Output examples

**Count mode:**
```bash
$ swarm list --count
3

$ swarm list -a --count
18

$ swarm list --name coder --count
5
```

**Last/Latest mode:**
```
$ swarm list --last 3
ID          NAME        PROMPT               MODEL              STATUS        ITERATION   STARTED
a7b3c9d1    reviewer    review-code          claude-opus-4...   running       2/5         2m ago
f8e2a4b6    coder       implement-feature    claude-opus-4...   running       5/10        15m ago
c1d5e9f3    planner     plan-sprint          claude-opus-4...   terminated    10/10       1h ago

$ swarm list --latest -q
a7b3c9d1
```

## Files to create/change

- Modify `cmd/list.go` - add new flags and logic

## Implementation details

### Changes to cmd/list.go

```go
var (
    listAll     bool
    listQuiet   bool
    listFormat  string
    listName    string
    listPrompt  string
    listModel   string
    listStatus  string
    listCount   bool    // NEW
    listLast    int     // NEW
    listLatest  bool    // NEW
)

var listCmd = &cobra.Command{
    // ... existing Use, Aliases, Short ...
    Long: `List running agents with their status and configuration.

By default, only shows running agents started in the current directory.
Use --all to include terminated agents.
Use --global to show agents from all directories.

Filter options:
  --name, -N      Filter by agent name (substring match, case-insensitive)
  --prompt, -p    Filter by prompt name (substring match, case-insensitive)
  --model, -m     Filter by model name (substring match, case-insensitive)
  --status        Filter by status (running, pausing, paused, or terminated)

Output options:
  --count         Output only the count of matching agents
  --last, -n      Show only the N most recently started agents
  --latest, -l    Show only the most recently started agent (same as --last 1)

Multiple filters are combined with AND logic (all conditions must match).`,
    Example: `  # List running agents in current project
  swarm list

  # Count running agents
  swarm list --count

  # Count all agents including terminated
  swarm list -a --count

  # Show 5 most recently started agents
  swarm list --last 5
  swarm list -n 5

  # Show the most recent agent
  swarm list --latest
  swarm list -l

  # Get ID of most recent agent (useful for scripting)
  swarm list -lq

  # Count agents matching a filter
  swarm list --name coder --count`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Handle --latest as alias for --last 1
        if listLatest {
            listLast = 1
        }

        // Validate flags
        if listCount && listQuiet {
            return fmt.Errorf("--count and --quiet cannot be used together")
        }
        if listLast < 0 {
            return fmt.Errorf("--last must be a positive number")
        }

        // ... existing validation and manager setup ...

        agents, err := mgr.List(onlyRunning)
        if err != nil {
            return fmt.Errorf("failed to list agents: %w", err)
        }

        // Apply filters
        agents = filterAgents(agents, listName, listPrompt, listModel, listStatus)

        // Apply --last limit (agents are sorted oldest-first, so we want last N)
        if listLast > 0 && len(agents) > listLast {
            agents = agents[len(agents)-listLast:]
        }

        // Reverse to show newest first when using --last or --latest
        if listLast > 0 {
            for i, j := 0, len(agents)-1; i < j; i, j = i+1, j-1 {
                agents[i], agents[j] = agents[j], agents[i]
            }
        }

        // Count mode - just output the number
        if listCount {
            fmt.Println(len(agents))
            return nil
        }

        // ... rest of existing logic (quiet mode, json format, table output) ...
    },
}

func init() {
    // ... existing flags ...
    
    // Count flag
    listCmd.Flags().BoolVar(&listCount, "count", false, "Output only the count of matching agents")
    
    // Last/Latest flags
    listCmd.Flags().IntVarP(&listLast, "last", "n", 0, "Show only the N most recently started agents")
    listCmd.Flags().BoolVarP(&listLatest, "latest", "l", false, "Show only the most recently started agent")
}
```

## Edge cases

1. **--count with --format json**: Should output `{"count": 3}` for consistency
2. **--last 0**: Treat as "no limit" (show all)
3. **--last with filters**: Apply filters first, then take last N of filtered results
4. **--last larger than result count**: Show all results, don't error
5. **--latest when no agents**: Output nothing (or 0 for count), no error
6. **--count with empty results**: Output `0`
7. **--last with --quiet**: Output only the N most recent IDs, one per line
8. **--latest and --last together**: --latest takes precedence (or error for clarity)

## Testing scenarios

```bash
# Setup: Have 10 agents, 3 running, 7 terminated

# Count tests
swarm list --count           # Outputs: 3
swarm list -a --count        # Outputs: 10
swarm list --status terminated --count  # Error: use -a flag hint
swarm list -a --status terminated --count  # Outputs: 7

# Last tests  
swarm list --last 2          # Shows 2 most recent running agents
swarm list -a --last 5       # Shows 5 most recent (any status)
swarm list --last 100        # Shows all 3 running (no error)

# Latest tests
swarm list --latest          # Shows most recent running agent
swarm list -l -q             # Outputs just the ID
swarm list -l --format json  # JSON output for 1 agent

# Combined
swarm list --name coder --last 3   # Last 3 agents with "coder" in name
swarm list --count --last 5        # Error: conflicting flags (or count of 5)
```

## Acceptance criteria

- `swarm list --count` outputs only the count of agents
- `swarm list --last N` shows N most recently started agents
- `swarm list --latest` or `-l` shows only the most recent agent
- `-n` is alias for `--last`
- `--last` and `--latest` reverse the sort order (newest first)
- `--count` works with all filters (--name, --status, etc.)
- `--last` works with `--quiet`, `--format json`, and filters
- `--count` and `--quiet` are mutually exclusive with clear error
- When no agents match, `--count` outputs `0`, `--last` outputs nothing
- All combinations work in both project and global scope
