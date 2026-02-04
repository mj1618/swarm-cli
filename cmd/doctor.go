package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/mj1618/swarm-cli/internal/config"
	"github.com/mj1618/swarm-cli/internal/prompt"
	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	doctorFormat string
	doctorCheck  string
)

// CheckResult represents the result of a single diagnostic check.
type CheckResult struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"` // "pass", "warn", "fail"
	Details     []string `json:"details"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// DoctorReport contains all check results and a summary.
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

		printDoctorReport(report)

		// Set exit code based on results
		if report.Summary.Failed > 0 {
			os.Exit(2)
		} else if report.Summary.Warnings > 0 {
			os.Exit(1)
		}

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

	result.Details = append(result.Details, fmt.Sprintf("Backend: %s", cfg.Backend))
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

	// Get the executable from command config
	executable := cfg.Command.Executable
	if executable == "" {
		result.Status = "fail"
		result.Details = append(result.Details, "No command configured")
		return result
	}

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
		// Truncate if too long
		if len(version) > 60 {
			version = version[:60] + "..."
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

	// Count stale (running status but process not actually running)
	stale := 0
	for _, agent := range allAgents {
		if agent.Status == "running" && !doctorIsProcessRunning(agent.PID) {
			stale++
		}
	}
	if stale > 0 {
		result.Status = "warn"
		result.Details = append(result.Details, fmt.Sprintf("Stale entries: %d (processes no longer running)", stale))
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

	// Warn if logs are large (> 1 GB)
	if totalSize > 1024*1024*1024 {
		result.Status = "warn"
		result.Details = append(result.Details, "Log directory is getting large")
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
			"Add a prompt file: swarm prompts new my-task")
		return result
	}

	result.Details = append(result.Details, fmt.Sprintf("Prompts directory: %s", promptsDir))

	// Count prompts
	prompts, err := prompt.ListPrompts(promptsDir)
	if err != nil {
		result.Status = "warn"
		result.Details = append(result.Details, fmt.Sprintf("Could not list prompts: %v", err))
		return result
	}

	if len(prompts) == 0 {
		result.Status = "warn"
		result.Details = append(result.Details, "No prompts found (*.md files)")
		result.Suggestions = append(result.Suggestions,
			"Add a prompt file: swarm prompts new my-task")
	} else {
		// Show first few prompt names
		display := prompts
		if len(display) > 5 {
			display = display[:5]
		}
		promptList := strings.Join(display, ", ")
		if len(prompts) > 5 {
			promptList += fmt.Sprintf(" ... and %d more", len(prompts)-5)
		}
		result.Details = append(result.Details, fmt.Sprintf("Prompts found: %d (%s)", len(prompts), promptList))
	}

	return result
}

func printDoctorReport(report DoctorReport) {
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

// doctorIsProcessRunning checks if a process with the given PID is still running.
func doctorIsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, sending signal 0 checks if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func init() {
	doctorCmd.Flags().StringVar(&doctorFormat, "format", "", "Output format: json or table (default)")
	doctorCmd.Flags().StringVar(&doctorCheck, "check", "", "Run specific check only (config, backend, state, disk, prompts)")
	rootCmd.AddCommand(doctorCmd)
}
