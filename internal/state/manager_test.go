package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	// Generate multiple IDs and ensure they're unique
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := GenerateID()
		if ids[id] {
			t.Errorf("GenerateID produced duplicate ID: %s", id)
		}
		ids[id] = true

		// Check ID format (should be 8 hex characters)
		if len(id) != 8 {
			t.Errorf("GenerateID produced ID with unexpected length: %d (expected 8)", len(id))
		}
	}
}

func TestNewManager(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("NewManager returned nil manager")
	}

	// Verify state path is set correctly
	homeDir, _ := os.UserHomeDir()
	expectedPath := filepath.Join(homeDir, ".swarm", "state.json")
	if mgr.statePath != expectedPath {
		t.Errorf("state path mismatch: got %s, want %s", mgr.statePath, expectedPath)
	}
}

func TestManagerRegisterAndGet(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	agent := &AgentState{
		ID:          GenerateID(),
		PID:         12345,
		Prompt:      "test-prompt",
		Model:       "test-model",
		StartedAt:   time.Now(),
		Iterations:  10,
		CurrentIter: 1,
		Status:      "running",
	}

	// Register the agent
	if err := mgr.Register(agent); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Get the agent
	retrieved, err := mgr.Get(agent.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Verify fields match
	if retrieved.ID != agent.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, agent.ID)
	}
	if retrieved.PID != agent.PID {
		t.Errorf("PID mismatch: got %d, want %d", retrieved.PID, agent.PID)
	}
	if retrieved.Prompt != agent.Prompt {
		t.Errorf("Prompt mismatch: got %s, want %s", retrieved.Prompt, agent.Prompt)
	}
	if retrieved.Model != agent.Model {
		t.Errorf("Model mismatch: got %s, want %s", retrieved.Model, agent.Model)
	}
	if retrieved.Iterations != agent.Iterations {
		t.Errorf("Iterations mismatch: got %d, want %d", retrieved.Iterations, agent.Iterations)
	}
	if retrieved.Status != agent.Status {
		t.Errorf("Status mismatch: got %s, want %s", retrieved.Status, agent.Status)
	}

	// Cleanup
	_ = mgr.Remove(agent.ID)
}

