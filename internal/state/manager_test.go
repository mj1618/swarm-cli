package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mj1618/swarm-cli/internal/scope"
)

// newTestManager creates a Manager backed by a temp directory so tests
// don't interfere with the real ~/swarm/state.json or each other.
func newTestManager(t *testing.T) *Manager {
	t.Helper()
	dir := t.TempDir()
	return &Manager{
		statePath: filepath.Join(dir, "state.json"),
		lockPath:  filepath.Join(dir, "state.lock"),
		scope:     scope.ScopeGlobal,
	}
}

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
	agents, err := mgr.List(false)
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
			// Create a copy of the agent to avoid data race on shared struct
			agentCopy := &AgentState{
				ID:          agent.ID,
				PID:         agent.PID,
				Prompt:      agent.Prompt,
				Model:       agent.Model,
				StartedAt:   agent.StartedAt,
				Iterations:  agent.Iterations,
				CurrentIter: iter,
				Status:      agent.Status,
			}
			_ = mgr.Update(agentCopy)
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

func TestManagerListSortingAndFiltering(t *testing.T) {
	mgr := newTestManager(t)

	// Create agents with different start times and statuses
	now := time.Now()
	agent1 := &AgentState{
		ID:        GenerateID(),
		PID:       os.Getpid(),
		Prompt:    "oldest-running",
		StartedAt: now.Add(-3 * time.Hour), // Oldest
		Status:    "running",
	}
	agent2 := &AgentState{
		ID:        GenerateID(),
		PID:       os.Getpid(),
		Prompt:    "middle-terminated",
		StartedAt: now.Add(-2 * time.Hour), // Middle
		Status:    "terminated",
	}
	agent3 := &AgentState{
		ID:        GenerateID(),
		PID:       os.Getpid(),
		Prompt:    "newest-running",
		StartedAt: now.Add(-1 * time.Hour), // Newest
		Status:    "running",
	}

	// Register in non-chronological order
	_ = mgr.Register(agent3)
	_ = mgr.Register(agent1)
	_ = mgr.Register(agent2)

	// Test: List all agents (onlyRunning=false) should include all 3, sorted by StartedAt
	allAgents, err := mgr.List(false)
	if err != nil {
		t.Fatalf("List(false) failed: %v", err)
	}

	if len(allAgents) != 3 {
		t.Fatalf("Expected 3 agents, got %d", len(allAgents))
	}

	// Verify sorting: oldest should be first
	if allAgents[0].ID != agent1.ID {
		t.Errorf("Expected oldest agent first, got %s", allAgents[0].Prompt)
	}
	if allAgents[1].ID != agent2.ID {
		t.Errorf("Expected middle agent second, got %s", allAgents[1].Prompt)
	}
	if allAgents[2].ID != agent3.ID {
		t.Errorf("Expected newest agent third, got %s", allAgents[2].Prompt)
	}

	// Test: List only running agents (onlyRunning=true) should exclude terminated
	runningAgents, err := mgr.List(true)
	if err != nil {
		t.Fatalf("List(true) failed: %v", err)
	}

	if len(runningAgents) != 2 {
		t.Fatalf("Expected 2 running agents, got %d", len(runningAgents))
	}

	// Verify terminated agent is not included
	for _, a := range runningAgents {
		if a.ID == agent2.ID {
			t.Error("Terminated agent should not be in running list")
		}
	}

	// Verify sorting still applies: oldest running should be first
	if runningAgents[0].ID != agent1.ID {
		t.Errorf("Expected oldest running agent first, got %s", runningAgents[0].Prompt)
	}
	if runningAgents[1].ID != agent3.ID {
		t.Errorf("Expected newest running agent second, got %s", runningAgents[1].Prompt)
	}
}

