package cmd

import (
	"fmt"

	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var stopAllCmd = &cobra.Command{
	Use:   "stop-all",
	Short: "Pause all running agents",
	Long: `Pause all running agents after their current iteration completes.

Each agent will finish its current iteration and then wait until resumed
with the 'start' or 'start-all' command. Use 'kill' or 'kill-all' to
terminate paused agents.

The command operates on agents in the current project directory by default.
Use --global to pause all agents across all projects.`,
	Example: `  # Pause all agents in current project
  swarm stop-all

  # Pause all agents globally
  swarm stop-all --global`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		agents, err := mgr.List(true) // only running agents
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		if len(agents) == 0 {
			fmt.Println("No running agents found")
			return nil
		}

		count := 0
		alreadyPaused := 0
		for _, agent := range agents {
			if agent.Paused {
				alreadyPaused++
				continue
			}

			agent.Paused = true
			if err := mgr.Update(agent); err != nil {
				fmt.Printf("Warning: failed to update agent %s: %v\n", agent.ID, err)
				continue
			}
			count++
		}

		if count > 0 {
			fmt.Printf("%d agent(s) will pause after current iteration\n", count)
		}
		if alreadyPaused > 0 {
			fmt.Printf("%d agent(s) already paused\n", alreadyPaused)
		}
		if count == 0 && alreadyPaused == 0 {
			fmt.Println("No agents to pause")
		}
		return nil
	},
}
