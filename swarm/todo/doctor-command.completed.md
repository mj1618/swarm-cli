# Add `swarm doctor` command for diagnostic troubleshooting

## Problem

Users encountering issues with swarm have no easy way to diagnose what's wrong. Common problems include:

1. **Missing CLI backends**: The configured agent command (cursor/claude) isn't installed or isn't in PATH
2. **Configuration errors**: Invalid TOML syntax, unknown settings, conflicting options
3. **Disk space issues**: Log directory is full or running low on space
4. **Stale state**: Orphaned state entries for processes that crashed
5. **Permission issues**: Can't write to log directory or state file
6. **Prompts directory missing**: No prompts configured for the project

Currently, users discover these issues only when commands fail, often with cryptic errors. There's no proactive health check.

## Solution

Add a `swarm doctor` command that performs diagnostic checks and reports issues with actionable suggestions.

### Proposed API

```bash
# Run all diagnostic checks
swarm doctor

# Output as JSON (for CI/automation)
swarm doctor --format json

# Check specific area only
swarm doctor --check config
swarm doctor --check backend
swarm doctor --check disk
```

### Default output

```
swarm doctor
============

✓ Configuration
  Global config: ~/.config/swarm/config.toml (found)
  Project config: swarm/.swarm.toml (not found, using defaults)
  Backend: cursor
  Model: claude-opus-4-20250514

✓ Agent Backend
  Command: cursor --dangerously-skip-permissions ...
  Executable: /usr/local/bin/cursor (found)
  Version: 0.50.0

✓ State
  State file: ~/.swarm/state.json (exists, 2.4 KB)
  Running agents: 2
  Terminated agents: 15
  Stale entries: 0

✓ Disk Space
  Log directory: ~/.swarm/logs/
  Log files: 17 files, 45 MB total
  Available space: 52 GB

✓ Prompts
  Prompts directory: ./swarm/prompts/
  Prompts found: 5 (planner, coder, reviewer, fixer, tester)

All checks passed!
```

### Output with issues

```
swarm doctor
============

✓ Configuration
  ...

✗ Agent Backend
  Command: claude --dangerously-skip-permissions ...
  Executable: claude (NOT FOUND in PATH)
  
  Suggestion: Install Claude Code CLI or switch backend:
    swarm config set-backend cursor

✓ State
  ...
  Stale entries: 3 (processes no longer running)
  
  Suggestion: Clean up stale entries:
    swarm prune

⚠ Disk Space
  Log directory: ~/.swarm/logs/
  Log files: 847 files, 12 GB total
  Available space: 1.2 GB (LOW)
  
  Suggestion: Remove old logs to free space:
    swarm prune --logs --older-than 7d

✗ Prompts
  Prompts directory: ./swarm/prompts/ (NOT FOUND)
  
  Suggestion: Create prompts directory and add prompt files:
    mkdir -p ./swarm/prompts/
    echo "Your prompt here" > ./swarm/prompts/my-task.md

2 checks failed, 1 warning
```

## Files to create/change

- Create `cmd/doctor.go` - new command implementation

## Implementation details

### cmd/doctor.go

```go
package cmd

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/fatih/color"
    "github.com/matt/swarm-cli/internal/config"
    "github.com/matt/swarm-cli/internal/state"
    "github.com/spf13/cobra"
)

var (
    doctorFormat string
    doctorCheck  string
)

type CheckResult struct {
    Name        string   `json:"name"`
    Status      string   `json:"status"` // "pass", "warn", "fail"
    Details     []string `json:"details"`
    Suggestions []string `json:"suggestions,omitempty"`
}

type DoctorReport struct {
    Checks  []CheckResult `json:"checks"`
    Summary struct {
        Passed   int `json:"passed"`
        Warnings int `json:"warnings"`
        Failed   int `json:"failed"`
    } `json:"summary"`
}

var doctorCmd = &cobra.Command{
    Use:   "doctor",
    Short: "Check system health and diagnose issues",
    Long: `Run diagnostic checks to identify configuration issues,
missing dependencies, and other problems.

