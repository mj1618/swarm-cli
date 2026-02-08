package dag

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mj1618/swarm-cli/internal/compose"
	"github.com/mj1618/swarm-cli/internal/config"
)

// testConfig returns a config that uses /bin/echo as the command backend.
// This allows testing pipeline mechanics without needing real AI agents.
func testConfig() *config.Config {
	return &config.Config{
		Backend: "test",
		Model:   "test-model",
		Command: config.CommandConfig{
			Executable: "/bin/echo",
			Args:       []string{"task-output"},
			RawOutput:  true,
		},
	}
}

func TestExecutor_RunPipeline_SequentialTasks(t *testing.T) {
	// Test that tasks with linear dependencies run in sequence: a → b → c
	tasks := map[string]compose.Task{
		"a": {PromptString: "step-a"},
		"b": {PromptString: "step-b", DependsOn: []compose.Dependency{{Task: "a"}}},
		"c": {PromptString: "step-c", DependsOn: []compose.Dependency{{Task: "b"}}},
	}

	pipeline := compose.Pipeline{
		Iterations: 1,
		Tasks:      []string{"a", "b", "c"},
	}

	var buf bytes.Buffer
	executor := NewExecutor(ExecutorConfig{
		AppConfig:  testConfig(),
		PromptsDir: t.TempDir(),
		WorkingDir: t.TempDir(),
		Output:     &buf,
	})

	err := executor.RunPipeline(pipeline, tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify all tasks ran
	if !strings.Contains(output, "3 succeeded") {
		t.Errorf("expected 3 succeeded tasks, output:\n%s", output)
	}
	if strings.Contains(output, "failed") && !strings.Contains(output, "0 failed") {
		t.Errorf("expected no failures, output:\n%s", output)
	}

	// Verify ordering: a's "Starting" appears before b's, and b's before c's
	aStart := strings.Index(output, "a | Starting")
	bStart := strings.Index(output, "b | Starting")
	cStart := strings.Index(output, "c | Starting")

	if aStart < 0 || bStart < 0 || cStart < 0 {
		t.Fatalf("expected all tasks to appear in output, got:\n%s", output)
	}
	if aStart >= bStart {
		t.Errorf("expected task 'a' to start before 'b', output:\n%s", output)
	}
	if bStart >= cStart {
		t.Errorf("expected task 'b' to start before 'c', output:\n%s", output)
	}
}

func TestExecutor_RunPipeline_MultipleIterations(t *testing.T) {
	// Test that a pipeline runs the specified number of iterations
	tasks := map[string]compose.Task{
		"step1": {PromptString: "do-step-1"},
		"step2": {PromptString: "do-step-2", DependsOn: []compose.Dependency{{Task: "step1"}}},
	}

	pipeline := compose.Pipeline{
		Iterations: 3,
		Tasks:      []string{"step1", "step2"},
	}

	var buf bytes.Buffer
	executor := NewExecutor(ExecutorConfig{
		AppConfig:  testConfig(),
		PromptsDir: t.TempDir(),
		WorkingDir: t.TempDir(),
		Output:     &buf,
	})

	err := executor.RunPipeline(pipeline, tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify all 3 iterations ran
	if !strings.Contains(output, "Iteration 1/3") {
		t.Errorf("expected iteration 1/3, output:\n%s", output)
	}
	if !strings.Contains(output, "Iteration 2/3") {
		t.Errorf("expected iteration 2/3, output:\n%s", output)
	}
	if !strings.Contains(output, "Iteration 3/3") {
		t.Errorf("expected iteration 3/3, output:\n%s", output)
	}

	// Verify completion message
	if !strings.Contains(output, "Pipeline completed successfully (3 iterations)") {
		t.Errorf("expected pipeline completion message, output:\n%s", output)
	}

	// Verify each iteration had 2 succeeded tasks
	count := strings.Count(output, "2 succeeded")
	if count != 3 {
		t.Errorf("expected '2 succeeded' to appear 3 times (once per iteration), got %d, output:\n%s", count, output)
	}
}

func TestExecutor_RunPipeline_ParallelTasks(t *testing.T) {
	// Test that independent tasks can run in parallel within a DAG level
	tasks := map[string]compose.Task{
		"root": {PromptString: "root-task"},
		"a":    {PromptString: "parallel-a", DependsOn: []compose.Dependency{{Task: "root"}}},
		"b":    {PromptString: "parallel-b", DependsOn: []compose.Dependency{{Task: "root"}}},
		"c":    {PromptString: "parallel-c", DependsOn: []compose.Dependency{{Task: "root"}}},
		"final": {PromptString: "final-task", DependsOn: []compose.Dependency{
			{Task: "a"}, {Task: "b"}, {Task: "c"},
		}},
	}

	pipeline := compose.Pipeline{
		Iterations: 1,
		Tasks:      []string{"root", "a", "b", "c", "final"},
	}

	var buf bytes.Buffer
	executor := NewExecutor(ExecutorConfig{
		AppConfig:  testConfig(),
		PromptsDir: t.TempDir(),
		WorkingDir: t.TempDir(),
		Output:     &buf,
	})

	err := executor.RunPipeline(pipeline, tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify all 5 tasks succeeded
	if !strings.Contains(output, "5 succeeded") {
		t.Errorf("expected 5 succeeded tasks, output:\n%s", output)
	}

	// Verify root ran before final
	rootIdx := strings.Index(output, "root")
	finalIdx := strings.Index(output, "final | Starting")
	if rootIdx < 0 || finalIdx < 0 {
		t.Fatalf("expected root and final in output, got:\n%s", output)
	}
	if rootIdx >= finalIdx {
		t.Errorf("expected root to start before final, output:\n%s", output)
	}
}

func TestExecutor_RunPipeline_ConditionalSkip(t *testing.T) {
	// Test that tasks with unsatisfiable conditions get skipped.
	// Use a command that fails (exit 1) so the "failure" condition task runs
	// and the "success" condition task gets skipped.
	failConfig := &config.Config{
		Backend: "test",
		Model:   "test-model",
		Command: config.CommandConfig{
			Executable: "/bin/sh",
			Args:       []string{"-c", "exit 1"},
			RawOutput:  true,
		},
	}

	tasks := map[string]compose.Task{
		"failing": {PromptString: "this-will-fail"},
		"on_success": {PromptString: "run-on-success", DependsOn: []compose.Dependency{
			{Task: "failing", Condition: compose.ConditionSuccess},
		}},
		"on_failure": {PromptString: "run-on-failure", DependsOn: []compose.Dependency{
			{Task: "failing", Condition: compose.ConditionFailure},
		}},
	}

	pipeline := compose.Pipeline{
		Iterations: 1,
		Tasks:      []string{"failing", "on_success", "on_failure"},
	}

	var buf bytes.Buffer
	executor := NewExecutor(ExecutorConfig{
		AppConfig:  failConfig,
		PromptsDir: t.TempDir(),
		WorkingDir: t.TempDir(),
		Output:     &buf,
	})

	err := executor.RunPipeline(pipeline, tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// failing task should fail, on_success should be skipped, on_failure should run
	// Note: on_failure also runs the same failing command, so it also fails.
	// The key verification is that on_success was skipped and on_failure was attempted.
	if !strings.Contains(output, "1 skipped") {
		t.Errorf("expected 1 skipped task (on_success), output:\n%s", output)
	}
	// on_failure should have been started (not skipped)
	if !strings.Contains(output, "on_failure | Starting") {
		t.Errorf("expected on_failure to start, output:\n%s", output)
	}
	// on_success should have been skipped
	if !strings.Contains(output, "on_success | Skipped") {
		t.Errorf("expected on_success to be skipped, output:\n%s", output)
	}
}

func TestExecutor_RunPipeline_DefaultIterations(t *testing.T) {
	// Test that a pipeline with 0 iterations defaults to 1
	tasks := map[string]compose.Task{
		"only": {PromptString: "only-task"},
	}

	pipeline := compose.Pipeline{
		Iterations: 0, // should default to 1
	}

	var buf bytes.Buffer
	executor := NewExecutor(ExecutorConfig{
		AppConfig:  testConfig(),
		PromptsDir: t.TempDir(),
		WorkingDir: t.TempDir(),
		Output:     &buf,
	})

	err := executor.RunPipeline(pipeline, tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "1 iteration(s)") {
		t.Errorf("expected 1 iteration, output:\n%s", output)
	}
	if !strings.Contains(output, "Pipeline completed successfully (1 iterations)") {
		t.Errorf("expected pipeline completion, output:\n%s", output)
	}
}

func TestExecutor_RunPipeline_CycleDetection(t *testing.T) {
	tasks := map[string]compose.Task{
		"a": {PromptString: "a", DependsOn: []compose.Dependency{{Task: "b"}}},
		"b": {PromptString: "b", DependsOn: []compose.Dependency{{Task: "a"}}},
	}

	pipeline := compose.Pipeline{
		Iterations: 1,
		Tasks:      []string{"a", "b"},
	}

	var buf bytes.Buffer
	executor := NewExecutor(ExecutorConfig{
		AppConfig:  testConfig(),
		PromptsDir: t.TempDir(),
		WorkingDir: t.TempDir(),
		Output:     &buf,
	})

	err := executor.RunPipeline(pipeline, tasks)
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected cycle error, got: %v", err)
	}
}
