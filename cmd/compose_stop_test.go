package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matt/swarm-cli/internal/compose"
)

func TestComposeStopCommandFlags(t *testing.T) {
	cmd := composeStopCmd

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

	noWaitFlag := cmd.Flags().Lookup("no-wait")
	if noWaitFlag == nil {
		t.Error("expected 'no-wait' flag to exist")
	} else {
		if noWaitFlag.DefValue != "false" {
			t.Errorf("no-wait flag default = %q, want %q", noWaitFlag.DefValue, "false")
		}
	}

	timeoutFlag := cmd.Flags().Lookup("timeout")
	if timeoutFlag == nil {
		t.Error("expected 'timeout' flag to exist")
	} else {
		if timeoutFlag.DefValue != "300" {
			t.Errorf("timeout flag default = %q, want %q", timeoutFlag.DefValue, "300")
		}
	}
}

func TestComposeStopCommandUsage(t *testing.T) {
	cmd := composeStopCmd

	if cmd.Use != "compose-stop [task...]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "compose-stop [task...]")
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

func TestComposeStopComposeFileIntegration(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "compose-stop-integration-test")
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

	// Load and validate - this tests the compose file loading logic used by compose-stop
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

	// Test filtering (used when specifying specific tasks to stop)
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

func TestComposeStopEffectiveNamesMatching(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "compose-stop-matching-test")
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

	// Build effective names map (same logic as compose-stop command)
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

func TestComposeStopTaskFiltering(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "compose-stop-filter-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a compose file with multiple tasks
	composeContent := `version: "1"
tasks:
  web:
    prompt-string: "Web server"
  api:
    prompt-string: "API server"
  db:
    prompt-string: "Database worker"
`
	composePath := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	cf, err := compose.Load(composePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test filtering to specific tasks
	filtered, err := cf.GetTasks([]string{"web", "api"})
	if err != nil {
		t.Fatalf("GetTasks() error = %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("GetTasks() returned %d tasks, want 2", len(filtered))
	}
	if _, ok := filtered["web"]; !ok {
		t.Error("GetTasks() should contain web")
	}
	if _, ok := filtered["api"]; !ok {
		t.Error("GetTasks() should contain api")
	}
	if _, ok := filtered["db"]; ok {
		t.Error("GetTasks() should not contain db")
	}

	// Test filtering with nonexistent task
	_, err = cf.GetTasks([]string{"nonexistent"})
	if err == nil {
		t.Error("GetTasks() should return error for nonexistent task")
	}
}
