package cmd

import (
	"fmt"

	"github.com/matt/swarm-cli/internal/process"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var rmForce bool

var rmCmd = &cobra.Command{
	Use:   "rm [agent-id-or-name...]",
	Short: "Remove one or more agents",
	Long: `Remove one or more agents from the state.

By default, only terminated agents can be removed. Use --force to remove
running agents (this will also terminate them).

The agents can be specified by their IDs or names.`,
	Example: `  # Remove a terminated agent by ID
  swarm rm abc123

  # Remove multiple agents
  swarm rm abc123 def456

  # Remove by name
  swarm rm my-agent

  # Force remove a running agent
  swarm rm abc123 --force`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		var errors []string
		removed := 0

		for _, identifier := range args {
			agent, err := mgr.GetByNameOrID(identifier)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: not found", identifier))
				continue
			}

			// Check if agent is running
			if agent.Status == "running" {
				if !rmForce {
					errors = append(errors, fmt.Sprintf("%s: agent is running (use --force to remove)", identifier))
					continue
				}

				// Force terminate the running agent
				if err := process.Kill(agent.PID); err != nil {
					fmt.Printf("Warning: could not send signal to process %d: %v\n", agent.PID, err)
				}
			}

			// Remove the agent
			if err := mgr.Remove(agent.ID); err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to remove: %v", identifier, err))
				continue
			}

			fmt.Println(agent.ID)
			removed++
		}

		// Print errors at the end
		for _, e := range errors {
			fmt.Printf("Error: %s\n", e)
		}

		if removed == 0 && len(errors) > 0 {
			return fmt.Errorf("no agents removed")
		}

		return nil
	},
}

func init() {
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Force removal of running agents")
	rootCmd.AddCommand(rmCmd)
}
