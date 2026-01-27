package cmd

import (
	"testing"
	"time"

	"github.com/matt/swarm-cli/internal/state"
)

func TestFilterAgentsByStatus(t *testing.T) {
	now := time.Now()
	pausedAt := now.Add(-5 * time.Minute)

	agents := []*state.AgentState{
		{
			ID:     "running-1",
			Status: "running",
			Paused: false,
		},
		{
			ID:       "pausing-1",
			Status:   "running",
			Paused:   true,
			PausedAt: nil, // Pause requested but not yet effective
		},
		{
			ID:       "paused-1",
			Status:   "running",
			Paused:   true,
			PausedAt: &pausedAt, // Actually paused
		},
		{
			ID:     "terminated-1",
			Status: "terminated",
			Paused: false,
		},
	}

	tests := []struct {
		name           string
		statusFilter   string
		expectedIDs    []string
		notExpectedIDs []string
	}{
		{
			name:           "filter running",
			statusFilter:   "running",
			expectedIDs:    []string{"running-1"},
			notExpectedIDs: []string{"pausing-1", "paused-1", "terminated-1"},
		},
		{
			name:           "filter pausing",
			statusFilter:   "pausing",
			expectedIDs:    []string{"pausing-1"},
			notExpectedIDs: []string{"running-1", "paused-1", "terminated-1"},
		},
		{
			name:           "filter paused",
			statusFilter:   "paused",
			expectedIDs:    []string{"paused-1"},
			notExpectedIDs: []string{"running-1", "pausing-1", "terminated-1"},
		},
		{
			name:           "filter terminated",
			statusFilter:   "terminated",
			expectedIDs:    []string{"terminated-1"},
			notExpectedIDs: []string{"running-1", "pausing-1", "paused-1"},
		},
		{
			name:           "no filter returns all",
			statusFilter:   "",
			expectedIDs:    []string{"running-1", "pausing-1", "paused-1", "terminated-1"},
			notExpectedIDs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterAgents(agents, "", "", "", tt.statusFilter)

			// Check expected IDs are present
			for _, expectedID := range tt.expectedIDs {
				found := false
				for _, a := range filtered {
					if a.ID == expectedID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected agent %s to be in filtered results", expectedID)
				}
			}

			// Check not-expected IDs are absent
			for _, notExpectedID := range tt.notExpectedIDs {
				for _, a := range filtered {
					if a.ID == notExpectedID {
						t.Errorf("agent %s should not be in filtered results for status %q", notExpectedID, tt.statusFilter)
					}
				}
			}
		})
	}
}

func TestFilterAgentsCaseInsensitive(t *testing.T) {
	now := time.Now()
	pausedAt := now.Add(-5 * time.Minute)

	agents := []*state.AgentState{
		{
			ID:       "pausing-1",
			Status:   "running",
			Paused:   true,
			PausedAt: nil,
		},
		{
			ID:       "paused-1",
			Status:   "running",
			Paused:   true,
			PausedAt: &pausedAt,
		},
	}

	// Test case variations
	testCases := []struct {
		filter     string
		expectedID string
	}{
		{"PAUSING", "pausing-1"},
		{"Pausing", "pausing-1"},
		{"pausing", "pausing-1"},
		{"PAUSED", "paused-1"},
		{"Paused", "paused-1"},
		{"paused", "paused-1"},
	}

	for _, tc := range testCases {
		t.Run(tc.filter, func(t *testing.T) {
			filtered := filterAgents(agents, "", "", "", tc.filter)
			if len(filtered) != 1 {
				t.Errorf("expected 1 result for filter %q, got %d", tc.filter, len(filtered))
				return
			}
			if filtered[0].ID != tc.expectedID {
				t.Errorf("expected agent %s for filter %q, got %s", tc.expectedID, tc.filter, filtered[0].ID)
			}
		})
	}
}

func TestFilterAgentsCombinedFilters(t *testing.T) {
	now := time.Now()
	pausedAt := now.Add(-5 * time.Minute)

	agents := []*state.AgentState{
		{
			ID:     "agent-1",
			Prompt: "coder",
			Model:  "sonnet",
			Status: "running",
			Paused: false,
		},
		{
			ID:       "agent-2",
			Prompt:   "coder",
			Model:    "opus",
			Status:   "running",
			Paused:   true,
			PausedAt: nil, // pausing
		},
		{
			ID:       "agent-3",
			Prompt:   "planner",
			Model:    "sonnet",
			Status:   "running",
			Paused:   true,
			PausedAt: &pausedAt, // paused
		},
	}

	// Test combined prompt + status filter
	t.Run("prompt and pausing status", func(t *testing.T) {
		filtered := filterAgents(agents, "", "coder", "", "pausing")
		if len(filtered) != 1 {
			t.Errorf("expected 1 result, got %d", len(filtered))
			return
		}
		if filtered[0].ID != "agent-2" {
			t.Errorf("expected agent-2, got %s", filtered[0].ID)
		}
	})

	// Test combined model + status filter
	t.Run("model and paused status", func(t *testing.T) {
		filtered := filterAgents(agents, "", "", "sonnet", "paused")
		if len(filtered) != 1 {
			t.Errorf("expected 1 result, got %d", len(filtered))
			return
		}
		if filtered[0].ID != "agent-3" {
			t.Errorf("expected agent-3, got %s", filtered[0].ID)
		}
	})
}

