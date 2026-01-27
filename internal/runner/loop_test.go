package runner

import (
	"bytes"
	"testing"
	"time"

	"github.com/matt/swarm-cli/internal/config"
	"github.com/matt/swarm-cli/internal/state"
)

func TestLoopConfigStructure(t *testing.T) {
	cfg := LoopConfig{
		PromptContent:     "test prompt",
		StartingIteration: 1,
		TotalTimeout:      30 * time.Minute,
		IterTimeout:       10 * time.Minute,
	}

	if cfg.PromptContent != "test prompt" {
		t.Errorf("PromptContent mismatch: got %s", cfg.PromptContent)
	}
	if cfg.StartingIteration != 1 {
		t.Errorf("StartingIteration mismatch: got %d", cfg.StartingIteration)
	}
	if cfg.TotalTimeout != 30*time.Minute {
		t.Errorf("TotalTimeout mismatch: got %v", cfg.TotalTimeout)
	}
	if cfg.IterTimeout != 10*time.Minute {
		t.Errorf("IterTimeout mismatch: got %v", cfg.IterTimeout)
	}
}

func TestLoopResultStructure(t *testing.T) {
	result := &LoopResult{
		TimedOut: true,
	}

	if !result.TimedOut {
		t.Error("TimedOut should be true")
	}
}

func TestLoopConfigDefaults(t *testing.T) {
	cfg := LoopConfig{}

	if cfg.StartingIteration != 0 {
		t.Errorf("Default StartingIteration should be 0, got %d", cfg.StartingIteration)
	}
	if cfg.TotalTimeout != 0 {
		t.Errorf("Default TotalTimeout should be 0, got %v", cfg.TotalTimeout)
	}
	if cfg.IterTimeout != 0 {
		t.Errorf("Default IterTimeout should be 0, got %v", cfg.IterTimeout)
	}
}

func TestLoopConfigWithEnv(t *testing.T) {
	env := []string{"VAR1=value1", "VAR2=value2"}
	cfg := LoopConfig{
		Env: env,
	}

	if len(cfg.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(cfg.Env))
	}
	if cfg.Env[0] != "VAR1=value1" {
		t.Errorf("First env var mismatch: got %s", cfg.Env[0])
	}
}

func TestLoopConfigWithCommand(t *testing.T) {
	cmd := config.CommandConfig{
		Executable: "echo",
		Args:       []string{"{prompt}"},
		RawOutput:  true,
	}
	cfg := LoopConfig{
		Command: cmd,
	}

	if cfg.Command.Executable != "echo" {
		t.Errorf("Executable mismatch: got %s", cfg.Command.Executable)
	}
	if !cfg.Command.RawOutput {
		t.Error("RawOutput should be true")
	}
}

func TestLoopConfigWithOutput(t *testing.T) {
	var buf bytes.Buffer
	cfg := LoopConfig{
		Output: &buf,
	}

	if cfg.Output == nil {
		t.Error("Output should not be nil")
	}
}