Checks performed:
- Configuration: Validates config files and settings
- Backend: Verifies agent CLI is installed and accessible
- State: Checks state file integrity and stale entries
- Disk: Reports log directory size and available space
- Prompts: Verifies prompts directory exists and has prompts`,
    Example: `  # Run all checks
  swarm doctor

  # Output as JSON
  swarm doctor --format json

  # Check only configuration
  swarm doctor --check config`,
    RunE: func(cmd *cobra.Command, args []string) error {
        report := DoctorReport{}

        checks := []func() CheckResult{
            checkConfig,
            checkBackend,
            checkState,
            checkDisk,
            checkPrompts,
        }

        // Filter to specific check if requested
        if doctorCheck != "" {
            switch doctorCheck {
            case "config":
                checks = []func() CheckResult{checkConfig}
            case "backend":
                checks = []func() CheckResult{checkBackend}
            case "state":
                checks = []func() CheckResult{checkState}
            case "disk":
                checks = []func() CheckResult{checkDisk}
            case "prompts":
                checks = []func() CheckResult{checkPrompts}
            default:
                return fmt.Errorf("unknown check: %s (valid: config, backend, state, disk, prompts)", doctorCheck)
            }
        }

        for _, check := range checks {
            result := check()
            report.Checks = append(report.Checks, result)
            switch result.Status {
            case "pass":
                report.Summary.Passed++
            case "warn":
                report.Summary.Warnings++
            case "fail":
                report.Summary.Failed++
            }
        }

        if doctorFormat == "json" {
            output, err := json.MarshalIndent(report, "", "  ")
            if err != nil {
                return err
            }
            fmt.Println(string(output))
            return nil
        }

        printReport(report)
        return nil
    },
}

func checkConfig() CheckResult {
    result := CheckResult{Name: "Configuration", Status: "pass"}

    // Check global config
    globalPath, err := config.GlobalConfigPath()
    if err != nil {
        result.Details = append(result.Details, fmt.Sprintf("Global config: error getting path: %v", err))
    } else {
        if _, err := os.Stat(globalPath); os.IsNotExist(err) {
            result.Details = append(result.Details, fmt.Sprintf("Global config: %s (not found, using defaults)", globalPath))
        } else {
            result.Details = append(result.Details, fmt.Sprintf("Global config: %s (found)", globalPath))
        }
    }

    // Check project config
    projectPath := config.ProjectConfigPath()
    if _, err := os.Stat(projectPath); os.IsNotExist(err) {
        result.Details = append(result.Details, fmt.Sprintf("Project config: %s (not found)", projectPath))
    } else {
        result.Details = append(result.Details, fmt.Sprintf("Project config: %s (found)", projectPath))
    }

    // Load and validate config
    cfg, err := config.Load()
    if err != nil {
        result.Status = "fail"
        result.Details = append(result.Details, fmt.Sprintf("Config error: %v", err))
        result.Suggestions = append(result.Suggestions, "Check config file syntax with: swarm config show")
        return result
    }

    result.Details = append(result.Details, fmt.Sprintf("Model: %s", cfg.Model))

    return result
}

func checkBackend() CheckResult {
    result := CheckResult{Name: "Agent Backend", Status: "pass"}

    cfg, err := config.Load()
    if err != nil {
        result.Status = "fail"
        result.Details = append(result.Details, "Could not load config to check backend")
        return result
    }

    // Extract the executable from the command
    parts := strings.Fields(cfg.Command)
    if len(parts) == 0 {
        result.Status = "fail"
        result.Details = append(result.Details, "No command configured")
        return result
    }

    executable := parts[0]
    result.Details = append(result.Details, fmt.Sprintf("Command: %s ...", executable))

    // Check if executable exists
    path, err := exec.LookPath(executable)
    if err != nil {
        result.Status = "fail"
        result.Details = append(result.Details, fmt.Sprintf("Executable: %s (NOT FOUND in PATH)", executable))
        result.Suggestions = append(result.Suggestions, 
            fmt.Sprintf("Install %s or check your PATH", executable),
            "Or switch backend: swarm config set-backend cursor")
        return result
    }

    result.Details = append(result.Details, fmt.Sprintf("Executable: %s (found)", path))

    // Try to get version
    cmd := exec.Command(executable, "--version")
    output, err := cmd.Output()
    if err == nil {
        version := strings.TrimSpace(string(output))
        // Take first line only
        if idx := strings.Index(version, "\n"); idx > 0 {
            version = version[:idx]
        }
        result.Details = append(result.Details, fmt.Sprintf("Version: %s", version))
    }

    return result
}

func checkState() CheckResult {
    result := CheckResult{Name: "State", Status: "pass"}

    homeDir, err := os.UserHomeDir()
    if err != nil {
        result.Status = "fail"
        result.Details = append(result.Details, fmt.Sprintf("Cannot get home directory: %v", err))
        return result
    }

    statePath := filepath.Join(homeDir, ".swarm", "state.json")
    info, err := os.Stat(statePath)
    if os.IsNotExist(err) {
        result.Details = append(result.Details, fmt.Sprintf("State file: %s (not found, will be created)", statePath))
        return result
    }
    if err != nil {
        result.Status = "fail"
        result.Details = append(result.Details, fmt.Sprintf("State file error: %v", err))
        return result
    }

    result.Details = append(result.Details, fmt.Sprintf("State file: %s (%s)", statePath, formatBytes(info.Size())))

    // Count agents
    mgr, err := state.NewManagerWithScope(GetScope(), "")
    if err != nil {
        result.Status = "warn"
        result.Details = append(result.Details, fmt.Sprintf("Could not load state: %v", err))
        return result
    }

    allAgents, _ := mgr.List(false)
    runningAgents, _ := mgr.List(true)

    result.Details = append(result.Details, fmt.Sprintf("Running agents: %d", len(runningAgents)))
    result.Details = append(result.Details, fmt.Sprintf("Total agents: %d", len(allAgents)))

    // Count stale (would need to check PIDs - simplified here)
    stale := 0
    for _, agent := range allAgents {
        if agent.Status == "running" && !isProcessRunning(agent.PID) {
            stale++
        }
    }
    if stale > 0 {
        result.Status = "warn"
        result.Details = append(result.Details, fmt.Sprintf("Stale entries: %d", stale))
        result.Suggestions = append(result.Suggestions, "Clean up stale entries: swarm prune")
    }

    return result
}

func checkDisk() CheckResult {
    result := CheckResult{Name: "Disk Space", Status: "pass"}

    homeDir, _ := os.UserHomeDir()
    logsDir := filepath.Join(homeDir, ".swarm", "logs")

    if _, err := os.Stat(logsDir); os.IsNotExist(err) {
        result.Details = append(result.Details, fmt.Sprintf("Log directory: %s (not found, will be created)", logsDir))
        return result
    }

    // Count files and total size
    var totalSize int64
    var fileCount int
    filepath.Walk(logsDir, func(path string, info os.FileInfo, err error) error {
        if err != nil || info.IsDir() {
            return nil
        }
        totalSize += info.Size()
        fileCount++
        return nil
    })

    result.Details = append(result.Details, fmt.Sprintf("Log directory: %s", logsDir))
    result.Details = append(result.Details, fmt.Sprintf("Log files: %d files, %s total", fileCount, formatBytes(totalSize)))

    // Check available space (simplified - would need syscall for actual impl)
    // For now, just warn if logs are large
    if totalSize > 10*1024*1024*1024 { // 10 GB
        result.Status = "warn"
        result.Suggestions = append(result.Suggestions, "Remove old logs: swarm prune --logs --older-than 7d")
    }

    return result
}

func checkPrompts() CheckResult {
    result := CheckResult{Name: "Prompts", Status: "pass"}

    promptsDir, err := GetPromptsDir()
    if err != nil {
        result.Status = "warn"
        result.Details = append(result.Details, fmt.Sprintf("Could not determine prompts directory: %v", err))
        return result
    }

    if _, err := os.Stat(promptsDir); os.IsNotExist(err) {
        result.Status = "fail"
        result.Details = append(result.Details, fmt.Sprintf("Prompts directory: %s (NOT FOUND)", promptsDir))
        result.Suggestions = append(result.Suggestions,
            fmt.Sprintf("Create prompts directory: mkdir -p %s", promptsDir),
            "Add a prompt file: echo 'Your task prompt' > " + filepath.Join(promptsDir, "my-task.md"))
        return result
    }

    result.Details = append(result.Details, fmt.Sprintf("Prompts directory: %s", promptsDir))

    // Count prompts
    entries, err := os.ReadDir(promptsDir)
    if err != nil {
        result.Status = "warn"
        result.Details = append(result.Details, fmt.Sprintf("Could not read prompts directory: %v", err))
        return result
    }

    var prompts []string
    for _, entry := range entries {
        if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
            name := strings.TrimSuffix(entry.Name(), ".md")
            prompts = append(prompts, name)
        }
    }

    if len(prompts) == 0 {
        result.Status = "warn"
        result.Details = append(result.Details, "No prompts found (*.md files)")
        result.Suggestions = append(result.Suggestions,
            "Add a prompt file: echo 'Your task prompt' > " + filepath.Join(promptsDir, "my-task.md"))
    } else {
        // Show first few prompt names
        display := prompts
        if len(display) > 5 {
            display = display[:5]
        }
        result.Details = append(result.Details, fmt.Sprintf("Prompts found: %d (%s)", len(prompts), strings.Join(display, ", ")))
    }

    return result
}

func printReport(report DoctorReport) {
    bold := color.New(color.Bold)
    green := color.New(color.FgGreen)
    yellow := color.New(color.FgYellow)
    red := color.New(color.FgRed)
    dim := color.New(color.Faint)

    bold.Println("swarm doctor")
    fmt.Println("============")
    fmt.Println()

    for _, check := range report.Checks {
        // Status icon
        switch check.Status {
        case "pass":
            green.Print("✓ ")
        case "warn":
            yellow.Print("⚠ ")
        case "fail":
            red.Print("✗ ")
        }

        bold.Println(check.Name)

        for _, detail := range check.Details {
            fmt.Printf("  %s\n", detail)
        }

        if len(check.Suggestions) > 0 {
            fmt.Println()
            dim.Println("  Suggestion:")
            for _, suggestion := range check.Suggestions {
                fmt.Printf("    %s\n", suggestion)
            }
        }

        fmt.Println()
    }

    // Summary
    if report.Summary.Failed > 0 {
        red.Printf("%d checks failed", report.Summary.Failed)
        if report.Summary.Warnings > 0 {
            fmt.Printf(", %d warnings", report.Summary.Warnings)
        }
        fmt.Println()
    } else if report.Summary.Warnings > 0 {
        yellow.Printf("%d warnings\n", report.Summary.Warnings)
    } else {
        green.Println("All checks passed!")
    }
}

func formatBytes(bytes int64) string {
    const unit = 1024
    if bytes < unit {
        return fmt.Sprintf("%d B", bytes)
    }
    div, exp := int64(unit), 0
    for n := bytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func init() {
    doctorCmd.Flags().StringVar(&doctorFormat, "format", "", "Output format: json or table (default)")
    doctorCmd.Flags().StringVar(&doctorCheck, "check", "", "Run specific check only (config, backend, state, disk, prompts)")
    rootCmd.AddCommand(doctorCmd)
}
```

