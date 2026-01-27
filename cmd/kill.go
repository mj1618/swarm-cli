package cmd

import (
	"fmt"

	"github.com/matt/swarm-cli/internal/process"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var killGraceful bool

var killCmd = &cobra.Command{
	Use:   "kill [agent-id-or-name]",
	Short: "Terminate a running agent",
	Long: `Terminate a running agent immediately or gracefully.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent

By default, the agent is terminated immediately. Use --graceful to allow
the current iteration to complete before terminating.`,
	Example: `  # Terminate immediately (by ID)
  swarm kill abc123

  # Terminate immediately (by name)
  swarm kill my-agent

  # Terminate the most recent agent
  swarm kill @last
  swarm kill _

  # Graceful termination (wait for current iteration)
  swarm kill abc123 --graceful`,
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

		if killGraceful {
			// Graceful termination: wait for current iteration to complete
			agent.TerminateMode = "after_iteration"
			if err := mgr.Update(agent); err != nil {
				return fmt.Errorf("failed to update agent state: %w", err)
			}
			fmt.Printf("Agent %s will terminate after current iteration\n", agent.ID)
			return nil
		}

		// Immediate termination
		agent.TerminateMode = "immediate"
		if err := mgr.Update(agent); err != nil {
			return fmt.Errorf("failed to update agent state: %w", err)
		}

		// Send termination signal to the process
		if err := process.Kill(agent.PID); err != nil {
			fmt.Printf("Warning: could not send signal to process %d: %v\n", agent.PID, err)
		}

		fmt.Printf("Sent termination signal to agent %s (PID: %d)\n", agent.ID, agent.PID)
		return nil
	},
}

func init() {
	killCmd.Flags().BoolVarP(&killGraceful, "graceful", "G", false, "Terminate after current iteration completes")

	// Add dynamic completion for agent identifier
	killCmd.ValidArgsFunction = completeRunningAgentIdentifier
}
