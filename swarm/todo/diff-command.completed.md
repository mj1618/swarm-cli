# Add `swarm diff` command to show agent code changes

## Problem

When an agent runs, especially for multiple iterations, it makes changes to the codebase. Currently, users have no easy way to:

1. **Review what an agent changed** before committing the work
2. **Understand agent behavior** by seeing the actual code modifications
3. **Compare agent output** across different runs or models
4. **Debug problematic changes** by identifying which agent made what changes

Users must manually run `git diff` or `git log` and correlate timestamps with agent start times, which is tedious and error-prone. This is especially problematic when multiple agents are running simultaneously.

Common workflow pain points:
```bash
# Currently: manual correlation required
swarm inspect abc123  # Get start time: 2026-01-28 14:30:00
git log --since="2026-01-28 14:30:00"  # Find commits
git diff HEAD~3  # Hope you counted right

# Or for uncommitted changes, just guess what the agent did
git diff  # Is this from agent abc123 or def456?
```

## Solution

Add a `swarm diff` command that shows code changes associated with an agent's run, integrating with git to provide meaningful diffs.

### Proposed API

```bash
# Show all changes (committed + uncommitted) since agent started
swarm diff abc123

# Show only uncommitted changes
swarm diff abc123 --uncommitted

# Show only commits made during agent run
swarm diff abc123 --commits

# Show diff stats only (summary view)
swarm diff abc123 --stat

# Show diff for the most recent agent
swarm diff @last
swarm diff _

# Filter to specific files/paths
swarm diff abc123 -- src/components/

# Output as patch file
swarm diff abc123 --output patch.diff
```

### Default output

```
$ swarm diff abc123

Agent: abc123 (frontend-task)
Started: 2026-01-28 14:30:00 (25 minutes ago)
Status: running (iteration 3/5)

Commits during run:
  a1b2c3d  fix: resolve button styling issue
  d4e5f6g  feat: add loading spinner component
  h7i8j9k  refactor: extract utility functions

Uncommitted changes:
  M  src/components/Button.tsx
  M  src/components/Spinner.tsx
  A  src/utils/helpers.ts

─────────────────────────────────────────────────────────
diff --git a/src/components/Button.tsx b/src/components/Button.tsx
index abc1234..def5678 100644
--- a/src/components/Button.tsx
+++ b/src/components/Button.tsx
@@ -1,5 +1,8 @@
 import React from 'react';
+import { Spinner } from './Spinner';
...
```

### Summary/stat mode

```
$ swarm diff abc123 --stat

Agent: abc123 (frontend-task)
Started: 2026-01-28 14:30:00 (25 minutes ago)

 3 commits, 5 files changed, 142 insertions(+), 23 deletions(-)

 src/components/Button.tsx    | 45 +++++++++++++++++++++++---
 src/components/Spinner.tsx   | 62 ++++++++++++++++++++++++++++++++++
 src/utils/helpers.ts         | 35 +++++++++++++++++++
 src/App.tsx                  |  8 ++---
 package.json                 | 15 ++++++++-
```

## Files to create/change

- Create `cmd/diff.go` - new command implementation

## Implementation details

### cmd/diff.go

```go
package cmd

import (
    "fmt"
    "os"
    "os/exec"
    "strings"
    "time"

    "github.com/fatih/color"
    "github.com/mj1618/swarm-cli/internal/state"
    "github.com/spf13/cobra"
)

var (
    diffUncommitted bool
    diffCommits     bool
    diffStat        bool
    diffOutput      string
)

var diffCmd = &cobra.Command{
    Use:   "diff [process-id-or-name] [-- path...]",
    Short: "Show code changes made during an agent's run",
    Long: `Show git diff of changes made since an agent started running.

By default, shows both commits made during the run and any uncommitted changes.
Use --commits to show only committed changes, or --uncommitted for only
uncommitted changes.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent

