package cmd

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view [agent-id-or-name]",
	Short: "View agent details and logs",
	Long: `View detailed information about a specific agent including its status and logs.

The agent can be specified by its ID or name.`,
	Example: `  # View by agent ID
  swarm view abc123

  # View by agent name
  swarm view my-agent`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentIdentifier := args[0]

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		agent, err := mgr.GetByNameOrID(agentIdentifier)
		if err != nil {
			return fmt.Errorf("agent not found: %w", err)
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
		switch agent.Status {
		case "running":
			statusColor = color.New(color.FgGreen)
		case "terminated":
			statusColor = color.New(color.FgRed)
		}
		fmt.Print("Status:        ")
		statusColor.Println(agent.Status)

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
