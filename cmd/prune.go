package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var pruneForce bool
var pruneLogs bool
var pruneOlderThan string

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove all terminated agents",
	Long: `Remove all terminated agents from the state.

This command removes all agents that are no longer running. By default,
it will prompt for confirmation. Use --force to skip the confirmation.

Use --logs to also delete the log files associated with pruned agents.

Use --older-than to only prune agents older than a specified duration.
Supported duration formats: 30s, 5m, 2h, 1d (days), 7d.`,
	Example: `  # Remove all terminated agents (with confirmation)
  swarm prune

  # Remove all terminated agents without confirmation
  swarm prune --force

  # Remove terminated agents and their log files
  swarm prune --logs

  # Remove agents and logs without confirmation
  swarm prune --logs --force

  # Remove agents older than 7 days
  swarm prune --older-than 7d

  # Remove agents and logs older than 24 hours
  swarm prune --logs --older-than 24h`,
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

		// Parse --older-than if specified
		var cutoffTime time.Time
		if pruneOlderThan != "" {
			duration, err := pruneParseDurationWithDays(pruneOlderThan)
			if err != nil {
				return fmt.Errorf("invalid --older-than format: %w (use 30s, 5m, 2h, 7d, etc.)", err)
			}
			cutoffTime = time.Now().Add(-duration)
		}

		// Filter to only terminated agents (and optionally by age)
		var terminated []*state.AgentState
		for _, agent := range agents {
			if agent.Status != "terminated" {
				continue
			}

			// Filter by age if --older-than specified
			if !cutoffTime.IsZero() {
				// Use StartedAt as the reference time
				if agent.StartedAt.After(cutoffTime) {
					continue // Skip agents newer than cutoff
				}
			}

			terminated = append(terminated, agent)
		}

		if len(terminated) == 0 {
			if pruneOlderThan != "" {
				fmt.Printf("No terminated agents older than %s to remove.\n", pruneOlderThan)
			} else {
				fmt.Println("No terminated agents to remove.")
			}
			return nil
		}

		// Confirm unless --force is specified
		if !pruneForce {
			if pruneOlderThan != "" {
				if pruneLogs {
					fmt.Printf("This will remove %d terminated agent(s) older than %s and their log files. Are you sure? [y/N] ", len(terminated), pruneOlderThan)
				} else {
					fmt.Printf("This will remove %d terminated agent(s) older than %s. Are you sure? [y/N] ", len(terminated), pruneOlderThan)
				}
			} else if pruneLogs {
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
	pruneCmd.Flags().StringVar(&pruneOlderThan, "older-than", "", "Only prune agents older than duration (e.g., 7d, 24h, 30m)")
	rootCmd.AddCommand(pruneCmd)
}

// pruneParseDurationWithDays handles durations with day support (e.g., "1d").
// Standard time.ParseDuration doesn't support 'd' for days.
func pruneParseDurationWithDays(s string) (time.Duration, error) {
	// Handle days specially since time.ParseDuration doesn't support 'd'
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}