func TestManagerUpdate(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	agent := &AgentState{
		ID:          GenerateID(),
		PID:         12345,
		Prompt:      "test-prompt",
		Model:       "test-model",
		StartedAt:   time.Now(),
		Iterations:  10,
		CurrentIter: 1,
		Status:      "running",
	}

	// Register the agent
	if err := mgr.Register(agent); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Update the agent
	agent.CurrentIter = 5
	agent.Model = "new-model"
	agent.TerminateMode = "immediate"
	if err := mgr.Update(agent); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	retrieved, err := mgr.Get(agent.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.CurrentIter != 5 {
		t.Errorf("CurrentIter not updated: got %d, want 5", retrieved.CurrentIter)
	}
	if retrieved.Model != "new-model" {
		t.Errorf("Model not updated: got %s, want new-model", retrieved.Model)
	}
	if retrieved.TerminateMode != "immediate" {
		t.Errorf("TerminateMode not updated: got %s, want immediate", retrieved.TerminateMode)
	}

	// Cleanup
	_ = mgr.Remove(agent.ID)
}

func TestManagerUpdateNonExistent(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	agent := &AgentState{
		ID:     "nonexistent-id",
		Status: "running",
	}

	err = mgr.Update(agent)
	if err == nil {
		t.Error("Update should fail for non-existent agent")
	}
}

func TestManagerList(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create multiple agents
	agent1 := &AgentState{
		ID:        GenerateID(),
		PID:       11111,
		Prompt:    "prompt-1",
		Model:     "model-1",
		StartedAt: time.Now(),
		Status:    "running",
	}
	agent2 := &AgentState{
		ID:        GenerateID(),
		PID:       22222,
		Prompt:    "prompt-2",
		Model:     "model-2",
		StartedAt: time.Now(),
		Status:    "running",
	}

	_ = mgr.Register(agent1)
	_ = mgr.Register(agent2)

	// List agents
	agents, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Verify our agents are in the list
	found1, found2 := false, false
	for _, a := range agents {
		if a.ID == agent1.ID {
			found1 = true
		}
		if a.ID == agent2.ID {
			found2 = true
		}
	}

	if !found1 {
		t.Error("agent1 not found in list")
	}
	if !found2 {
		t.Error("agent2 not found in list")
	}

	// Cleanup
	_ = mgr.Remove(agent1.ID)
	_ = mgr.Remove(agent2.ID)
}

func TestManagerRemove(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	agent := &AgentState{
		ID:        GenerateID(),
		PID:       12345,
		Prompt:    "test-prompt",
		StartedAt: time.Now(),
		Status:    "running",
	}

	// Register
	if err := mgr.Register(agent); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Remove
	if err := mgr.Remove(agent.ID); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify removal
	_, err = mgr.Get(agent.ID)
	if err == nil {
		t.Error("Get should fail after Remove")
	}
}

func TestManagerGetNonExistent(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	_, err = mgr.Get("nonexistent-id")
	if err == nil {
		t.Error("Get should fail for non-existent agent")
	}
}

func TestIsProcessRunning(t *testing.T) {
	// Test with current process (should be running)
	if !isProcessRunning(os.Getpid()) {
		t.Error("isProcessRunning returned false for current process")
	}

	// Test with invalid PID
	if isProcessRunning(0) {
		t.Error("isProcessRunning returned true for PID 0")
	}

	if isProcessRunning(-1) {
		t.Error("isProcessRunning returned true for negative PID")
	}

	// Test with a PID that almost certainly doesn't exist
	if isProcessRunning(9999999) {
		t.Error("isProcessRunning returned true for non-existent PID")
	}
}

func TestAgentStateFields(t *testing.T) {
	now := time.Now()
	agent := &AgentState{
		ID:            "test-id",
		PID:           12345,
		Prompt:        "test-prompt",
		Model:         "opus-4.5-thinking",
		StartedAt:     now,
		Iterations:    20,
		CurrentIter:   5,
		Status:        "running",
		TerminateMode: "immediate",
		LogFile:       "/var/log/agent.log",
	}

	if agent.ID != "test-id" {
		t.Errorf("ID mismatch: got %s", agent.ID)
	}
	if agent.PID != 12345 {
		t.Errorf("PID mismatch: got %d", agent.PID)
	}
	if agent.Prompt != "test-prompt" {
		t.Errorf("Prompt mismatch: got %s", agent.Prompt)
	}
	if agent.Model != "opus-4.5-thinking" {
		t.Errorf("Model mismatch: got %s", agent.Model)
	}
	if !agent.StartedAt.Equal(now) {
		t.Errorf("StartedAt mismatch: got %v", agent.StartedAt)
	}
	if agent.Iterations != 20 {
		t.Errorf("Iterations mismatch: got %d", agent.Iterations)
	}
	if agent.CurrentIter != 5 {
		t.Errorf("CurrentIter mismatch: got %d", agent.CurrentIter)
	}
	if agent.Status != "running" {
		t.Errorf("Status mismatch: got %s", agent.Status)
	}
	if agent.TerminateMode != "immediate" {
		t.Errorf("TerminateMode mismatch: got %s", agent.TerminateMode)
	}
	if agent.LogFile != "/var/log/agent.log" {
		t.Errorf("LogFile mismatch: got %s", agent.LogFile)
	}
}

func TestConcurrentAccess(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Register an agent
	agent := &AgentState{
		ID:          GenerateID(),
		PID:         12345,
		Prompt:      "test-prompt",
		Model:       "test-model",
		StartedAt:   time.Now(),
		Iterations:  100,
		CurrentIter: 0,
		Status:      "running",
	}

	if err := mgr.Register(agent); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Concurrent updates
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(iter int) {
			agent.CurrentIter = iter
			_ = mgr.Update(agent)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Cleanup
	_ = mgr.Remove(agent.ID)
}
