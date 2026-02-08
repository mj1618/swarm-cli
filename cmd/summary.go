package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/mj1618/swarm-cli/internal/logsummary"
	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	summaryFormat  string
	summaryVerbose bool
)

var summaryCmd = &cobra.Command{
	Use:   "summary [task-id-or-name]",
	Short: "Show a summary of an agent's run",
	Long: `Display a concise summary of what an agent accomplished.

Parses agent logs to extract key information including:
- Duration and iteration timing
- Files created, modified, and deleted
- Errors and warnings encountered
- Key milestones and events

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent`,
	Example: `  # Summary of agent by ID
  swarm summary abc123

  # Summary of agent by name
  swarm summary my-agent

  # Summary of most recent agent
  swarm summary @last

  # Output as JSON
  swarm summary my-agent --format json

  # More detailed output
  swarm summary my-agent --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentIdentifier := args[0]

		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		agent, err := ResolveAgentIdentifier(mgr, agentIdentifier)
		if err != nil {
			return err
		}

		// Parse logs and generate summary
		summary, err := logsummary.Parse(agent)
		if err != nil {
			return fmt.Errorf("failed to parse logs: %w", err)
		}

		if summaryFormat == "json" {
			output, err := json.MarshalIndent(summary, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal summary: %w", err)
			}
			fmt.Println(string(output))
			return nil
		}

		printSummary(agent, summary, summaryVerbose)
		return nil
	},
}

func printSummary(agent *state.AgentState, s *logsummary.Summary, verbose bool) {
	bold := color.New(color.Bold)

	// Header
	name := agent.Name
	if name == "" {
		name = agent.ID
	}
	bold.Printf("Agent Summary: %s (%s)\n", name, agent.ID)
	fmt.Println("───────────────────────────────────────────────────────────────")
	fmt.Println()

	// Status section
	statusColor := color.New(color.FgGreen)
	statusStr := agent.Status
	switch agent.Status {
	case "running":
		if agent.Paused {
			statusStr = "paused"
			statusColor = color.New(color.FgYellow)
		}
	case "terminated":
		statusColor = color.New(color.FgRed)
	}

	fmt.Print("Status:       ")
	statusColor.Println(statusStr)
	fmt.Printf("Duration:     %s\n", s.FormatDuration())
	fmt.Printf("Iterations:   %d/%d completed\n", s.IterationsCompleted, agent.Iterations)
	fmt.Println()

	// Iteration breakdown
	if s.IterationsCompleted > 0 && s.AvgIterationSeconds > 0 {
		bold.Println("Iteration Breakdown:")
		fmt.Printf("  Avg duration:  %s\n", s.FormatAvgIteration())
		if s.FastestIteration > 0 {
			fmt.Printf("  Fastest:       %s (iteration %d)\n", s.FormatFastestIteration(), s.FastestIterationNum)
		}
		if s.SlowestIteration > 0 {
			fmt.Printf("  Slowest:       %s (iteration %d)\n", s.FormatSlowestIteration(), s.SlowestIterationNum)
		}
		fmt.Println()
	}

	// Activity summary
	bold.Println("Activity Summary:")
	fmt.Printf("  Files created:    %d\n", s.FilesCreated)
	fmt.Printf("  Files modified:   %d\n", s.FilesModified)
	fmt.Printf("  Files deleted:    %d\n", s.FilesDeleted)
	fmt.Printf("  Tool calls:       %d\n", s.ToolCalls)
	fmt.Printf("  Errors:           %d\n", len(s.Errors))
	fmt.Println()

	// Errors
	if len(s.Errors) > 0 {
		errColor := color.New(color.FgRed)
		bold.Println("Errors Encountered:")
		limit := 5
		if verbose {
			limit = len(s.Errors)
		}
		for i, e := range s.Errors {
			if i >= limit {
				fmt.Printf("  ... and %d more errors\n", len(s.Errors)-limit)
				break
			}
			errColor.Printf("  [iter %d]  ", e.Iteration)
			fmt.Println(e.Message)
		}
		fmt.Println()
	}

	// Key events (in verbose mode or if few events)
	if len(s.Events) > 0 && (verbose || len(s.Events) <= 5) {
		bold.Println("Key Events:")
		limit := 10
		if !verbose && len(s.Events) > 5 {
			limit = 5
		}
		for i, e := range s.Events {
			if i >= limit {
				fmt.Printf("  ... and %d more events\n", len(s.Events)-limit)
				break
			}
			fmt.Printf("  [iter %d]  %s: %s\n", e.Iteration, e.Type, e.Message)
		}
		fmt.Println()
	}

	// Final state
	if s.LastAction != "" {
		bold.Println("Final State:")
		fmt.Printf("  Last action: %q\n", s.LastAction)
	}
}

func init() {
	summaryCmd.Flags().StringVar(&summaryFormat, "format", "", "Output format: json or table (default)")
	summaryCmd.Flags().BoolVarP(&summaryVerbose, "verbose", "v", false, "Show more detailed output")

	// Add completion
	summaryCmd.ValidArgsFunction = completeAgentIdentifier
}
