package cmd

import (
	"fmt"

	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop [agent-id-or-name]",
	Short: "Pause a running agent",
	Long: `Pause a running agent after the current iteration completes.

The agent can be specified by its ID or name.

The agent will finish its current iteration and then wait until resumed
with the 'start' command. Use 'kill' to terminate a paused agent.`,
	Example: `  # Stop an agent by ID
  swarm stop abc123

  # Stop an agent by name
  swarm stop my-agent`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentIdentifier := args[0]

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		agent, err := mgr.GetByNameOrID(agentIdentifier)
		if err != nil {
			return fmt.Errorf("agent not found: %w", err)
		}

		if agent.Status != "running" {
			return fmt.Errorf("agent is not running (status: %s)", agent.Status)
		}

		if agent.Paused {
			fmt.Printf("Agent %s is already paused\n", agent.ID)
			return nil
		}

		agent.Paused = true
		if err := mgr.Update(agent); err != nil {
			return fmt.Errorf("failed to update agent state: %w", err)
		}

		fmt.Printf("Agent %s will pause after current iteration\n", agent.ID)
		if agent.Name != "" {
			fmt.Printf("Name: %s\n", agent.Name)
		}
		return nil
	},
}
