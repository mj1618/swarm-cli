package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	diffUncommitted bool
	diffCommits     bool
	diffStat        bool
	diffOutput      string
)

var diffCmd = &cobra.Command{
	Use:   "diff [agent-id-or-name] [-- path...]",
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

		// Check for -- separator in args after the identifier
		for i := 1; i < len(args); i++ {
			if args[i] == "--" && i+1 < len(args) {
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

		// Get commits made during the agent's run
		var commits []commitInfo
		if !diffUncommitted {
			commits, err = getCommitsSince(agent.WorkingDir, agent.StartedAt)
			if err != nil {
				// Non-fatal, just warn
				fmt.Fprintf(os.Stderr, "Warning: could not get commits: %v\n", err)
			}
		}

		// Print header (unless outputting to file)
		if diffOutput == "" {
			printDiffHeader(agent)
		}

		// Show commits summary
		if len(commits) > 0 && diffOutput == "" && !diffStat {
			bold := color.New(color.Bold)
			bold.Println("\nCommits during run:")
			for _, c := range commits {
				fmt.Printf("  %s  %s\n", c.ShortHash, truncateDiffString(c.Subject, 60))
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

	if len(pathFilters) > 0 {
		args = append(args, "--")
		args = append(args, pathFilters...)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = agent.WorkingDir
	cmd.Stdout = output
	cmd.Stderr = os.Stderr

	if output == os.Stdout {
		bold.Printf("\n %d commits", len(commits))

		// Get summary line
		summaryArgs := []string{"diff", "--shortstat"}
		if len(commits) > 0 && !diffUncommitted {
			summaryArgs = append(summaryArgs, commits[len(commits)-1].Hash+"^")
		}
		if len(pathFilters) > 0 {
			summaryArgs = append(summaryArgs, "--")
			summaryArgs = append(summaryArgs, pathFilters...)
		}
		summaryCmd := exec.Command("git", summaryArgs...)
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
	// Use --color=always only for terminal output
	args := []string{"diff"}
	if output == os.Stdout {
		args = append(args, "--color=always")
	}

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

	if len(pathFilters) > 0 {
		args = append(args, "--")
		args = append(args, pathFilters...)
	}

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

func truncateDiffString(s string, max int) string {
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

	// Add dynamic completion for agent identifier
	diffCmd.ValidArgsFunction = completeAgentIdentifier
}
