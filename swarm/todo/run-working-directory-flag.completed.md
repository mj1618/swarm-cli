# Add `--working-dir` flag to `swarm run` command

## Problem

Currently, the `swarm run` command always uses the current working directory as the agent's working directory. Users who want to run an agent in a different directory must first `cd` to that directory.

This is inconvenient when:

1. Running multiple agents in different directories from a single terminal
2. Scripting agent runs across multiple projects
3. Running an agent from a parent directory into a subdirectory

For example, to run agents in frontend and backend directories, users must:

```bash
# Current approach - verbose and changes shell state
cd frontend && swarm run -p coder -d && cd ..
cd backend && swarm run -p coder -d && cd ..

# Or use subshells
(cd frontend && swarm run -p coder -d)
(cd backend && swarm run -p coder -d)
```

## Solution

Add a `--working-dir` (short form `-C`) flag to the `swarm run` command that specifies the directory where the agent should run. This follows the convention of Git's `-C` flag and Docker's `-w` flag.

### Proposed API

```bash
# Run agent in a specific directory
swarm run -p coder -C /path/to/project

# Run agent in a relative directory
swarm run -p frontend -C ./frontend -d

# Short form
swarm run -p coder -C backend

# Multiple agents in different directories from one terminal
swarm run -p frontend -C ./frontend -d
swarm run -p backend -C ./backend -d
swarm run -p e2e -C ./e2e-tests -d
```

### Behavior

- The specified directory becomes the agent's working directory
- The agent process will `chdir` to that directory before running
- Prompts directory is resolved relative to the specified working directory (for project scope)
- The directory must exist and be accessible
- Relative paths are resolved relative to the current working directory
- The `WorkingDir` field in agent state is set to the resolved absolute path

## Files to change

- `cmd/run.go` - Add the `--working-dir` flag and handling logic
- `internal/detach/detach.go` - Ensure detached processes inherit the working directory correctly (may already work via existing `workingDir` param)

## Implementation details

### cmd/run.go changes

```go
var (
    // ... existing vars ...
    runWorkingDir string
)

var runCmd = &cobra.Command{
    // ... existing fields ...
    Example: `  # ... existing examples ...

  # Run agent in a specific directory
  swarm run -p coder -C /path/to/project

  # Run agent in a subdirectory
  swarm run -p frontend -C ./frontend -d`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Get working directory (from flag or current)
        var workingDir string
        var err error
        
        if runWorkingDir != "" {
            // Resolve relative to current directory
            if filepath.IsAbs(runWorkingDir) {
                workingDir = runWorkingDir
            } else {
                cwd, err := os.Getwd()
                if err != nil {
                    return fmt.Errorf("failed to get current directory: %w", err)
                }
                workingDir = filepath.Join(cwd, runWorkingDir)
            }
            
            // Verify directory exists
            info, err := os.Stat(workingDir)
            if err != nil {
                if os.IsNotExist(err) {
                    return fmt.Errorf("working directory does not exist: %s", workingDir)
                }
                return fmt.Errorf("failed to access working directory: %w", err)
            }
            if !info.IsDir() {
                return fmt.Errorf("not a directory: %s", workingDir)
            }
            
            // Get absolute path for consistency
            workingDir, err = filepath.Abs(workingDir)
            if err != nil {
                return fmt.Errorf("failed to resolve working directory: %w", err)
            }
        } else {
            workingDir, err = scope.CurrentWorkingDir()
            if err != nil {
                return fmt.Errorf("failed to get working directory: %w", err)
            }
        }

        // Get prompts directory based on scope AND working directory
        // (need to update GetPromptsDir to accept workingDir param)
        promptsDir, err := GetPromptsDirForPath(workingDir)
        if err != nil {
            return fmt.Errorf("failed to get prompts directory: %w", err)
        }
        
        // ... rest of existing logic, using resolved workingDir ...
    },
}

func init() {
    // ... existing flags ...
    runCmd.Flags().StringVarP(&runWorkingDir, "working-dir", "C", "", "Run agent in specified directory")
}
```

### For detached mode

The detached mode already passes `workingDir` to `detach.StartDetached()`, so it should work with minimal changes. The flag value just needs to be passed through:

```go
if runDetach && !runInternalDetached {
    // ... existing code ...
    
    // Build args for the detached process
    detachedArgs := []string{"run", "--_internal-detached", "--_internal-task-id", taskID}
    
    // Pass working dir to child if specified
    if runWorkingDir != "" {
        detachedArgs = append(detachedArgs, "--working-dir", workingDir) // Use resolved absolute path
    }
    
    // ... rest of detached args ...
    
    // Start detached process in the specified directory
    pid, err := detach.StartDetached(detachedArgs, logFile, workingDir)
}
```

## Edge cases

1. **Non-existent directory**: Return clear error "working directory does not exist: /path"
2. **File instead of directory**: Return error "not a directory: /path/file"
3. **No permissions**: Return error from os.Stat with permission denied
4. **Relative path resolution**: Resolve relative to current directory before passing to agent
5. **Detached mode**: Pass resolved absolute path to child process
6. **Prompt discovery**: In project scope, prompts dir should be relative to the working directory
7. **State tracking**: Agent's WorkingDir field uses the resolved path for filtering

## Use cases

### Multi-project orchestration script

```bash
#!/bin/bash
# Run agents across multiple services
SERVICES=("frontend" "backend" "api-gateway" "worker")

for service in "${SERVICES[@]}"; do
    swarm run -p "$service" -C "./$service" -d -n 10 -N "$service-agent"
done

# Wait for all to complete
swarm wait $(swarm list -q | tr '\n' ' ')
```

### CI/CD integration

```bash
# Run tests in the test directory without changing shell state
swarm run -p test-runner -C ./tests --model sonnet -n 1
```

### Monorepo workflow

```bash
# From monorepo root, run agents in specific packages
swarm run -p coder -C packages/core -d
swarm run -p coder -C packages/ui -d
swarm run -p coder -C packages/cli -d
```

## Acceptance criteria

- `swarm run -p prompt -C /path/to/dir` runs the agent in the specified directory
- `swarm run -p prompt -C ./relative/path` resolves relative paths correctly
- Error message when directory doesn't exist is clear
- Error message when path is a file (not directory) is clear
- Detached mode (`-d`) works correctly with `-C` flag
- Agent state shows correct WorkingDir
- `swarm list` filtering by project scope works correctly
- Prompts are discovered from the specified working directory in project scope

---

## Completion Notes (Agent cd59a862)

**Completed:** 2026-01-28

### Changes Made

1. **cmd/run.go**:
   - Added `runWorkingDir` variable for the new flag
   - Added `--working-dir` / `-C` flag definition in `init()`
   - Added working directory validation and resolution logic at the start of `RunE`:
     - Resolves relative paths against current directory
     - Validates directory exists and is actually a directory
     - Returns clear error messages for edge cases
   - Updated prompts directory resolution to use custom working directory for project scope
   - Added passing of `--working-dir` flag to detached child process (using resolved absolute path)
   - Added examples in command help text

2. **Imports**: Added `path/filepath` import

### Testing Performed

- Verified `--help` shows the new `-C`/`--working-dir` flag
- Tested error case: non-existent directory → "working directory does not exist: /path"
- Tested error case: file instead of directory → "not a directory: /path/file"
- Tested valid directory: command runs successfully with custom working directory
- All existing Go tests pass (`go test ./...`)
