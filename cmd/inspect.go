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
		if agent.TerminatedAt != nil {
			fmt.Printf("Terminated:    %s\n", agent.TerminatedAt.Format(time.RFC3339))
			duration := agent.TerminatedAt.Sub(agent.StartedAt).Round(time.Second)
			fmt.Printf("Runtime:       %s\n", duration)
		} else {
			fmt.Printf("Running for:   %s\n", time.Since(agent.StartedAt).Round(time.Second))
		}

		if agent.ExitReason != "" {
			fmt.Printf("Exit reason:   %s\n", agent.ExitReason)
		}

		if agent.Iterations == 0 {
			fmt.Printf("Iteration:     %d (unlimited)\n", agent.CurrentIter)
		} else {
			fmt.Printf("Iteration:     %d/%d\n", agent.CurrentIter, agent.Iterations)
		}

		// Show iteration breakdown if there were any iterations
		if agent.SuccessfulIters > 0 || agent.FailedIters > 0 {
			fmt.Printf("Successful:    %d\n", agent.SuccessfulIters)
			fmt.Printf("Failed:        %d\n", agent.FailedIters)
			total := agent.SuccessfulIters + agent.FailedIters
			if total > 0 {
				rate := float64(agent.SuccessfulIters) / float64(total) * 100
				fmt.Printf("Success rate:  %.0f%%\n", rate)
			}
		}

		if agent.WorkingDir != "" {
			fmt.Printf("Directory:     %s\n", agent.WorkingDir)
		}

		if agent.TerminateMode != "" {
			fmt.Printf("Terminate:     %s\n", agent.TerminateMode)
		}

		if agent.TimeoutAt != nil {
			remaining := time.Until(*agent.TimeoutAt)
			if remaining > 0 {
				fmt.Printf("Timeout:       %s remaining\n", remaining.Round(time.Second))
			} else {
				fmt.Printf("Timeout:       expired\n")
			}
		}

		if agent.TimeoutReason != "" {
			fmt.Printf("Timeout reason: %s\n", agent.TimeoutReason)
		}

		if len(agent.EnvNames) > 0 {
			fmt.Println()
			bold.Println("Environment Variables")
			fmt.Println("─────────────────────────────────")
			for _, name := range agent.EnvNames {
				fmt.Printf("  %s\n", name)
			}
		}

		if agent.LastError != "" {
			fmt.Println()
			bold.Println("Last Error")
			fmt.Println("─────────────────────────────────")
			// Truncate very long errors
			errMsg := agent.LastError
			if len(errMsg) > 500 {
				errMsg = errMsg[:500] + "..."
			}
			fmt.Println(errMsg)
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

	// Add dynamic completion for agent identifier
	inspectCmd.ValidArgsFunction = completeAgentIdentifier
}
