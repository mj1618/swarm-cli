package cmd

import (
	"fmt"

	"github.com/matt/swarm-cli/internal/label"
	"github.com/matt/swarm-cli/internal/process"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	killGraceful bool
	killLabels   []string
)

var killCmd = &cobra.Command{
	Use:   "kill [task-id-or-name]",
	Short: "Terminate a running agent",
	Long: `Terminate a running agent immediately or gracefully.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent

By default, the agent is terminated immediately. Use --graceful to allow
the current iteration to complete before terminating.

Use --label to kill all running agents matching the specified labels.
When using --label, the task-id-or-name argument is not required.`,
	Example: `  # Terminate immediately (by ID)
  swarm kill abc123

  # Terminate immediately (by name)
  swarm kill my-agent

  # Terminate the most recent agent
  swarm kill @last
  swarm kill _

  # Graceful termination (wait for current iteration)
  swarm kill abc123 --graceful

  # Kill all agents with a specific label
  swarm kill --label team=frontend

  # Kill all agents with multiple labels (AND logic)
  swarm kill --label env=staging --label priority=low

  # Graceful kill by label
  swarm kill --label team=backend --graceful`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// Handle label-based batch kill
		if len(killLabels) > 0 {
			labelFilters, err := label.ParseMultiple(killLabels)
			if err != nil {
				return fmt.Errorf("invalid label filter: %w", err)
			}

			// Get all running agents
			agents, err := mgr.List(true) // true = only running
			if err != nil {
				return fmt.Errorf("failed to list agents: %w", err)
			}

			// Filter by labels
			var matched []*state.AgentState
			for _, agent := range agents {
				if label.Match(agent.Labels, labelFilters) {
					matched = append(matched, agent)
				}
			}

			if len(matched) == 0 {
				fmt.Println("No running agents found matching the specified labels")
				return nil
			}

			// Kill all matching agents
			killed := 0
			for _, agent := range matched {
				if killGraceful {
					agent.TerminateMode = "after_iteration"
					if err := mgr.Update(agent); err != nil {
						fmt.Printf("Warning: failed to update agent %s: %v\n", agent.ID, err)
						continue
					}
					fmt.Printf("Agent %s will terminate after current iteration\n", agent.ID)
				} else {
					agent.TerminateMode = "immediate"
					if err := mgr.Update(agent); err != nil {
						fmt.Printf("Warning: failed to update agent %s: %v\n", agent.ID, err)
						continue
					}
					if err := process.Kill(agent.PID); err != nil {
						fmt.Printf("Warning: could not send signal to agent %s (PID %d): %v\n", agent.ID, agent.PID, err)
					}
					fmt.Printf("Sent termination signal to agent %s (PID: %d)\n", agent.ID, agent.PID)
				}
				killed++
			}

			fmt.Printf("Killed %d agent(s)\n", killed)
			return nil
		}

		// Single agent mode - require argument
		if len(args) == 0 {
			return fmt.Errorf("task-id-or-name is required (or use --label for batch operations)")
		}

		agentIdentifier := args[0]
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
	killCmd.Flags().StringArrayVarP(&killLabels, "label", "l", nil, "Kill agents matching label (can be repeated for AND logic)")

	// Add dynamic completion for agent identifier
	killCmd.ValidArgsFunction = completeRunningAgentIdentifier
}