// computeEffectiveStatus is a helper that mirrors the logic in filterAgents
// to make test assertions clearer
func computeEffectiveStatus(agent *state.AgentState) string {
	if agent.Status == "running" && agent.Paused {
		if agent.PausedAt != nil {
			return "paused"
		}
		return "pausing"
	}
	return agent.Status
}

func TestEffectiveStatusComputation(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		agent          *state.AgentState
		expectedStatus string
	}{
		{
			name: "running agent",
			agent: &state.AgentState{
				Status: "running",
				Paused: false,
			},
			expectedStatus: "running",
		},
		{
			name: "pausing agent (Paused=true, PausedAt=nil)",
			agent: &state.AgentState{
				Status:   "running",
				Paused:   true,
				PausedAt: nil,
			},
			expectedStatus: "pausing",
		},
		{
			name: "paused agent (Paused=true, PausedAt!=nil)",
			agent: &state.AgentState{
				Status:   "running",
				Paused:   true,
				PausedAt: &now,
			},
			expectedStatus: "paused",
		},
		{
			name: "terminated agent",
			agent: &state.AgentState{
				Status: "terminated",
				Paused: false,
			},
			expectedStatus: "terminated",
		},
		{
			name: "terminated agent with Paused=true (edge case)",
			agent: &state.AgentState{
				Status: "terminated",
				Paused: true,
			},
			expectedStatus: "terminated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := computeEffectiveStatus(tt.agent)
			if status != tt.expectedStatus {
				t.Errorf("expected status %q, got %q", tt.expectedStatus, status)
			}
		})
	}
}

func TestFilterAgentsByName(t *testing.T) {
	agents := []*state.AgentState{
		{ID: "1", Name: "coder-frontend", Prompt: "task1", Model: "opus", Status: "running"},
		{ID: "2", Name: "coder-backend", Prompt: "task2", Model: "sonnet", Status: "running"},
		{ID: "3", Name: "reviewer", Prompt: "task3", Model: "opus", Status: "running"},
		{ID: "4", Name: "", Prompt: "task4", Model: "opus", Status: "running"}, // no name
	}

	// Test name filter
	t.Run("filter by name substring", func(t *testing.T) {
		filtered := filterAgents(agents, "coder", "", "", "")
		if len(filtered) != 2 {
			t.Errorf("expected 2 agents, got %d", len(filtered))
		}
		// Verify both coder agents are included
		foundFrontend, foundBackend := false, false
		for _, a := range filtered {
			if a.ID == "1" {
				foundFrontend = true
			}
			if a.ID == "2" {
				foundBackend = true
			}
		}
		if !foundFrontend || !foundBackend {
			t.Errorf("expected both coder agents, found frontend=%v backend=%v", foundFrontend, foundBackend)
		}
	})

	// Test case insensitivity
	t.Run("case insensitive match", func(t *testing.T) {
		filtered := filterAgents(agents, "CODER", "", "", "")
		if len(filtered) != 2 {
			t.Errorf("expected 2 agents with case-insensitive match, got %d", len(filtered))
		}
	})

	// Test combined name + model filter
	t.Run("name and model combined", func(t *testing.T) {
		filtered := filterAgents(agents, "coder", "", "opus", "")
		if len(filtered) != 1 {
			t.Errorf("expected 1 agent matching name AND model, got %d", len(filtered))
		}
		if len(filtered) > 0 && filtered[0].ID != "1" {
			t.Errorf("expected agent 1 (coder-frontend with opus), got %s", filtered[0].ID)
		}
	})

	// Test combined name + status filter
	t.Run("name and status combined", func(t *testing.T) {
		filtered := filterAgents(agents, "coder", "", "", "running")
		if len(filtered) != 2 {
			t.Errorf("expected 2 agents matching name AND status, got %d", len(filtered))
		}
	})

	// Test no match
	t.Run("no match for nonexistent name", func(t *testing.T) {
		filtered := filterAgents(agents, "nonexistent", "", "", "")
		if len(filtered) != 0 {
			t.Errorf("expected 0 agents, got %d", len(filtered))
		}
	})

	// Test empty name agents don't match
	t.Run("empty name agents don't match filter", func(t *testing.T) {
		// Filtering for "task" should not match the empty-named agent by name
		filtered := filterAgents(agents, "task", "", "", "")
		if len(filtered) != 0 {
			t.Errorf("expected 0 agents (empty name shouldn't match), got %d", len(filtered))
		}
	})

	// Test exact name match
	t.Run("exact name match", func(t *testing.T) {
		filtered := filterAgents(agents, "reviewer", "", "", "")
		if len(filtered) != 1 {
			t.Errorf("expected 1 agent, got %d", len(filtered))
		}
		if len(filtered) > 0 && filtered[0].ID != "3" {
			t.Errorf("expected agent 3 (reviewer), got %s", filtered[0].ID)
		}
	})
}
