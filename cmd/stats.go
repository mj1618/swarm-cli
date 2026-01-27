package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/fatih/color"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var statsFormat string

// Stats represents aggregate statistics about agents.
type Stats struct {
	Running    int `json:"running"`
	Paused     int `json:"paused"`
	Terminated int `json:"terminated"`
	Total      int `json:"total"`

	IterationsCompleted  int     `json:"iterations_completed"`
	IterationsTotal      int     `json:"iterations_total"`
	IterationsSuccessful int     `json:"iterations_successful"`
	IterationsFailed     int     `json:"iterations_failed"`
	SuccessRate          float64 `json:"success_rate"`

	PromptStats []PromptStat `json:"prompt_stats"`
	ModelStats  []ModelStat  `json:"model_stats"`

	TotalRuntimeSeconds   int64 `json:"total_runtime_seconds"`
	AverageRuntimeSeconds int64 `json:"average_runtime_seconds"`
}

// PromptStat represents statistics for a single prompt.
type PromptStat struct {
	Name       string `json:"name"`
	RunCount   int    `json:"run_count"`
	Iterations int    `json:"iterations"`
}

// ModelStat represents statistics for a single model.
type ModelStat struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show agent usage statistics",
	Long: `Display aggregate statistics about agent usage.

Shows counts of running/paused/terminated agents, iteration totals,
prompt usage frequency, and model distribution.`,
	Example: `  # Show stats for current project
  swarm stats

  # Show stats across all projects
  swarm stats --global

  # Output as JSON
  swarm stats --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// Get all agents (not just running)
		agents, err := mgr.List(false)
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		stats := calculateStats(agents)

		if statsFormat == "json" {
			output, err := json.MarshalIndent(stats, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal stats: %w", err)
			}
			fmt.Println(string(output))
			return nil
		}

		printStats(stats)
		return nil
	},
}

func calculateStats(agents []*state.AgentState) Stats {
	stats := Stats{}
	promptMap := make(map[string]*PromptStat)
	modelMap := make(map[string]int)

	now := time.Now()

	for _, agent := range agents {
		stats.Total++

		// Status counts
		switch agent.Status {
		case "running":
			if agent.Paused {
				stats.Paused++
			} else {
				stats.Running++
			}
		case "terminated":
			stats.Terminated++
		}

		// Iteration counts
		stats.IterationsCompleted += agent.CurrentIter
		stats.IterationsTotal += agent.Iterations
		stats.IterationsSuccessful += agent.SuccessfulIters
		stats.IterationsFailed += agent.FailedIters

		// Prompt stats
		promptName := agent.Prompt
		if promptName == "" {
			promptName = "(none)"
		}
		if ps, ok := promptMap[promptName]; ok {
			ps.RunCount++
			ps.Iterations += agent.CurrentIter
		} else {
			promptMap[promptName] = &PromptStat{
				Name:       promptName,
				RunCount:   1,
				Iterations: agent.CurrentIter,
			}
		}

		// Model stats
		modelName := agent.Model
		if modelName == "" {
			modelName = "(unknown)"
		}
		modelMap[modelName]++

		// Runtime calculation
		var endTime time.Time
		if agent.Status == "terminated" {
			if agent.TerminatedAt != nil {
				// Use actual termination time if available
				endTime = *agent.TerminatedAt
			} else {
				// For terminated agents without TerminatedAt, use an estimate based on iterations
				// Each iteration is roughly 5 minutes (heuristic)
				endTime = agent.StartedAt.Add(time.Duration(agent.CurrentIter) * 5 * time.Minute)
				// But cap it at now if that would be in the future
				if endTime.After(now) {
					endTime = now
				}
			}
		} else {
			endTime = now
		}
		stats.TotalRuntimeSeconds += int64(endTime.Sub(agent.StartedAt).Seconds())
	}

	// Convert maps to sorted slices
	for _, ps := range promptMap {
		stats.PromptStats = append(stats.PromptStats, *ps)
	}
	sort.Slice(stats.PromptStats, func(i, j int) bool {
		return stats.PromptStats[i].RunCount > stats.PromptStats[j].RunCount
	})

	for model, count := range modelMap {
		stats.ModelStats = append(stats.ModelStats, ModelStat{Name: model, Count: count})
	}
	sort.Slice(stats.ModelStats, func(i, j int) bool {
		return stats.ModelStats[i].Count > stats.ModelStats[j].Count
	})

	// Calculate average
	if stats.Total > 0 {
		stats.AverageRuntimeSeconds = stats.TotalRuntimeSeconds / int64(stats.Total)
	}

	// Calculate success rate
	totalIterOutcomes := stats.IterationsSuccessful + stats.IterationsFailed
	if totalIterOutcomes > 0 {
		stats.SuccessRate = float64(stats.IterationsSuccessful) / float64(totalIterOutcomes) * 100
	}

	return stats
}

func printStats(stats Stats) {
	bold := color.New(color.Bold)

	bold.Println("Agent Statistics")
	fmt.Println("─────────────────────────────────")
	fmt.Println()

	bold.Println("Status Summary")
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	fmt.Print("  Running:    ")
	green.Println(stats.Running)
	fmt.Print("  Paused:     ")
	yellow.Println(stats.Paused)
	fmt.Print("  Terminated: ")
	red.Println(stats.Terminated)
	fmt.Printf("  Total:      %d\n", stats.Total)
	fmt.Println()

	bold.Println("Iterations")
	fmt.Printf("  Completed:  %d\n", stats.IterationsCompleted)
	fmt.Printf("  Total:      %d\n", stats.IterationsTotal)
	if stats.IterationsSuccessful > 0 || stats.IterationsFailed > 0 {
		fmt.Printf("  Successful: %d\n", stats.IterationsSuccessful)
		fmt.Printf("  Failed:     %d\n", stats.IterationsFailed)
		fmt.Printf("  Success rate: %.1f%%\n", stats.SuccessRate)
	}
	fmt.Println()

	if len(stats.PromptStats) > 0 {
		bold.Println("Top Prompts (by run count)")
		limit := 5
		if len(stats.PromptStats) < limit {
			limit = len(stats.PromptStats)
		}
		for i := 0; i < limit; i++ {
			ps := stats.PromptStats[i]
			fmt.Printf("  %-16s %d runs  (%d iterations)\n", ps.Name, ps.RunCount, ps.Iterations)
		}
		fmt.Println()
	}

	if len(stats.ModelStats) > 0 {
		bold.Println("Models Used")
		for _, ms := range stats.ModelStats {
			fmt.Printf("  %-30s %d agents\n", ms.Name, ms.Count)
		}
		fmt.Println()
	}

	bold.Println("Runtime")
	fmt.Printf("  Total:   %s\n", formatStatsDuration(time.Duration(stats.TotalRuntimeSeconds)*time.Second))
	fmt.Printf("  Average: %s per agent\n", formatStatsDuration(time.Duration(stats.AverageRuntimeSeconds)*time.Second))
}

func formatStatsDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return "0m"
}

func init() {
	statsCmd.Flags().StringVar(&statsFormat, "format", "", "Output format: json or table (default)")
	rootCmd.AddCommand(statsCmd)
}
