package cmd

import (
	"testing"
	"time"

	"github.com/mj1618/swarm-cli/internal/state"
)

func TestCalculateStats_Empty(t *testing.T) {
	agents := []*state.AgentState{}
	stats := calculateStats(agents)

	if stats.Total != 0 {
		t.Errorf("expected Total=0, got %d", stats.Total)
	}
	if stats.Running != 0 {
		t.Errorf("expected Running=0, got %d", stats.Running)
	}
	if stats.Paused != 0 {
		t.Errorf("expected Paused=0, got %d", stats.Paused)
	}
	if stats.Terminated != 0 {
		t.Errorf("expected Terminated=0, got %d", stats.Terminated)
	}
	if len(stats.PromptStats) != 0 {
		t.Errorf("expected empty PromptStats, got %d entries", len(stats.PromptStats))
	}
	if len(stats.ModelStats) != 0 {
		t.Errorf("expected empty ModelStats, got %d entries", len(stats.ModelStats))
	}
}

func TestCalculateStats_StatusCounts(t *testing.T) {
	now := time.Now()
	pausedAt := now.Add(-5 * time.Minute)

	agents := []*state.AgentState{
		{ID: "1", Status: "running", Paused: false, StartedAt: now, Prompt: "p1", Model: "m1", Iterations: 10, CurrentIter: 5},
		{ID: "2", Status: "running", Paused: false, StartedAt: now, Prompt: "p1", Model: "m1", Iterations: 10, CurrentIter: 3},
		{ID: "3", Status: "running", Paused: true, PausedAt: &pausedAt, StartedAt: now, Prompt: "p2", Model: "m2", Iterations: 5, CurrentIter: 2},
		{ID: "4", Status: "terminated", StartedAt: now.Add(-1 * time.Hour), Prompt: "p2", Model: "m1", Iterations: 5, CurrentIter: 5},
		{ID: "5", Status: "terminated", StartedAt: now.Add(-2 * time.Hour), Prompt: "p3", Model: "m2", Iterations: 3, CurrentIter: 3},
	}

	stats := calculateStats(agents)

	if stats.Total != 5 {
		t.Errorf("expected Total=5, got %d", stats.Total)
	}
	if stats.Running != 2 {
		t.Errorf("expected Running=2, got %d", stats.Running)
	}
	if stats.Paused != 1 {
		t.Errorf("expected Paused=1, got %d", stats.Paused)
	}
	if stats.Terminated != 2 {
		t.Errorf("expected Terminated=2, got %d", stats.Terminated)
	}
}

func TestCalculateStats_IterationCounts(t *testing.T) {
	now := time.Now()

	agents := []*state.AgentState{
		{ID: "1", Status: "running", StartedAt: now, Prompt: "p1", Model: "m1", Iterations: 10, CurrentIter: 5},
		{ID: "2", Status: "running", StartedAt: now, Prompt: "p1", Model: "m1", Iterations: 20, CurrentIter: 8},
		{ID: "3", Status: "terminated", StartedAt: now, Prompt: "p2", Model: "m2", Iterations: 5, CurrentIter: 5},
	}

	stats := calculateStats(agents)

	// 5 + 8 + 5 = 18 completed
	if stats.IterationsCompleted != 18 {
		t.Errorf("expected IterationsCompleted=18, got %d", stats.IterationsCompleted)
	}
	// 10 + 20 + 5 = 35 total
	if stats.IterationsTotal != 35 {
		t.Errorf("expected IterationsTotal=35, got %d", stats.IterationsTotal)
	}
}

func TestCalculateStats_PromptStats(t *testing.T) {
	now := time.Now()

	agents := []*state.AgentState{
		{ID: "1", Status: "running", StartedAt: now, Prompt: "coder", Model: "m1", Iterations: 10, CurrentIter: 5},
		{ID: "2", Status: "running", StartedAt: now, Prompt: "coder", Model: "m1", Iterations: 20, CurrentIter: 8},
		{ID: "3", Status: "running", StartedAt: now, Prompt: "coder", Model: "m1", Iterations: 5, CurrentIter: 2},
		{ID: "4", Status: "terminated", StartedAt: now, Prompt: "planner", Model: "m2", Iterations: 5, CurrentIter: 5},
		{ID: "5", Status: "terminated", StartedAt: now, Prompt: "reviewer", Model: "m2", Iterations: 3, CurrentIter: 3},
	}

	stats := calculateStats(agents)

	if len(stats.PromptStats) != 3 {
		t.Errorf("expected 3 prompt stats, got %d", len(stats.PromptStats))
	}

	// Should be sorted by run count (coder: 3, planner: 1, reviewer: 1)
	if stats.PromptStats[0].Name != "coder" {
		t.Errorf("expected first prompt to be 'coder', got %s", stats.PromptStats[0].Name)
	}
	if stats.PromptStats[0].RunCount != 3 {
		t.Errorf("expected coder run count=3, got %d", stats.PromptStats[0].RunCount)
	}
	// 5 + 8 + 2 = 15 iterations for coder
	if stats.PromptStats[0].Iterations != 15 {
		t.Errorf("expected coder iterations=15, got %d", stats.PromptStats[0].Iterations)
	}
}

