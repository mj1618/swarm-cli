package dag

import (
	"errors"
	"testing"
	"time"
)

func TestNewStateTracker(t *testing.T) {
	tasks := []string{"a", "b", "c"}
	tracker := NewStateTracker(tasks)

	for _, task := range tasks {
		state := tracker.Get(task)
		if state == nil {
			t.Errorf("expected state for task %s", task)
			continue
		}
		if state.Status != TaskPending {
			t.Errorf("expected pending status for %s, got %s", task, state.Status)
		}
	}
}

func TestStateTracker_Transitions(t *testing.T) {
	tracker := NewStateTracker([]string{"task1"})

	// Initially pending
	state := tracker.Get("task1")
	if state.Status != TaskPending {
		t.Errorf("expected pending, got %s", state.Status)
	}

	// Set to running
	tracker.SetRunning("task1")
	state = tracker.Get("task1")
	if state.Status != TaskRunning {
		t.Errorf("expected running, got %s", state.Status)
	}
	if state.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}

	// Set to succeeded
	tracker.SetSucceeded("task1")
	state = tracker.Get("task1")
	if state.Status != TaskSucceeded {
		t.Errorf("expected succeeded, got %s", state.Status)
	}
	if state.CompletedAt.IsZero() {
		t.Error("expected CompletedAt to be set")
	}
}

func TestStateTracker_SetFailed(t *testing.T) {
	tracker := NewStateTracker([]string{"task1"})

	testErr := errors.New("test error")
	tracker.SetFailed("task1", testErr)

	state := tracker.Get("task1")
	if state.Status != TaskFailed {
		t.Errorf("expected failed, got %s", state.Status)
	}
	if state.Error != testErr {
		t.Errorf("expected error %v, got %v", testErr, state.Error)
	}
}

func TestStateTracker_SetSkipped(t *testing.T) {
	tracker := NewStateTracker([]string{"task1"})

	tracker.SetSkipped("task1")

	state := tracker.Get("task1")
	if state.Status != TaskSkipped {
		t.Errorf("expected skipped, got %s", state.Status)
	}
}

func TestStateTracker_Reset(t *testing.T) {
	tracker := NewStateTracker([]string{"task1", "task2"})

	// Set various states
	tracker.SetSucceeded("task1")
	tracker.SetFailed("task2", errors.New("error"))

	// Reset
	tracker.Reset()

	// All should be pending again
	for _, task := range []string{"task1", "task2"} {
		state := tracker.Get(task)
		if state.Status != TaskPending {
			t.Errorf("expected pending after reset for %s, got %s", task, state.Status)
		}
	}
}

func TestStateTracker_AllTerminal(t *testing.T) {
	tracker := NewStateTracker([]string{"a", "b", "c"})

	// Not all terminal initially
	if tracker.AllTerminal() {
		t.Error("expected not all terminal initially")
	}

	// Set some to terminal
	tracker.SetSucceeded("a")
	tracker.SetFailed("b", nil)
	if tracker.AllTerminal() {
		t.Error("expected not all terminal with one pending")
	}

	// Set all to terminal
	tracker.SetSkipped("c")
	if !tracker.AllTerminal() {
		t.Error("expected all terminal")
	}
}

func TestStateTracker_GetSummary(t *testing.T) {
	tracker := NewStateTracker([]string{"a", "b", "c", "d", "e"})

	tracker.SetRunning("a")
	tracker.SetSucceeded("b")
	tracker.SetFailed("c", nil)
	tracker.SetSkipped("d")
	// e remains pending

	summary := tracker.GetSummary()

	if summary.Pending != 1 {
		t.Errorf("expected 1 pending, got %d", summary.Pending)
	}
	if summary.Running != 1 {
		t.Errorf("expected 1 running, got %d", summary.Running)
	}
	if summary.Succeeded != 1 {
		t.Errorf("expected 1 succeeded, got %d", summary.Succeeded)
	}
	if summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", summary.Failed)
	}
	if summary.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", summary.Skipped)
	}
}

func TestStateTracker_GetAll_ReturnsCopy(t *testing.T) {
	tracker := NewStateTracker([]string{"task1"})
	tracker.SetRunning("task1")

	// Get all states
	states := tracker.GetAll()

	// Modify the returned state
	states["task1"].Status = TaskSucceeded

	// Original should be unchanged
	original := tracker.Get("task1")
	if original.Status != TaskRunning {
		t.Error("modifying returned map should not affect tracker")
	}
}

func TestTaskState_IsTerminal(t *testing.T) {
	testCases := []struct {
		status   TaskStatus
		terminal bool
	}{
		{TaskPending, false},
		{TaskRunning, false},
		{TaskSucceeded, true},
		{TaskFailed, true},
		{TaskSkipped, true},
	}

	for _, tc := range testCases {
		state := &TaskState{Status: tc.status}
		if state.IsTerminal() != tc.terminal {
			t.Errorf("expected IsTerminal()=%v for status %s", tc.terminal, tc.status)
		}
	}
}

func TestTaskState_Duration(t *testing.T) {
	state := &TaskState{}

	// Not started - should return 0
	if state.Duration() != 0 {
		t.Error("expected 0 duration when not started")
	}

	// Started but not completed - should return time since start
	state.StartedAt = time.Now().Add(-1 * time.Second)
	d := state.Duration()
	if d < time.Second || d > 2*time.Second {
		t.Errorf("expected ~1s duration while running, got %v", d)
	}

	// Completed - should return fixed duration
	state.CompletedAt = state.StartedAt.Add(500 * time.Millisecond)
	d = state.Duration()
	if d != 500*time.Millisecond {
		t.Errorf("expected 500ms duration when completed, got %v", d)
	}
}
