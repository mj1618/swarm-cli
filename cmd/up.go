package cmd

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mj1618/swarm-cli/internal/agent"
	"github.com/mj1618/swarm-cli/internal/compose"
	"github.com/mj1618/swarm-cli/internal/dag"
	"github.com/mj1618/swarm-cli/internal/detach"
	"github.com/mj1618/swarm-cli/internal/logparser"
	"github.com/mj1618/swarm-cli/internal/output"
	"github.com/mj1618/swarm-cli/internal/process"
	"github.com/mj1618/swarm-cli/internal/prompt"
	"github.com/mj1618/swarm-cli/internal/scope"
	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	upFile              string
	upDetach            bool
	upPipeline          string
	upInternalDetached  bool
	upInternalTaskID    string
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

  # Run specific tasks only
  swarm up frontend backend

  # Run a specific pipeline by name
  swarm up development

  # Mix tasks and pipelines
  swarm up development frontend

  # Run in detached mode
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

		// If running as a detached child, run the pipeline directly
		if upInternalDetached && upPipeline != "" {
			return runPipeline(cf, upPipeline, promptsDir, workingDir)
		}

		// If a specific pipeline is requested via flag, run only that pipeline
		if upPipeline != "" {
			if upDetach {
				return runPipelineDetached(cf, upPipeline, promptsDir, workingDir)
			}
			return runPipeline(cf, upPipeline, promptsDir, workingDir)
		}

		// If specific tasks/pipelines are requested via args, run them
		if len(args) > 0 {
			// Separate args into task names and pipeline names
			var taskArgs []string
			var pipelineArgNames []string
			for _, arg := range args {
				if _, exists := cf.Pipelines[arg]; exists {
					pipelineArgNames = append(pipelineArgNames, arg)
				} else {
					taskArgs = append(taskArgs, arg)
				}
			}

			// Run requested pipelines
			for _, pipelineName := range pipelineArgNames {
				if upDetach {
					if err := runPipelineDetached(cf, pipelineName, promptsDir, workingDir); err != nil {
						return fmt.Errorf("pipeline %q failed to start: %w", pipelineName, err)
					}
				} else {
					if err := runPipeline(cf, pipelineName, promptsDir, workingDir); err != nil {
						return fmt.Errorf("pipeline %q failed: %w", pipelineName, err)
					}
				}
			}

			// Run requested tasks
			if len(taskArgs) > 0 {
				tasks, err := cf.GetTasks(taskArgs)
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

			return nil
		}

		// Default behavior: run all pipelines + standalone tasks
		return runAllPipelinesAndStandaloneTasks(cf, promptsDir, workingDir)
	},
}

func init() {
	upCmd.Flags().StringVarP(&upFile, "file", "f", compose.DefaultPath(), "Path to compose file")
	upCmd.Flags().BoolVarP(&upDetach, "detach", "d", false, "Run all tasks in background")
	upCmd.Flags().StringVarP(&upPipeline, "pipeline", "p", "", "Run a named pipeline (DAG with iterations)")
	upCmd.Flags().BoolVar(&upInternalDetached, "_internal-detached", false, "Internal flag for detached execution")
	upCmd.Flags().MarkHidden("_internal-detached")
	upCmd.Flags().StringVar(&upInternalTaskID, "_internal-task-id", "", "Internal flag for passing task ID to detached child")
	upCmd.Flags().MarkHidden("_internal-task-id")
}

// runPipeline runs a named pipeline using the DAG executor.
// When parallelism > 1 (and not running as a detached child), it spawns
// multiple concurrent instances of the pipeline.
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

	parallelism := pipeline.EffectiveParallelism()

	// Detached children are already a single instance — don't re-expand
	if parallelism <= 1 || upInternalDetached {
		fmt.Printf("Running pipeline %q from %s\n", pipelineName, upFile)
		return runSinglePipelineInstance(cf, pipelineName, *pipeline, promptsDir, workingDir, os.Stdout)
	}

	// Multiple parallel instances
	fmt.Printf("Running pipeline %q from %s (parallelism: %d)\n", pipelineName, upFile, parallelism)

	var instanceNames []string
	for i := 1; i <= parallelism; i++ {
		instanceNames = append(instanceNames, fmt.Sprintf("%s.%d", pipelineName, i))
	}

	writers := output.NewWriterGroup(os.Stdout, instanceNames)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for i := 1; i <= parallelism; i++ {
		instanceName := fmt.Sprintf("%s.%d", pipelineName, i)
		writer := writers.Get(instanceName)

		wg.Add(1)
		go func(name string, out *output.PrefixedWriter) {
			defer wg.Done()
			defer out.Flush()

			if err := runSinglePipelineInstance(cf, name, *pipeline, promptsDir, workingDir, out); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("%s: %w", name, err))
				mu.Unlock()
			}
		}(instanceName, writer)
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("%d pipeline instance(s) failed", len(errors))
	}
	return nil
}

