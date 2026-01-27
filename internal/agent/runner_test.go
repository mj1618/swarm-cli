package agent

import (
	"bytes"
	"testing"

	"github.com/matt/swarm-cli/internal/config"
)

// CommandConfig is an alias for config.CommandConfig
type CommandConfig = config.CommandConfig

func TestConfig(t *testing.T) {
	cfg := Config{
		Model:  "test-model",
		Prompt: "test prompt content",
	}

	if cfg.Model != "test-model" {
		t.Errorf("Model mismatch: got %s, want test-model", cfg.Model)
	}
	if cfg.Prompt != "test prompt content" {
		t.Errorf("Prompt mismatch: got %s", cfg.Prompt)
	}
}

func TestDefaultModelFromConfig(t *testing.T) {
	// DefaultModel is now in the config package
	// This test verifies the runner accepts model from config
	cfg := Config{
		Model:  "opus-4.5-thinking",
		Prompt: "test",
	}
	runner := NewRunner(cfg)
	if runner.config.Model != "opus-4.5-thinking" {
		t.Errorf("Expected model 'opus-4.5-thinking', got '%s'", runner.config.Model)
	}
}

func TestNewRunner(t *testing.T) {
	cfg := Config{
		Model:  "test-model",
		Prompt: "test prompt",
	}

	runner := NewRunner(cfg)
	if runner == nil {
		t.Fatal("NewRunner returned nil")
	}

	if runner.config.Model != cfg.Model {
		t.Errorf("Runner config.Model mismatch: got %s, want %s", runner.config.Model, cfg.Model)
	}
	if runner.config.Prompt != cfg.Prompt {
		t.Errorf("Runner config.Prompt mismatch")
	}
}

func TestRunnerPIDBeforeRun(t *testing.T) {
	runner := NewRunner(Config{Model: "test", Prompt: "test"})

	// Before running, PID should be 0
	if runner.PID() != 0 {
		t.Errorf("PID before run should be 0, got %d", runner.PID())
	}
}

func TestRunnerKillBeforeRun(t *testing.T) {
	runner := NewRunner(Config{Model: "test", Prompt: "test"})

	// Kill before run should not error (no-op)
	err := runner.Kill()
	if err != nil {
		t.Errorf("Kill before run should not error, got %v", err)
	}
}

func TestConfigEmptyValues(t *testing.T) {
	cfg := Config{}

	if cfg.Model != "" {
		t.Errorf("Empty Config should have empty Model")
	}
	if cfg.Prompt != "" {
		t.Errorf("Empty Config should have empty Prompt")
	}
}

func TestRunnerWithMockCommand(t *testing.T) {
	// Test that NewRunner properly stores the config
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "basic config",
			config: Config{
				Model:  "test-model",
				Prompt: "test prompt",
			},
		},
		{
			name: "empty model",
			config: Config{
				Model:  "",
				Prompt: "prompt only",
			},
		},
		{
			name: "multiline prompt",
			config: Config{
				Model:  "model-name",
				Prompt: "line1\nline2\nline3",
			},
		},
		{
			name: "unicode prompt",
			config: Config{
				Model:  "test",
				Prompt: "测试提示 🎉",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewRunner(tt.config)
			if runner.config.Model != tt.config.Model {
				t.Errorf("Model mismatch: got %s, want %s", runner.config.Model, tt.config.Model)
			}
			if runner.config.Prompt != tt.config.Prompt {
				t.Errorf("Prompt mismatch")
			}
		})
	}
}

// TestRunWithNonExistentCommand tests that Run fails gracefully when the agent command doesn't exist
func TestRunWithNonExistentCommand(t *testing.T) {
	cfg := Config{
		Model:  "test-model",
		Prompt: "test prompt",
		Command: CommandConfig{
			Executable: "nonexistent-command-that-does-not-exist",
			Args:       []string{"{prompt}"},
		},
	}

	runner := NewRunner(cfg)
	var buf bytes.Buffer

	// This should fail because the command doesn't exist
	err := runner.Run(&buf)

	// We expect an error (command not found)
	if err == nil {
		t.Error("Expected error for non-existent command")
	}
}

func TestRunnerStructure(t *testing.T) {
	runner := &Runner{}

	// cmd should be nil initially
	if runner.cmd != nil {
		t.Error("New runner should have nil cmd")
	}

	// Config should be zero value
	if runner.config.Model != "" || runner.config.Prompt != "" {
		t.Error("New runner should have empty config")
	}
}

func TestMultipleRunners(t *testing.T) {
	// Test that multiple runners can be created independently
	cfg1 := Config{Model: "model1", Prompt: "prompt1"}
	cfg2 := Config{Model: "model2", Prompt: "prompt2"}

	runner1 := NewRunner(cfg1)
	runner2 := NewRunner(cfg2)

	if runner1.config.Model == runner2.config.Model {
		t.Error("Runners should have independent configs")
	}

	// Modifying one shouldn't affect the other
	runner1.config.Model = "modified"
	if runner2.config.Model == "modified" {
		t.Error("Modifying runner1 affected runner2")
	}
}