func TestRegisterUniqueNameSuffix(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create first agent with name "my-task"
	agent1 := &AgentState{
		ID:        GenerateID(),
		Name:      "my-task",
		PID:       os.Getpid(),
		Prompt:    "test-prompt",
		Model:     "test-model",
		StartedAt: time.Now(),
		Status:    "running",
	}

	if err := mgr.Register(agent1); err != nil {
		t.Fatalf("Register agent1 failed: %v", err)
	}

	// Name should remain "my-task" (no conflict)
	if agent1.Name != "my-task" {
		t.Errorf("Expected name 'my-task', got '%s'", agent1.Name)
	}

	// Create second agent with same name - should get suffix
	agent2 := &AgentState{
		ID:        GenerateID(),
		Name:      "my-task",
		PID:       os.Getpid(),
		Prompt:    "test-prompt-2",
		Model:     "test-model",
		StartedAt: time.Now(),
		Status:    "running",
	}

	if err := mgr.Register(agent2); err != nil {
		t.Fatalf("Register agent2 failed: %v", err)
	}

	// Name should be "my-task-2"
	if agent2.Name != "my-task-2" {
		t.Errorf("Expected name 'my-task-2', got '%s'", agent2.Name)
	}

	// Create third agent with same name - should get next suffix
	agent3 := &AgentState{
		ID:        GenerateID(),
		Name:      "my-task",
		PID:       os.Getpid(),
		Prompt:    "test-prompt-3",
		Model:     "test-model",
		StartedAt: time.Now(),
		Status:    "running",
	}

	if err := mgr.Register(agent3); err != nil {
		t.Fatalf("Register agent3 failed: %v", err)
	}

	// Name should be "my-task-3"
	if agent3.Name != "my-task-3" {
		t.Errorf("Expected name 'my-task-3', got '%s'", agent3.Name)
	}

	// Terminate agent1 - its name should become available
	agent1.Status = "terminated"
	if err := mgr.Update(agent1); err != nil {
		t.Fatalf("Update agent1 failed: %v", err)
	}

	// Create fourth agent with same name - should reuse "my-task" (since terminated)
	agent4 := &AgentState{
		ID:        GenerateID(),
		Name:      "my-task",
		PID:       os.Getpid(),
		Prompt:    "test-prompt-4",
		Model:     "test-model",
		StartedAt: time.Now(),
		Status:    "running",
	}

	if err := mgr.Register(agent4); err != nil {
		t.Fatalf("Register agent4 failed: %v", err)
	}

	// Name should be "my-task" (reused from terminated agent)
	if agent4.Name != "my-task" {
		t.Errorf("Expected name 'my-task' (reused), got '%s'", agent4.Name)
	}

	// Cleanup
	_ = mgr.Remove(agent1.ID)
	_ = mgr.Remove(agent2.ID)
	_ = mgr.Remove(agent3.ID)
	_ = mgr.Remove(agent4.ID)
}

func TestGetLast(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create an agent that is definitely the newest (started "now")
	now := time.Now()
	newestAgent := &AgentState{
		ID:        GenerateID(),
		PID:       99999,
		Prompt:    "test-newest-agent",
		StartedAt: now, // Right now - should be newest
		Status:    "running",
	}

	// Register the agent
	if err := mgr.Register(newestAgent); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// GetLast should return our agent since it has the most recent StartedAt
	latest, err := mgr.GetLast()
	if err != nil {
		t.Fatalf("GetLast failed: %v", err)
	}

	if latest.ID != newestAgent.ID {
		t.Errorf("GetLast returned wrong agent: got %s (started %v), want %s (started %v)",
			latest.ID, latest.StartedAt, newestAgent.ID, newestAgent.StartedAt)
	}

	// Also create some older agents to verify sorting
	olderAgent := &AgentState{
		ID:        GenerateID(),
		PID:       88888,
		Prompt:    "test-older-agent",
		StartedAt: now.Add(-1 * time.Hour), // 1 hour ago
		Status:    "running",
	}
	if err := mgr.Register(olderAgent); err != nil {
		t.Fatalf("Register older agent failed: %v", err)
	}

	// GetLast should still return the newest agent
	latest, err = mgr.GetLast()
	if err != nil {
		t.Fatalf("GetLast failed after adding older agent: %v", err)
	}

	if latest.ID != newestAgent.ID {
		t.Errorf("GetLast returned older agent: got %s, want %s", latest.ID, newestAgent.ID)
	}

	// Cleanup
	_ = mgr.Remove(newestAgent.ID)
	_ = mgr.Remove(olderAgent.ID)
}

func TestGetLastNoAgents(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Get all agents and remove them to ensure a clean state for this test
	allAgents, _ := mgr.List(false)

	// If there are no agents, GetLast should return an error
	if len(allAgents) == 0 {
		_, err := mgr.GetLast()
		if err == nil {
			t.Error("GetLast should return error when no agents exist")
		}
	}
	// Note: This test is best-effort since other tests may leave agents in global state
}
