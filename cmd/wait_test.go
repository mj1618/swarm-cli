package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
)

func TestWaitCommandFlagsExist(t *testing.T) {
	// Test that all expected flags exist
	flags := waitCmd.Flags()

	if flags.Lookup("timeout") == nil {
		t.Error("expected --timeout flag to exist")
	}
	if flags.Lookup("interval") == nil {
		t.Error("expected --interval flag to exist")
	}
	if flags.Lookup("any") == nil {
		t.Error("expected --any flag to exist")
	}
	if flags.Lookup("verbose") == nil {
		t.Error("expected --verbose flag to exist")
	}
}

func TestWaitCommandRequiresArgs(t *testing.T) {
	// Verify the command requires at least one argument
	if waitCmd.Args == nil {
		t.Error("expected Args validation to be set")
		return
	}

	// Test with no args - should fail
	err := waitCmd.Args(waitCmd, []string{})
	if err == nil {
		t.Error("expected error when no args provided")
	}

	// Test with one arg - should pass
	err = waitCmd.Args(waitCmd, []string{"agent1"})
	if err != nil {
		t.Errorf("unexpected error with one arg: %v", err)
	}

	// Test with multiple args - should pass
	err = waitCmd.Args(waitCmd, []string{"agent1", "agent2", "agent3"})
	if err != nil {
		t.Errorf("unexpected error with multiple args: %v", err)
	}
}

