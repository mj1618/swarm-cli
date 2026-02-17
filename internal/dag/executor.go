package dag

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mj1618/swarm-cli/internal/agent"
	"github.com/mj1618/swarm-cli/internal/compose"
	"github.com/mj1618/swarm-cli/internal/config"
	"github.com/mj1618/swarm-cli/internal/logparser"
	"github.com/mj1618/swarm-cli/internal/output"
	"github.com/mj1618/swarm-cli/internal/prompt"
	"github.com/mj1618/swarm-cli/internal/state"
)

// ExecutorConfig holds the configuration for running a pipeline.
type ExecutorConfig struct {
	// AppConfig is the application configuration (model, command, etc.)
	AppConfig *config.Config

	// PromptsDir is the directory containing prompt files
	PromptsDir string

	// WorkingDir is the working directory for agent execution
	WorkingDir string

	// Output is the writer for pipeline output (defaults to os.Stdout)
	Output io.Writer

	// Verbose enables verbose output
	Verbose bool

	// StateManager is the state manager for persisting pipeline progress (optional)
	StateManager *state.Manager

	// TaskID is the agent state ID to update during execution (optional)
	TaskID string
}

// Executor runs pipelines with DAG-ordered task execution.
type Executor struct {
	cfg ExecutorConfig

	// Cumulative usage stats across all completed tasks (protected by mu)
	mu           sync.Mutex
	inputTokens  int64
	outputTokens int64
	totalCostUSD float64
	taskStats    map[string]logparser.UsageStats // running tasks' current stats
}

// NewExecutor creates a new pipeline executor.
func NewExecutor(cfg ExecutorConfig) *Executor {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	return &Executor{
		cfg:       cfg,
		taskStats: make(map[string]logparser.UsageStats),
	}
}

// RunPipeline runs a pipeline with the given tasks for the specified iterations.
func (e *Executor) RunPipeline(pipeline compose.Pipeline, tasks map[string]compose.Task) error {
	// Initialize cumulative stats from persisted state (so costs persist between iterations)
	if e.cfg.StateManager != nil && e.cfg.TaskID != "" {
		if agentState, err := e.cfg.StateManager.Get(e.cfg.TaskID); err == nil {
			e.inputTokens = agentState.InputTokens
			e.outputTokens = agentState.OutputTokens
			e.totalCostUSD = agentState.TotalCost
		}
	}

	// Get task names for this pipeline
	taskNames := pipeline.GetPipelineTasks(tasks)

	// Build and validate the DAG
	graph := NewGraph(tasks, taskNames)
	if err := graph.Validate(); err != nil {
		return fmt.Errorf("invalid DAG: %w", err)
	}

	iterations := pipeline.EffectiveIterations()
	fmt.Fprintf(e.cfg.Output, "Running pipeline with %d iteration(s) and %d task(s)\n", iterations, len(taskNames))

	terminated := false

	// Run each iteration
	for i := 1; i <= iterations; i++ {
		// Check for pause/terminate between iterations
		if e.checkPipelineControl() {
			terminated = true
			break
		}

		// Create a unique, time-sortable output directory per iteration
		runID := time.Now().Format("20060102-150405") + "-" + state.GenerateID()
		outputDir := filepath.Join(os.TempDir(), "swarm", "outputs", runID)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Update state with current iteration and check for iteration limit changes
		if e.cfg.StateManager != nil && e.cfg.TaskID != "" {
			if agentState, err := e.cfg.StateManager.Get(e.cfg.TaskID); err == nil {
				agentState.CurrentIter = i
				_ = e.cfg.StateManager.MergeUpdate(agentState)

				// Re-read state to pick up externally changed iteration limit
				if updated, err := e.cfg.StateManager.Get(e.cfg.TaskID); err == nil {
					if updated.Iterations != 0 && updated.Iterations != iterations {
						iterations = updated.Iterations
						if i > iterations {
							break
						}
					}
				}
			}
		}

		fmt.Fprintf(e.cfg.Output, "\n=== Pipeline Iteration %d/%d ===\n", i, iterations)

		dagTerminated, err := e.runDAG(graph, taskNames, i, iterations, outputDir)
		if err != nil {
			return fmt.Errorf("iteration %d failed: %w", i, err)
		}
		if dagTerminated {
			terminated = true
			break
		}

		fmt.Fprintf(e.cfg.Output, "--- Iteration %d complete ---\n", i)
	}

	// Mark pipeline as terminated on completion
	if e.cfg.StateManager != nil && e.cfg.TaskID != "" {
		if agentState, err := e.cfg.StateManager.Get(e.cfg.TaskID); err == nil {
			agentState.Status = "terminated"
			now := time.Now()
			agentState.TerminatedAt = &now
			if terminated {
				agentState.ExitReason = "killed"
			} else {
				agentState.ExitReason = "completed"
			}
			_ = e.cfg.StateManager.MergeUpdate(agentState)
		}
	}

	if terminated {
		fmt.Fprintf(e.cfg.Output, "\nPipeline terminated\n")
	} else {
		fmt.Fprintf(e.cfg.Output, "\nPipeline completed successfully (%d iterations)\n", iterations)
	}
	return nil
}