Use -- to pass path filters to git diff.`,
    Example: `  # Show all changes since agent started
  swarm diff abc123

  # Show changes for most recent agent
  swarm diff @last

  # Show only uncommitted changes
  swarm diff abc123 --uncommitted

  # Show only commits made during run
  swarm diff abc123 --commits

  # Show summary statistics
  swarm diff abc123 --stat

  # Filter to specific directory
  swarm diff abc123 -- src/

  # Save diff to file
  swarm diff abc123 --output changes.diff`,
    Args: cobra.MinimumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        // Parse args - first arg is agent identifier, rest are path filters
        agentIdentifier := args[0]
        var pathFilters []string
        
        // Check for -- separator
        for i, arg := range args {
            if arg == "--" && i+1 < len(args) {
                pathFilters = args[i+1:]
                break
            }
        }

        // Create state manager with scope
        mgr, err := state.NewManagerWithScope(GetScope(), "")
        if err != nil {
            return fmt.Errorf("failed to initialize state manager: %w", err)
        }

        agent, err := ResolveAgentIdentifier(mgr, agentIdentifier)
        if err != nil {
            return fmt.Errorf("agent not found: %w", err)
        }

        // Check if we're in a git repository
        if !isGitRepo(agent.WorkingDir) {
            return fmt.Errorf("agent working directory is not a git repository: %s", agent.WorkingDir)
        }

        // Prepare output writer
        var output *os.File
        if diffOutput != "" {
            output, err = os.Create(diffOutput)
            if err != nil {
                return fmt.Errorf("failed to create output file: %w", err)
            }
            defer output.Close()
        } else {
            output = os.Stdout
        }

        // Print header (unless outputting to file)
        if diffOutput == "" {
            printDiffHeader(agent)
        }

        // Get commits made during the agent's run
        var commits []commitInfo
        if !diffUncommitted {
            commits, err = getCommitsSince(agent.WorkingDir, agent.StartedAt)
            if err != nil {
                // Non-fatal, just warn
                fmt.Fprintf(os.Stderr, "Warning: could not get commits: %v\n", err)
            }
        }

        // Show commits summary
        if len(commits) > 0 && diffOutput == "" && !diffStat {
            bold := color.New(color.Bold)
            bold.Println("\nCommits during run:")
            for _, c := range commits {
                fmt.Printf("  %s  %s\n", c.ShortHash, truncateString(c.Subject, 60))
            }
        }

        // Show uncommitted changes summary
        if !diffCommits && diffOutput == "" && !diffStat {
            uncommitted, _ := getUncommittedFiles(agent.WorkingDir)
            if len(uncommitted) > 0 {
                bold := color.New(color.Bold)
                bold.Println("\nUncommitted changes:")
                for _, f := range uncommitted {
                    fmt.Printf("  %s  %s\n", f.Status, f.Path)
                }
            }
        }

        // Generate the diff
        if diffOutput == "" && !diffStat {
            fmt.Println("\n─────────────────────────────────────────────────────────")
        }

        if diffStat {
            return showDiffStat(agent, commits, pathFilters, output)
        }

        return showFullDiff(agent, commits, pathFilters, output)
    },
}

type commitInfo struct {
    Hash      string
    ShortHash string
    Subject   string
    Author    string
    Date      time.Time
}

type fileStatus struct {
    Status string // "M", "A", "D", "??"
    Path   string
}

func isGitRepo(dir string) bool {
    cmd := exec.Command("git", "rev-parse", "--git-dir")
    cmd.Dir = dir
    return cmd.Run() == nil
}

func getCommitsSince(dir string, since time.Time) ([]commitInfo, error) {
    // Format: hash|short_hash|subject|author|date
    format := "%H|%h|%s|%an|%aI"
    sinceStr := since.Format(time.RFC3339)
    
    cmd := exec.Command("git", "log", "--since="+sinceStr, "--format="+format)
    cmd.Dir = dir
    
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    lines := strings.Split(strings.TrimSpace(string(output)), "\n")
    var commits []commitInfo
    
    for _, line := range lines {
        if line == "" {
            continue
        }
        parts := strings.SplitN(line, "|", 5)
        if len(parts) < 5 {
            continue
        }
        
        date, _ := time.Parse(time.RFC3339, parts[4])
        commits = append(commits, commitInfo{
            Hash:      parts[0],
            ShortHash: parts[1],
            Subject:   parts[2],
            Author:    parts[3],
            Date:      date,
        })
    }
    
    return commits, nil
}

func getUncommittedFiles(dir string) ([]fileStatus, error) {
    cmd := exec.Command("git", "status", "--porcelain")
    cmd.Dir = dir
    
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    
    lines := strings.Split(strings.TrimSpace(string(output)), "\n")
    var files []fileStatus
    
    for _, line := range lines {
        if len(line) < 3 {
            continue
        }
        status := strings.TrimSpace(line[:2])
        path := strings.TrimSpace(line[3:])
        files = append(files, fileStatus{Status: status, Path: path})
    }
    
    return files, nil
}

func showDiffStat(agent *state.AgentState, commits []commitInfo, pathFilters []string, output *os.File) error {
    bold := color.New(color.Bold)
    
    // Build git diff command for stats
    args := []string{"diff", "--stat"}
    
    if len(commits) > 0 && !diffUncommitted {
        // Show diff from before first commit to HEAD (plus uncommitted)
        args = append(args, commits[len(commits)-1].Hash+"^")
    }
    
    args = append(args, pathFilters...)
    
    cmd := exec.Command("git", args...)
    cmd.Dir = agent.WorkingDir
    cmd.Stdout = output
    cmd.Stderr = os.Stderr
    
    if output == os.Stdout {
        bold.Printf("\n %d commits", len(commits))
        
        // Get summary line
        summaryCmd := exec.Command("git", append([]string{"diff", "--shortstat"}, args[2:]...)...)
        summaryCmd.Dir = agent.WorkingDir
        summary, _ := summaryCmd.Output()
        if len(summary) > 0 {
            fmt.Printf(",%s\n", strings.TrimSpace(string(summary)))
        } else {
            fmt.Println()
        }
        fmt.Println()
    }
    
    return cmd.Run()
}

func showFullDiff(agent *state.AgentState, commits []commitInfo, pathFilters []string, output *os.File) error {
    // Build git diff command
    args := []string{"diff", "--color=always"}
    
    if diffCommits && len(commits) > 0 {
        // Show only committed changes: from before first commit to last commit
        args = append(args, commits[len(commits)-1].Hash+"^", commits[0].Hash)
    } else if diffUncommitted {
        // Show only uncommitted changes (default git diff behavior)
        // No additional args needed
    } else if len(commits) > 0 {
        // Show everything from before first commit to current state (including uncommitted)
        args = append(args, commits[len(commits)-1].Hash+"^")
    }
    // If no commits, just show uncommitted changes (default)
    
    args = append(args, "--")
    args = append(args, pathFilters...)
    
    cmd := exec.Command("git", args...)
    cmd.Dir = agent.WorkingDir
    cmd.Stdout = output
    cmd.Stderr = os.Stderr
    
    return cmd.Run()
}

func printDiffHeader(agent *state.AgentState) {
    bold := color.New(color.Bold)
    dim := color.New(color.Faint)
    
    name := agent.ID
    if agent.Name != "" {
        name = fmt.Sprintf("%s (%s)", agent.ID, agent.Name)
    }
    
    bold.Printf("Agent: %s\n", name)
    
    elapsed := time.Since(agent.StartedAt).Round(time.Second)
    fmt.Printf("Started: %s (%s ago)\n", agent.StartedAt.Format("2006-01-02 15:04:05"), elapsed)
    
    statusStr := agent.Status
    if agent.Status == "running" {
        if agent.Paused {
            statusStr = "paused"
        }
        statusStr += fmt.Sprintf(" (iteration %d/%d)", agent.CurrentIter, agent.Iterations)
    }
    dim.Printf("Status: %s\n", statusStr)
}

func truncateString(s string, max int) string {
    if len(s) <= max {
        return s
    }
    return s[:max-3] + "..."
}

func init() {
    diffCmd.Flags().BoolVar(&diffUncommitted, "uncommitted", false, "Show only uncommitted changes")
    diffCmd.Flags().BoolVar(&diffCommits, "commits", false, "Show only committed changes during run")
    diffCmd.Flags().BoolVar(&diffStat, "stat", false, "Show diffstat summary instead of full diff")
    diffCmd.Flags().StringVarP(&diffOutput, "output", "o", "", "Write diff to file instead of stdout")
    rootCmd.AddCommand(diffCmd)
}
```

