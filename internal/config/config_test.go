package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Model != "opus-4.5-thinking" {
		t.Errorf("expected default model 'opus-4.5-thinking', got '%s'", cfg.Model)
	}

	if cfg.Iterations != 20 {
		t.Errorf("expected default iterations 20, got %d", cfg.Iterations)
	}

	if cfg.Command.Executable != "agent" {
		t.Errorf("expected default executable 'agent', got '%s'", cfg.Command.Executable)
	}

	if len(cfg.Command.Args) == 0 {
		t.Error("expected non-empty default args")
	}
}

func TestExpandArgs(t *testing.T) {
	cmd := CommandConfig{
		Executable: "agent",
		Args: []string{
			"--model", "{model}",
			"--prompt", "{prompt}",
			"--other", "value",
		},
	}

	expanded := cmd.ExpandArgs("gpt-5", "Hello world")

	expected := []string{
		"--model", "gpt-5",
		"--prompt", "Hello world",
		"--other", "value",
	}

	if len(expanded) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(expanded))
	}

	for i, arg := range expanded {
		if arg != expected[i] {
			t.Errorf("arg[%d]: expected '%s', got '%s'", i, expected[i], arg)
		}
	}
}

func TestExpandArgsMultiplePlaceholders(t *testing.T) {
	cmd := CommandConfig{
		Args: []string{
			"model={model},prompt={prompt}",
		},
	}

	expanded := cmd.ExpandArgs("opus", "test prompt")

	if expanded[0] != "model=opus,prompt=test prompt" {
		t.Errorf("expected combined expansion, got '%s'", expanded[0])
	}
}

func TestLoadConfigFile(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()

	// Write a test config
	configContent := `
model = "test-model"
iterations = 50

[command]
executable = "custom-agent"
args = ["--custom", "{model}", "{prompt}"]
`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load into default config
	cfg := DefaultConfig()
	if err := loadConfigFile(configPath, cfg); err != nil {
		t.Fatalf("loadConfigFile failed: %v", err)
	}

	if cfg.Model != "test-model" {
		t.Errorf("expected model 'test-model', got '%s'", cfg.Model)
	}

	if cfg.Iterations != 50 {
		t.Errorf("expected iterations 50, got %d", cfg.Iterations)
	}

	if cfg.Command.Executable != "custom-agent" {
		t.Errorf("expected executable 'custom-agent', got '%s'", cfg.Command.Executable)
	}

	if len(cfg.Command.Args) != 3 {
		t.Errorf("expected 3 args, got %d", len(cfg.Command.Args))
	}
}

func TestLoadConfigFileMerge(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()

	// Write a partial config (only model)
	configContent := `model = "partial-model"`
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load into default config
	cfg := DefaultConfig()
	if err := loadConfigFile(configPath, cfg); err != nil {
		t.Fatalf("loadConfigFile failed: %v", err)
	}

	// Model should be overridden
	if cfg.Model != "partial-model" {
		t.Errorf("expected model 'partial-model', got '%s'", cfg.Model)
	}

	// Other values should remain defaults
	if cfg.Iterations != 20 {
		t.Errorf("expected iterations to remain 20, got %d", cfg.Iterations)
	}

	if cfg.Command.Executable != "agent" {
		t.Errorf("expected executable to remain 'agent', got '%s'", cfg.Command.Executable)
	}
}

func TestLoadWithProjectOverride(t *testing.T) {
	// Create temp dir and change to it
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}

	// Write project config
	projectConfig := `
model = "project-model"
iterations = 100
`
	if err := os.WriteFile(".swarm.toml", []byte(projectConfig), 0644); err != nil {
		t.Fatalf("failed to write project config: %v", err)
	}

	// Load config
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Model != "project-model" {
		t.Errorf("expected project model 'project-model', got '%s'", cfg.Model)
	}

	if cfg.Iterations != 100 {
		t.Errorf("expected project iterations 100, got %d", cfg.Iterations)
	}

	// Command should remain default (not specified in project config)
	if cfg.Command.Executable != "agent" {
		t.Errorf("expected default executable 'agent', got '%s'", cfg.Command.Executable)
	}
}

func TestToTOML(t *testing.T) {
	cfg := DefaultConfig()
	toml := cfg.ToTOML()

	// Check that key elements are present
	if !contains(toml, "model = \"opus-4.5-thinking\"") {
		t.Error("TOML output missing model")
	}

	if !contains(toml, "iterations = 20") {
		t.Error("TOML output missing iterations")
	}

	if !contains(toml, "[command]") {
		t.Error("TOML output missing [command] section")
	}

	if !contains(toml, "executable = \"agent\"") {
		t.Error("TOML output missing executable")
	}

	if !contains(toml, "args = [") {
		t.Error("TOML output missing args")
	}
}

func TestGlobalConfigPath(t *testing.T) {
	path, err := GlobalConfigPath()
	if err != nil {
		t.Fatalf("GlobalConfigPath failed: %v", err)
	}

	if path == "" {
		t.Error("GlobalConfigPath returned empty string")
	}

	// Should end with config.toml
	if filepath.Base(path) != "config.toml" {
		t.Errorf("expected path to end with config.toml, got '%s'", path)
	}

	// Should contain 'swarm' directory
	if filepath.Base(filepath.Dir(path)) != "swarm" {
		t.Errorf("expected path to contain swarm directory, got '%s'", path)
	}
}

func TestProjectConfigPath(t *testing.T) {
	path := ProjectConfigPath()
	if path != ".swarm.toml" {
		t.Errorf("expected '.swarm.toml', got '%s'", path)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
