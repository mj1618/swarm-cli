package dag

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/mj1618/swarm-cli/internal/agent"
	"github.com/mj1618/swarm-cli/internal/compose"
	"github.com/mj1618/swarm-cli/internal/config"
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
}

// Executor runs pipelines with DAG-ordered task execution.
type Executor struct {
	cfg ExecutorConfig
}

// NewExecutor creates a new pipeline executor.
func NewExecutor(cfg ExecutorConfig) *Executor {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	return &Executor{cfg: cfg}
}

// RunPipeline runs a pipeline with the given tasks for the specified iterations.
func (e *Executor) RunPipeline(pipeline compose.Pipeline, tasks map[string]compose.Task) error {
	// Get task names for this pipeline
	taskNames := pipeline.GetPipelineTasks(tasks)

	// Build and validate the DAG
	graph := NewGraph(tasks, taskNames)
	if err := graph.Validate(); err != nil {
		return fmt.Errorf("invalid DAG: %w", err)
	}

	iterations := pipeline.EffectiveIterations()
	fmt.Fprintf(e.cfg.Output, "Running pipeline with %d iteration(s) and %d task(s)\n", iterations, len(taskNames))

	// Run each iteration
	for i := 1; i <= iterations; i++ {
		fmt.Fprintf(e.cfg.Output, "\n=== Pipeline Iteration %d/%d ===\n", i, iterations)

		if err := e.runDAG(graph, taskNames, i); err != nil {
			return fmt.Errorf("iteration %d failed: %w", i, err)
		}

		fmt.Fprintf(e.cfg.Output, "--- Iteration %d complete ---\n", i)
	}

	fmt.Fprintf(e.cfg.Output, "\nPipeline completed successfully (%d iterations)\n", iterations)
	return nil
}

// runDAG executes a single DAG iteration.
func (e *Executor) runDAG(graph *Graph, taskNames []string, iteration int) error {
	// Initialize state tracker
	states := NewStateTracker(taskNames)

	// Create prefixed writers for parallel output
	writers := output.NewWriterGroup(e.cfg.Output, taskNames)

	for {
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
				return fmt.Errorf("deadlock: %d pending task(s) but none ready", summary.Pending)
			}
			break
		}

		// Execute ready tasks in parallel
		if err := e.executeTasks(graph, readyTasks, states, writers, iteration); err != nil {
			// Log error but continue - individual task failures don't stop the DAG
			fmt.Fprintf(e.cfg.Output, "Warning: task execution error: %v\n", err)
		}
	}

	// Report final summary
	summary := states.GetSummary()
	fmt.Fprintf(e.cfg.Output, "Tasks: %d succeeded, %d failed, %d skipped\n",
		summary.Succeeded, summary.Failed, summary.Skipped)

	return nil
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
func (e *Executor) executeTasks(graph *Graph, taskNames []string, tracker *StateTracker, writers *output.WriterGroup, iteration int) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, taskName := range taskNames {
		task, ok := graph.GetTask(taskName)
		if !ok {
			continue
		}

		writer := writers.Get(taskName)
		tracker.SetRunning(taskName)

		wg.Add(1)
		go func(name string, t compose.Task, out *output.PrefixedWriter) {
			defer wg.Done()
			defer out.Flush()

			fmt.Fprintf(out, "Starting (iteration %d)\n", iteration)

			err := e.runTask(name, t, out)
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
func (e *Executor) runTask(taskName string, task compose.Task, out io.Writer) error {
	// Generate task ID
	taskID := state.GenerateID()

	// Load prompt content
	promptContent, _, err := e.loadTaskPrompt(task)
	if err != nil {
		return err
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

	// Create and run the agent
	cfg := agent.Config{
		Model:   effectiveModel,
		Prompt:  promptContent,
		Command: e.cfg.AppConfig.Command,
	}

	runner := agent.NewRunner(cfg)
	return runner.Run(out)
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