## Use cases

### Reviewing agent work before committing

```bash
# Agent finishes a coding task
swarm run -p implement-feature -n 10 -d --name feature-x

# Later, review what it did
swarm diff feature-x

# Looks good, commit it
git add -A && git commit -m "feat: implement feature X (agent-assisted)"
```

### Comparing agent runs

```bash
# Run same task with different models
swarm run -p refactor-code -n 5 -d --name sonnet-run -m claude-sonnet-4-20250514
swarm wait sonnet-run
swarm diff sonnet-run --output sonnet.diff

swarm run -p refactor-code -n 5 -d --name opus-run -m claude-opus-4-20250514
swarm wait opus-run  
swarm diff opus-run --output opus.diff

# Compare the outputs
diff sonnet.diff opus.diff
```

### Quick check on running agent progress

```bash
# Agent is running, want to see what it's done so far
swarm diff @last --stat

# See full changes in a specific directory
swarm diff @last -- src/components/
```

### Debugging problematic changes

```bash
# Something broke, which agent did it?
swarm list -a
# ID        NAME        ...
# abc123    frontend    ...
# def456    backend     ...

swarm diff abc123 --stat
swarm diff def456 --stat

# Found it - backend agent made breaking changes
swarm diff def456
```

## Edge cases

1. **Agent not in git repo**: Show clear error message suggesting the feature requires git.

