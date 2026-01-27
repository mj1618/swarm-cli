package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/matt/swarm-cli/internal/agent"
	"github.com/matt/swarm-cli/internal/compose"
	"github.com/matt/swarm-cli/internal/detach"
	"github.com/matt/swarm-cli/internal/prompt"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	upFile   string
	upDetach bool
)

var upCmd = &cobra.Command{
	Use:   "up [task...]",
	Short: "Run tasks defined in a compose file",
	Long: `Run one or more tasks from a compose file (./swarm/swarm.yaml by default).

Similar to docker compose up, this command reads task definitions from a YAML
file and runs them. By default, all tasks are run in parallel.

Each task can specify:
  - prompt: Name of a prompt from the prompts directory
  - prompt-file: Path to an arbitrary prompt file
  - prompt-string: Direct prompt text
  - model: Model to use (optional, overrides config)
  - iterations: Number of iterations (optional, default 1)
  - name: Custom agent name (optional, defaults to task name)`,
	Example: `  # Run all tasks from ./swarm/swarm.yaml
  swarm up

  # Run specific tasks only
  swarm up frontend backend

  # Run in detached mode (background)
  swarm up -d

  # Use a custom compose file
  swarm up -f custom.yaml

  # Combine options
  swarm up -d -f deploy.yaml frontend`,
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

		// Get tasks to run (filtered by args if provided)
		tasks, err := cf.GetTasks(args)
		if err != nil {
			return err
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

		// Sort task names for consistent output
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
	},
}

func init() {
	upCmd.Flags().StringVarP(&upFile, "file", "f", compose.DefaultPath(), "Path to compose file")
	upCmd.Flags().BoolVarP(&upDetach, "detach", "d", false, "Run all tasks in background")
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

	var wg sync.WaitGroup
	var mu sync.Mutex
	var failedTasks []string
	var skippedTasks []string
	var startedCount int

	for _, taskName := range taskNames {
		task := tasks[taskName]

		// Check if task is already running
		effectiveName := task.EffectiveName(taskName)
		if runningNames[effectiveName] {
			fmt.Printf("  [%s] Already running, skipping\n", taskName)
			skippedTasks = append(skippedTasks, taskName)
			continue
		}

		startedCount++
		wg.Add(1)

		go func(name string, t compose.Task) {
			defer wg.Done()

			if err := runSingleTask(name, t, promptsDir, workingDir); err != nil {
				mu.Lock()
				failedTasks = append(failedTasks, name)
				mu.Unlock()
				fmt.Printf("\n[%s] Error: %v\n", name, err)
			}
		}(taskName, task)
	}

	wg.Wait()

	fmt.Println()
	if len(skippedTasks) > 0 {
		fmt.Printf("Skipped %d task(s) already running: %v\n", len(skippedTasks), skippedTasks)
	}
	if len(failedTasks) > 0 {
		return fmt.Errorf("%d task(s) failed: %v", len(failedTasks), failedTasks)
	}

	if startedCount > 0 {
		fmt.Println("All tasks completed successfully.")
	}
	return nil
}

// runSingleTask runs a single task in the foreground.
func runSingleTask(taskName string, task compose.Task, promptsDir, workingDir string) error {
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

	fmt.Printf("\n[%s] Starting (model: %s, iterations: %d)\n", taskName, effectiveModel, effectiveIterations)

	// For single iteration, run directly
	if effectiveIterations == 1 {
		cfg := agent.Config{
			Model:   effectiveModel,
			Prompt:  promptContent,
			Command: appConfig.Command,
		}
		runner := agent.NewRunner(cfg)
		if err := runner.Run(os.Stdout); err != nil {
			return err
		}
		fmt.Printf("\n[%s] Completed\n", taskName)
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
				fmt.Printf("\n[%s] Received termination signal\n", taskName)
				return nil
			}
			if currentState.TerminateMode == "after_iteration" && i > 1 {
				fmt.Printf("\n[%s] Terminating after iteration\n", taskName)
				return nil
			}
		}

		agentState.CurrentIter = i
		_ = mgr.Update(agentState)

		fmt.Printf("\n[%s] === Iteration %d/%d ===\n", taskName, i, agentState.Iterations)

		cfg := agent.Config{
			Model:   agentState.Model,
			Prompt:  promptContent,
			Command: appConfig.Command,
		}

		runner := agent.NewRunner(cfg)
		if err := runner.Run(os.Stdout); err != nil {
			fmt.Printf("\n[%s] Agent error (continuing): %v\n", taskName, err)
		}
	}

	fmt.Printf("\n[%s] Completed (%d iterations)\n", taskName, agentState.Iterations)
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
	return
}
