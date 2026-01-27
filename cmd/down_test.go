package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matt/swarm-cli/internal/compose"
)

func TestDownCommandFlags(t *testing.T) {
	cmd := downCmd

	fileFlag := cmd.Flags().Lookup("file")
	if fileFlag == nil {
		t.Error("expected 'file' flag to exist")
	} else {
		if fileFlag.Shorthand != "f" {
			t.Errorf("file flag shorthand = %q, want %q", fileFlag.Shorthand, "f")
		}
		if fileFlag.DefValue != "./swarm/swarm.yaml" {
			t.Errorf("file flag default = %q, want %q", fileFlag.DefValue, "./swarm/swarm.yaml")
		}
	}

	gracefulFlag := cmd.Flags().Lookup("graceful")
	if gracefulFlag == nil {
		t.Error("expected 'graceful' flag to exist")
	} else {
		if gracefulFlag.Shorthand != "G" {
			t.Errorf("graceful flag shorthand = %q, want %q", gracefulFlag.Shorthand, "G")
		}
		if gracefulFlag.DefValue != "false" {
			t.Errorf("graceful flag default = %q, want %q", gracefulFlag.DefValue, "false")
		}
	}
}

func TestDownCommandUsage(t *testing.T) {
	cmd := downCmd

	if cmd.Use != "down [task...]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "down [task...]")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if cmd.Example == "" {
		t.Error("Example should not be empty")
	}
}

func TestDownComposeFileIntegration(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "down-integration-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid compose file
	composeContent := `version: "1"
tasks:
  task1:
    prompt-string: "Do task 1"
    iterations: 2
  task2:
    prompt-string: "Do task 2"
    model: sonnet-4.5
    name: custom-name
`
	composePath := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	// Load and validate - this tests the compose file loading logic used by down
	cf, err := compose.Load(composePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cf.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Test effective names (used for agent matching)
	task1 := cf.Tasks["task1"]
	if task1.EffectiveName("task1") != "task1" {
		t.Errorf("task1.EffectiveName() = %q, want %q", task1.EffectiveName("task1"), "task1")
	}

	task2 := cf.Tasks["task2"]
	if task2.EffectiveName("task2") != "custom-name" {
		t.Errorf("task2.EffectiveName() = %q, want %q", task2.EffectiveName("task2"), "custom-name")
	}

	// Test filtering (used when specifying specific tasks to down)
	filtered, err := cf.GetTasks([]string{"task1"})
	if err != nil {
		t.Fatalf("GetTasks() error = %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("GetTasks() returned %d tasks, want 1", len(filtered))
	}
	if _, ok := filtered["task1"]; !ok {
		t.Error("GetTasks() should contain task1")
	}
}

func TestDownEffectiveNamesMatching(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "down-matching-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a compose file with various name configurations
	composeContent := `version: "1"
tasks:
  frontend:
    prompt-string: "Frontend task"
  backend:
    prompt-string: "Backend task"
    name: api-server
  worker:
    prompt-string: "Worker task"
    name: background-worker
`
	composePath := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	cf, err := compose.Load(composePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tasks, err := cf.GetTasks(nil)
	if err != nil {
		t.Fatalf("GetTasks() error = %v", err)
	}

	// Build effective names map (same logic as down command)
	effectiveNames := make(map[string]bool)
	for taskName, task := range tasks {
		effectiveNames[task.EffectiveName(taskName)] = true
	}

	// Verify expected effective names
	expectedNames := []string{"frontend", "api-server", "background-worker"}
	for _, name := range expectedNames {
		if !effectiveNames[name] {
			t.Errorf("expected effective name %q to be present", name)
		}
	}

	// Verify task names without custom names use the key
	unexpectedNames := []string{"backend", "worker"}
	for _, name := range unexpectedNames {
		if effectiveNames[name] {
			t.Errorf("did not expect effective name %q (should use custom name)", name)
		}
	}
}
