package compose

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFrontendTaskE2E is an end-to-end test for loading and processing a frontend task
func TestFrontendTaskE2E(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "frontend-e2e-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a realistic compose file for a frontend project
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
	composePath := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	// Test 1: Load the compose file
	cf, err := Load(composePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cf.Version != "1" {
		t.Errorf("Version = %q, want '1'", cf.Version)
	}

	// Test 2: Validate the compose file
	if err := cf.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Test 3: Verify frontend task properties
	frontend, ok := cf.Tasks["frontend"]
	if !ok {
		t.Fatal("frontend task not found in compose file")
	}

	// Check all frontend task properties
	expectedPrompt := "Review the frontend code and check for any React best practices violations"
	if frontend.PromptString != expectedPrompt {
		t.Errorf("frontend.PromptString = %q, want %q", frontend.PromptString, expectedPrompt)
	}

	if frontend.Model != "sonnet-4.5" {
		t.Errorf("frontend.Model = %q, want 'sonnet-4.5'", frontend.Model)
	}

	if frontend.Iterations != 1 {
		t.Errorf("frontend.Iterations = %d, want 1", frontend.Iterations)
	}

	if frontend.Name != "frontend-reviewer" {
		t.Errorf("frontend.Name = %q, want 'frontend-reviewer'", frontend.Name)
	}

	// Test 4: Verify effective values
	if frontend.EffectiveIterations() != 1 {
		t.Errorf("frontend.EffectiveIterations() = %d, want 1", frontend.EffectiveIterations())
	}

	if frontend.EffectiveName("frontend") != "frontend-reviewer" {
		t.Errorf("frontend.EffectiveName('frontend') = %q, want 'frontend-reviewer'", frontend.EffectiveName("frontend"))
	}

	// Test 5: Verify task filtering works
	// Get only frontend task
	frontendOnly, err := cf.GetTasks([]string{"frontend"})
	if err != nil {
		t.Fatalf("GetTasks([frontend]) error = %v", err)
	}

	if len(frontendOnly) != 1 {
		t.Errorf("GetTasks([frontend]) returned %d tasks, want 1", len(frontendOnly))
	}

	// Get frontend and documentation
	subset, err := cf.GetTasks([]string{"frontend", "documentation"})
	if err != nil {
		t.Fatalf("GetTasks([frontend, documentation]) error = %v", err)
	}

	if len(subset) != 2 {
		t.Errorf("GetTasks([frontend, documentation]) returned %d tasks, want 2", len(subset))
	}

	// Get all tasks
	allTasks, err := cf.GetTasks(nil)
	if err != nil {
		t.Fatalf("GetTasks(nil) error = %v", err)
	}

	if len(allTasks) != 3 {
		t.Errorf("GetTasks(nil) returned %d tasks, want 3", len(allTasks))
	}

	// Test 6: Verify validation catches errors
	invalidCf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"bad-task": {
				Model:      "sonnet-4.5",
				Iterations: 1,
				// Missing prompt source
			},
		},
	}

	if err := invalidCf.Validate(); err == nil {
		t.Error("Validate() expected error for task without prompt source, got nil")
	}
}

// TestFrontendTaskWithPromptFile tests a frontend task that uses a prompt file
func TestFrontendTaskWithPromptFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "frontend-prompt-file-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a prompt file
	promptContent := `# Frontend Code Review

Please review the following aspects of the frontend code:

## React Best Practices
- Check for proper use of hooks
- Verify state management patterns
- Look for unnecessary re-renders

## Performance
- Check for expensive computations that should be memoized
- Verify proper use of React.memo
- Look for large bundle sizes

## Accessibility
- Check for proper ARIA labels
- Verify keyboard navigation
- Test with screen readers
`
	promptPath := filepath.Join(tmpDir, "frontend-review.md")
	if err := os.WriteFile(promptPath, []byte(promptContent), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	// Create compose file referencing the prompt file
	composeContent := `version: "1"
tasks:
  frontend-review:
    prompt-file: ` + promptPath + `
    model: opus-4.5
    iterations: 2
    name: detailed-frontend-reviewer
`
	composePath := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	// Load and validate
	cf, err := Load(composePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cf.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Check the task
	task := cf.Tasks["frontend-review"]
	if task.PromptFile != promptPath {
		t.Errorf("task.PromptFile = %q, want %q", task.PromptFile, promptPath)
	}

	if task.Model != "opus-4.5" {
		t.Errorf("task.Model = %q, want 'opus-4.5'", task.Model)
	}

	if task.Iterations != 2 {
		t.Errorf("task.Iterations = %d, want 2", task.Iterations)
	}

	if task.Name != "detailed-frontend-reviewer" {
		t.Errorf("task.Name = %q, want 'detailed-frontend-reviewer'", task.Name)
	}
}

// TestFrontendOnlyCompose tests a compose file with only frontend tasks
func TestFrontendOnlyCompose(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "frontend-only-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a compose file with only frontend tasks
	composeContent := `version: "1"
tasks:
  unit-tests:
    prompt-string: "Run all frontend unit tests and report failures"
    model: sonnet-4.5
    iterations: 1
  
  integration-tests:
    prompt-string: "Run integration tests for the frontend"
    model: sonnet-4.5
    iterations: 2
  
  e2e-tests:
    prompt-string: "Run end-to-end tests using Playwright"
    model: opus-4.5
    iterations: 3
  
  lint:
    prompt-string: "Run ESLint and fix auto-fixable issues"
    model: sonnet-4.5
    iterations: 1
  
  type-check:
    prompt-string: "Run TypeScript type checking and fix errors"
    model: opus-4.5
    iterations: 2
`
	composePath := filepath.Join(tmpDir, "frontend-tests.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	// Load and validate
	cf, err := Load(composePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cf.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Should have 5 frontend-related tasks
	expectedTasks := []string{"unit-tests", "integration-tests", "e2e-tests", "lint", "type-check"}
	if len(cf.Tasks) != len(expectedTasks) {
		t.Errorf("expected %d tasks, got %d", len(expectedTasks), len(cf.Tasks))
	}

	for _, taskName := range expectedTasks {
		if _, ok := cf.Tasks[taskName]; !ok {
			t.Errorf("task %q not found", taskName)
		}
	}

	// Verify we can run a subset of tests
	testTasks, err := cf.GetTasks([]string{"unit-tests", "lint"})
	if err != nil {
		t.Fatalf("GetTasks([unit-tests, lint]) error = %v", err)
	}

	if len(testTasks) != 2 {
		t.Errorf("GetTasks([unit-tests, lint]) returned %d tasks, want 2", len(testTasks))
	}

	// Verify e2e-tests uses the more capable model
	e2eTask := cf.Tasks["e2e-tests"]
	if e2eTask.Model != "opus-4.5" {
		t.Errorf("e2e-tests.Model = %q, want 'opus-4.5'", e2eTask.Model)
	}

	if e2eTask.EffectiveIterations() != 3 {
		t.Errorf("e2e-tests.EffectiveIterations() = %d, want 3", e2eTask.EffectiveIterations())
	}
}
