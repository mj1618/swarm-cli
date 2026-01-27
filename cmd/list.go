package cmd

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List running agents",
	Long: `List running agents with their status and configuration.

By default, only shows agents started in the current directory.
Use --global to show agents from all directories.`,
	Example: `  # List agents in current project
  swarm list

  # List all agents across all projects
  swarm list -g`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		agents, err := mgr.List()
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		if len(agents) == 0 {
			if GetScope() == scope.ScopeProject {
				fmt.Println("No agents found in this project. Use --global to list all agents.")
			} else {
				fmt.Println("No running agents found.")
			}
			return nil
		}

		// Column widths
		const (
			colID        = 10
			colName      = 15
			colPrompt    = 20
			colModel     = 18
			colStatus    = 12
			colIteration = 10
			colDir       = 30
		)

		// Header - include DIRECTORY column in global mode
		header := color.New(color.Bold)
		if GetScope() == scope.ScopeGlobal {
			header.Printf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %s\n",
				colID, "ID", colName, "NAME", colPrompt, "PROMPT", colModel, "MODEL", colStatus, "STATUS", colIteration, "ITERATION", colDir, "DIRECTORY", "STARTED")
		} else {
			header.Printf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %s\n",
				colID, "ID", colName, "NAME", colPrompt, "PROMPT", colModel, "MODEL", colStatus, "STATUS", colIteration, "ITERATION", "STARTED")
		}

		for _, a := range agents {
			statusColor := color.New(color.FgWhite)
			switch a.Status {
			case "running":
				statusColor = color.New(color.FgGreen)
			case "terminated":
				statusColor = color.New(color.FgRed)
			}

			duration := time.Since(a.StartedAt).Round(time.Second)
			iterStr := fmt.Sprintf("%d/%d", a.CurrentIter, a.Iterations)

			// Truncate prompt if too long
			prompt := a.Prompt
			if len(prompt) > colPrompt {
				prompt = prompt[:colPrompt-3] + "..."
			}

			// Display name or "-" if not set
			name := a.Name
			if name == "" {
				name = "-"
			}
			if len(name) > colName {
				name = name[:colName-3] + "..."
			}

			// Print fixed-width columns, with status colored separately
			fmt.Printf("%-*s  %-*s  %-*s  %-*s  ", colID, a.ID, colName, name, colPrompt, prompt, colModel, a.Model)
			statusColor.Printf("%-*s", colStatus, a.Status)
			if GetScope() == scope.ScopeGlobal {
				dir := a.WorkingDir
				if len(dir) > colDir {
					dir = "..." + dir[len(dir)-colDir+3:]
				}
				fmt.Printf("  %-*s  %-*s  %s ago\n", colIteration, iterStr, colDir, dir, duration)
			} else {
				fmt.Printf("  %-*s  %s ago\n", colIteration, iterStr, duration)
			}
		}

		return nil
	},
}