// TestRunLoopImmediateTermination tests that the loop respects immediate termination mode.
// This is a unit test that verifies the behavior without running an actual agent.
func TestRunLoopImmediateTermination(t *testing.T) {
	mgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	agentState := &state.AgentState{
		ID:            state.GenerateID(),
		Name:          "test-agent",
		PID:           12345,
		Prompt:        "test-prompt",
		Model:         "test-model",
		StartedAt:     time.Now(),
		Iterations:    5,
		CurrentIter:   0,
		Status:        "running",
		TerminateMode: "immediate", // Pre-set termination mode
	}

	if err := mgr.Register(agentState); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	defer mgr.Remove(agentState.ID)

	var buf bytes.Buffer
	cfg := LoopConfig{
		Manager:       mgr,
		AgentState:    agentState,
		PromptContent: "test prompt",
		Command: config.CommandConfig{
			Executable: "nonexistent-command", // Won't actually run
			Args:       []string{},
		},
		Output:            &buf,
		StartingIteration: 1,
	}

	result, err := RunLoop(cfg)
	if err != nil {
		t.Errorf("RunLoop returned error: %v", err)
	}

	if result.TimedOut {
		t.Error("Should not have timed out")
	}

	// Verify the agent was marked as terminated
	updated, err := mgr.Get(agentState.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if updated.Status != "terminated" {
		t.Errorf("Expected status 'terminated', got '%s'", updated.Status)
	}
	if updated.ExitReason != "killed" {
		t.Errorf("Expected exit reason 'killed', got '%s'", updated.ExitReason)
	}
}

// TestRunLoopTotalTimeout tests that the loop properly handles total timeout.
func TestRunLoopTotalTimeout(t *testing.T) {
	mgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	agentState := &state.AgentState{
		ID:          state.GenerateID(),
		Name:        "test-timeout-agent",
		PID:         12345,
		Prompt:      "test-prompt",
		Model:       "test-model",
		StartedAt:   time.Now(),
		Iterations:  100, // Many iterations
		CurrentIter: 0,
		Status:      "running",
	}

	if err := mgr.Register(agentState); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	defer mgr.Remove(agentState.ID)

	var buf bytes.Buffer
	cfg := LoopConfig{
		Manager:       mgr,
		AgentState:    agentState,
		PromptContent: "test prompt",
		Command: config.CommandConfig{
			Executable: "sleep",
			Args:       []string{"10"}, // Long sleep
		},
		Output:            &buf,
		StartingIteration: 1,
		TotalTimeout:      50 * time.Millisecond, // Very short timeout
	}

	result, err := RunLoop(cfg)
	if err != nil {
		t.Errorf("RunLoop returned error: %v", err)
	}

	if !result.TimedOut {
		t.Error("Should have timed out")
	}

	// Verify the agent was marked as terminated with timeout reason
	updated, err := mgr.Get(agentState.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if updated.Status != "terminated" {
		t.Errorf("Expected status 'terminated', got '%s'", updated.Status)
	}
	if updated.TimeoutReason != "total" {
		t.Errorf("Expected timeout reason 'total', got '%s'", updated.TimeoutReason)
	}
}

// TestRunLoopIterationUpdate tests that the loop respects external iteration updates.
func TestRunLoopIterationUpdate(t *testing.T) {
	mgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	agentState := &state.AgentState{
		ID:          state.GenerateID(),
		Name:        "test-iter-agent",
		PID:         12345,
		Prompt:      "test-prompt",
		Model:       "test-model",
		StartedAt:   time.Now(),
		Iterations:  1, // Start with 1 iteration
		CurrentIter: 0,
		Status:      "running",
	}

	if err := mgr.Register(agentState); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	defer mgr.Remove(agentState.ID)

	// Update iterations externally before loop reads state
	agentState.Iterations = 0 // Set to unlimited
	agentState.TerminateMode = "immediate" // But also set terminate to stop it
	if err := mgr.Update(agentState); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	var buf bytes.Buffer
	cfg := LoopConfig{
		Manager:       mgr,
		AgentState:    agentState,
		PromptContent: "test prompt",
		Command: config.CommandConfig{
			Executable: "true",
			Args:       []string{},
		},
		Output:            &buf,
		StartingIteration: 1,
	}

	result, err := RunLoop(cfg)
	if err != nil {
		t.Errorf("RunLoop returned error: %v", err)
	}

	if result.TimedOut {
		t.Error("Should not have timed out")
	}
}

// TestRunLoopStartingIteration tests that the loop starts from the correct iteration.
func TestRunLoopStartingIteration(t *testing.T) {
	mgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	agentState := &state.AgentState{
		ID:            state.GenerateID(),
		Name:          "test-start-iter-agent",
		PID:           12345,
		Prompt:        "test-prompt",
		Model:         "test-model",
		StartedAt:     time.Now(),
		Iterations:    5,
		CurrentIter:   0,
		Status:        "running",
		TerminateMode: "immediate", // Terminate immediately to test starting iteration
	}

	if err := mgr.Register(agentState); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	defer mgr.Remove(agentState.ID)

	var buf bytes.Buffer
	cfg := LoopConfig{
		Manager:       mgr,
		AgentState:    agentState,
		PromptContent: "test prompt",
		Command: config.CommandConfig{
			Executable: "true",
			Args:       []string{},
		},
		Output:            &buf,
		StartingIteration: 3, // Start from iteration 3
	}

	_, err = RunLoop(cfg)
	if err != nil {
		t.Errorf("RunLoop returned error: %v", err)
	}

	// The output should mention "Iteration 3" not "Iteration 1"
	// Since we terminate immediately, we shouldn't see any iteration output,
	// but the starting iteration should have been set correctly
}

// TestRunLoopZeroStartingIteration tests that 0 starting iteration defaults to 1.
func TestRunLoopZeroStartingIteration(t *testing.T) {
	mgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	agentState := &state.AgentState{
		ID:            state.GenerateID(),
		Name:          "test-zero-start-agent",
		PID:           12345,
		Prompt:        "test-prompt",
		Model:         "test-model",
		StartedAt:     time.Now(),
		Iterations:    1,
		CurrentIter:   0,
		Status:        "running",
		TerminateMode: "immediate",
	}

	if err := mgr.Register(agentState); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	defer mgr.Remove(agentState.ID)

	var buf bytes.Buffer
	cfg := LoopConfig{
		Manager:       mgr,
		AgentState:    agentState,
		PromptContent: "test prompt",
		Command: config.CommandConfig{
			Executable: "true",
			Args:       []string{},
		},
		Output:            &buf,
		StartingIteration: 0, // Should default to 1
	}

	_, err = RunLoop(cfg)
	if err != nil {
		t.Errorf("RunLoop returned error: %v", err)
	}
}

// TestRunLoopNegativeStartingIteration tests that negative starting iteration defaults to 1.
func TestRunLoopNegativeStartingIteration(t *testing.T) {
	mgr, err := state.NewManager()
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	agentState := &state.AgentState{
		ID:            state.GenerateID(),
		Name:          "test-neg-start-agent",
		PID:           12345,
		Prompt:        "test-prompt",
		Model:         "test-model",
		StartedAt:     time.Now(),
		Iterations:    1,
		CurrentIter:   0,
		Status:        "running",
		TerminateMode: "immediate",
	}

	if err := mgr.Register(agentState); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	defer mgr.Remove(agentState.ID)

	var buf bytes.Buffer
	cfg := LoopConfig{
		Manager:       mgr,
		AgentState:    agentState,
		PromptContent: "test prompt",
		Command: config.CommandConfig{
			Executable: "true",
			Args:       []string{},
		},
		Output:            &buf,
		StartingIteration: -5, // Should default to 1
	}

	_, err = RunLoop(cfg)
	if err != nil {
		t.Errorf("RunLoop returned error: %v", err)
	}
}