## Edge cases

1. **First-time user**: No config, no state, no prompts - show helpful setup suggestions
2. **Missing home directory**: Graceful error explaining the issue
3. **Permission denied**: Clear error with suggestion to check permissions
4. **Corrupted state file**: Detect JSON parse errors and suggest fix (backup + delete)
5. **Backend command with spaces**: Parse command correctly even if path has spaces

## Dependencies

No new dependencies required. Uses existing:
- `os/exec` for checking executable
- `github.com/fatih/color` for colored output

## Acceptance criteria

- `swarm doctor` runs all diagnostic checks
- Each check shows pass (✓), warn (⚠), or fail (✗) status
- Failing checks include actionable suggestions
- `--format json` outputs machine-readable JSON
- `--check <name>` runs only the specified check
- Exit code is 0 if all pass, 1 if any warnings, 2 if any failures
- Works on first run (no existing config/state)
- Handles permission and disk errors gracefully

---

## Completion Notes (cd59a862)

Implemented `swarm doctor` command in `cmd/doctor.go`.

**Features implemented:**
- All 5 diagnostic checks: Configuration, Backend, State, Disk Space, Prompts
- JSON output format with `--format json`
- Single check mode with `--check <name>`
- Exit codes: 0 (pass), 1 (warnings), 2 (failures)
- Color-coded output with ✓/⚠/✗ status indicators
- Actionable suggestions for failed/warning checks

**Testing:**
- Verified all checks pass in healthy environment
- Verified JSON output is valid
- Verified --check flag filters correctly
- Verified invalid check names produce helpful error
- All existing tests pass