// checkPipelineControl checks for pause/terminate signals from state.
// If paused, it blocks until resumed or terminated.
// Returns true if the pipeline should be terminated.
func (e *Executor) checkPipelineControl() bool {
	if e.cfg.StateManager == nil || e.cfg.TaskID == "" {
		return false
	}

	agentState, err := e.cfg.StateManager.Get(e.cfg.TaskID)
	if err != nil {
		return false
	}

	// Check for termination
	if agentState.TerminateMode == "immediate" {
		fmt.Fprintf(e.cfg.Output, "\n[swarm] Received termination signal\n")
		return true
	}

	if !agentState.Paused {
		return false
	}

	// Enter pause state — set PausedAt to acknowledge
	fmt.Fprintf(e.cfg.Output, "\n[swarm] Pipeline paused, waiting for resume...\n")
	now := time.Now()
	agentState.PausedAt = &now
	_ = e.cfg.StateManager.MergeUpdate(agentState)

	// Sleep loop until resumed or terminated
	for {
		time.Sleep(1 * time.Second)
		agentState, err = e.cfg.StateManager.Get(e.cfg.TaskID)
		if err != nil {
			break
		}

		// Allow termination while paused
		if agentState.TerminateMode == "immediate" {
			fmt.Fprintf(e.cfg.Output, "\n[swarm] Received termination signal\n")
			return true
		}

		// Check for resume
		if !agentState.Paused {
			break
		}
	}

	// Resumed — clear PausedAt
	fmt.Fprintf(e.cfg.Output, "\n[swarm] Pipeline resumed\n")
	agentState, err = e.cfg.StateManager.Get(e.cfg.TaskID)
	if err == nil {
		agentState.PausedAt = nil
		_ = e.cfg.StateManager.MergeUpdate(agentState)
	}

	return false
}

// runDAG executes a single DAG iteration.
// Returns (terminated, error) where terminated is true if a terminate signal was received.
func (e *Executor) runDAG(graph *Graph, taskNames []string, iteration, totalIterations int, outputDir string) (bool, error) {
	// Initialize state tracker
	states := NewStateTracker(taskNames)

	// Create prefixed writers for parallel output
	writers := output.NewWriterGroup(e.cfg.Output, taskNames)

	for {
		// Check for pause/terminate before scheduling new tasks
		if e.checkPipelineControl() {
			return true, nil
		}

		// Get current states
		currentStates := states.GetAll()

		// Check for tasks that should be skipped
		e.skipBlockedTasks(graph, states, currentStates, writers)

		// Find tasks ready to run
		readyTasks := graph.FindReadyTasks(currentStates)

		if len(readyTasks) == 0 {
			// No more ready tasks - check if we're done
			if states.AllTerminal() {
				break
			}

			// If there are pending tasks but none ready, there might be a deadlock
			summary := states.GetSummary()
			if summary.Pending > 0 {
				return false, fmt.Errorf("deadlock: %d pending task(s) but none ready", summary.Pending)
			}
			break
		}

		// Execute ready tasks in parallel
		if err := e.executeTasks(graph, readyTasks, states, writers, iteration, totalIterations, outputDir); err != nil {
			// Log error but continue - individual task failures don't stop the DAG
			fmt.Fprintf(e.cfg.Output, "Warning: task execution error: %v\n", err)
		}
	}

	// Report final summary
	summary := states.GetSummary()
	fmt.Fprintf(e.cfg.Output, "Tasks: %d succeeded, %d failed, %d skipped\n",
		summary.Succeeded, summary.Failed, summary.Skipped)

	return false, nil
}

