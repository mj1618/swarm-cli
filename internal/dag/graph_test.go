package dag

import (
	"testing"

	"github.com/mj1618/swarm-cli/internal/compose"
)

func TestNewGraph(t *testing.T) {
	tasks := map[string]compose.Task{
		"a": {Prompt: "a"},
		"b": {Prompt: "b", DependsOn: []compose.Dependency{{Task: "a"}}},
		"c": {Prompt: "c", DependsOn: []compose.Dependency{{Task: "b"}}},
	}

	graph := NewGraph(tasks, []string{"a", "b", "c"})

	nodes := graph.GetNodes()
	if len(nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(nodes))
	}

	roots := graph.GetRootTasks()
	if len(roots) != 1 || roots[0] != "a" {
		t.Errorf("expected root task 'a', got %v", roots)
	}
}

func TestGraphValidate_NoCycle(t *testing.T) {
	tasks := map[string]compose.Task{
		"a": {Prompt: "a"},
		"b": {Prompt: "b", DependsOn: []compose.Dependency{{Task: "a"}}},
		"c": {Prompt: "c", DependsOn: []compose.Dependency{{Task: "b"}}},
	}

	graph := NewGraph(tasks, []string{"a", "b", "c"})
	if err := graph.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestGraphValidate_DirectCycle(t *testing.T) {
	tasks := map[string]compose.Task{
		"a": {Prompt: "a", DependsOn: []compose.Dependency{{Task: "b"}}},
		"b": {Prompt: "b", DependsOn: []compose.Dependency{{Task: "a"}}},
	}

	graph := NewGraph(tasks, []string{"a", "b"})
	err := graph.Validate()
	if err == nil {
		t.Error("expected cycle error, got nil")
	}
}

func TestGraphValidate_IndirectCycle(t *testing.T) {
	tasks := map[string]compose.Task{
		"a": {Prompt: "a", DependsOn: []compose.Dependency{{Task: "c"}}},
		"b": {Prompt: "b", DependsOn: []compose.Dependency{{Task: "a"}}},
		"c": {Prompt: "c", DependsOn: []compose.Dependency{{Task: "b"}}},
	}

	graph := NewGraph(tasks, []string{"a", "b", "c"})
	err := graph.Validate()
	if err == nil {
		t.Error("expected cycle error, got nil")
	}
}

func TestTopologicalSort(t *testing.T) {
	tasks := map[string]compose.Task{
		"a": {Prompt: "a"},
		"b": {Prompt: "b", DependsOn: []compose.Dependency{{Task: "a"}}},
		"c": {Prompt: "c", DependsOn: []compose.Dependency{{Task: "a"}}},
		"d": {Prompt: "d", DependsOn: []compose.Dependency{{Task: "b"}, {Task: "c"}}},
	}

	graph := NewGraph(tasks, []string{"a", "b", "c", "d"})
	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that dependencies come before dependents
	positions := make(map[string]int)
	for i, name := range sorted {
		positions[name] = i
	}

	// a must come before b, c, d
	if positions["a"] >= positions["b"] {
		t.Error("a should come before b")
	}
	if positions["a"] >= positions["c"] {
		t.Error("a should come before c")
	}
	if positions["a"] >= positions["d"] {
		t.Error("a should come before d")
	}

	// b and c must come before d
	if positions["b"] >= positions["d"] {
		t.Error("b should come before d")
	}
	if positions["c"] >= positions["d"] {
		t.Error("c should come before d")
	}
}

func TestFindReadyTasks_AllPending(t *testing.T) {
	tasks := map[string]compose.Task{
		"a": {Prompt: "a"},
		"b": {Prompt: "b", DependsOn: []compose.Dependency{{Task: "a"}}},
	}

	graph := NewGraph(tasks, []string{"a", "b"})
	states := map[string]*TaskState{
		"a": {Name: "a", Status: TaskPending},
		"b": {Name: "b", Status: TaskPending},
	}

	ready := graph.FindReadyTasks(states)
	if len(ready) != 1 || ready[0] != "a" {
		t.Errorf("expected only 'a' ready, got %v", ready)
	}
}

func TestFindReadyTasks_AfterDependencySuccess(t *testing.T) {
	tasks := map[string]compose.Task{
		"a": {Prompt: "a"},
		"b": {Prompt: "b", DependsOn: []compose.Dependency{{Task: "a"}}},
	}

	graph := NewGraph(tasks, []string{"a", "b"})
	states := map[string]*TaskState{
		"a": {Name: "a", Status: TaskSucceeded},
		"b": {Name: "b", Status: TaskPending},
	}

	ready := graph.FindReadyTasks(states)
	if len(ready) != 1 || ready[0] != "b" {
		t.Errorf("expected only 'b' ready, got %v", ready)
	}
}

func TestFindReadyTasks_ParallelTasks(t *testing.T) {
	tasks := map[string]compose.Task{
		"a":  {Prompt: "a"},
		"b1": {Prompt: "b1", DependsOn: []compose.Dependency{{Task: "a"}}},
		"b2": {Prompt: "b2", DependsOn: []compose.Dependency{{Task: "a"}}},
	}

	graph := NewGraph(tasks, []string{"a", "b1", "b2"})
	states := map[string]*TaskState{
		"a":  {Name: "a", Status: TaskSucceeded},
		"b1": {Name: "b1", Status: TaskPending},
		"b2": {Name: "b2", Status: TaskPending},
	}

	ready := graph.FindReadyTasks(states)
	if len(ready) != 2 {
		t.Errorf("expected 2 ready tasks, got %v", ready)
	}
}

func TestFindReadyTasks_ConditionFailure(t *testing.T) {
	tasks := map[string]compose.Task{
		"a": {Prompt: "a"},
		"b": {Prompt: "b", DependsOn: []compose.Dependency{
			{Task: "a", Condition: compose.ConditionFailure},
		}},
	}

	graph := NewGraph(tasks, []string{"a", "b"})

	// When a succeeds, b should NOT be ready (it needs a to fail)
	states := map[string]*TaskState{
		"a": {Name: "a", Status: TaskSucceeded},
		"b": {Name: "b", Status: TaskPending},
	}
	ready := graph.FindReadyTasks(states)
	if len(ready) != 0 {
		t.Errorf("expected no ready tasks when dependency succeeded, got %v", ready)
	}

	// When a fails, b should be ready
	states["a"].Status = TaskFailed
	ready = graph.FindReadyTasks(states)
	if len(ready) != 1 || ready[0] != "b" {
		t.Errorf("expected 'b' ready when dependency failed, got %v", ready)
	}
}

func TestFindReadyTasks_ConditionAny(t *testing.T) {
	tasks := map[string]compose.Task{
		"a": {Prompt: "a"},
		"b": {Prompt: "b", DependsOn: []compose.Dependency{
			{Task: "a", Condition: compose.ConditionAny},
		}},
	}

	graph := NewGraph(tasks, []string{"a", "b"})

	// b should be ready whether a succeeds or fails
	for _, status := range []TaskStatus{TaskSucceeded, TaskFailed} {
		states := map[string]*TaskState{
			"a": {Name: "a", Status: status},
			"b": {Name: "b", Status: TaskPending},
		}
		ready := graph.FindReadyTasks(states)
		if len(ready) != 1 || ready[0] != "b" {
			t.Errorf("expected 'b' ready with condition 'any' and status %s, got %v", status, ready)
		}
	}

	// b should NOT be ready when a is still running
	states := map[string]*TaskState{
		"a": {Name: "a", Status: TaskRunning},
		"b": {Name: "b", Status: TaskPending},
	}
	ready := graph.FindReadyTasks(states)
	if len(ready) != 0 {
		t.Errorf("expected no ready tasks while dependency running, got %v", ready)
	}
}

func TestShouldSkip_FailureConditionWithSuccess(t *testing.T) {
	tasks := map[string]compose.Task{
		"a": {Prompt: "a"},
		"b": {Prompt: "b", DependsOn: []compose.Dependency{
			{Task: "a", Condition: compose.ConditionFailure},
		}},
	}

	graph := NewGraph(tasks, []string{"a", "b"})

	// When a succeeds, b should be skipped (needs failure)
	states := map[string]*TaskState{
		"a": {Name: "a", Status: TaskSucceeded},
		"b": {Name: "b", Status: TaskPending},
	}

	if !graph.ShouldSkip("b", states) {
		t.Error("expected 'b' to be skipped when dependency succeeded (needs failure)")
	}
}

func TestShouldSkip_SuccessConditionWithFailure(t *testing.T) {
	tasks := map[string]compose.Task{
		"a": {Prompt: "a"},
		"b": {Prompt: "b", DependsOn: []compose.Dependency{
			{Task: "a", Condition: compose.ConditionSuccess},
		}},
	}

	graph := NewGraph(tasks, []string{"a", "b"})

	// When a fails, b should be skipped (needs success)
	states := map[string]*TaskState{
		"a": {Name: "a", Status: TaskFailed},
		"b": {Name: "b", Status: TaskPending},
	}

	if !graph.ShouldSkip("b", states) {
		t.Error("expected 'b' to be skipped when dependency failed (needs success)")
	}
}

func TestConditionalBranching(t *testing.T) {
	// Test the classic pattern: coder -> tester -> (fixer OR reviewer)
	tasks := map[string]compose.Task{
		"coder":    {Prompt: "coder"},
		"tester":   {Prompt: "tester", DependsOn: []compose.Dependency{{Task: "coder"}}},
		"fixer":    {Prompt: "fixer", DependsOn: []compose.Dependency{{Task: "tester", Condition: compose.ConditionFailure}}},
		"reviewer": {Prompt: "reviewer", DependsOn: []compose.Dependency{{Task: "tester", Condition: compose.ConditionSuccess}}},
	}

	graph := NewGraph(tasks, []string{"coder", "tester", "fixer", "reviewer"})

	// Scenario 1: tester succeeds -> reviewer runs, fixer skipped
	states := map[string]*TaskState{
		"coder":    {Status: TaskSucceeded},
		"tester":   {Status: TaskSucceeded},
		"fixer":    {Status: TaskPending},
		"reviewer": {Status: TaskPending},
	}

	ready := graph.FindReadyTasks(states)
	if len(ready) != 1 || ready[0] != "reviewer" {
		t.Errorf("expected reviewer ready after tester success, got %v", ready)
	}

	if !graph.ShouldSkip("fixer", states) {
		t.Error("expected fixer to be skipped when tester succeeded")
	}

	// Scenario 2: tester fails -> fixer runs, reviewer skipped
	states["tester"].Status = TaskFailed

	ready = graph.FindReadyTasks(states)
	if len(ready) != 1 || ready[0] != "fixer" {
		t.Errorf("expected fixer ready after tester failure, got %v", ready)
	}

	if !graph.ShouldSkip("reviewer", states) {
		t.Error("expected reviewer to be skipped when tester failed")
	}
}
