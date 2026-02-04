package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mj1618/swarm-cli/internal/compose"
)

// TestFrontendTaskCompose tests loading and validating a compose file with a frontend task
func TestFrontendTaskCompose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "frontend-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a compose file with a frontend task
	composeContent := `version: "1"
tasks:
  frontend:
    prompt-string: "Review the frontend code and check for any React best practices violations"
    model: sonnet-4.5
    iterations: 1
    name: frontend-reviewer
  
  backend:
    prompt-string: "Check the Go code for potential improvements"
    iterations: 1
  
  documentation:
    prompt-string: "Review README.md and ensure it's up to date"
    iterations: 1
`
	composePath := filepath.Join(tmpDir, "test-swarm.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	// Load and validate
	cf, err := compose.Load(composePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cf.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Verify we have 3 tasks
	if len(cf.Tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(cf.Tasks))
	}

	// Test frontend task specifically
	frontend, ok := cf.Tasks["frontend"]
	if !ok {
		t.Fatal("frontend task not found")
	}

	if frontend.PromptString != "Review the frontend code and check for any React best practices violations" {
		t.Errorf("frontend.PromptString = %q, want 'Review the frontend code...'", frontend.PromptString)
	}

	if frontend.Model != "sonnet-4.5" {
		t.Errorf("frontend.Model = %q, want 'sonnet-4.5'", frontend.Model)
	}

	if frontend.EffectiveIterations() != 1 {
		t.Errorf("frontend.EffectiveIterations() = %d, want 1", frontend.EffectiveIterations())
	}

	if frontend.EffectiveName("frontend") != "frontend-reviewer" {
		t.Errorf("frontend.EffectiveName() = %q, want 'frontend-reviewer'", frontend.EffectiveName("frontend"))
	}

	// Test GetTasks filtering - get only frontend task
	filtered, err := cf.GetTasks([]string{"frontend"})
	if err != nil {
		t.Fatalf("GetTasks([frontend]) error = %v", err)
	}

	if len(filtered) != 1 {
		t.Errorf("GetTasks([frontend]) returned %d tasks, want 1", len(filtered))
	}

	if _, ok := filtered["frontend"]; !ok {
		t.Error("GetTasks([frontend]) should contain frontend task")
	}

	// Test GetTasks with multiple tasks including frontend
	multiFiltered, err := cf.GetTasks([]string{"frontend", "backend"})
	if err != nil {
		t.Fatalf("GetTasks([frontend, backend]) error = %v", err)
	}

	if len(multiFiltered) != 2 {
		t.Errorf("GetTasks([frontend, backend]) returned %d tasks, want 2", len(multiFiltered))
	}

	// Test GetTasks with no filter returns all tasks
	allTasks, err := cf.GetTasks(nil)
	if err != nil {
		t.Fatalf("GetTasks(nil) error = %v", err)
	}

	if len(allTasks) != 3 {
		t.Errorf("GetTasks(nil) returned %d tasks, want 3", len(allTasks))
	}
}

// TestFrontendTaskPromptLoading tests loading the prompt for a frontend task
func TestFrontendTaskPromptLoading(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "frontend-prompt-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a prompts directory
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	// Test 1: Frontend task with prompt-string
	task1 := compose.Task{
		PromptString: "Test the frontend application",
		Model:        "sonnet-4.5",
		Iterations:   1,
	}

	content1, label1, err := loadTaskPrompt(task1, promptsDir)
	if err != nil {
		t.Fatalf("loadTaskPrompt(prompt-string) error = %v", err)
	}

	if content1 != "Test the frontend application" {
		t.Errorf("content = %q, want 'Test the frontend application'", content1)
	}

	if label1 != "<string>" {
		t.Errorf("label = %q, want '<string>'", label1)
	}

	// Test 2: Frontend task with named prompt
	frontendPromptContent := "# Frontend Testing\n\nTest all React components for proper rendering"
	frontendPromptPath := filepath.Join(promptsDir, "frontend-test.md")
	if err := os.WriteFile(frontendPromptPath, []byte(frontendPromptContent), 0644); err != nil {
		t.Fatalf("failed to write frontend prompt: %v", err)
	}

	task2 := compose.Task{
		Prompt:     "frontend-test",
		Model:      "sonnet-4.5",
		Iterations: 2,
	}

	content2, label2, err := loadTaskPrompt(task2, promptsDir)
	if err != nil {
		t.Fatalf("loadTaskPrompt(prompt) error = %v", err)
	}

	if content2 != frontendPromptContent {
		t.Errorf("content = %q, want frontend prompt content", content2)
	}

	if label2 != "frontend-test" {
		t.Errorf("label = %q, want 'frontend-test'", label2)
	}
}

