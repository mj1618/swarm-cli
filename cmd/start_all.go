package cmd

import (
	"fmt"

	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var startAllCmd = &cobra.Command{
	Use:   "start-all",
	Short: "Resume all paused agents",
	Long: `Resume all paused agents.

Each agent will continue from the next iteration after being resumed.

The command operates on agents in the current project directory by default.
Use --global to resume all agents across all projects.`,
	Example: `  # Resume all paused agents in current project
  swarm start-all

  # Resume all paused agents globally
  swarm start-all --global`,
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

		// Use atomic method for control field to avoid race conditions
		count := 0
		notPaused := 0
		for _, agent := range agents {
			if !agent.Paused {
				notPaused++
				continue
			}

			if err := mgr.SetPaused(agent.ID, false); err != nil {
				fmt.Printf("Warning: failed to update agent %s: %v\n", agent.ID, err)
				continue
			}
			count++
		}

		if count > 0 {
			fmt.Printf("%d agent(s) resumed\n", count)
		}
		if count == 0 {
			fmt.Println("No paused agents found")
		}
		return nil
	},
}
