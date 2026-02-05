package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/mj1618/swarm-cli/internal/scope"
	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	resumeAllName   string
	resumeAllPrompt string
	resumeAllModel  string
	resumeAllDryRun bool
	resumeAllYes    bool
)

var resumeAllCmd = &cobra.Command{
	Use:   "resume-all",
	Short: "Resume all paused agents",
	Long: `Resume all paused agents in the current project.

Paused agents will continue from their next iteration.
Use --global to resume agents across all projects.

Filter options allow you to resume only specific agents:
  --name, -N      Filter by agent name (substring match, case-insensitive)
  --prompt, -p    Filter by prompt name (substring match, case-insensitive)
  --model, -m     Filter by model name (substring match, case-insensitive)

Multiple filters are combined with AND logic (all conditions must match).`,
	Example: `  # Resume all paused agents
  swarm resume-all

  # Resume only agents with matching name
  swarm resume-all --name coder

  # Resume agents using a specific model
  swarm resume-all --model opus

  # Resume agents with a specific prompt
  swarm resume-all --prompt planner

  # Combine filters
  swarm resume-all --name coder --model sonnet

  # Preview what would be resumed
  swarm resume-all --dry-run

  # Skip confirmation
  swarm resume-all -y

  # Resume all agents globally
  swarm resume-all --global`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// Get all running agents
		agents, err := mgr.List(true)
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		// Filter to only paused agents and apply filters
		var toResume []*state.AgentState
		for _, agent := range agents {
			if agent.Status != "running" || !agent.Paused {
				continue
			}

			// Apply name filter (substring, case-insensitive)
			if resumeAllName != "" && !strings.Contains(strings.ToLower(agent.Name), strings.ToLower(resumeAllName)) {
				continue
			}

			// Apply prompt filter (substring, case-insensitive)
			if resumeAllPrompt != "" && !strings.Contains(strings.ToLower(agent.Prompt), strings.ToLower(resumeAllPrompt)) {
				continue
			}

			// Apply model filter (substring, case-insensitive)
			if resumeAllModel != "" && !strings.Contains(strings.ToLower(agent.Model), strings.ToLower(resumeAllModel)) {
				continue
			}

			toResume = append(toResume, agent)
		}

		if len(toResume) == 0 {
			fmt.Println("No paused agents to resume.")
			return nil
		}

		// Dry run mode
		if resumeAllDryRun {
			fmt.Printf("Would resume %d agent(s):\n", len(toResume))
			for _, agent := range toResume {
				name := agent.Name
				if name == "" {
					name = "-"
				}
				pausedDur := ""
				if agent.PausedAt != nil {
					pausedDur = fmt.Sprintf(" (paused %s)", time.Since(*agent.PausedAt).Round(time.Second))
				}
				fmt.Printf("  %s (%s)%s\n", agent.ID, name, pausedDur)
			}
			fmt.Println("\nRun without --dry-run to resume.")
			return nil
		}

		// Confirmation (unless -y)
		if !resumeAllYes {
			scopeStr := "in this project"
			if GetScope() == scope.ScopeGlobal {
				scopeStr = "globally (all projects)"
			}

			filterDesc := ""
			if resumeAllName != "" || resumeAllPrompt != "" || resumeAllModel != "" {
				var filters []string
				if resumeAllName != "" {
					filters = append(filters, fmt.Sprintf("name=%q", resumeAllName))
				}
				if resumeAllPrompt != "" {
					filters = append(filters, fmt.Sprintf("prompt=%q", resumeAllPrompt))
				}
				if resumeAllModel != "" {
					filters = append(filters, fmt.Sprintf("model=%q", resumeAllModel))
				}
				filterDesc = fmt.Sprintf(" matching %s", strings.Join(filters, ", "))
			}

			fmt.Printf("Resume %d paused agent(s)%s %s", len(toResume), filterDesc, scopeStr)

			// List agents if small number (5 or fewer)
			if len(toResume) <= 5 {
				fmt.Println(":")
				for _, agent := range toResume {
					name := agent.ID
					if agent.Name != "" {
						name = fmt.Sprintf("%s (%s)", agent.Name, agent.ID)
					}
					pausedDur := ""
					if agent.PausedAt != nil {
						pausedDur = fmt.Sprintf(" (paused %s)", time.Since(*agent.PausedAt).Round(time.Second))
					}
					fmt.Printf("  - %s%s\n", name, pausedDur)
				}
			} else {
				fmt.Println(".")
			}

			// Check if stdin is a terminal (interactive mode)
			if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
				fmt.Println("Non-interactive mode detected. Use -y to skip confirmation.")
				fmt.Println("Aborted.")
				return nil
			}

			fmt.Print("Are you sure? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		// Resume agents (use atomic method for control field)
		fmt.Printf("Resuming %d paused agent(s)...\n", len(toResume))
		var resumed int
		for _, agent := range toResume {
			if err := mgr.SetPaused(agent.ID, false); err != nil {
				name := agent.Name
				if name == "" {
					name = "-"
				}
				fmt.Printf("  %s (%s)  failed: %v\n", agent.ID, name, err)
				continue
			}

			name := agent.Name
			if name == "" {
				name = "-"
			}
			fmt.Printf("  %s (%s)  resumed\n", agent.ID, name)
			resumed++
		}

		if resumed > 0 {
			fmt.Printf("\n%d agent(s) resumed.\n", resumed)
		}
		return nil
	},
}

func init() {
	resumeAllCmd.Flags().StringVarP(&resumeAllName, "name", "N", "", "Filter by agent name (substring match)")
	resumeAllCmd.Flags().StringVarP(&resumeAllPrompt, "prompt", "p", "", "Filter by prompt name (substring match)")
	resumeAllCmd.Flags().StringVarP(&resumeAllModel, "model", "m", "", "Filter by model name (substring match)")
	resumeAllCmd.Flags().BoolVar(&resumeAllDryRun, "dry-run", false, "Show what would be resumed without resuming")
	resumeAllCmd.Flags().BoolVarP(&resumeAllYes, "yes", "y", false, "Skip confirmation prompt")
}
