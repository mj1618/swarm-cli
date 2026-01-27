package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var pruneForce bool
var pruneLogs bool

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove all terminated agents",
	Long: `Remove all terminated agents from the state.

This command removes all agents that are no longer running. By default,
it will prompt for confirmation. Use --force to skip the confirmation.

Use --logs to also delete the log files associated with pruned agents.`,
	Example: `  # Remove all terminated agents (with confirmation)
  swarm prune

  # Remove all terminated agents without confirmation
  swarm prune --force

  # Remove terminated agents and their log files
  swarm prune --logs

  # Remove agents and logs without confirmation
  swarm prune --logs --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// Get all agents including terminated
		agents, err := mgr.List(false) // false = include terminated
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		// Filter to only terminated agents
		var terminated []*state.AgentState
		for _, agent := range agents {
			if agent.Status == "terminated" {
				terminated = append(terminated, agent)
			}
		}

		if len(terminated) == 0 {
			fmt.Println("No terminated agents to remove.")
			return nil
		}

		// Confirm unless --force is specified
		if !pruneForce {
			if pruneLogs {
				fmt.Printf("This will remove %d terminated agent(s) and their log files. Are you sure? [y/N] ", len(terminated))
			} else {
				fmt.Printf("This will remove %d terminated agent(s). Are you sure? [y/N] ", len(terminated))
			}
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// Remove all terminated agents
		removed := 0
		logsRemoved := 0
		for _, agent := range terminated {
			if err := mgr.Remove(agent.ID); err != nil {
				fmt.Printf("Warning: failed to remove agent %s: %v\n", agent.ID, err)
				continue
			}

			// Clean up log file if requested
			if pruneLogs && agent.LogFile != "" {
				if err := os.Remove(agent.LogFile); err != nil {
					if !os.IsNotExist(err) {
						fmt.Printf("Warning: failed to remove log file %s: %v\n", agent.LogFile, err)
					}
				} else {
					logsRemoved++
				}
			}

			fmt.Println(agent.ID)
			removed++
		}

		if pruneLogs && logsRemoved > 0 {
			fmt.Printf("Removed %d agent(s) and %d log file(s).\n", removed, logsRemoved)
		} else {
			fmt.Printf("Removed %d agent(s).\n", removed)
		}
		return nil
	},
}

func init() {
	pruneCmd.Flags().BoolVarP(&pruneForce, "force", "f", false, "Do not prompt for confirmation")
	pruneCmd.Flags().BoolVar(&pruneLogs, "logs", false, "Also delete log files for pruned agents")
	rootCmd.AddCommand(pruneCmd)
}
