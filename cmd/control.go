package cmd

import (
	"fmt"
	"syscall"

	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	ctrlIterations     int
	ctrlModel          string
	ctrlTerminate      bool
	ctrlTerminateAfter bool
)

var controlCmd = &cobra.Command{
	Use:   "control [agent-id-or-name]",
	Short: "Control a running agent",
	Long: `Control a running agent by changing its configuration or terminating it.

The agent can be specified by its ID or name.

Examples:
  swarm control abc123 --terminate           # Terminate immediately (by ID)
  swarm control my-agent --terminate         # Terminate immediately (by name)
  swarm control abc123 --terminate-after     # Terminate after current iteration
  swarm control abc123 --iterations 50       # Change iteration count
  swarm control my-agent --model gpt-5.2     # Change model for next iteration`,
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
		if ctrlTerminate {
			agent.TerminateMode = "immediate"
			if err := mgr.Update(agent); err != nil {
				return fmt.Errorf("failed to update agent state: %w", err)
			}

			// Send SIGTERM to the process
			if err := syscall.Kill(agent.PID, syscall.SIGTERM); err != nil {
				fmt.Printf("Warning: could not send signal to process %d: %v\n", agent.PID, err)
			}

			fmt.Printf("Sent termination signal to agent %s (PID: %d)\n", agent.ID, agent.PID)
			return nil
		}

		if ctrlTerminateAfter {
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
			agent.Iterations = ctrlIterations
			updated = true
			fmt.Printf("Updated iterations to %d\n", ctrlIterations)
		}

		if cmd.Flags().Changed("model") {
			agent.Model = ctrlModel
			updated = true
			fmt.Printf("Updated model to %s (will apply on next iteration)\n", ctrlModel)
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
	controlCmd.Flags().IntVarP(&ctrlIterations, "iterations", "n", 0, "Set new iteration count")
	controlCmd.Flags().StringVarP(&ctrlModel, "model", "m", "", "Set model for next iteration")
	controlCmd.Flags().BoolVar(&ctrlTerminate, "terminate", false, "Terminate agent immediately")
	controlCmd.Flags().BoolVar(&ctrlTerminateAfter, "terminate-after", false, "Terminate after current iteration")
}