// TestFrontendTaskValidation tests validation of frontend task configurations
func TestFrontendTaskValidation(t *testing.T) {
	tests := []struct {
		name    string
		task    compose.Task
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid frontend task with prompt-string",
			task: compose.Task{
				PromptString: "Test frontend",
				Model:        "sonnet-4.5",
				Iterations:   1,
			},
			wantErr: false,
		},
		{
			name: "valid frontend task with prompt",
			task: compose.Task{
				Prompt:     "frontend-task",
				Model:      "opus-4.5",
				Iterations: 5,
			},
			wantErr: false,
		},
		{
			name: "invalid frontend task - no prompt",
			task: compose.Task{
				Model:      "sonnet-4.5",
				Iterations: 1,
			},
			wantErr: true,
			errMsg:  "no prompt source",
		},
		{
			name: "invalid frontend task - negative iterations",
			task: compose.Task{
				PromptString: "Test frontend",
				Iterations:   -1,
			},
			wantErr: true,
			errMsg:  "iterations cannot be negative",
		},
		{
			name: "valid frontend task - zero iterations defaults to 1",
			task: compose.Task{
				PromptString: "Test frontend",
				Iterations:   0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate("frontend")
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() == "" {
					t.Errorf("expected error message to contain %q", tt.errMsg)
				}
			}

			// Test effective iterations
			if !tt.wantErr {
				expectedIter := tt.task.Iterations
				if expectedIter == 0 {
					expectedIter = 1
				}
				if tt.task.EffectiveIterations() != expectedIter {
					t.Errorf("EffectiveIterations() = %d, want %d", tt.task.EffectiveIterations(), expectedIter)
				}
			}
		})
	}
}

// TestMultipleTasksIncludingFrontend tests running multiple tasks including frontend
func TestMultipleTasksIncludingFrontend(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "multi-task-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a compose file with multiple tasks including frontend
	composeContent := `version: "1"
tasks:
  frontend:
    prompt-string: "Test React components"
    model: sonnet-4.5
    iterations: 2
    name: react-tester
  
  frontend-lint:
    prompt-string: "Run ESLint on frontend code"
    model: sonnet-4.5
    iterations: 1
  
  frontend-e2e:
    prompt-string: "Run end-to-end tests on the frontend"
    model: opus-4.5
    iterations: 3
    name: e2e-tester
`
	composePath := filepath.Join(tmpDir, "multi-frontend.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	// Load and validate
	cf, err := compose.Load(composePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cf.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Verify all tasks are frontend-related
	expectedTasks := []string{"frontend", "frontend-lint", "frontend-e2e"}
	if len(cf.Tasks) != len(expectedTasks) {
		t.Errorf("expected %d tasks, got %d", len(expectedTasks), len(cf.Tasks))
	}

	for _, taskName := range expectedTasks {
		if _, ok := cf.Tasks[taskName]; !ok {
			t.Errorf("task %q not found", taskName)
		}
	}

	// Test filtering to get only frontend-e2e
	filtered, err := cf.GetTasks([]string{"frontend-e2e"})
	if err != nil {
		t.Fatalf("GetTasks([frontend-e2e]) error = %v", err)
	}

	if len(filtered) != 1 {
		t.Errorf("GetTasks([frontend-e2e]) returned %d tasks, want 1", len(filtered))
	}

	e2eTask := filtered["frontend-e2e"]
	if e2eTask.Model != "opus-4.5" {
		t.Errorf("frontend-e2e.Model = %q, want 'opus-4.5'", e2eTask.Model)
	}

	if e2eTask.EffectiveName("frontend-e2e") != "e2e-tester" {
		t.Errorf("frontend-e2e.EffectiveName() = %q, want 'e2e-tester'", e2eTask.EffectiveName("frontend-e2e"))
	}

	// Test filtering multiple frontend tasks
	multiFiltered, err := cf.GetTasks([]string{"frontend", "frontend-lint"})
	if err != nil {
		t.Fatalf("GetTasks([frontend, frontend-lint]) error = %v", err)
	}

	if len(multiFiltered) != 2 {
		t.Errorf("GetTasks([frontend, frontend-lint]) returned %d tasks, want 2", len(multiFiltered))
	}
}
