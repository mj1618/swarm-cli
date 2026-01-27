# Add label/tag support for agents

## Completion Notes

Implemented label/tag support for agents (completed by cd59a862):

### Changes Made:
- Created `internal/label/label.go` with Parse, ParseMultiple, Match, Format, and Merge functions
- Created `internal/label/label_test.go` with comprehensive tests
- Modified `internal/state/manager.go` to add Labels field to AgentState
- Modified `cmd/run.go` to add `--label/-l` flag for attaching labels
- Modified `cmd/list.go` to add `--label/-L` filter flag and `--show-labels` display flag  
- Modified `cmd/inspect.go` to display labels in output
- Modified `cmd/kill.go` to add `--label/-l` flag for batch kill operations
- Modified `cmd/stop.go` to add `--label/-l` flag for batch stop operations
- Modified `cmd/update.go` to add `--filter-label` and `--set-label/-l` flags for batch updates and label modification
- Modified `cmd/restart.go` to preserve labels and allow adding new ones via `--label/-l`
- Updated `cmd/list_test.go` to work with new filterAgents signature

All tests pass. Labels can now be used for organizing and filtering agents across all operations.

---

## Problem

When running multiple agents, especially across different projects or teams, there's no way to organize or categorize them beyond the name. Users who want to:

1. Run agents for different purposes (testing, deployment, refactoring) and filter by purpose
2. Assign agents to teams or owners and find all agents for a specific team
3. Mark agents with priority levels and batch-kill low-priority agents
4. Track which feature or ticket an agent is working on

...have no clean way to do this. The current filtering options (`--name`, `--prompt`, `--model`, `--status`) are useful but don't support arbitrary categorization.

Common workarounds are clunky:
```bash
# Currently: encode tags in the name (ugly, limited)
swarm run -p task -N "frontend-team-high-priority-TICKET-123"

# Or: maintain external tracking
echo "abc123: team=frontend, priority=high" >> agent-tracker.txt
```

## Solution

Add label/tag support similar to Docker labels or Kubernetes labels. Labels are key-value pairs that can be attached to agents at creation time and used for filtering.

### Proposed API

#### Adding labels when running

```bash
# Single label
swarm run -p task -l team=frontend

# Multiple labels
swarm run -p task -l team=frontend -l priority=high -l ticket=PROJ-123

# Long form
swarm run -p task --label team=frontend --label env=staging
```

#### Filtering by labels in list

```bash
# Filter by label (exact match)
swarm list --label team=frontend

# Multiple label filters (AND logic)
swarm list --label team=frontend --label priority=high

# Label existence check (has the label, any value)
swarm list --label team

# Combine with existing filters
swarm list --label team=frontend --status running --last 5
```

#### Viewing labels

```bash
# In list output (optional column, shown with -L flag)
swarm list -L
# ID        NAME      LABELS                          STATUS    ...
# abc123    worker    team=frontend,priority=high     running   ...

# In inspect output (always shown if present)
swarm inspect abc123
# ...
# Labels:        team=frontend, priority=high
# ...

# In JSON output (always included)
swarm list --format json
# [{"id": "abc123", "labels": {"team": "frontend", "priority": "high"}, ...}]
```

#### Batch operations with labels

```bash
# Kill all agents with a specific label
swarm kill --label priority=low

# Pause all agents for a team
swarm stop --label team=backend

# Update iterations for all labeled agents
swarm update --label env=staging -n 5
```

### Label format

- Keys: alphanumeric, hyphens, underscores, dots (like `team`, `app.kubernetes.io/name`)
- Values: alphanumeric, hyphens, underscores, dots, slashes
- Max key length: 63 characters
- Max value length: 253 characters
- Reserved prefix: `swarm.` (for future internal use)

Examples of valid labels:
- `team=frontend`
- `priority=high`
- `app.kubernetes.io/name=my-app`
- `ticket=PROJ-123`
- `owner=alice`

## Files to create/change