func TestCalculateStats_ModelStats(t *testing.T) {
	now := time.Now()

	agents := []*state.AgentState{
		{ID: "1", Status: "running", StartedAt: now, Prompt: "p1", Model: "claude-opus-4-20250514", Iterations: 10, CurrentIter: 5},
		{ID: "2", Status: "running", StartedAt: now, Prompt: "p1", Model: "claude-opus-4-20250514", Iterations: 20, CurrentIter: 8},
		{ID: "3", Status: "running", StartedAt: now, Prompt: "p2", Model: "claude-sonnet-4-20250514", Iterations: 5, CurrentIter: 2},
		{ID: "4", Status: "terminated", StartedAt: now, Prompt: "p2", Model: "claude-opus-4-20250514", Iterations: 5, CurrentIter: 5},
	}

	stats := calculateStats(agents)

	if len(stats.ModelStats) != 2 {
		t.Errorf("expected 2 model stats, got %d", len(stats.ModelStats))
	}

	// Should be sorted by count (opus: 3, sonnet: 1)
	if stats.ModelStats[0].Name != "claude-opus-4-20250514" {
		t.Errorf("expected first model to be 'claude-opus-4-20250514', got %s", stats.ModelStats[0].Name)
	}
	if stats.ModelStats[0].Count != 3 {
		t.Errorf("expected opus count=3, got %d", stats.ModelStats[0].Count)
	}
	if stats.ModelStats[1].Name != "claude-sonnet-4-20250514" {
		t.Errorf("expected second model to be 'claude-sonnet-4-20250514', got %s", stats.ModelStats[1].Name)
	}
	if stats.ModelStats[1].Count != 1 {
		t.Errorf("expected sonnet count=1, got %d", stats.ModelStats[1].Count)
	}
}

func TestCalculateStats_EmptyPromptAndModel(t *testing.T) {
	now := time.Now()

	agents := []*state.AgentState{
		{ID: "1", Status: "running", StartedAt: now, Prompt: "", Model: "", Iterations: 5, CurrentIter: 2},
	}

	stats := calculateStats(agents)

	// Empty prompt should be grouped as "(none)"
	if len(stats.PromptStats) != 1 {
		t.Errorf("expected 1 prompt stat, got %d", len(stats.PromptStats))
	}
	if stats.PromptStats[0].Name != "(none)" {
		t.Errorf("expected prompt name to be '(none)', got %s", stats.PromptStats[0].Name)
	}

	// Empty model should be grouped as "(unknown)"
	if len(stats.ModelStats) != 1 {
		t.Errorf("expected 1 model stat, got %d", len(stats.ModelStats))
	}
	if stats.ModelStats[0].Name != "(unknown)" {
		t.Errorf("expected model name to be '(unknown)', got %s", stats.ModelStats[0].Name)
	}
}

func TestCalculateStats_AverageRuntime(t *testing.T) {
	now := time.Now()

	// Two agents that started 1 hour ago
	agents := []*state.AgentState{
		{ID: "1", Status: "running", StartedAt: now.Add(-1 * time.Hour), Prompt: "p1", Model: "m1", Iterations: 10, CurrentIter: 5},
		{ID: "2", Status: "running", StartedAt: now.Add(-1 * time.Hour), Prompt: "p1", Model: "m1", Iterations: 10, CurrentIter: 5},
	}

	stats := calculateStats(agents)

	// Total should be roughly 2 hours (7200 seconds)
	// Allow some tolerance for test execution time
	if stats.TotalRuntimeSeconds < 7100 || stats.TotalRuntimeSeconds > 7300 {
		t.Errorf("expected TotalRuntimeSeconds ~7200, got %d", stats.TotalRuntimeSeconds)
	}

	// Average should be roughly 1 hour (3600 seconds)
	if stats.AverageRuntimeSeconds < 3500 || stats.AverageRuntimeSeconds > 3700 {
		t.Errorf("expected AverageRuntimeSeconds ~3600, got %d", stats.AverageRuntimeSeconds)
	}
}

func TestFormatStatsDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{0, "0m"},
		{30 * time.Second, "0m"},
		{1 * time.Minute, "1m"},
		{59 * time.Minute, "59m"},
		{1 * time.Hour, "1h 0m"},
		{1*time.Hour + 30*time.Minute, "1h 30m"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
		{24 * time.Hour, "24h 0m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatStatsDuration(tt.input)
			if result != tt.expected {
				t.Errorf("formatStatsDuration(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
