package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var inspectFormat string

var inspectCmd = &cobra.Command{
	Use:     "inspect [process-id-or-name]",
	Aliases: []string{"view"},
	Short:   "Display detailed information about an agent",
	Long: `Display detailed information about a specific agent including its status, configuration, and logs.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent`,
	Example: `  # Inspect by process ID
  swarm inspect abc123

  # Inspect by agent name
  swarm inspect my-agent

  # Inspect the most recent agent
  swarm inspect @last
  swarm inspect _

  # Output as JSON
  swarm inspect abc123 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		processIdentifier := args[0]

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		agent, err := ResolveAgentIdentifier(mgr, processIdentifier)
		if err != nil {
			return fmt.Errorf("agent not found: %w", err)
		}

		// JSON format output
		if inspectFormat == "json" {
			output, err := json.MarshalIndent(agent, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal agent to JSON: %w", err)
			}
			fmt.Println(string(output))
			return nil
		}

		// Print agent details
		bold := color.New(color.Bold)

		bold.Println("Agent Details")
		fmt.Println("─────────────────────────────────")
		fmt.Printf("ID:            %s\n", agent.ID)
		if agent.Name != "" {
			fmt.Printf("Name:          %s\n", agent.Name)
		}
		fmt.Printf("PID:           %d\n", agent.PID)
		fmt.Printf("Prompt:        %s\n", agent.Prompt)
		fmt.Printf("Model:         %s\n", agent.Model)

		statusColor := color.New(color.FgWhite)
		statusStr := agent.Status
		switch agent.Status {
		case "running":
			if agent.Paused {
				if agent.PausedAt != nil {
					statusStr = "paused"
					statusColor = color.New(color.FgYellow)
				} else {
					statusStr = "pausing"
					statusColor = color.New(color.FgYellow)
				}
			} else {
				statusColor = color.New(color.FgGreen)
			}
		case "terminated":
			statusColor = color.New(color.FgRed)
		}
		fmt.Print("Status:        ")
		statusColor.Println(statusStr)

		fmt.Printf("Started:       %s\n", agent.StartedAt.Format(time.RFC3339))
		fmt.Printf("Running for:   %s\n", time.Since(agent.StartedAt).Round(time.Second))
		fmt.Printf("Iteration:     %d/%d\n", agent.CurrentIter, agent.Iterations)

		if agent.WorkingDir != "" {
			fmt.Printf("Directory:     %s\n", agent.WorkingDir)
		}

		if agent.TerminateMode != "" {
			fmt.Printf("Terminate:     %s\n", agent.TerminateMode)
		}

		if agent.LogFile != "" {
			fmt.Println()
			bold.Println("Log File")
			fmt.Println("─────────────────────────────────")
			fmt.Println(agent.LogFile)
		}

		return nil
	},
}

func init() {
	inspectCmd.Flags().StringVar(&inspectFormat, "format", "", "Output format: json or table (default)")
}