- Modify `internal/state/manager.go` - add Labels field to AgentState
- Modify `cmd/run.go` - add `--label/-l` flag
- Modify `cmd/restart.go` - preserve labels, allow adding new ones
- Modify `cmd/list.go` - add `--label` filter flag and `-L` display flag
- Modify `cmd/inspect.go` - display labels
- Modify `cmd/kill.go` - add `--label` flag for batch operations
- Modify `cmd/stop.go` - add `--label` flag for batch operations
- Modify `cmd/update.go` - add `--label` flag for batch operations, allow modifying labels

## Implementation details

### State changes

```go
// internal/state/manager.go

type AgentState struct {
    ID            string            `json:"id"`
    Name          string            `json:"name,omitempty"`
    Labels        map[string]string `json:"labels,omitempty"` // NEW
    PID           int               `json:"pid"`
    // ... rest unchanged
}
```

### Label parsing helper

```go
// internal/label/label.go

package label

import (
    "fmt"
    "regexp"
    "strings"
)

var (
    keyRegex   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9._-]{0,62}$`)
    valueRegex = regexp.MustCompile(`^[a-zA-Z0-9._/-]{0,253}$`)
)

// Parse parses a label string in the format "key=value" or "key".
// Returns key, value, error. If no value, returns empty string for value.
func Parse(s string) (string, string, error) {
    parts := strings.SplitN(s, "=", 2)
    key := parts[0]
    
    if !keyRegex.MatchString(key) {
        return "", "", fmt.Errorf("invalid label key: %s", key)
    }
    
    if strings.HasPrefix(key, "swarm.") {
        return "", "", fmt.Errorf("label key cannot use reserved prefix 'swarm.'")
    }
    
    if len(parts) == 1 {
        return key, "", nil
    }
    
    value := parts[1]
    if !valueRegex.MatchString(value) {
        return "", "", fmt.Errorf("invalid label value: %s", value)
    }
    
    return key, value, nil
}

// ParseMultiple parses multiple label strings.
func ParseMultiple(labels []string) (map[string]string, error) {
    result := make(map[string]string)
    for _, l := range labels {
        key, value, err := Parse(l)
        if err != nil {
            return nil, err
        }
        result[key] = value
    }
    return result, nil
}

// Match checks if an agent's labels match the filter labels.
// For filters with values, exact match is required.
// For filters without values (key only), label existence is checked.
func Match(agentLabels, filterLabels map[string]string) bool {
    for key, filterValue := range filterLabels {
        agentValue, exists := agentLabels[key]
        if !exists {
            return false
        }
        // If filter has a value, it must match exactly
        if filterValue != "" && agentValue != filterValue {
            return false
        }
    }
    return true
}

// Format formats labels for display.
func Format(labels map[string]string) string {
    if len(labels) == 0 {
        return "-"
    }
    
    pairs := make([]string, 0, len(labels))
    for k, v := range labels {
        if v == "" {
            pairs = append(pairs, k)
        } else {
            pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
        }
    }
    sort.Strings(pairs)
    return strings.Join(pairs, ",")
}
```

### Run command changes

```go
// cmd/run.go

var (
    // ... existing vars
    runLabels []string // NEW
)

var runCmd = &cobra.Command{
    // ...
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... existing code ...
        
        // Parse labels
        labels, err := label.ParseMultiple(runLabels)
        if err != nil {
            return fmt.Errorf("invalid label: %w", err)
        }
        
        // Include labels in agent state
        agentState := &state.AgentState{
            // ... existing fields ...
            Labels: labels, // NEW
        }
        
        // ... rest unchanged
    },
}

func init() {
    // ... existing flags ...
    runCmd.Flags().StringArrayVarP(&runLabels, "label", "l", nil, "Label to attach (key=value format, can be repeated)")
}
```

### List command changes

```go
// cmd/list.go

var (
    // ... existing vars
    listLabels     []string // NEW: filter by labels
    listShowLabels bool     // NEW: show labels column
)

