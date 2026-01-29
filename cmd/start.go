package cmd

import (
	"fmt"

	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

func init() {
	// Add dynamic completion for agent identifier
	startCmd.ValidArgsFunction = completeRunningAgentIdentifier
}

var startCmd = &cobra.Command{
	Use:   "start [task-id-or-name]",
	Short: "Resume a paused agent",
	Long: `Resume a paused agent.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent

The agent will continue from the next iteration after being resumed.`,
	Example: `  # Resume an agent by ID
  swarm start abc123

  # Resume an agent by name
  swarm start my-agent

  # Resume the most recent agent
  swarm start @last
  swarm start _`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentIdentifier := args[0]

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		agent, err := ResolveAgentIdentifier(mgr, agentIdentifier)
		if err != nil {
			return fmt.Errorf("agent not found: %w", err)
		}

		if agent.Status != "running" {
			return fmt.Errorf("agent is not running (status: %s)", agent.Status)
		}

		if !agent.Paused {
			fmt.Printf("Agent %s is not paused\n", agent.ID)
			return nil
		}

		agent.Paused = false
		if err := mgr.Update(agent); err != nil {
			return fmt.Errorf("failed to update agent state: %w", err)
		}

		fmt.Printf("Agent %s resumed\n", agent.ID)
		if agent.Name != "" {
			fmt.Printf("Name: %s\n", agent.Name)
		}
		return nil
	},
}
