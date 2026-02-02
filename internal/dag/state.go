package dag

import (
	"sync"
	"time"
)

// TaskStatus represents the execution status of a task within a DAG iteration.
type TaskStatus string

const (
	// TaskPending indicates the task has not started yet
	TaskPending TaskStatus = "pending"

	// TaskRunning indicates the task is currently executing
	TaskRunning TaskStatus = "running"

	// TaskSucceeded indicates the task completed successfully
	TaskSucceeded TaskStatus = "succeeded"

	// TaskFailed indicates the task failed
	TaskFailed TaskStatus = "failed"

	// TaskSkipped indicates the task was skipped due to dependency conditions
	TaskSkipped TaskStatus = "skipped"
)

// TaskState tracks the execution state of a task within a single DAG iteration.
type TaskState struct {
	// Name is the task name
	Name string

	// Status is the current execution status
	Status TaskStatus

	// Error holds any error from task execution
	Error error

	// StartedAt is when the task started running
	StartedAt time.Time

	// CompletedAt is when the task finished (success, failure, or skipped)
	CompletedAt time.Time
}

// IsTerminal returns true if the task is in a terminal state (not pending or running).
func (ts *TaskState) IsTerminal() bool {
	return ts.Status == TaskSucceeded || ts.Status == TaskFailed || ts.Status == TaskSkipped
}

// Duration returns how long the task ran. Returns 0 if not started or still running.
func (ts *TaskState) Duration() time.Duration {
	if ts.StartedAt.IsZero() {
		return 0
	}
	if ts.CompletedAt.IsZero() {
		return time.Since(ts.StartedAt)
	}
	return ts.CompletedAt.Sub(ts.StartedAt)
}

// StateTracker manages task states for a DAG execution.
type StateTracker struct {
	mu     sync.RWMutex
	states map[string]*TaskState
}

// NewStateTracker creates a new state tracker initialized with all tasks in pending state.
func NewStateTracker(taskNames []string) *StateTracker {
	st := &StateTracker{
		states: make(map[string]*TaskState),
	}

	for _, name := range taskNames {
		st.states[name] = &TaskState{
			Name:   name,
			Status: TaskPending,
		}
	}

	return st
}

// Get returns a copy of the state for a task to avoid race conditions.
func (st *StateTracker) Get(name string) *TaskState {
	st.mu.RLock()
	defer st.mu.RUnlock()
	state := st.states[name]
	if state == nil {
		return nil
	}
	// Return a copy to avoid race conditions
	stateCopy := *state
	return &stateCopy
}

// GetAll returns a copy of all states.
func (st *StateTracker) GetAll() map[string]*TaskState {
	st.mu.RLock()
	defer st.mu.RUnlock()

	result := make(map[string]*TaskState, len(st.states))
	for name, state := range st.states {
		// Return a copy to avoid race conditions
		stateCopy := *state
		result[name] = &stateCopy
	}
	return result
}

// SetRunning marks a task as running.
func (st *StateTracker) SetRunning(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if state, ok := st.states[name]; ok {
		state.Status = TaskRunning
		state.StartedAt = time.Now()
	}
}

// SetSucceeded marks a task as succeeded.
func (st *StateTracker) SetSucceeded(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if state, ok := st.states[name]; ok {
		state.Status = TaskSucceeded
		state.CompletedAt = time.Now()
		state.Error = nil
	}
}

// SetFailed marks a task as failed with an error.
func (st *StateTracker) SetFailed(name string, err error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if state, ok := st.states[name]; ok {
		state.Status = TaskFailed
		state.CompletedAt = time.Now()
		state.Error = err
	}
}

// SetSkipped marks a task as skipped.
func (st *StateTracker) SetSkipped(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if state, ok := st.states[name]; ok {
		state.Status = TaskSkipped
		state.CompletedAt = time.Now()
	}
}

// Reset resets all task states to pending for a new iteration.
func (st *StateTracker) Reset() {
	st.mu.Lock()
	defer st.mu.Unlock()

	for name := range st.states {
		st.states[name] = &TaskState{
			Name:   name,
			Status: TaskPending,
		}
	}
}

// AllTerminal returns true if all tasks are in a terminal state.
func (st *StateTracker) AllTerminal() bool {
	st.mu.RLock()
	defer st.mu.RUnlock()

	for _, state := range st.states {
		if !state.IsTerminal() {
			return false
		}
	}
	return true
}

// CountByStatus returns counts of tasks by status.
func (st *StateTracker) CountByStatus() map[TaskStatus]int {
	st.mu.RLock()
	defer st.mu.RUnlock()

	counts := make(map[TaskStatus]int)
	for _, state := range st.states {
		counts[state.Status]++
	}
	return counts
}

// Summary returns a summary of the current state.
type Summary struct {
	Pending   int
	Running   int
	Succeeded int
	Failed    int
	Skipped   int
}

// GetSummary returns a summary of task states.
func (st *StateTracker) GetSummary() Summary {
	counts := st.CountByStatus()
	return Summary{
		Pending:   counts[TaskPending],
		Running:   counts[TaskRunning],
		Succeeded: counts[TaskSucceeded],
		Failed:    counts[TaskFailed],
		Skipped:   counts[TaskSkipped],
	}
}