var listCmd = &cobra.Command{
    // ... update Long description ...
    RunE: func(cmd *cobra.Command, args []string) error {
        // ... existing code ...
        
        // Parse label filters
        labelFilters, err := label.ParseMultiple(listLabels)
        if err != nil {
            return fmt.Errorf("invalid label filter: %w", err)
        }
        
        // Apply filters including labels
        agents = filterAgents(agents, listName, listPrompt, listModel, listStatus, labelFilters)
        
        // ... rest with optional label column ...
    },
}

// Update filterAgents to include label filtering
func filterAgents(agents []*state.AgentState, nameFilter, promptFilter, modelFilter, statusFilter string, labelFilters map[string]string) []*state.AgentState {
    // ... existing filter code ...
    
    // Check label filters
    if len(labelFilters) > 0 && !label.Match(agent.Labels, labelFilters) {
        continue
    }
    
    // ... rest unchanged
}

func init() {
    // ... existing flags ...
    listCmd.Flags().StringArrayVarP(&listLabels, "label", "l", nil, "Filter by label (key=value or key, can be repeated)")
    listCmd.Flags().BoolVarP(&listShowLabels, "show-labels", "L", false, "Show labels column in output")
}
```

## Use cases

### Team organization

```bash
# Frontend team starts their agents
swarm run -p frontend-task -l team=frontend -l owner=alice -d
swarm run -p frontend-task -l team=frontend -l owner=bob -d

# Backend team starts their agents  
swarm run -p backend-task -l team=backend -l owner=charlie -d

# View all frontend team agents
swarm list --label team=frontend

# View Alice's agents
swarm list --label owner=alice
```

### Priority-based management

```bash
# Start agents with different priorities
swarm run -p urgent-fix -l priority=critical -n 1 -d
swarm run -p feature-work -l priority=normal -n 20 -d
swarm run -p cleanup -l priority=low -n 50 -d

# When resources are tight, kill low-priority agents
swarm kill --label priority=low

# Check on critical work
swarm list --label priority=critical
```

### Feature/ticket tracking

```bash
# Track which ticket each agent is working on
swarm run -p implement-auth -l ticket=PROJ-456 -l feature=auth -d
swarm run -p implement-auth -l ticket=PROJ-456 -l feature=auth -d

# Find all agents working on a ticket
swarm list --label ticket=PROJ-456

# Kill all agents for a cancelled feature
swarm kill --label feature=deprecated-feature
```

### Environment separation

```bash
# Development agents
swarm run -p dev-task -l env=dev -d

# Staging agents  
swarm run -p staging-task -l env=staging -d

# Kill all staging agents before deployment
swarm kill --label env=staging
```

## Edge cases

1. **Empty labels map**: Agents without labels should work exactly as they do today. Filter `--label key` should not match agents without that key.

2. **Label conflicts on restart**: When restarting, preserve original labels but allow adding/overriding with new `--label` flags.

3. **Duplicate label keys**: Later `-l` flags override earlier ones: `-l team=a -l team=b` results in `team=b`.

4. **Special characters**: Labels with special characters should be rejected with a clear error message.

5. **Case sensitivity**: Labels are case-sensitive (`Team=A` and `team=A` are different).

6. **Backward compatibility**: Existing state files without labels should load correctly (Labels will be nil/empty).

## Acceptance criteria

- `swarm run -l key=value` attaches a label to the agent
- Multiple `-l` flags can be used to attach multiple labels
- `swarm list --label key=value` filters to agents with that exact label
- `swarm list --label key` filters to agents that have that key (any value)
- Multiple `--label` filters use AND logic
- `swarm list -L` shows a labels column
- `swarm inspect` shows labels in output
- `swarm list --format json` includes labels in JSON output
- `swarm kill --label key=value` kills all matching agents
- `swarm stop --label key=value` stops all matching agents
- Invalid label formats are rejected with helpful error messages
- Labels starting with `swarm.` are rejected (reserved)
- Existing agents without labels continue to work
