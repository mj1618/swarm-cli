package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	waitTimeout  time.Duration
	waitInterval time.Duration
	waitAny      bool
	waitVerbose  bool
)

var waitCmd = &cobra.Command{
	Use:   "wait [agent-id-or-name...]",
	Short: "Wait for agent(s) to terminate",
	Long: `Wait for one or more agents to terminate.

Blocks until all specified agents have terminated, or until the timeout
is reached (if specified). Useful for scripting and orchestration.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent`,
	Example: `  # Wait for a single agent
  swarm wait abc123

  # Wait for agent by name
  swarm wait my-agent

  # Wait for the most recent agent
  swarm wait @last
  swarm wait _

  # Wait for multiple agents
  swarm wait abc123 def456

  # Wait with 30 minute timeout
  swarm wait abc123 --timeout 30m

  # Wait for any agent to finish (first wins)
  swarm wait --any abc123 def456

  # Custom polling interval
  swarm wait abc123 --interval 2s`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// Resolve all agent identifiers to AgentStates and collect IDs
		agentIDs := make([]string, 0, len(args))
		agentNames := make(map[string]string) // ID -> display name (name or ID)

		for _, identifier := range args {
			agent, err := ResolveAgentIdentifier(mgr, identifier)
			if err != nil {
				return fmt.Errorf("agent not found: %s", identifier)
			}
			agentIDs = append(agentIDs, agent.ID)
			if agent.Name != "" {
				agentNames[agent.ID] = agent.Name
			} else {
				agentNames[agent.ID] = agent.ID
			}
		}

		if waitVerbose {
			if len(agentIDs) == 1 {
				fmt.Printf("Waiting for agent %s...\n", agentNames[agentIDs[0]])
			} else if waitAny {
				fmt.Printf("Waiting for any of %d agents to terminate...\n", len(agentIDs))
			} else {
				fmt.Printf("Waiting for %d agents to terminate...\n", len(agentIDs))
			}
		}

		// Set up timeout if specified
		var deadline time.Time
		if waitTimeout > 0 {
			deadline = time.Now().Add(waitTimeout)
		}

		startTimes := make(map[string]time.Time)
		for _, id := range agentIDs {
			agent, err := mgr.Get(id)
			if err == nil && agent != nil {
				startTimes[id] = agent.StartedAt
			}
		}

		// Polling loop
		for {
			allTerminated := true
			anyTerminated := false

			for _, id := range agentIDs {
				agent, err := mgr.Get(id)
				if err != nil {
					// Agent state removed = terminated
					anyTerminated = true
					if waitVerbose {
						fmt.Printf("Agent %s terminated (state removed)\n", agentNames[id])
					}
					continue
				}
				if agent.Status == "terminated" {
					anyTerminated = true
					if waitVerbose {
						runtime := time.Since(startTimes[id]).Round(time.Second)
						fmt.Printf("Agent %s terminated (was running for %s)\n", agentNames[id], runtime)
					}
				} else {
					allTerminated = false
				}
			}

			// Check exit conditions
			if waitAny && anyTerminated {
				return nil
			}
			if !waitAny && allTerminated {
				return nil
			}

			// Check timeout
			if !deadline.IsZero() && time.Now().After(deadline) {
				fmt.Fprintf(os.Stderr, "Timeout waiting for agent(s) after %s\n", waitTimeout)
				os.Exit(2)
			}

			time.Sleep(waitInterval)
		}
	},
}

func init() {
	waitCmd.Flags().DurationVar(&waitTimeout, "timeout", 0, "Maximum time to wait (e.g., 30m, 1h)")
	waitCmd.Flags().DurationVar(&waitInterval, "interval", time.Second, "Polling interval")
	waitCmd.Flags().BoolVar(&waitAny, "any", false, "Return when any agent terminates")
	waitCmd.Flags().BoolVarP(&waitVerbose, "verbose", "v", false, "Print status updates")
	rootCmd.AddCommand(waitCmd)

	// Add dynamic completion for agent identifier
	waitCmd.ValidArgsFunction = completeRunningAgentIdentifier
}
