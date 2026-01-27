package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Integration tests that test the full state manager lifecycle

func TestManagerFullLifecycle(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// 1. Start with empty state
	agents, _ := mgr.List()
	initialCount := len(agents)

	// 2. Register multiple agents
	agents1 := []*AgentState{
		{
			ID:          GenerateID(),
			PID:         os.Getpid(), // Use current process so it's "running"
			Prompt:      "prompt-a",
			Model:       "model-a",
			StartedAt:   time.Now(),
			Iterations:  5,
			CurrentIter: 0,
			Status:      "running",
		},
		{
			ID:          GenerateID(),
			PID:         99999, // Non-existent process
			Prompt:      "prompt-b",
			Model:       "model-b",
			StartedAt:   time.Now(),
			Iterations:  10,
			CurrentIter: 0,
			Status:      "running",
		},
	}

	for _, a := range agents1 {
		if err := mgr.Register(a); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	// 3. Verify registration
	listed, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(listed) < initialCount+2 {
		t.Errorf("Expected at least %d agents, got %d", initialCount+2, len(listed))
	}

	// 4. Update first agent through multiple iterations
	for i := 1; i <= 5; i++ {
		agents1[0].CurrentIter = i
		if err := mgr.Update(agents1[0]); err != nil {
			t.Fatalf("Update iteration %d failed: %v", i, err)
		}

		retrieved, err := mgr.Get(agents1[0].ID)
		if err != nil {
			t.Fatalf("Get after update failed: %v", err)
		}
		if retrieved.CurrentIter != i {
			t.Errorf("CurrentIter should be %d, got %d", i, retrieved.CurrentIter)
		}
	}

	// 5. Test termination modes
	agents1[0].TerminateMode = "after_iteration"
	if err := mgr.Update(agents1[0]); err != nil {
		t.Fatalf("Update terminate mode failed: %v", err)
	}

	retrieved, _ := mgr.Get(agents1[0].ID)
	if retrieved.TerminateMode != "after_iteration" {
		t.Errorf("TerminateMode should be 'after_iteration', got %s", retrieved.TerminateMode)
	}

	// 6. Remove agents
	for _, a := range agents1 {
		if err := mgr.Remove(a.ID); err != nil {
			t.Fatalf("Remove failed: %v", err)
		}
	}

	// 7. Verify removal
	for _, a := range agents1 {
		_, err := mgr.Get(a.ID)
		if err == nil {
			t.Errorf("Agent %s should be removed", a.ID)
		}
	}
}

func TestStateFilePersistence(t *testing.T) {
	mgr1, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager 1 failed: %v", err)
	}

	// Create an agent with first manager
	agent := &AgentState{
		ID:          GenerateID(),
		PID:         os.Getpid(),
		Prompt:      "persistence-test",
		Model:       "test-model",
		StartedAt:   time.Now(),
		Iterations:  100,
		CurrentIter: 42,
		Status:      "running",
		LogFile:     "/tmp/test.log",
	}

	if err := mgr1.Register(agent); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Create a new manager instance (simulates restart)
	mgr2, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager 2 failed: %v", err)
	}

	// Should be able to retrieve the agent
	retrieved, err := mgr2.Get(agent.ID)
	if err != nil {
		t.Fatalf("Get from second manager failed: %v", err)
	}

	if retrieved.Prompt != agent.Prompt {
		t.Errorf("Prompt mismatch: got %s, want %s", retrieved.Prompt, agent.Prompt)
	}
	if retrieved.CurrentIter != agent.CurrentIter {
		t.Errorf("CurrentIter mismatch: got %d, want %d", retrieved.CurrentIter, agent.CurrentIter)
	}
	if retrieved.LogFile != agent.LogFile {
		t.Errorf("LogFile mismatch: got %s, want %s", retrieved.LogFile, agent.LogFile)
	}

	// Cleanup
	_ = mgr2.Remove(agent.ID)
}

func TestStateJSONFormat(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	agent := &AgentState{
		ID:          GenerateID(),
		PID:         12345,
		Prompt:      "json-test",
		Model:       "test-model",
		StartedAt:   time.Now(),
		Iterations:  10,
		CurrentIter: 3,
		Status:      "running",
	}

	if err := mgr.Register(agent); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Read the state file directly
	homeDir, _ := os.UserHomeDir()
	statePath := filepath.Join(homeDir, ".swarm", "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	// Verify it's valid JSON
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("State file is not valid JSON: %v", err)
	}

	// Verify our agent is in the state
	if state.Agents == nil {
		t.Fatal("State.Agents is nil")
	}

	savedAgent, exists := state.Agents[agent.ID]
	if !exists {
		t.Fatalf("Agent %s not found in state file", agent.ID)
	}

	if savedAgent.Prompt != "json-test" {
		t.Errorf("Saved agent prompt mismatch")
	}

	// Cleanup
	_ = mgr.Remove(agent.ID)
}

func TestCleanupOnNewManager(t *testing.T) {
	mgr1, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager 1 failed: %v", err)
	}

	// Create an agent with a non-existent PID
	agent := &AgentState{
		ID:          GenerateID(),
		PID:         9999999, // Almost certainly doesn't exist
		Prompt:      "cleanup-test",
		Model:       "test-model",
		StartedAt:   time.Now(),
		Iterations:  10,
		CurrentIter: 0,
		Status:      "running",
	}

	if err := mgr1.Register(agent); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Create a new manager - this triggers cleanup
	mgr2, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager 2 failed: %v", err)
	}

	// The agent should now be marked as terminated
	retrieved, err := mgr2.Get(agent.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Status != "terminated" {
		t.Errorf("Agent with non-existent PID should be marked terminated, got status: %s", retrieved.Status)
	}

	// Cleanup
	_ = mgr2.Remove(agent.ID)
}

func TestEmptyStateFile(t *testing.T) {
	// Test behavior with empty/missing state file
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// List should work even with empty state
	agents, err := mgr.List()
	if err != nil {
		t.Fatalf("List on empty state failed: %v", err)
	}

	// Should return empty list, not error
	if agents == nil {
		// nil is OK, just means empty
	}
}

func TestTimestampPersistence(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	startTime := time.Now().Add(-1 * time.Hour) // 1 hour ago
	agent := &AgentState{
		ID:          GenerateID(),
		PID:         os.Getpid(),
		Prompt:      "timestamp-test",
		Model:       "test-model",
		StartedAt:   startTime,
		Iterations:  10,
		CurrentIter: 5,
		Status:      "running",
	}

	if err := mgr.Register(agent); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Retrieve and verify timestamp is preserved
	retrieved, err := mgr.Get(agent.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Allow 1 second tolerance for JSON serialization
	diff := retrieved.StartedAt.Sub(startTime)
	if diff > time.Second || diff < -time.Second {
		t.Errorf("StartedAt not preserved correctly. Got %v, want %v (diff: %v)", 
			retrieved.StartedAt, startTime, diff)
	}

	// Cleanup
	_ = mgr.Remove(agent.ID)
}