2. **No changes made**: Show message "No changes detected since agent started."

3. **Agent working directory changed/deleted**: Show error with helpful message.

4. **Commits made by other processes**: The diff includes ALL commits since agent start time. This is intentional for simplicity, but could add `--author` filter in future.

5. **Agent still running**: Works fine - shows changes up to current moment.

6. **Very old agent**: Git log/diff performance may degrade. Consider adding `--max-commits` limit.

7. **Binary files**: Handled by git's default diff behavior (shows "Binary files differ").

8. **Large diffs**: Consider adding pager support (pipe to `less` by default for terminal output).

## Dependencies

No new dependencies. Uses:
- `os/exec` for git commands
- Existing `state` package for agent lookup
- `github.com/fatih/color` for colored output (already in use)

## Future enhancements (out of scope)

1. **`--author` filter**: Only show commits by a specific author
2. **Automatic git stash tracking**: Track stash state at agent start for more accurate diffs
3. **Integration with `swarm inspect`**: Show diff summary in inspect output
4. **Web UI diff viewer**: Pretty HTML diff output
5. **Diff between two agents**: `swarm diff abc123..def456`

## Acceptance criteria

- `swarm diff <agent>` shows combined committed + uncommitted changes since agent started
- `swarm diff @last` works for most recent agent
- `swarm diff <agent> --uncommitted` shows only uncommitted changes
- `swarm diff <agent> --commits` shows only committed changes
- `swarm diff <agent> --stat` shows summary statistics
- `swarm diff <agent> -- path/` filters to specific paths
- `swarm diff <agent> -o file.diff` writes to file
- Colored output in terminal (respects NO_COLOR env var)
- Clear error when not in git repository
- Works with running, paused, and terminated agents
- Header shows agent info (ID, name, start time, status)

## Completion Notes

**Completed by agent cd59a862 on 2026-01-28**

Implementation:
- Created `cmd/diff.go` with the `swarm diff` command
- Registered the command in `cmd/root.go`
- Supports all specified features:
  - Show combined committed + uncommitted changes since agent started
  - `swarm diff @last` works for most recent agent
  - `--uncommitted` flag shows only uncommitted changes
  - `--commits` flag shows only committed changes
  - `--stat` flag shows summary statistics
  - Path filtering with `-- path/` syntax
  - `-o file.diff` flag writes to file
  - Colored output in terminal
  - Clear error messages for edge cases (not in git repo, agent not found)
  - Works with running, paused, and terminated agents
  - Header shows agent info (ID, name, start time, status)
  - Dynamic shell completion for agent identifiers
- All existing tests pass
- Code compiles successfully
