package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	pauseAllName   string
	pauseAllPrompt string
	pauseAllModel  string
	pauseAllDryRun bool
	pauseAllYes    bool
)

var pauseAllCmd = &cobra.Command{
	Use:   "pause-all",
	Short: "Pause all running agents",
	Long: `Pause all running agents in the current project.

Paused agents will stop after completing their current iteration.
Use 'swarm resume-all' to continue paused agents.

Use --global to pause agents across all projects.

Filter options allow you to pause only specific agents:
  --name, -N      Filter by agent name (substring match, case-insensitive)
  --prompt, -p    Filter by prompt name (substring match, case-insensitive)
  --model, -m     Filter by model name (substring match, case-insensitive)

Multiple filters are combined with AND logic (all conditions must match).`,
	Example: `  # Pause all running agents
  swarm pause-all

  # Pause only agents with matching name
  swarm pause-all --name coder

  # Pause agents using a specific model
  swarm pause-all --model opus

  # Pause agents with a specific prompt
  swarm pause-all --prompt planner

  # Combine filters
  swarm pause-all --name coder --model sonnet

  # Preview what would be paused
  swarm pause-all --dry-run

  # Skip confirmation
  swarm pause-all -y

  # Pause all agents globally
  swarm pause-all --global`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// Get all running agents (not terminated)
		agents, err := mgr.List(true)
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		// Filter to only running (not already paused) and apply filters
		var toPause []*state.AgentState
		for _, agent := range agents {
			if agent.Status != "running" || agent.Paused {
				continue
			}

			// Apply name filter (substring, case-insensitive)
			if pauseAllName != "" && !strings.Contains(strings.ToLower(agent.Name), strings.ToLower(pauseAllName)) {
				continue
			}

			// Apply prompt filter (substring, case-insensitive)
			if pauseAllPrompt != "" && !strings.Contains(strings.ToLower(agent.Prompt), strings.ToLower(pauseAllPrompt)) {
				continue
			}

			// Apply model filter (substring, case-insensitive)
			if pauseAllModel != "" && !strings.Contains(strings.ToLower(agent.Model), strings.ToLower(pauseAllModel)) {
				continue
			}

			toPause = append(toPause, agent)
		}

		if len(toPause) == 0 {
			fmt.Println("No running agents to pause.")
			return nil
		}

		// Dry run mode
		if pauseAllDryRun {
			fmt.Printf("Would pause %d agent(s):\n", len(toPause))
			for _, agent := range toPause {
				name := agent.Name
				if name == "" {
					name = "-"
				}
				fmt.Printf("  %s (%s)\n", agent.ID, name)
			}
			fmt.Println("\nRun without --dry-run to pause.")
			return nil
		}

		// Confirmation (unless -y)
		if !pauseAllYes {
			scopeStr := "in this project"
			if GetScope() == scope.ScopeGlobal {
				scopeStr = "globally (all projects)"
			}

			filterDesc := ""
			if pauseAllName != "" || pauseAllPrompt != "" || pauseAllModel != "" {
				var filters []string
				if pauseAllName != "" {
					filters = append(filters, fmt.Sprintf("name=%q", pauseAllName))
				}
				if pauseAllPrompt != "" {
					filters = append(filters, fmt.Sprintf("prompt=%q", pauseAllPrompt))
				}
				if pauseAllModel != "" {
					filters = append(filters, fmt.Sprintf("model=%q", pauseAllModel))
				}
				filterDesc = fmt.Sprintf(" matching %s", strings.Join(filters, ", "))
			}

			fmt.Printf("Pause %d running agent(s)%s %s", len(toPause), filterDesc, scopeStr)

			// List agents if small number (5 or fewer)
			if len(toPause) <= 5 {
				fmt.Println(":")
				for _, agent := range toPause {
					name := agent.ID
					if agent.Name != "" {
						name = fmt.Sprintf("%s (%s)", agent.Name, agent.ID)
					}
					fmt.Printf("  - %s\n", name)
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

		// Pause agents (use atomic method for control field)
		fmt.Printf("Pausing %d running agent(s)...\n", len(toPause))
		var paused int
		for _, agent := range toPause {
			if err := mgr.SetPaused(agent.ID, true); err != nil {
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
			fmt.Printf("  %s (%s)  paused\n", agent.ID, name)
			paused++
		}

		if paused > 0 {
			fmt.Printf("\n%d agent(s) paused. Use 'swarm resume-all' to continue.\n", paused)
		}
		return nil
	},
}

func init() {
	pauseAllCmd.Flags().StringVarP(&pauseAllName, "name", "N", "", "Filter by agent name (substring match)")
	pauseAllCmd.Flags().StringVarP(&pauseAllPrompt, "prompt", "p", "", "Filter by prompt name (substring match)")
	pauseAllCmd.Flags().StringVarP(&pauseAllModel, "model", "m", "", "Filter by model name (substring match)")
	pauseAllCmd.Flags().BoolVar(&pauseAllDryRun, "dry-run", false, "Show what would be paused without pausing")
	pauseAllCmd.Flags().BoolVarP(&pauseAllYes, "yes", "y", false, "Skip confirmation prompt")
}
