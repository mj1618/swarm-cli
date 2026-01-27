package cmd

import (
	"fmt"

	"github.com/matt/swarm-cli/internal/process"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var killAllGraceful bool

var killAllCmd = &cobra.Command{
	Use:   "kill-all",
	Short: "Terminate all running agents",
	Long: `Terminate all running agents immediately or gracefully.

By default, agents are terminated immediately. Use --graceful to allow
each agent's current iteration to complete before terminating.

The command operates on agents in the current project directory by default.
Use --global to terminate all agents across all projects.`,
	Example: `  # Terminate all agents immediately
  swarm kill-all

  # Graceful termination (wait for current iterations)
  swarm kill-all --graceful

  # Terminate all agents globally
  swarm kill-all --global`,
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
		for _, agent := range agents {
			if killAllGraceful {
				// Graceful termination: wait for current iteration to complete
				agent.TerminateMode = "after_iteration"
				if err := mgr.Update(agent); err != nil {
					fmt.Printf("Warning: failed to update agent %s: %v\n", agent.ID, err)
					continue
				}
			} else {
				// Immediate termination
				agent.TerminateMode = "immediate"
				if err := mgr.Update(agent); err != nil {
					fmt.Printf("Warning: failed to update agent %s: %v\n", agent.ID, err)
					continue
				}

				// Send termination signal to the process
				if err := process.Kill(agent.PID); err != nil {
					fmt.Printf("Warning: could not send signal to process %d: %v\n", agent.PID, err)
				}
			}
			count++
		}

		if killAllGraceful {
			fmt.Printf("%d agent(s) will terminate after current iteration\n", count)
		} else {
			fmt.Printf("Sent termination signal to %d agent(s)\n", count)
		}
		return nil
	},
}

func init() {
	killAllCmd.Flags().BoolVarP(&killAllGraceful, "graceful", "G", false, "Terminate after current iteration completes")
}
