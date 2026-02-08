package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mj1618/swarm-cli/internal/config"
	"github.com/mj1618/swarm-cli/internal/logparser"
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

// TestRunnerClaudeCodeParsedStream tests end-to-end streaming output parsing
// using a mock process that emits claude-code format JSONL.
func TestRunnerClaudeCodeParsedStream(t *testing.T) {
	// Build a shell script that emits claude-code JSONL lines to stdout
	jsonLines := []string{
		`{"type":"system","subtype":"init","model":"opus","cwd":"/tmp","session_id":"test123"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Working on it."}]},"usage":{"input_tokens":500,"output_tokens":30}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"echo hello"}}]},"usage":{"input_tokens":800,"output_tokens":20}}`,
		`{"type":"tool_result","content":"hello"}`,
		`{"type":"result","subtype":"success","result":"done","duration_ms":2000,"usage":{"input_tokens":200,"output_tokens":10}}`,
	}

	// Use printf with \n to emit each line (echo adds trailing newline per line)
	var script string
	for _, line := range jsonLines {
		script += `printf '%s\n' '` + line + `'; `
	}

	cfg := Config{
		Model:  "opus",
		Prompt: "test",
		Command: CommandConfig{
			Executable: "sh",
			Args:       []string{"-c", script},
			RawOutput:  false, // Use the parser (non-raw mode)
		},
	}

	runner := NewRunner(cfg)

	var usageCallbackCalled bool
	var finalStats = make(map[string]int64)
	runner.SetUsageCallback(func(stats logparser.UsageStats) {
		usageCallbackCalled = true
		finalStats["input"] = stats.InputTokens
		finalStats["output"] = stats.OutputTokens
	})

	var buf bytes.Buffer
	err := runner.Run(&buf)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := buf.String()

	// Verify parsed output contains expected elements
	if !strings.Contains(output, "System init") {
		t.Errorf("Output should contain 'System init', got: %q", output)
	}
	if !strings.Contains(output, "Working on it") {
		t.Errorf("Output should contain assistant text, got: %q", output)
	}
	if !strings.Contains(output, "Shell: echo hello") {
		t.Errorf("Output should contain tool use summary, got: %q", output)
	}
	if !strings.Contains(output, "Result") {
		t.Errorf("Output should contain result, got: %q", output)
	}

	// Verify usage stats were tracked
	if !usageCallbackCalled {
		t.Error("Usage callback was never called")
	}
	if finalStats["input"] == 0 {
		t.Error("Expected non-zero input tokens")
	}
	if finalStats["output"] == 0 {
		t.Error("Expected non-zero output tokens")
	}
}

// TestRunnerClaudeCodeRawStream tests the raw output mode (direct streaming with usage extraction).
func TestRunnerClaudeCodeRawStream(t *testing.T) {
	jsonLines := []string{
		`{"type":"system","subtype":"init","model":"opus"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hello"}]},"usage":{"input_tokens":100,"output_tokens":20}}`,
		`{"type":"result","subtype":"success","result":"done","usage":{"input_tokens":50,"output_tokens":10}}`,
	}

	var script string
	for _, line := range jsonLines {
		script += `printf '%s\n' '` + line + `'; `
	}

	cfg := Config{
		Model:  "opus",
		Prompt: "test",
		Command: CommandConfig{
			Executable: "sh",
			Args:       []string{"-c", script},
			RawOutput:  true, // Raw mode - streams directly
		},
	}

	runner := NewRunner(cfg)

	var lastInputTokens int64
	runner.SetUsageCallback(func(stats logparser.UsageStats) {
		lastInputTokens = stats.InputTokens
	})

	var buf bytes.Buffer
	err := runner.Run(&buf)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := buf.String()

	// In raw mode, output should contain the raw JSON lines (not parsed/pretty-printed)
	if !strings.Contains(output, `"type":"system"`) {
		t.Errorf("Raw output should contain raw JSON, got: %q", output)
	}

	// Usage stats should still be extracted even in raw mode
	if lastInputTokens != 150 {
		t.Errorf("Expected 150 input tokens extracted in raw mode, got %d", lastInputTokens)
	}
}

// TestRunnerClaudeCodeStreamingOrder verifies that output appears in the correct order.
func TestRunnerClaudeCodeStreamingOrder(t *testing.T) {
	jsonLines := []string{
		`{"type":"system","subtype":"init","model":"opus"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Step one."}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"Read","input":{"file_path":"/src/main.go"}}]}}`,
		`{"type":"tool_result","content":"package main"}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Step two."}]}}`,
		`{"type":"result","subtype":"success","result":"complete"}`,
	}

	var script string
	for _, line := range jsonLines {
		script += `printf '%s\n' '` + line + `'; `
	}

	cfg := Config{
		Model:  "opus",
		Prompt: "test",
		Command: CommandConfig{
			Executable: "sh",
			Args:       []string{"-c", script},
			RawOutput:  false,
		},
	}

	runner := NewRunner(cfg)
	var buf bytes.Buffer
	err := runner.Run(&buf)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := buf.String()

	// Verify ordering: "Step one" should appear before "Read file" which should appear before "Step two"
	stepOneIdx := strings.Index(output, "Step one")
	readIdx := strings.Index(output, "Read file")
	stepTwoIdx := strings.Index(output, "Step two")
	resultIdx := strings.Index(output, "complete")

	if stepOneIdx == -1 || readIdx == -1 || stepTwoIdx == -1 || resultIdx == -1 {
		t.Fatalf("Missing expected content in output: %q", output)
	}

	if stepOneIdx >= readIdx {
		t.Errorf("'Step one' (%d) should appear before 'Read file' (%d)", stepOneIdx, readIdx)
	}
	if readIdx >= stepTwoIdx {
		t.Errorf("'Read file' (%d) should appear before 'Step two' (%d)", readIdx, stepTwoIdx)
	}
	if stepTwoIdx >= resultIdx {
		t.Errorf("'Step two' (%d) should appear before result (%d)", stepTwoIdx, resultIdx)
	}
}
