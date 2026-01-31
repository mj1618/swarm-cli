package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/matt/swarm-cli/internal/agent"
	"github.com/matt/swarm-cli/internal/compose"
	"github.com/matt/swarm-cli/internal/dag"
	"github.com/matt/swarm-cli/internal/detach"
	"github.com/matt/swarm-cli/internal/output"
	"github.com/matt/swarm-cli/internal/prompt"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	upFile     string
	upDetach   bool
	upPipeline string
)

var upCmd = &cobra.Command{
	Use:   "up [task...]",
	Short: "Run tasks defined in a compose file",
	Long: `Run tasks from a compose file (./swarm/swarm.yaml by default).

By default, 'swarm up' runs:
  1. All defined pipelines (in DAG order with iterations)
  2. All standalone tasks (tasks not in pipelines and without dependencies)

Tasks that are part of a pipeline or have dependencies are only run via their
pipeline - they won't run as standalone parallel tasks.

Each task can specify:
  - prompt: Name of a prompt from the prompts directory
  - prompt-file: Path to an arbitrary prompt file
  - prompt-string: Direct prompt text
  - model: Model to use (optional, overrides config)
  - iterations: Number of iterations (for standalone tasks)
  - name: Custom agent name (optional, defaults to task name)
  - depends_on: Task dependencies with optional conditions

Pipelines define DAG workflows with iteration cycles:
  - Each iteration runs the entire DAG to completion before the next
  - Tasks can have conditional dependencies (success, failure, any, always)`,
	Example: `  # Run all pipelines and standalone tasks
  swarm up

  # Run specific tasks only (bypass pipeline logic)
  swarm up frontend backend

  # Run only a specific pipeline
  swarm up --pipeline development

  # Run in detached mode (standalone tasks only)
  swarm up -d

  # Use a custom compose file
  swarm up -f custom.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load compose file
		cf, err := compose.Load(upFile)
		if err != nil {
			return fmt.Errorf("failed to load compose file %s: %w", upFile, err)
		}

		// Validate compose file
		if err := cf.Validate(); err != nil {
			return fmt.Errorf("invalid compose file: %w", err)
		}

		// Get prompts directory based on scope
		promptsDir, err := GetPromptsDir()
		if err != nil {
			return fmt.Errorf("failed to get prompts directory: %w", err)
		}

		// Get current working directory
		workingDir, err := scope.CurrentWorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// If a specific pipeline is requested, run only that pipeline
		if upPipeline != "" {
			if upDetach {
				return fmt.Errorf("detached mode is not yet supported for pipelines")
			}
			return runPipeline(cf, upPipeline, promptsDir, workingDir)
		}

		// If specific tasks are requested via args, run only those tasks
		if len(args) > 0 {
			tasks, err := cf.GetTasks(args)
			if err != nil {
				return err
			}
			taskNames := make([]string, 0, len(tasks))
			for name := range tasks {
				taskNames = append(taskNames, name)
			}
			sort.Strings(taskNames)

			fmt.Printf("Starting %d task(s) from %s\n", len(tasks), upFile)

			if upDetach {
				return runTasksDetached(taskNames, tasks, promptsDir, workingDir)
			}
			return runTasksForeground(taskNames, tasks, promptsDir, workingDir)
		}

		// Default behavior: run all pipelines + standalone tasks
		return runAllPipelinesAndStandaloneTasks(cf, promptsDir, workingDir)
	},
}

func init() {
	upCmd.Flags().StringVarP(&upFile, "file", "f", compose.DefaultPath(), "Path to compose file")
	upCmd.Flags().BoolVarP(&upDetach, "detach", "d", false, "Run all tasks in background")
	upCmd.Flags().StringVarP(&upPipeline, "pipeline", "p", "", "Run a named pipeline (DAG with iterations)")
}

// runPipeline runs a named pipeline using the DAG executor.
func runPipeline(cf *compose.ComposeFile, pipelineName, promptsDir, workingDir string) error {
	// Get the pipeline definition
	pipeline, err := cf.GetPipeline(pipelineName)
	if err != nil {
		// List available pipelines if any exist
		if cf.HasPipelines() {
			var names []string
			for name := range cf.Pipelines {
				names = append(names, name)
			}
			return fmt.Errorf("%w\nAvailable pipelines: %v", err, names)
		}
		return fmt.Errorf("%w\nNo pipelines defined in compose file", err)
	}

	fmt.Printf("Running pipeline %q from %s\n", pipelineName, upFile)

	// Create the executor
	executor := dag.NewExecutor(dag.ExecutorConfig{
		AppConfig:  appConfig,
		PromptsDir: promptsDir,
		WorkingDir: workingDir,
		Output:     os.Stdout,
	})

	// Run the pipeline
	return executor.RunPipeline(*pipeline, cf.Tasks)
}

// runAllPipelinesAndStandaloneTasks runs all defined pipelines and standalone tasks.
// Standalone tasks are tasks that are not part of any pipeline and have no dependencies.
func runAllPipelinesAndStandaloneTasks(cf *compose.ComposeFile, promptsDir, workingDir string) error {
	// Get standalone tasks (not in any pipeline, no dependencies, not depended upon)
	standaloneTasks := cf.GetStandaloneTasks()

	// Sort pipeline names for consistent output
	var pipelineNames []string
	for name := range cf.Pipelines {
		pipelineNames = append(pipelineNames, name)
	}
	sort.Strings(pipelineNames)

	// Sort standalone task names for consistent output
	var standaloneNames []string
	for name := range standaloneTasks {
		standaloneNames = append(standaloneNames, name)
	}
	sort.Strings(standaloneNames)

	// Report what we're going to run
	fmt.Printf("From %s:\n", upFile)
	if len(pipelineNames) > 0 {
		fmt.Printf("  Pipelines: %v\n", pipelineNames)
	}
	if len(standaloneNames) > 0 {
		fmt.Printf("  Standalone tasks: %v\n", standaloneNames)
	}
	if len(pipelineNames) == 0 && len(standaloneNames) == 0 {
		fmt.Println("  No pipelines or standalone tasks to run")
		return nil
	}
	fmt.Println()

	// Run pipelines sequentially (each pipeline runs its full iteration cycle)
	for _, pipelineName := range pipelineNames {
		pipeline := cf.Pipelines[pipelineName]

		fmt.Printf("=== Pipeline: %s ===\n", pipelineName)

		executor := dag.NewExecutor(dag.ExecutorConfig{
			AppConfig:  appConfig,
			PromptsDir: promptsDir,
			WorkingDir: workingDir,
			Output:     os.Stdout,
		})

		if err := executor.RunPipeline(pipeline, cf.Tasks); err != nil {
			return fmt.Errorf("pipeline %q failed: %w", pipelineName, err)
		}
	}

	// Run standalone tasks in parallel (they have no dependencies)
	if len(standaloneNames) > 0 {
		fmt.Printf("=== Standalone Tasks ===\n")
		if upDetach {
			return runTasksDetached(standaloneNames, standaloneTasks, promptsDir, workingDir)
		}
		return runTasksForeground(standaloneNames, standaloneTasks, promptsDir, workingDir)
	}

	return nil
}

// runTasksDetached spawns all tasks as detached agents and returns immediately.
func runTasksDetached(taskNames []string, tasks map[string]compose.Task, promptsDir, workingDir string) error {
	mgr, err := state.NewManagerWithScope(GetScope(), workingDir)
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	// Get running agents to check for already-running tasks
	runningAgents, _ := mgr.List(true) // true = only running
	runningNames := make(map[string]bool)
	for _, a := range runningAgents {
		runningNames[a.Name] = true
	}

	var startedTasks []string
	var skippedTasks []string
	var failedTasks []string

	for _, taskName := range taskNames {
		task := tasks[taskName]

		// Check if task is already running
		effectiveName := task.EffectiveName(taskName)
		if runningNames[effectiveName] {
			fmt.Printf("  [%s] Already running, skipping\n", taskName)
			skippedTasks = append(skippedTasks, taskName)
			continue
		}

		// Generate task ID
		taskID := state.GenerateID()

		// Load prompt content
		promptContent, promptLabel, err := loadTaskPrompt(task, promptsDir)
		if err != nil {
			fmt.Printf("  [%s] Error: %v\n", taskName, err)
			failedTasks = append(failedTasks, taskName)
			continue
		}

		// Inject task ID into prompt
		promptContent = prompt.InjectTaskID(promptContent, taskID)

		// Determine effective values
		effectiveModel := appConfig.Model
		if task.Model != "" {
			effectiveModel = task.Model
		}
		effectiveIterations := task.EffectiveIterations()

		// Create log file
		logFile, err := detach.LogFilePath(taskID)
		if err != nil {
			fmt.Printf("  [%s] Error creating log file: %v\n", taskName, err)
			failedTasks = append(failedTasks, taskName)
			continue
		}

		// Build args for the detached process
		detachedArgs := []string{"run", "--_internal-detached", "--_internal-task-id", taskID}
		if globalFlag {
			detachedArgs = append(detachedArgs, "--global")
		}
		if task.Model != "" {
			detachedArgs = append(detachedArgs, "--model", task.Model)
		}
		if task.Prompt != "" {
			detachedArgs = append(detachedArgs, "--prompt", task.Prompt)
		}
		if task.PromptFile != "" {
			detachedArgs = append(detachedArgs, "--prompt-file", task.PromptFile)
		}
		if task.PromptString != "" {
			detachedArgs = append(detachedArgs, "--prompt-string", task.PromptString)
		}
		detachedArgs = append(detachedArgs, "--iterations", strconv.Itoa(effectiveIterations))
		if task.Name != "" {
			detachedArgs = append(detachedArgs, "--name", task.Name)
		} else {
			detachedArgs = append(detachedArgs, "--name", taskName)
		}
		// Pass prefix/suffix to child
		if task.Prefix != "" {
			detachedArgs = append(detachedArgs, "--_internal-prefix", task.Prefix)
		}
		if task.Suffix != "" {
			detachedArgs = append(detachedArgs, "--_internal-suffix", task.Suffix)
		}

		// Start detached process
		pid, err := detach.StartDetached(detachedArgs, logFile, workingDir)
		if err != nil {
			fmt.Printf("  [%s] Error starting: %v\n", taskName, err)
			failedTasks = append(failedTasks, taskName)
			continue
		}

		// Register agent state
		agentState := &state.AgentState{
			ID:          taskID,
			Name:        effectiveName,
			PID:         pid,
			Prompt:      promptLabel,
			Model:       effectiveModel,
			StartedAt:   time.Now(),
			Iterations:  effectiveIterations,
			CurrentIter: 0,
			Status:      "running",
			LogFile:     logFile,
			WorkingDir:  workingDir,
		}

		if err := mgr.Register(agentState); err != nil {
			fmt.Printf("  [%s] Error registering state: %v\n", taskName, err)
			failedTasks = append(failedTasks, taskName)
			continue
		}

		fmt.Printf("  [%s] Started (ID: %s, PID: %d, iterations: %d)\n", taskName, taskID, pid, effectiveIterations)
		startedTasks = append(startedTasks, taskName)
	}

	fmt.Println()
	if len(startedTasks) > 0 {
		fmt.Printf("Started %d task(s) in background. Use 'swarm list' to view status.\n", len(startedTasks))
	}
	if len(skippedTasks) > 0 {
		fmt.Printf("Skipped %d task(s) already running: %v\n", len(skippedTasks), skippedTasks)
	}
	if len(failedTasks) > 0 {
		fmt.Printf("Failed to start %d task(s): %v\n", len(failedTasks), failedTasks)
	}

	return nil
}

// runTasksForeground runs all tasks in parallel and waits for them to complete.
func runTasksForeground(taskNames []string, tasks map[string]compose.Task, promptsDir, workingDir string) error {
	// Initialize state manager to check for already-running tasks
	mgr, err := state.NewManagerWithScope(GetScope(), workingDir)
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	// Get running agents to check for already-running tasks
	runningAgents, _ := mgr.List(true) // true = only running
	runningNames := make(map[string]bool)
	for _, a := range runningAgents {
		runningNames[a.Name] = true
	}

	// Collect tasks that will actually run (not skipped)
	var tasksToRun []string
	var skippedTasks []string

	for _, taskName := range taskNames {
		task := tasks[taskName]
		effectiveName := task.EffectiveName(taskName)
		if runningNames[effectiveName] {
			fmt.Printf("  [%s] Already running, skipping\n", taskName)
			skippedTasks = append(skippedTasks, taskName)
			continue
		}
		tasksToRun = append(tasksToRun, taskName)
	}

	if len(tasksToRun) == 0 {
		if len(skippedTasks) > 0 {
			fmt.Printf("\nSkipped %d task(s) already running: %v\n", len(skippedTasks), skippedTasks)
		}
		return nil
	}

	// Create prefixed writer group for colored, synchronized output
	writers := output.NewWriterGroup(os.Stdout, tasksToRun)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var failedTasks []string

	for _, taskName := range tasksToRun {
		task := tasks[taskName]
		writer := writers.Get(taskName)

		wg.Add(1)

		go func(name string, t compose.Task, out *output.PrefixedWriter) {
			defer wg.Done()
			defer out.Flush()

			if err := runSingleTask(name, t, promptsDir, workingDir, out); err != nil {
				mu.Lock()
				failedTasks = append(failedTasks, name)
				mu.Unlock()
				fmt.Fprintf(out, "Error: %v\n", err)
			}
		}(taskName, task, writer)
	}

	wg.Wait()

	fmt.Println()
	if len(skippedTasks) > 0 {
		fmt.Printf("Skipped %d task(s) already running: %v\n", len(skippedTasks), skippedTasks)
	}
	if len(failedTasks) > 0 {
		return fmt.Errorf("%d task(s) failed: %v", len(failedTasks), failedTasks)
	}

	fmt.Println("All tasks completed successfully.")
	return nil
}

// runSingleTask runs a single task in the foreground.
// The out parameter is used for all task output (supports prefixed writers for parallel execution).
func runSingleTask(taskName string, task compose.Task, promptsDir, workingDir string, out io.Writer) error {
	// Generate task ID
	taskID := state.GenerateID()

	// Load prompt content
	promptContent, promptLabel, err := loadTaskPrompt(task, promptsDir)
	if err != nil {
		return err
	}

	// Inject task ID into prompt
	promptContent = prompt.InjectTaskID(promptContent, taskID)

	// Determine effective values
	effectiveModel := appConfig.Model
	if task.Model != "" {
		effectiveModel = task.Model
	}
	effectiveName := task.EffectiveName(taskName)
	effectiveIterations := task.EffectiveIterations()

	fmt.Fprintf(out, "Starting (model: %s, iterations: %d)\n", effectiveModel, effectiveIterations)

	// For single iteration, run directly
	if effectiveIterations == 1 {
		// Generate a per-iteration agent ID and inject it into the prompt.
		iterationAgentID := state.GenerateID()
		iterationPrompt := prompt.InjectAgentID(promptContent, iterationAgentID)

		cfg := agent.Config{
			Model:   effectiveModel,
			Prompt:  iterationPrompt,
			Command: appConfig.Command,
		}
		runner := agent.NewRunner(cfg)
		if err := runner.Run(out); err != nil {
			return err
		}
		fmt.Fprintf(out, "Completed\n")
		return nil
	}

	// Multi-iteration mode with state management
	mgr, err := state.NewManagerWithScope(GetScope(), workingDir)
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	agentState := &state.AgentState{
		ID:          taskID,
		Name:        effectiveName,
		PID:         os.Getpid(),
		Prompt:      promptLabel,
		Model:       effectiveModel,
		StartedAt:   time.Now(),
		Iterations:  effectiveIterations,
		CurrentIter: 0,
		Status:      "running",
		WorkingDir:  workingDir,
	}

	if err := mgr.Register(agentState); err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}

	defer func() {
		agentState.Status = "terminated"
		_ = mgr.Update(agentState)
	}()

	// Run iterations
	for i := 1; i <= agentState.Iterations; i++ {
		// Check for control signals from state
		currentState, err := mgr.Get(agentState.ID)
		if err == nil && currentState != nil {
			if currentState.Iterations != agentState.Iterations {
				agentState.Iterations = currentState.Iterations
			}
			if currentState.Model != agentState.Model {
				agentState.Model = currentState.Model
			}
			if currentState.TerminateMode == "immediate" {
				fmt.Fprintf(out, "Received termination signal\n")
				return nil
			}
			if currentState.TerminateMode == "after_iteration" && i > 1 {
				fmt.Fprintf(out, "Terminating after iteration\n")
				return nil
			}
		}

		agentState.CurrentIter = i
		_ = mgr.Update(agentState)

		fmt.Fprintf(out, "=== Iteration %d/%d ===\n", i, agentState.Iterations)

		// Generate a per-iteration agent ID and inject it into the prompt.
		iterationAgentID := state.GenerateID()
		iterationPrompt := prompt.InjectAgentID(promptContent, iterationAgentID)

		cfg := agent.Config{
			Model:   agentState.Model,
			Prompt:  iterationPrompt,
			Command: appConfig.Command,
		}

		runner := agent.NewRunner(cfg)
		if err := runner.Run(out); err != nil {
			fmt.Fprintf(out, "Agent error (continuing): %v\n", err)
		}
	}

	fmt.Fprintf(out, "Completed (%d iterations)\n", agentState.Iterations)
	return nil
}

// loadTaskPrompt loads the prompt content for a task.
// Returns the content and a label for display.
func loadTaskPrompt(task compose.Task, promptsDir string) (content, label string, err error) {
	switch {
	case task.PromptFile != "":
		label = task.PromptFile
		content, err = prompt.LoadPromptFromFile(task.PromptFile)
	case task.PromptString != "":
		label = "<string>"
		content = prompt.WrapPromptString(task.PromptString)
	case task.Prompt != "":
		label = task.Prompt
		content, err = prompt.LoadPrompt(promptsDir, task.Prompt)
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