// skipBlockedTasks marks tasks as skipped if their dependency conditions can't be met.
func (e *Executor) skipBlockedTasks(graph *Graph, tracker *StateTracker, currentStates map[string]*TaskState, writers *output.WriterGroup) {
	for _, task := range graph.GetNodes() {
		state := currentStates[task]
		if state == nil || state.Status != TaskPending {
			continue
		}

		if graph.ShouldSkip(task, currentStates) {
			tracker.SetSkipped(task)
			writer := writers.Get(task)
			fmt.Fprintf(writer, "Skipped (dependency condition not met)\n")
			writer.Flush()
		}
	}
}

// executeTasks runs multiple tasks in parallel.
func (e *Executor) executeTasks(graph *Graph, taskNames []string, tracker *StateTracker, writers *output.WriterGroup, iteration, totalIterations int, outputDir string) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for i, taskName := range taskNames {
		task, ok := graph.GetTask(taskName)
		if !ok {
			continue
		}

		// Stagger parallel task starts by 5 seconds to avoid resource contention
		if i > 0 {
			time.Sleep(5 * time.Second)
		}

		writer := writers.Get(taskName)
		tracker.SetRunning(taskName)

		wg.Add(1)
		go func(name string, t compose.Task, out *output.PrefixedWriter) {
			defer wg.Done()
			defer out.Flush()

			// Acquire concurrency slot (blocks if limit reached)
			concurrencyLimit := t.EffectiveConcurrency()
			if concurrencyLimit > 0 {
				fmt.Fprintf(out, "Waiting for concurrency slot...\n")
			}
			AcquireTaskSlot(name, concurrencyLimit)
			defer ReleaseTaskSlot(name, concurrencyLimit)

			fmt.Fprintf(out, "Starting (iteration %d)\n", iteration)

			err := e.runTask(name, t, out, iteration, totalIterations, outputDir)
			if err != nil {
				tracker.SetFailed(name, err)
				fmt.Fprintf(out, "Failed: %v\n", err)
				mu.Lock()
				errors = append(errors, fmt.Errorf("%s: %w", name, err))
				mu.Unlock()
			} else {
				tracker.SetSucceeded(name)
				fmt.Fprintf(out, "Completed\n")
			}
		}(taskName, task, writer)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("%d task(s) failed", len(errors))
	}
	return nil
}