// runSinglePipelineInstance runs a single instance of a pipeline using the DAG executor.
func runSinglePipelineInstance(cf *compose.ComposeFile, name string, pipeline compose.Pipeline, promptsDir, workingDir string, out io.Writer) error {
	execCfg := dag.ExecutorConfig{
		AppConfig:  appConfig,
		PromptsDir: promptsDir,
		WorkingDir: workingDir,
		Output:     out,
	}

	// If running as a detached child, set up state tracking
	if upInternalTaskID != "" {
		mgr, err := state.NewManagerWithScope(GetScope(), workingDir)
		if err == nil {
			execCfg.StateManager = mgr
			execCfg.TaskID = upInternalTaskID
		}
	}

	// Create the executor
	executor := dag.NewExecutor(execCfg)

	// Run the pipeline
	return executor.RunPipeline(pipeline, cf.Tasks)
}

// runPipelineDetached spawns a pipeline as a detached background process.
// When parallelism > 1, spawns multiple independent detached processes.
// On re-run, skips already-running instances and kills excess instances
// when parallelism has been reduced.
func runPipelineDetached(cf *compose.ComposeFile, pipelineName, promptsDir, workingDir string) error {
	// Verify the pipeline exists
	pipeline, err := cf.GetPipeline(pipelineName)
	if err != nil {
		return err
	}

	parallelism := pipeline.EffectiveParallelism()

	mgr, err := state.NewManagerWithScope(GetScope(), workingDir)
	if err != nil {
		return fmt.Errorf("failed to initialize state manager: %w", err)
	}

	// Get running agents to check for already-running and excess instances
	runningAgents, _ := mgr.List(true)
	runningByName := make(map[string]*state.AgentState)
	for _, a := range runningAgents {
		runningByName[a.Name] = a
	}

	// Compute desired instance names
	desiredNames := make(map[string]bool)
	for i := 1; i <= parallelism; i++ {
		instanceName := pipelineName
		if parallelism > 1 {
			instanceName = fmt.Sprintf("%s.%d", pipelineName, i)
		}
		desiredNames[fmt.Sprintf("pipeline:%s", instanceName)] = true
	}

	// Kill excess instances (running instances of this pipeline not in desired set)
	for _, a := range runningAgents {
		if !isPipelineInstance(a.Name, pipelineName) {
			continue
		}
		if desiredNames[a.Name] {
			continue
		}
		fmt.Printf("Killing excess pipeline instance %q (ID: %s, PID: %d)\n", a.Name, a.ID, a.PID)
		killAgentAndDescendants(mgr, a)
	}

	effectiveIterations := pipeline.EffectiveIterations()

	var startedCount, skippedCount int
	for i := 1; i <= parallelism; i++ {
		instanceName := pipelineName
		if parallelism > 1 {
			instanceName = fmt.Sprintf("%s.%d", pipelineName, i)
		}

		agentName := fmt.Sprintf("pipeline:%s", instanceName)

		// Skip if already running
		if _, running := runningByName[agentName]; running {
			fmt.Printf("Pipeline %q already running, skipping\n", instanceName)
			skippedCount++
			continue
		}

		taskID := state.GenerateID()

		logFile, err := detach.LogFilePath(taskID)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}

		// Build args for the detached process
		detachedArgs := []string{"up", "--_internal-detached", "--_internal-task-id", taskID, "--pipeline", pipelineName}
		if globalFlag {
			detachedArgs = append(detachedArgs, "--global")
		}
		if upFile != compose.DefaultPath() {
			detachedArgs = append(detachedArgs, "--file", upFile)
		}

		agentState := &state.AgentState{
			ID:          taskID,
			Name:        agentName,
			Prompt:      fmt.Sprintf("pipeline:%s", pipelineName),
			Model:       appConfig.Model,
			StartedAt:   time.Now(),
			Iterations:  effectiveIterations,
			CurrentIter: 0,
			Status:      "running",
			LogFile:     logFile,
			WorkingDir:  workingDir,
		}

		// Start detached process
		pid, err := detach.StartDetached(detachedArgs, logFile, workingDir)
		if err != nil {
			return fmt.Errorf("failed to start detached process for %s: %w", instanceName, err)
		}

		agentState.PID = pid
		if err := mgr.Register(agentState); err != nil {
			return fmt.Errorf("failed to register state for %s: %w", instanceName, err)
		}

		fmt.Printf("Started pipeline %q in background (ID: %s, PID: %d)\n", instanceName, taskID, pid)
		startedCount++
	}

	if skippedCount > 0 {
		fmt.Printf("Skipped %d already-running pipeline instance(s)\n", skippedCount)
	}

	return nil
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
		for _, w := range cf.Warnings() {
			fmt.Printf("  Warning: %s\n", w)
		}
		return nil
	}
	fmt.Println()

	// Run pipelines
	if upDetach {
		// Detached mode: spawn each pipeline as a background process
		for _, pipelineName := range pipelineNames {
			if err := runPipelineDetached(cf, pipelineName, promptsDir, workingDir); err != nil {
				return fmt.Errorf("pipeline %q failed to start: %w", pipelineName, err)
			}
		}
	} else {
		// Foreground mode: run pipelines sequentially
		for _, pipelineName := range pipelineNames {
			fmt.Printf("=== Pipeline: %s ===\n", pipelineName)

			if err := runPipeline(cf, pipelineName, promptsDir, workingDir); err != nil {
				return fmt.Errorf("pipeline %q failed: %w", pipelineName, err)
			}
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
// On re-run, skips already-running instances and kills excess instances
// when parallelism has been reduced.
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

	// Scale-down: kill excess instances for tasks whose parallelism has been reduced
	for _, taskName := range taskNames {
		task := tasks[taskName]
		baseName := taskName
		if task.Name != "" {
			baseName = task.Name
		}
		p := task.EffectiveParallelism()

		// Compute desired names for this task
		desiredNames := make(map[string]bool)
		if p == 1 {
			desiredNames[task.EffectiveName(taskName)] = true
		} else {
			for j := 1; j <= p; j++ {
				if task.Name != "" {
					desiredNames[fmt.Sprintf("%s.%d", task.Name, j)] = true
				} else {
					desiredNames[fmt.Sprintf("%s.%d", taskName, j)] = true
				}
			}
		}

		// Find and kill excess instances
		for _, a := range runningAgents {
			if !isTaskInstance(a.Name, baseName) {
				continue
			}
			if desiredNames[a.Name] {
				continue
			}
			fmt.Printf("  [%s] Killing excess instance (ID: %s, PID: %d)\n", a.Name, a.ID, a.PID)
			killAgentAndDescendants(mgr, a)
			delete(runningNames, a.Name)
		}
	}

	// Expand tasks with parallelism > 1 into multiple instances
	var expandedNames []string
	expandedTasks := make(map[string]compose.Task)
	for _, taskName := range taskNames {
		task := tasks[taskName]
		p := task.EffectiveParallelism()
		if p == 1 {
			expandedNames = append(expandedNames, taskName)
			expandedTasks[taskName] = task
		} else {
			for j := 1; j <= p; j++ {
				instanceName := fmt.Sprintf("%s.%d", taskName, j)
				expandedNames = append(expandedNames, instanceName)
				expandedTask := task
				if task.Name != "" {
					expandedTask.Name = fmt.Sprintf("%s.%d", task.Name, j)
				}
				expandedTasks[instanceName] = expandedTask
			}
		}
	}

	var startedTasks []string
	var skippedTasks []string
	var failedTasks []string

	for _, taskName := range expandedNames {
		task := expandedTasks[taskName]

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
	// Initialize state manager (shared across all parallel tasks)
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

	// Expand tasks with parallelism > 1 into multiple instances BEFORE checking
	// for already-running tasks, so individual instances can be skipped independently
	var expandedNames []string
	expandedTasks := make(map[string]compose.Task)
	for _, taskName := range taskNames {
		task := tasks[taskName]
		p := task.EffectiveParallelism()
		if p == 1 {
			expandedNames = append(expandedNames, taskName)
			expandedTasks[taskName] = task
		} else {
			for j := 1; j <= p; j++ {
				instanceName := fmt.Sprintf("%s.%d", taskName, j)
				expandedNames = append(expandedNames, instanceName)
				expandedTask := task
				if task.Name != "" {
					expandedTask.Name = fmt.Sprintf("%s.%d", task.Name, j)
				}
				expandedTasks[instanceName] = expandedTask
			}
		}
	}

	// Check for already-running tasks on expanded instance names
	var tasksToRun []string
	var skippedTasks []string

	for _, taskName := range expandedNames {
		task := expandedTasks[taskName]
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
		task := expandedTasks[taskName]
		writer := writers.Get(taskName)

		wg.Add(1)

		go func(name string, t compose.Task, out *output.PrefixedWriter) {
			defer wg.Done()
			defer out.Flush()

			if err := runSingleTask(name, t, promptsDir, workingDir, out, mgr); err != nil {
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
// If mgr is non-nil, it is reused for state management instead of creating a new one.
func runSingleTask(taskName string, task compose.Task, promptsDir, workingDir string, out io.Writer, mgr *state.Manager) error {
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

	// Track cumulative usage across iterations
	var cumulativeInputTokens int64
	var cumulativeOutputTokens int64
	var cumulativeCostUSD float64

	// Multi-iteration mode with state management
	if mgr == nil {
		mgr, err = state.NewManagerWithScope(GetScope(), workingDir)
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}
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
				if i > agentState.Iterations {
					fmt.Fprintf(out, "Iteration limit reduced to %d, stopping\n", agentState.Iterations)
					return nil
				}
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

		// Set up usage callback to update state in real time
		iterStartInput := cumulativeInputTokens
		iterStartOutput := cumulativeOutputTokens
		iterStartCost := cumulativeCostUSD
		runner.SetUsageCallback(func(stats logparser.UsageStats) {
			agentState.InputTokens = iterStartInput + stats.InputTokens
			agentState.OutputTokens = iterStartOutput + stats.OutputTokens
			agentState.CurrentTask = stats.CurrentTask
			if stats.TotalCostUSD > 0 {
				agentState.TotalCost = iterStartCost + stats.TotalCostUSD
			}
			_ = mgr.MergeUpdate(agentState)
		})

		if err := runner.Run(out); err != nil {
			fmt.Fprintf(out, "Agent error (continuing): %v\n", err)
		}

		// Accumulate final stats from this iteration
		finalStats := runner.UsageStats()
		cumulativeInputTokens += finalStats.InputTokens
		cumulativeOutputTokens += finalStats.OutputTokens
		cumulativeCostUSD += finalStats.TotalCostUSD
		agentState.InputTokens = cumulativeInputTokens
		agentState.OutputTokens = cumulativeOutputTokens
		if cumulativeCostUSD > 0 {
			agentState.TotalCost = cumulativeCostUSD
		}
		_ = mgr.MergeUpdate(agentState)
	}

	fmt.Fprintf(out, "Completed (%d iterations)\n", agentState.Iterations)
	return nil
}

// isPipelineInstance returns true if agentName is an instance of the given pipeline.
// Matches "pipeline:name" (single instance) and "pipeline:name.N" (parallel instances).
func isPipelineInstance(agentName, pipelineName string) bool {
	base := fmt.Sprintf("pipeline:%s", pipelineName)
	if agentName == base {
		return true
	}
	prefix := base + "."
	if strings.HasPrefix(agentName, prefix) {
		_, err := strconv.Atoi(agentName[len(prefix):])
		return err == nil
	}
	return false
}

// isTaskInstance returns true if agentName is an instance of the given task base name.
// Matches "baseName" (single instance) and "baseName.N" (parallel instances).
func isTaskInstance(agentName, baseName string) bool {
	if agentName == baseName {
		return true
	}
	prefix := baseName + "."
	if strings.HasPrefix(agentName, prefix) {
		_, err := strconv.Atoi(agentName[len(prefix):])
		return err == nil
	}
	return false
}

// killAgentAndDescendants kills a running agent and all its running descendants.
func killAgentAndDescendants(mgr *state.Manager, a *state.AgentState) {
	// Kill descendants first
	descendants, err := mgr.GetDescendants(a.ID)
	if err == nil {
		for _, d := range descendants {
			if d.Status == "running" {
				_ = mgr.SetTerminateMode(d.ID, "immediate")
				_ = process.ForceKill(d.PID)
				now := time.Now()
				d.Status = "terminated"
				d.ExitReason = "killed"
				d.TerminatedAt = &now
				_ = mgr.Update(d)
			}
		}
	}

	_ = mgr.SetTerminateMode(a.ID, "immediate")
	_ = process.ForceKill(a.PID)
	now := time.Now()
	a.Status = "terminated"
	a.ExitReason = "killed"
	a.TerminatedAt = &now
	_ = mgr.Update(a)
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
