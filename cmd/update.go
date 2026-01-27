package cmd

import (
	"fmt"

	"github.com/matt/swarm-cli/internal/process"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	updateIterations     int
	updateModel          string
	updateTerminate      bool
	updateTerminateAfter bool
)

var updateCmd = &cobra.Command{
	Use:     "update [agent-id-or-name]",
	Aliases: []string{"control"},
	Short:   "Update configuration of a running agent",
	Long: `Update the configuration of a running agent or terminate it.

The agent can be specified by its ID or name.`,
	Example: `  # Terminate immediately (by ID)
  swarm update abc123 --terminate

  # Terminate immediately (by name)
  swarm update my-agent --terminate

  # Terminate after current iteration
  swarm update abc123 --terminate-after

  # Change iteration count
  swarm update abc123 --iterations 50

  # Change model for next iteration
  swarm update my-agent --model claude-sonnet-4-20250514`,
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

		// Handle termination
		if updateTerminate {
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
		}

		if updateTerminateAfter {
			agent.TerminateMode = "after_iteration"
			if err := mgr.Update(agent); err != nil {
				return fmt.Errorf("failed to update agent state: %w", err)
			}
			fmt.Printf("Agent %s will terminate after current iteration\n", agent.ID)
			return nil
		}

		// Handle configuration changes
		updated := false

		if cmd.Flags().Changed("iterations") {
			agent.Iterations = updateIterations
			updated = true
			fmt.Printf("Updated iterations to %d\n", updateIterations)
		}

		if cmd.Flags().Changed("model") {
			agent.Model = updateModel
			updated = true
			fmt.Printf("Updated model to %s (will apply on next iteration)\n", updateModel)
		}

		if updated {
			if err := mgr.Update(agent); err != nil {
				return fmt.Errorf("failed to update agent state: %w", err)
			}
		} else {
			fmt.Println("No changes specified. Use --help to see available options.")
		}

		return nil
	},
}

func init() {
	updateCmd.Flags().IntVarP(&updateIterations, "iterations", "n", 0, "Set new iteration count")
	updateCmd.Flags().StringVarP(&updateModel, "model", "m", "", "Set model for next iteration")
	updateCmd.Flags().BoolVar(&updateTerminate, "terminate", false, "Terminate agent immediately")
	updateCmd.Flags().BoolVar(&updateTerminateAfter, "terminate-after", false, "Terminate after current iteration")
}
