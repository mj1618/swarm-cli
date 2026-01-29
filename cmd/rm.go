package cmd

import (
	"fmt"
	"os"

	"github.com/matt/swarm-cli/internal/process"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var rmForce bool
var rmLogs bool

var rmCmd = &cobra.Command{
	Use:   "rm [task-id-or-name...]",
	Short: "Remove one or more agents",
	Long: `Remove one or more agents from the state.

By default, only terminated agents can be removed. Use --force to remove
running agents (this will also terminate them).

Use --logs to also delete the log files associated with removed agents.

The agents can be specified by their IDs, names, or special identifiers:
  - @last or _ : the most recently started agent`,
	Example: `  # Remove a terminated agent by ID
  swarm rm abc123

  # Remove multiple agents
  swarm rm abc123 def456

  # Remove by name
  swarm rm my-agent

  # Remove the most recent agent
  swarm rm @last
  swarm rm _

  # Force remove a running agent
  swarm rm abc123 --force

  # Remove agent and its log file
  swarm rm abc123 --logs

  # Force remove and delete logs
  swarm rm abc123 --force --logs`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		var errors []string
		removed := 0
		logsRemoved := 0

		for _, identifier := range args {
			agent, err := ResolveAgentIdentifier(mgr, identifier)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: not found", identifier))
				continue
			}

			// Check if agent is running
			if agent.Status == "running" {
				if !rmForce {
					errors = append(errors, fmt.Sprintf("%s: agent is running (use --force to remove)", identifier))
					continue
				}

				// Force terminate the running agent
				if err := process.Kill(agent.PID); err != nil {
					fmt.Printf("Warning: could not send signal to process %d: %v\n", agent.PID, err)
				}
			}

			// Remove the agent
			if err := mgr.Remove(agent.ID); err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to remove: %v", identifier, err))
				continue
			}

			// Clean up log file if requested
			if rmLogs && agent.LogFile != "" {
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

		// Print errors at the end
		for _, e := range errors {
			fmt.Printf("Error: %s\n", e)
		}

		if removed == 0 && len(errors) > 0 {
			return fmt.Errorf("no agents removed")
		}

		// Print summary when logs were removed
		if rmLogs && logsRemoved > 0 {
			fmt.Printf("Removed %d agent(s) and %d log file(s).\n", removed, logsRemoved)
		}

		return nil
	},
}

func init() {
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Force removal of running agents")
	rmCmd.Flags().BoolVar(&rmLogs, "logs", false, "Also delete log files for removed agents")
	rootCmd.AddCommand(rmCmd)

	// Add dynamic completion for agent identifier
	rmCmd.ValidArgsFunction = completeAgentIdentifier
}