// runTask executes a single task.
func (e *Executor) runTask(taskName string, task compose.Task, out io.Writer, iteration, totalIterations int, outputDir string) error {
	// Generate task ID
	taskID := state.GenerateID()

	// Load prompt content
	promptContent, _, err := e.loadTaskPrompt(task)
	if err != nil {
		return err
	}

	// Process {{output:task_name}} directives before other injections
	promptContent, err = prompt.ProcessOutputDirectives(promptContent, outputDir)
	if err != nil {
		return fmt.Errorf("failed to process output directives: %w", err)
	}

	// Inject task ID into prompt
	promptContent = prompt.InjectTaskID(promptContent, taskID)

	// Determine effective model
	effectiveModel := e.cfg.AppConfig.Model
	if task.Model != "" {
		effectiveModel = task.Model
	}

	// Generate agent ID and inject it
	agentID := state.GenerateID()
	promptContent = prompt.InjectAgentID(promptContent, agentID)
	promptContent = prompt.InjectIteration(promptContent, iteration, totalIterations)

	// Inject the output directory so the agent can write its own state
	promptContent = prompt.InjectOutputDir(promptContent, outputDir, taskName)

	// Create and run the agent
	cfg := agent.Config{
		Model:   effectiveModel,
		Prompt:  promptContent,
		Command: e.cfg.AppConfig.Command,
	}

	runner := agent.NewRunner(cfg)

	// Set up real-time usage callback
	runner.SetUsageCallback(func(stats logparser.UsageStats) {
		e.mu.Lock()
		e.taskStats[taskName] = stats
		e.persistUsageState()
		e.mu.Unlock()
	})

	err = runner.Run(out)

	// Move this task's final stats from running to completed
	stats := runner.UsageStats()
	e.mu.Lock()
	delete(e.taskStats, taskName)
	e.inputTokens += stats.InputTokens
	e.outputTokens += stats.OutputTokens
	e.totalCostUSD += stats.TotalCostUSD
	e.persistUsageState()
	e.mu.Unlock()

	return err
}

// persistUsageState writes the current total usage (completed + running tasks) to pipeline state.
// Must be called with e.mu held.
func (e *Executor) persistUsageState() {
	if e.cfg.StateManager == nil || e.cfg.TaskID == "" {
		return
	}
	agentState, err := e.cfg.StateManager.Get(e.cfg.TaskID)
	if err != nil {
		return
	}

	// Sum completed + all running tasks
	totalInput := e.inputTokens
	totalOutput := e.outputTokens
	totalCost := e.totalCostUSD
	for _, s := range e.taskStats {
		totalInput += s.InputTokens
		totalOutput += s.OutputTokens
		totalCost += s.TotalCostUSD
	}

	agentState.InputTokens = totalInput
	agentState.OutputTokens = totalOutput
	if totalCost > 0 {
		agentState.TotalCost = totalCost
	} else if e.cfg.AppConfig != nil {
		pricing := e.cfg.AppConfig.GetPricing(agentState.Model)
		agentState.TotalCost = pricing.CalculateCost(totalInput, totalOutput)
	}
	_ = e.cfg.StateManager.MergeUpdate(agentState)
}

// loadTaskPrompt loads the prompt content for a task.
func (e *Executor) loadTaskPrompt(task compose.Task) (content, label string, err error) {
	switch {
	case task.PromptFile != "":
		label = task.PromptFile
		content, err = prompt.LoadPromptFromFile(task.PromptFile)
	case task.PromptString != "":
		label = "<string>"
		content = prompt.WrapPromptString(task.PromptString)
	case task.Prompt != "":
		label = task.Prompt
		content, err = prompt.LoadPrompt(e.cfg.PromptsDir, task.Prompt)
	default:
		err = fmt.Errorf("no prompt source specified")
	}
	if err != nil {
		return
	}
	// Apply prefix/suffix if specified
	content = prompt.ApplyPrefixSuffix(content, task.Prefix, task.Suffix)
	return
}

// PipelineResult holds the results of a pipeline execution.
type PipelineResult struct {
	// Iterations is the number of iterations completed
	Iterations int

	// IterationResults holds results for each iteration
	IterationResults []IterationResult

	// Duration is the total pipeline duration
	Duration time.Duration
}

// IterationResult holds the results of a single DAG iteration.
type IterationResult struct {
	// Iteration number (1-indexed)
	Iteration int

	// TaskResults maps task name to its result
	TaskResults map[string]TaskResult

	// Duration is how long this iteration took
	Duration time.Duration
}

// TaskResult holds the result of a single task execution.
type TaskResult struct {
	Status   TaskStatus
	Error    error
	Duration time.Duration
}
