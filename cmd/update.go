package cmd

import (
	"fmt"

	"github.com/matt/swarm-cli/internal/label"
	"github.com/matt/swarm-cli/internal/process"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	updateIterations     int
	updateModel          string
	updateName           string
	updateTerminate      bool
	updateTerminateAfter bool
	updateFilterLabels   []string
	updateSetLabels      []string
)

var updateCmd = &cobra.Command{
	Use:     "update [process-id-or-name]",
	Aliases: []string{"control"},
	Short:   "Update configuration of a running agent",
	Long: `Update the configuration of a running agent or terminate it.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent

Use --filter-label to update all agents matching the specified labels.
When using --filter-label, the process-id-or-name argument is not required.

Use --set-label to add or update labels on an agent.`,
	Example: `  # Terminate immediately (by ID)
  swarm update abc123 --terminate

  # Terminate immediately (by name)
  swarm update my-agent --terminate

  # Update the most recent agent
  swarm update @last --iterations 50
  swarm update _ -m claude-sonnet-4-20250514

  # Terminate after current iteration
  swarm update abc123 --terminate-after

  # Change iteration count
  swarm update abc123 --iterations 50

  # Change model for next iteration
  swarm update my-agent --model claude-sonnet-4-20250514

  # Rename an agent
  swarm update abc123 --name new-name
  swarm update my-agent -N better-name

  # Add or update labels on an agent
  swarm update abc123 --set-label team=frontend
  swarm update abc123 --set-label priority=high --set-label env=staging

  # Update iterations for all agents with a specific label
  swarm update --filter-label team=frontend --iterations 50

  # Update model for all agents with multiple labels
  swarm update --filter-label env=staging --filter-label priority=high -m claude-sonnet-4-20250514`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// Parse set-labels
		var labelsToSet map[string]string
		if len(updateSetLabels) > 0 {
			labelsToSet, err = label.ParseMultiple(updateSetLabels)
			if err != nil {
				return fmt.Errorf("invalid set-label: %w", err)
			}
		}

		// Handle label-based batch update
		if len(updateFilterLabels) > 0 {
			labelFilters, err := label.ParseMultiple(updateFilterLabels)
			if err != nil {
				return fmt.Errorf("invalid filter-label: %w", err)
			}

			// For batch operations, we need at least one actual update
			if !cmd.Flags().Changed("iterations") && !cmd.Flags().Changed("model") && len(labelsToSet) == 0 {
				return fmt.Errorf("batch update requires at least one change (--iterations, --model, or --set-label)")
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

			// Update all matching agents
			updated := 0
			for _, agent := range matched {
				if cmd.Flags().Changed("iterations") {
					agent.Iterations = updateIterations
				}
				if cmd.Flags().Changed("model") {
					agent.Model = updateModel
				}
				if len(labelsToSet) > 0 {
					agent.Labels = label.Merge(agent.Labels, labelsToSet)
				}

				if err := mgr.Update(agent); err != nil {
					fmt.Printf("Warning: failed to update agent %s: %v\n", agent.ID, err)
					continue
				}
				fmt.Printf("Updated agent %s\n", agent.ID)
				updated++
			}

			fmt.Printf("Updated %d agent(s)\n", updated)
			return nil
		}

		// Single agent mode - require argument
		if len(args) == 0 {
			return fmt.Errorf("process-id-or-name is required (or use --filter-label for batch operations)")
		}

		agentIdentifier := args[0]
		agent, err := ResolveAgentIdentifier(mgr, agentIdentifier)
		if err != nil {
			return fmt.Errorf("agent not found: %w", err)
		}

		// Handle name update first (works for both running and terminated agents)
		nameUpdated := false
		if cmd.Flags().Changed("name") {
			if updateName == "" {
				return fmt.Errorf("name cannot be empty")
			}
			if updateName != agent.Name {
				// Check for name conflicts with other running agents
				allAgents, err := mgr.List(true) // true = only running
				if err != nil {
					return fmt.Errorf("failed to check name availability: %w", err)
				}
				for _, other := range allAgents {
					if other.ID != agent.ID && other.Name == updateName {
						return fmt.Errorf("name '%s' is already in use by agent %s", updateName, other.ID)
					}
				}
				oldName := agent.Name
				agent.Name = updateName
				nameUpdated = true
				fmt.Printf("Renamed agent from '%s' to '%s'\n", oldName, updateName)
			}
			// If same name, skip silently (no error, no message)
		}

		// Handle label updates (works for both running and terminated agents)
		labelsUpdated := false
		if len(labelsToSet) > 0 {
			agent.Labels = label.Merge(agent.Labels, labelsToSet)
			labelsUpdated = true
			fmt.Printf("Updated labels: %s\n", label.Format(agent.Labels))
		}

		// For operations other than rename and labels, agent must be running
		requiresRunning := updateTerminate || updateTerminateAfter ||
			cmd.Flags().Changed("iterations") || cmd.Flags().Changed("model")
		if requiresRunning && agent.Status != "running" {
			if nameUpdated || labelsUpdated {
				// Save changes even if we can't do other operations
				if err := mgr.Update(agent); err != nil {
					return fmt.Errorf("failed to update agent state: %w", err)
				}
			}
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
		updated := nameUpdated || labelsUpdated

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
	updateCmd.Flags().StringVarP(&updateName, "name", "N", "", "Set new name for the agent")
	updateCmd.Flags().BoolVar(&updateTerminate, "terminate", false, "Terminate agent immediately")
	updateCmd.Flags().BoolVar(&updateTerminateAfter, "terminate-after", false, "Terminate after current iteration")
	updateCmd.Flags().StringArrayVar(&updateFilterLabels, "filter-label", nil, "Filter agents by label for batch operations (can be repeated)")
	updateCmd.Flags().StringArrayVarP(&updateSetLabels, "set-label", "l", nil, "Set or update label on agent (key=value format, can be repeated)")

	// Add dynamic completion for agent identifier and model flag
	updateCmd.ValidArgsFunction = completeAgentIdentifier
	updateCmd.RegisterFlagCompletionFunc("model", completeModelName)
}
