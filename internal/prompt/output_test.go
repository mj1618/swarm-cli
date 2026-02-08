package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessOutputDirectives_NoDirectives(t *testing.T) {
	content := "Hello world, no directives here."
	result, err := ProcessOutputDirectives(content, "/tmp/fake")
	if err != nil {
		t.Fatal(err)
	}
	if result != content {
		t.Errorf("expected unchanged content, got %q", result)
	}
}

func TestProcessOutputDirectives_PresentOutput(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "planner.txt"), []byte("the plan output"), 0644); err != nil {
		t.Fatal(err)
	}

	content := "Before\n{{output:planner}}\nAfter"
	result, err := ProcessOutputDirectives(content, dir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, `--- Output from task "planner" ---`) {
		t.Errorf("expected output header, got:\n%s", result)
	}
	if !strings.Contains(result, "the plan output") {
		t.Errorf("expected output content, got:\n%s", result)
	}
	if !strings.Contains(result, `--- End output from task "planner" ---`) {
		t.Errorf("expected output footer, got:\n%s", result)
	}
	if !strings.Contains(result, "Before\n") {
		t.Errorf("expected 'Before' preserved, got:\n%s", result)
	}
	if !strings.Contains(result, "\nAfter") {
		t.Errorf("expected 'After' preserved, got:\n%s", result)
	}
}

func TestProcessOutputDirectives_MissingOutput(t *testing.T) {
	dir := t.TempDir()

	content := "{{output:nonexistent}}"
	result, err := ProcessOutputDirectives(content, dir)
	if err != nil {
		t.Fatal(err)
	}

	expected := `(No output available from task "nonexistent")`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessOutputDirectives_EmptyOutputDir(t *testing.T) {
	content := "{{output:planner}}"
	result, err := ProcessOutputDirectives(content, "")
	if err != nil {
		t.Fatal(err)
	}

	expected := `(No output available from task "planner" — not running in a pipeline)`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestProcessOutputDirectives_MultipleDirectives(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "planner.txt"), []byte("plan"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "evaluator.txt"), []byte("evaluation"), 0644); err != nil {
		t.Fatal(err)
	}

	content := "Plan:\n{{output:planner}}\n\nEval:\n{{output:evaluator}}"
	result, err := ProcessOutputDirectives(content, dir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "plan") {
		t.Errorf("expected planner output, got:\n%s", result)
	}
	if !strings.Contains(result, "evaluation") {
		t.Errorf("expected evaluator output, got:\n%s", result)
	}
	if strings.Count(result, "--- Output from task") != 2 {
		t.Errorf("expected 2 output headers, got:\n%s", result)
	}
}

func TestProcessOutputDirectives_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "empty.txt"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	content := "{{output:empty}}"
	result, err := ProcessOutputDirectives(content, dir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, `--- Output from task "empty" ---`) {
		t.Errorf("expected output header for empty file, got:\n%s", result)
	}
}

func TestProcessOutputDirectives_WhitespaceInName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "my-task.txt"), []byte("output"), 0644); err != nil {
		t.Fatal(err)
	}

	content := "{{output: my-task }}"
	result, err := ProcessOutputDirectives(content, dir)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "output") {
		t.Errorf("expected task output with trimmed name, got:\n%s", result)
	}
}
