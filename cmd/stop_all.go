package cmd

import (
	"fmt"
	"time"

	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	stopAllNoWait  bool
	stopAllTimeout int
)

var stopAllCmd = &cobra.Command{
	Use:   "stop-all",
	Short: "Pause all running agents",
	Long: `Pause all running agents after their current iteration completes.

Each agent will finish its current iteration and then wait until resumed
with the 'start' or 'start-all' command. Use 'kill' or 'kill-all' to
terminate paused agents.

By default, the command waits until all agents have finished their current
iteration and entered the paused state. Use --no-wait to return immediately.

The command operates on agents in the current project directory by default.
Use --global to pause all agents across all projects.`,
	Example: `  # Pause all agents in current project (waits for pause)
  swarm stop-all

  # Pause all agents globally
  swarm stop-all --global

  # Return immediately without waiting
  swarm stop-all --no-wait

  # Custom timeout (default 300 seconds)
  swarm stop-all --timeout 60`,
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

		// Track which agents we're setting to paused (for waiting)
		waitingFor := make(map[string]bool)
		count := 0
		alreadyPaused := 0

		// Use atomic method for control field to avoid race conditions
		for _, agent := range agents {
			if agent.Paused {
				alreadyPaused++
				continue
			}

			if err := mgr.SetPaused(agent.ID, true); err != nil {
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
		if !stopAllNoWait && count > 0 {
			fmt.Printf("Waiting for %d agent(s) to pause...\n", count)

			deadline := time.Now().Add(time.Duration(stopAllTimeout) * time.Second)
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
	stopAllCmd.Flags().BoolVar(&stopAllNoWait, "no-wait", false, "Return immediately without waiting for agents to pause")
	stopAllCmd.Flags().IntVar(&stopAllTimeout, "timeout", 300, "Maximum seconds to wait for agents to pause")
}
