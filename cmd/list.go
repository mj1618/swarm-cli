package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/matt/swarm-cli/internal/label"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var listAll bool
var listQuiet bool
var listFormat string
var listName string
var listPrompt string
var listModel string
var listStatus string
var listCount bool
var listLast int
var listLatest bool
var listLabels []string
var listShowLabels bool

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ps", "ls"},
	Short:   "List running agents",
	Long: `List running agents with their status and configuration.

By default, only shows running agents started in the current directory.
Use --all to include terminated agents.
Use --global to show agents from all directories.

Filter options:
  --name, -N      Filter by agent name (substring match, case-insensitive)
  --prompt, -p    Filter by prompt name (substring match, case-insensitive)
  --model, -m     Filter by model name (substring match, case-insensitive)
  --status        Filter by status (running, pausing, paused, or terminated)
  --label, -L     Filter by label (key=value for exact match, key for existence check)

Output options:
  --count         Output only the count of matching agents
  --last, -n      Show only the N most recently started agents
  --latest, -l    Show only the most recently started agent (same as --last 1)
  --show-labels   Show labels column in table output

Multiple filters are combined with AND logic (all conditions must match).`,
	Example: `  # List running agents in current project
  swarm list

  # List all agents (including terminated) in current project
  swarm list -a

  # List all agents across all projects
  swarm list -g -a

  # Output only agent IDs (useful for scripting)
  swarm list -q

  # Get all agent IDs including terminated
  swarm list -aq

  # Output as JSON
  swarm list --format json

  # Count running agents
  swarm list --count

  # Count all agents including terminated
  swarm list -a --count

  # Show 5 most recently started agents
  swarm list --last 5
  swarm list -n 5

  # Show the most recent agent
  swarm list --latest
  swarm list -l

  # Get ID of most recent agent (useful for scripting)
  swarm list -lq

  # Filter by name
  swarm list --name coder
  swarm list -N frontend

  # Filter by prompt name
  swarm list --prompt coder
  swarm list -p planner

  # Filter by model
  swarm list --model sonnet
  swarm list -m opus

  # Filter by status
  swarm list --status paused
  swarm list --status terminated -a

  # Combine filters
  swarm list --name coder --status running
  swarm list --prompt coder --model sonnet
  swarm list -a --status terminated --prompt planner
  swarm list --name coder --last 3

  # Filter by label (exact match)
  swarm list --label team=frontend

  # Filter by label existence (has the label, any value)
  swarm list --label team

  # Multiple label filters (AND logic)
  swarm list --label team=frontend --label priority=high

  # Show labels column
  swarm list --show-labels

  # Combine label filter with other filters
  swarm list --label team=frontend --status running --last 5`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle --latest as alias for --last 1
		if listLatest {
			listLast = 1
		}

		// Validate flags
		if listCount && listQuiet {
			return fmt.Errorf("--count and --quiet cannot be used together")
		}
		if listLast < 0 {
			return fmt.Errorf("--last must be a positive number")
		}

		// Validate status filter if provided
		if listStatus != "" {
			validStatuses := []string{"running", "pausing", "paused", "terminated"}
			isValid := false
			for _, s := range validStatuses {
				if strings.ToLower(listStatus) == s {
					isValid = true
					break
				}
			}
			if !isValid {
				return fmt.Errorf("invalid status filter %q: must be one of 'running', 'pausing', 'paused', or 'terminated'", listStatus)
			}
		}

		// Parse label filters
		labelFilters, err := label.ParseMultiple(listLabels)
		if err != nil {
			return fmt.Errorf("invalid label filter: %w", err)
		}

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// By default show only running agents, use --all to show all
		onlyRunning := !listAll
		agents, err := mgr.List(onlyRunning)
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		// Apply filters
		agents = filterAgents(agents, listName, listPrompt, listModel, listStatus, labelFilters)

		// Apply --last limit (agents are sorted oldest-first, so we want last N)
		if listLast > 0 && len(agents) > listLast {
			agents = agents[len(agents)-listLast:]
		}

		// Reverse to show newest first when using --last or --latest
		if listLast > 0 {
			for i, j := 0, len(agents)-1; i < j; i, j = i+1, j-1 {
				agents[i], agents[j] = agents[j], agents[i]
			}
		}

		// Count mode - just output the number
		if listCount {
			if listFormat == "json" {
				fmt.Printf("{\"count\": %d}\n", len(agents))
			} else {
				fmt.Println(len(agents))
			}
			return nil
		}

		// Check for helpful hints when no agents match
		if len(agents) == 0 && (listName != "" || listPrompt != "" || listModel != "" || listStatus != "" || len(listLabels) > 0) {
			// Check if filtering for terminated without -a flag
			if strings.ToLower(listStatus) == "terminated" && !listAll {
				if !listQuiet {
					fmt.Println("No agents found matching filters. Use -a to include terminated agents.")
				}
				return nil
			}
		}

		if len(agents) == 0 {
			// In quiet mode, output nothing for empty list
			if listQuiet {
				return nil
			}
			if GetScope() == scope.ScopeProject {
				if onlyRunning {
					fmt.Println("No running agents found in this project. Use --all to show terminated agents, or --global to list all projects.")
				} else {
					fmt.Println("No agents found in this project. Use --global to list all agents.")
				}
			} else {
				if onlyRunning {
					fmt.Println("No running agents found. Use --all to show terminated agents.")
				} else {
					fmt.Println("No agents found.")
				}
			}
			return nil
		}

		// Quiet mode: output only IDs, one per line
		if listQuiet {
			for _, a := range agents {
				fmt.Println(a.ID)
			}
			return nil
		}

		// JSON format output
		if listFormat == "json" {
			output, err := json.MarshalIndent(agents, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal agents to JSON: %w", err)
			}
			fmt.Println(string(output))
			return nil
		}

		// Column widths
		const (
			colID        = 10
			colName      = 15
			colParent    = 10
			colPrompt    = 20
			colModel     = 18
			colStatus    = 12
			colIteration = 10
			colDir       = 30
			colLabels    = 30
		)

		// Header - include DIRECTORY column in global mode, LABELS column if --show-labels
		header := color.New(color.Bold)
		if GetScope() == scope.ScopeGlobal {
			if listShowLabels {
				header.Printf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %s\n",
					colID, "ID", colName, "NAME", colParent, "PARENT", colLabels, "LABELS", colPrompt, "PROMPT", colModel, "MODEL", colStatus, "STATUS", colIteration, "ITERATION", colDir, "DIRECTORY", "STARTED")
			} else {
				header.Printf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %s\n",
					colID, "ID", colName, "NAME", colParent, "PARENT", colPrompt, "PROMPT", colModel, "MODEL", colStatus, "STATUS", colIteration, "ITERATION", colDir, "DIRECTORY", "STARTED")
			}
		} else {
			if listShowLabels {
				header.Printf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %s\n",
					colID, "ID", colName, "NAME", colParent, "PARENT", colLabels, "LABELS", colPrompt, "PROMPT", colModel, "MODEL", colStatus, "STATUS", colIteration, "ITERATION", "STARTED")
			} else {
				header.Printf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %-*s  %s\n",
					colID, "ID", colName, "NAME", colParent, "PARENT", colPrompt, "PROMPT", colModel, "MODEL", colStatus, "STATUS", colIteration, "ITERATION", "STARTED")
			}
		}

		for _, a := range agents {
			statusColor := color.New(color.FgWhite)
			statusStr := a.Status
		switch a.Status {
		case "running":
			if a.Paused {
				if a.PausedAt != nil {
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

			duration := time.Since(a.StartedAt).Round(time.Second)
			var iterStr string
			if a.Iterations == 0 {
				iterStr = fmt.Sprintf("%d/∞", a.CurrentIter)
			} else {
				iterStr = fmt.Sprintf("%d/%d", a.CurrentIter, a.Iterations)
			}

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

			// Format labels for display
			labelsStr := label.Format(a.Labels)
			if len(labelsStr) > colLabels {
				labelsStr = labelsStr[:colLabels-3] + "..."
			}

			// Format parent for display
			parent := a.ParentID
			if parent == "" {
				parent = "-"
			}
			if len(parent) > colParent {
				parent = parent[:colParent-3] + "..."
			}

			// Print fixed-width columns, with status colored separately
			if listShowLabels {
				fmt.Printf("%-*s  %-*s  %-*s  %-*s  %-*s  %-*s  ", colID, a.ID, colName, name, colParent, parent, colLabels, labelsStr, colPrompt, prompt, colModel, a.Model)
			} else {
				fmt.Printf("%-*s  %-*s  %-*s  %-*s  %-*s  ", colID, a.ID, colName, name, colParent, parent, colPrompt, prompt, colModel, a.Model)
			}
			statusColor.Printf("%-*s", colStatus, statusStr)
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

// filterAgents applies name, prompt, model, status, and label filters to the agent list.
// All non-empty filters must match (AND logic).
func filterAgents(agents []*state.AgentState, nameFilter, promptFilter, modelFilter, statusFilter string, labelFilters map[string]string) []*state.AgentState {
	if nameFilter == "" && promptFilter == "" && modelFilter == "" && statusFilter == "" && len(labelFilters) == 0 {
		return agents
	}

	nameFilter = strings.ToLower(nameFilter)
	promptFilter = strings.ToLower(promptFilter)
	modelFilter = strings.ToLower(modelFilter)
	statusFilter = strings.ToLower(statusFilter)

	var filtered []*state.AgentState
	for _, agent := range agents {
		// Check name filter (substring, case-insensitive)
		if nameFilter != "" && !strings.Contains(strings.ToLower(agent.Name), nameFilter) {
			continue
		}

		// Check prompt filter (substring, case-insensitive)
		if promptFilter != "" && !strings.Contains(strings.ToLower(agent.Prompt), promptFilter) {
			continue
		}

		// Check model filter (substring, case-insensitive)
		if modelFilter != "" && !strings.Contains(strings.ToLower(agent.Model), modelFilter) {
			continue
		}

		// Check status filter (exact match for running/terminated, special handling for pausing/paused)
		if statusFilter != "" {
			effectiveStatus := agent.Status
			if agent.Status == "running" && agent.Paused {
				if agent.PausedAt != nil {
					effectiveStatus = "paused"
				} else {
					effectiveStatus = "pausing"
				}
			}
			if strings.ToLower(effectiveStatus) != statusFilter {
				continue
			}
		}

		// Check label filters
		if len(labelFilters) > 0 && !label.Match(agent.Labels, labelFilters) {
			continue
		}

		filtered = append(filtered, agent)
	}

	return filtered
}

func init() {
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "Show all agents including terminated")
	listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "Only display agent IDs")
	listCmd.Flags().StringVar(&listFormat, "format", "", "Output format: json or table (default)")

	// Filter flags
	listCmd.Flags().StringVarP(&listName, "name", "N", "", "Filter by agent name (substring match)")
	listCmd.Flags().StringVarP(&listPrompt, "prompt", "p", "", "Filter by prompt name (substring match)")
	listCmd.Flags().StringVarP(&listModel, "model", "m", "", "Filter by model name (substring match)")
	listCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status: running, pausing, paused, or terminated")

	// Count flag
	listCmd.Flags().BoolVar(&listCount, "count", false, "Output only the count of matching agents")

	// Last/Latest flags
	listCmd.Flags().IntVarP(&listLast, "last", "n", 0, "Show only the N most recently started agents")
	listCmd.Flags().BoolVarP(&listLatest, "latest", "l", false, "Show only the most recently started agent")

	// Label flags
	listCmd.Flags().StringArrayVarP(&listLabels, "label", "L", nil, "Filter by label (key=value for exact match, key for existence check)")
	listCmd.Flags().BoolVar(&listShowLabels, "show-labels", false, "Show labels column in table output")
}
