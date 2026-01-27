package cmd

import (
	"fmt"
	"time"

	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	stopNoWait  bool
	stopTimeout int
)

var stopCmd = &cobra.Command{
	Use:   "stop [agent-id-or-name]",
	Short: "Pause a running agent",
	Long: `Pause a running agent after the current iteration completes.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent

The agent will finish its current iteration and then wait until resumed
with the 'start' command. Use 'kill' to terminate a paused agent.

By default, the command waits until the agent has finished its current
iteration and entered the paused state. Use --no-wait to return immediately.`,
	Example: `  # Stop an agent by ID (waits for pause)
  swarm stop abc123

  # Stop an agent by name
  swarm stop my-agent

  # Stop the most recent agent
  swarm stop @last
  swarm stop _

  # Return immediately without waiting
  swarm stop my-agent --no-wait

  # Custom timeout (default 300 seconds)
  swarm stop my-agent --timeout 60`,
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

		if agent.Paused {
			fmt.Printf("Agent %s is already paused\n", agent.ID)
			return nil
		}

		agentID := agent.ID
		agent.Paused = true
		if err := mgr.Update(agent); err != nil {
			return fmt.Errorf("failed to update agent state: %w", err)
		}

		fmt.Printf("Agent %s will pause after current iteration\n", agentID)
		if agent.Name != "" {
			fmt.Printf("Name: %s\n", agent.Name)
		}

		// Wait for agent to actually enter paused state
		if !stopNoWait {
			fmt.Println("Waiting for agent to pause...")

			deadline := time.Now().Add(time.Duration(stopTimeout) * time.Second)
			paused := false

			for time.Now().Before(deadline) {
				time.Sleep(500 * time.Millisecond)

				agent, err := mgr.Get(agentID)
				if err != nil || agent.Status != "running" {
					// Agent terminated or error reading state
					fmt.Println("Agent terminated")
					return nil
				}
				if agent.PausedAt != nil {
					// Agent has entered the pause loop
					paused = true
					break
				}
			}

			if paused {
				fmt.Println("Agent paused")
			} else {
				fmt.Println("Warning: agent did not pause within timeout")
			}
		}

		return nil
	},
}

func init() {
	stopCmd.Flags().BoolVar(&stopNoWait, "no-wait", false, "Return immediately without waiting for agent to pause")
	stopCmd.Flags().IntVar(&stopTimeout, "timeout", 300, "Maximum seconds to wait for agent to pause")

	// Add dynamic completion for agent identifier
	stopCmd.ValidArgsFunction = completeRunningAgentIdentifier
}
