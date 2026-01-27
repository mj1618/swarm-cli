package cmd

import (
	"fmt"
	"time"

	"github.com/matt/swarm-cli/internal/compose"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	composeStopFile    string
	composeStopNoWait  bool
	composeStopTimeout int
)

var composeStopCmd = &cobra.Command{
	Use:   "compose-stop [task...]",
	Short: "Pause agents started from a compose file",
	Long: `Pause agents that were started from a compose file.

Similar to docker compose stop, this command reads task definitions from a YAML
file and pauses the matching running agents after their current iteration completes.

Each agent will finish its current iteration and then wait until resumed
with the 'start' command. Use 'down' to terminate agents instead.

By default, the command waits until all matching agents have finished their current
iteration and entered the paused state. Use --no-wait to return immediately.

Agents are matched by name and working directory to ensure only agents
started from the specified compose file are affected.`,
	Example: `  # Pause all agents from ./swarm/swarm.yaml
  swarm compose-stop

  # Pause specific tasks only
  swarm compose-stop frontend backend

  # Return immediately without waiting
  swarm compose-stop --no-wait

  # Use a custom compose file
  swarm compose-stop -f custom.yaml

  # Custom timeout (default 300 seconds)
  swarm compose-stop --timeout 60`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load compose file
		cf, err := compose.Load(composeStopFile)
		if err != nil {
			return fmt.Errorf("failed to load compose file %s: %w", composeStopFile, err)
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

		// List running agents
		agents, err := mgr.List(true) // only running agents
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		// Filter for agents that match our compose file tasks
		var matchingAgents []*state.AgentState
		for _, agent := range agents {
			if agent.WorkingDir == workingDir && effectiveNames[agent.Name] {
				matchingAgents = append(matchingAgents, agent)
			}
		}

		if len(matchingAgents) == 0 {
			fmt.Println("No matching agents found")
			return nil
		}

		// Track which agents we're setting to paused (for waiting)
		waitingFor := make(map[string]bool)
		count := 0
		alreadyPaused := 0

		for _, agent := range matchingAgents {
			if agent.Paused {
				alreadyPaused++
				continue
			}

			agent.Paused = true
			if err := mgr.Update(agent); err != nil {
				fmt.Printf("Warning: failed to update agent %s: %v\n", agent.ID, err)
				continue
			}
			waitingFor[agent.ID] = true
			count++
		}

		if count > 0 {
			fmt.Printf("%d agent(s) will pause after current iteration\n", count)
		}
		if alreadyPaused > 0 {
			fmt.Printf("%d agent(s) already paused\n", alreadyPaused)
		}
		if count == 0 && alreadyPaused == 0 {
			fmt.Println("No agents to pause")
			return nil
		}

		// Wait for agents to actually enter paused state
		if !composeStopNoWait && count > 0 {
			fmt.Printf("Waiting for %d agent(s) to pause...\n", count)

			deadline := time.Now().Add(time.Duration(composeStopTimeout) * time.Second)
			for len(waitingFor) > 0 && time.Now().Before(deadline) {
				time.Sleep(500 * time.Millisecond)

				for id := range waitingFor {
					agent, err := mgr.Get(id)
					if err != nil || agent.Status != "running" {
						// Agent terminated or error reading state
						delete(waitingFor, id)
						continue
					}
					if agent.PausedAt != nil {
						// Agent has entered the pause loop
						delete(waitingFor, id)
					}
				}
			}

			if len(waitingFor) > 0 {
				fmt.Printf("Warning: %d agent(s) did not pause within timeout\n", len(waitingFor))
			} else {
				fmt.Println("All agents paused")
			}
		}

		return nil
	},
}

func init() {
	composeStopCmd.Flags().StringVarP(&composeStopFile, "file", "f", compose.DefaultPath(), "Path to compose file")
	composeStopCmd.Flags().BoolVar(&composeStopNoWait, "no-wait", false, "Return immediately without waiting for agents to pause")
	composeStopCmd.Flags().IntVar(&composeStopTimeout, "timeout", 300, "Maximum seconds to wait for agents to pause")
}
