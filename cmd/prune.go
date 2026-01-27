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

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove all terminated agents",
	Long: `Remove all terminated agents from the state.

This command removes all agents that are no longer running. By default,
it will prompt for confirmation. Use --force to skip the confirmation.`,
	Example: `  # Remove all terminated agents (with confirmation)
  swarm prune

  # Remove all terminated agents without confirmation
  swarm prune --force`,
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
			fmt.Printf("This will remove %d terminated agent(s). Are you sure? [y/N] ", len(terminated))
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
		for _, agent := range terminated {
			if err := mgr.Remove(agent.ID); err != nil {
				fmt.Printf("Warning: failed to remove agent %s: %v\n", agent.ID, err)
				continue
			}
			fmt.Println(agent.ID)
			removed++
		}

		fmt.Printf("Removed %d agent(s).\n", removed)
		return nil
	},
}

func init() {
	pruneCmd.Flags().BoolVarP(&pruneForce, "force", "f", false, "Do not prompt for confirmation")
	rootCmd.AddCommand(pruneCmd)
}
