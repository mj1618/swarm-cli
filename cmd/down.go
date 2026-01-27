package cmd

import (
	"fmt"

	"github.com/matt/swarm-cli/internal/compose"
	"github.com/matt/swarm-cli/internal/process"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	downFile     string
	downGraceful bool
)

var downCmd = &cobra.Command{
	Use:   "down [task...]",
	Short: "Terminate agents started from a compose file",
	Long: `Terminate agents that were started from a compose file.

Similar to docker compose down, this command reads task definitions from a YAML
file and terminates the matching running agents.

By default, agents are terminated immediately. Use --graceful to allow
each agent's current iteration to complete before terminating.

Agents are matched by name and working directory to ensure only agents
started from the specified compose file are affected.`,
	Example: `  # Terminate all agents from ./swarm/swarm.yaml
  swarm down

  # Terminate specific tasks only
  swarm down frontend backend

  # Graceful termination (wait for current iteration)
  swarm down --graceful

  # Use a custom compose file
  swarm down -f custom.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load compose file
		cf, err := compose.Load(downFile)
		if err != nil {
			return fmt.Errorf("failed to load compose file %s: %w", downFile, err)
		}

		// Validate compose file
		if err := cf.Validate(); err != nil {
			return fmt.Errorf("invalid compose file: %w", err)
		}

		// Get tasks (filtered by args if provided)
		tasks, err := cf.GetTasks(args)
		if err != nil {
			return err
		}

		// Get current working directory
		workingDir, err := scope.CurrentWorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Build set of effective task names
		effectiveNames := make(map[string]bool)
		for taskName, task := range tasks {
			effectiveNames[task.EffectiveName(taskName)] = true
		}

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

		// Filter for running agents that match our compose file tasks
		var matchingAgents []*state.AgentState
		for _, agent := range allAgents {
			if agent.Status == "running" && agent.WorkingDir == workingDir && effectiveNames[agent.Name] {
				matchingAgents = append(matchingAgents, agent)
			}
		}

		if len(matchingAgents) == 0 {
			fmt.Println("No matching agents found")
			return nil
		}

		count := 0
		for _, agent := range matchingAgents {
			if downGraceful {
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

		if downGraceful {
			fmt.Printf("%d agent(s) will terminate after current iteration\n", count)
		} else {
			fmt.Printf("Terminated %d agent(s)\n", count)
		}
		return nil
	},
}

func init() {
	downCmd.Flags().StringVarP(&downFile, "file", "f", compose.DefaultPath(), "Path to compose file")
	downCmd.Flags().BoolVarP(&downGraceful, "graceful", "G", false, "Terminate after current iteration completes")
}
