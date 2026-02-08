package cmd

import (
	"fmt"
	"time"

	"github.com/mj1618/swarm-cli/internal/compose"
	"github.com/mj1618/swarm-cli/internal/process"
	"github.com/mj1618/swarm-cli/internal/scope"
	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	downFile string
)

var downCmd = &cobra.Command{
	Use:   "down [task...]",
	Short: "Kill agents started from a compose file",
	Long: `Kill agents that were started from a compose file.

Similar to docker compose down, this command reads task definitions from a YAML
file and immediately kills the matching running agents using SIGKILL.

All matching agents and their descendant sub-agents are killed immediately.

Agents are matched by name and working directory to ensure only agents
started from the specified compose file are affected.`,
	Example: `  # Kill all agents from ./swarm/swarm.yaml
  swarm down

  # Kill specific tasks only
  swarm down frontend backend

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

		// Collect all agents to kill: matching agents + their descendants
		var agentsToKill []*state.AgentState
		for _, a := range matchingAgents {
			agentsToKill = append(agentsToKill, a)
			descendants, err := mgr.GetDescendants(a.ID)
			if err == nil {
				for _, d := range descendants {
					if d.Status == "running" {
						agentsToKill = append(agentsToKill, d)
					}
				}
			}
		}

		// Kill all agents immediately
		count := 0
		for _, a := range agentsToKill {
			// Set terminate mode and force kill
			if err := mgr.SetTerminateMode(a.ID, "immediate"); err != nil {
				fmt.Printf("Warning: failed to update agent %s: %v\n", a.ID, err)
				continue
			}

			// Force kill the process immediately (SIGKILL on Unix)
			if err := process.ForceKill(a.PID); err != nil {
				fmt.Printf("Warning: could not kill process %d: %v\n", a.PID, err)
			}

			// Mark as terminated in state so it's immediately reflected
			now := time.Now()
			a.Status = "terminated"
			a.ExitReason = "killed"
			a.TerminatedAt = &now
			_ = mgr.Update(a)

			count++
		}

		fmt.Printf("Killed %d agent(s)\n", count)
		return nil
	},
}

func init() {
	downCmd.Flags().StringVarP(&downFile, "file", "f", compose.DefaultPath(), "Path to compose file")
}