func TestWaitAgentTermination(t *testing.T) {
	// Create a temporary directory for the test state
	tmpDir, err := os.MkdirTemp("", "swarm-wait-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the .swarm directory
	swarmDir := filepath.Join(tmpDir, ".swarm")
	if err := os.MkdirAll(swarmDir, 0755); err != nil {
		t.Fatalf("failed to create swarm dir: %v", err)
	}

	// Temporarily override HOME for the test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create state manager
	mgr, err := state.NewManagerWithScope(scope.ScopeGlobal, "")
	if err != nil {
		t.Fatalf("failed to create state manager: %v", err)
	}

	// Register a running agent
	agent := &state.AgentState{
		ID:          "test-wait-agent",
		Name:        "test-wait",
		PID:         99999, // Non-existent PID
		Prompt:      "test",
		Model:       "test-model",
		StartedAt:   time.Now(),
		Iterations:  1,
		CurrentIter: 1,
		Status:      "running",
		WorkingDir:  tmpDir,
	}
	if err := mgr.Register(agent); err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// Verify agent is running
	retrieved, err := mgr.Get("test-wait-agent")
	if err != nil {
		t.Fatalf("failed to get agent: %v", err)
	}
	if retrieved.Status != "running" {
		t.Errorf("expected status 'running', got %s", retrieved.Status)
	}

	// Terminate the agent
	agent.Status = "terminated"
	if err := mgr.Update(agent); err != nil {
		t.Fatalf("failed to update agent: %v", err)
	}

	// Verify agent is terminated
	retrieved, err = mgr.Get("test-wait-agent")
	if err != nil {
		t.Fatalf("failed to get agent: %v", err)
	}
	if retrieved.Status != "terminated" {
		t.Errorf("expected status 'terminated', got %s", retrieved.Status)
	}
}

func TestWaitMultipleAgents(t *testing.T) {
	// Create a temporary directory for the test state
	tmpDir, err := os.MkdirTemp("", "swarm-wait-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the .swarm directory
	swarmDir := filepath.Join(tmpDir, ".swarm")
	if err := os.MkdirAll(swarmDir, 0755); err != nil {
		t.Fatalf("failed to create swarm dir: %v", err)
	}

	// Temporarily override HOME for the test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create state manager
	mgr, err := state.NewManagerWithScope(scope.ScopeGlobal, "")
	if err != nil {
		t.Fatalf("failed to create state manager: %v", err)
	}

	// Register multiple running agents
	agents := []*state.AgentState{
		{
			ID:          "agent-1",
			Name:        "first",
			PID:         99991,
			Prompt:      "test1",
			Model:       "test-model",
			StartedAt:   time.Now(),
			Iterations:  1,
			CurrentIter: 1,
			Status:      "running",
			WorkingDir:  tmpDir,
		},
		{
			ID:          "agent-2",
			Name:        "second",
			PID:         99992,
			Prompt:      "test2",
			Model:       "test-model",
			StartedAt:   time.Now(),
			Iterations:  1,
			CurrentIter: 1,
			Status:      "running",
			WorkingDir:  tmpDir,
		},
		{
			ID:          "agent-3",
			Name:        "third",
			PID:         99993,
			Prompt:      "test3",
			Model:       "test-model",
			StartedAt:   time.Now(),
			Iterations:  1,
			CurrentIter: 1,
			Status:      "running",
			WorkingDir:  tmpDir,
		},
	}

	for _, a := range agents {
		if err := mgr.Register(a); err != nil {
			t.Fatalf("failed to register agent %s: %v", a.ID, err)
		}
	}

	// Check all are running
	for _, a := range agents {
		retrieved, err := mgr.Get(a.ID)
		if err != nil {
			t.Fatalf("failed to get agent %s: %v", a.ID, err)
		}
		if retrieved.Status != "running" {
			t.Errorf("agent %s: expected status 'running', got %s", a.ID, retrieved.Status)
		}
	}

	// Simulate --any behavior: terminate just one
	agents[0].Status = "terminated"
	if err := mgr.Update(agents[0]); err != nil {
		t.Fatalf("failed to update agent: %v", err)
	}

	// Verify first is terminated, others still running
	a1, _ := mgr.Get("agent-1")
	a2, _ := mgr.Get("agent-2")
	a3, _ := mgr.Get("agent-3")

	if a1.Status != "terminated" {
		t.Errorf("expected agent-1 to be terminated")
	}
	if a2.Status != "running" {
		t.Errorf("expected agent-2 to still be running")
	}
	if a3.Status != "running" {
		t.Errorf("expected agent-3 to still be running")
	}

	// Terminate all for full wait behavior test
	agents[1].Status = "terminated"
	agents[2].Status = "terminated"
	if err := mgr.Update(agents[1]); err != nil {
		t.Fatalf("failed to update agent: %v", err)
	}
	if err := mgr.Update(agents[2]); err != nil {
		t.Fatalf("failed to update agent: %v", err)
	}

	// Verify all terminated
	for _, a := range agents {
		retrieved, err := mgr.Get(a.ID)
		if err != nil {
			t.Fatalf("failed to get agent %s: %v", a.ID, err)
		}
		if retrieved.Status != "terminated" {
			t.Errorf("agent %s: expected status 'terminated', got %s", a.ID, retrieved.Status)
		}
	}
}

func TestWaitResolvesByName(t *testing.T) {
	// Create a temporary directory for the test state
	tmpDir, err := os.MkdirTemp("", "swarm-wait-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create the .swarm directory
	swarmDir := filepath.Join(tmpDir, ".swarm")
	if err := os.MkdirAll(swarmDir, 0755); err != nil {
		t.Fatalf("failed to create swarm dir: %v", err)
	}

	// Temporarily override HOME for the test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create state manager
	mgr, err := state.NewManagerWithScope(scope.ScopeGlobal, "")
	if err != nil {
		t.Fatalf("failed to create state manager: %v", err)
	}

	// Register an agent with a name
	agent := &state.AgentState{
		ID:          "abc12345",
		Name:        "my-named-agent",
		PID:         99999,
		Prompt:      "test",
		Model:       "test-model",
		StartedAt:   time.Now(),
		Iterations:  1,
		CurrentIter: 1,
		Status:      "running",
		WorkingDir:  tmpDir,
	}
	if err := mgr.Register(agent); err != nil {
		t.Fatalf("failed to register agent: %v", err)
	}

	// Test resolution by ID
	retrieved, err := mgr.GetByNameOrID("abc12345")
	if err != nil {
		t.Fatalf("failed to resolve by ID: %v", err)
	}
	if retrieved.Name != "my-named-agent" {
		t.Errorf("expected name 'my-named-agent', got %s", retrieved.Name)
	}

	// Test resolution by name
	retrieved, err = mgr.GetByNameOrID("my-named-agent")
	if err != nil {
		t.Fatalf("failed to resolve by name: %v", err)
	}
	if retrieved.ID != "abc12345" {
		t.Errorf("expected ID 'abc12345', got %s", retrieved.ID)
	}
}
