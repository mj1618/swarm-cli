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
	Short: "Terminate all running and paused agents",
	Long: `Terminate all running and paused agents immediately or gracefully.

By default, agents are terminated immediately using SIGKILL. Use --graceful to allow
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

		// List all agents (including paused ones)
		allAgents, err := mgr.List(false)
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		// Filter for running agents (which includes paused agents since they have status "running")
		var agents []*state.AgentState
		for _, agent := range allAgents {
			if agent.Status == "running" {
				agents = append(agents, agent)
			}
		}

		if len(agents) == 0 {
			fmt.Println("No running or paused agents found")
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
				// Immediate termination using SIGKILL
				agent.TerminateMode = "immediate"
				agent.Status = "terminated"
				if err := mgr.Update(agent); err != nil {
					fmt.Printf("Warning: failed to update agent %s: %v\n", agent.ID, err)
					continue
				}

				// Force kill the process immediately (SIGKILL on Unix)
				if err := process.ForceKill(agent.PID); err != nil {
					fmt.Printf("Warning: could not kill process %d: %v\n", agent.PID, err)
				}
			}
			count++
		}

		if killAllGraceful {
			fmt.Printf("%d agent(s) will terminate after current iteration\n", count)
		} else {
			fmt.Printf("Killed %d agent(s)\n", count)
		}
		return nil
	},
}

func init() {
	killAllCmd.Flags().BoolVarP(&killAllGraceful, "graceful", "G", false, "Terminate after current iteration completes")
}
