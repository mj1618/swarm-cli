package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/matt/swarm-cli/internal/agent"
	"github.com/matt/swarm-cli/internal/detach"
	"github.com/matt/swarm-cli/internal/prompt"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	runModel               string
	runPrompt              string
	runPromptFile          string
	runPromptString        string
	runStdin               bool
	runIterations          int
	runName                string
	runDetach              bool
	runInternalDetached    bool
	runInternalTaskID      string
	runInternalStdin       string
	runEnv                 []string
	runInternalEnv         []string
	runTimeout             string
	runIterTimeout         string
	runInternalTimeout     string
	runInternalIterTimeout string
	runWorkingDir          string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run an agent",
	Long: `Run an agent with a specified prompt and model.

By default, runs a single iteration. Use -n to run multiple iterations.
When running multiple iterations, agent failures do not stop the run.`,
	Example: `  # Interactive prompt selection (single iteration)
  swarm run

  # Use a named prompt from the prompts directory
  swarm run -p my-prompt

  # Run 10 iterations
  swarm run -p my-prompt -n 10

  # Run with a name for easy reference
  swarm run -p my-prompt -n 5 -N my-agent

  # Use a specific prompt file
  swarm run -f ./prompts/custom.md

  # Use an inline prompt string
  swarm run -s "Review the code for bugs"

  # Read prompt from stdin
  echo "Fix the bug in auth.go" | swarm run --stdin

  # Pipe file contents as prompt
  cat README.md | swarm run --stdin

  # Combine stdin with a named prompt template
  git diff | swarm run --stdin -p code-reviewer

  # Run with a specific model
  swarm run -p my-prompt -m claude-sonnet-4-20250514

  # Run in background (detached)
  swarm run -p my-prompt -n 20 -d

  # Run agent in a specific directory
  swarm run -p coder -C /path/to/project

  # Run agent in a subdirectory
  swarm run -p frontend -C ./frontend -d`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get working directory (from flag or current)
		var workingDir string
		var err error

		if runWorkingDir != "" {
			// Resolve relative to current directory
			if filepath.IsAbs(runWorkingDir) {
				workingDir = runWorkingDir
			} else {
				cwd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}
				workingDir = filepath.Join(cwd, runWorkingDir)
			}

			// Verify directory exists
			info, err := os.Stat(workingDir)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("working directory does not exist: %s", workingDir)
				}
				return fmt.Errorf("failed to access working directory: %w", err)
			}
			if !info.IsDir() {
				return fmt.Errorf("not a directory: %s", workingDir)
			}

			// Get absolute path for consistency
			workingDir, err = filepath.Abs(workingDir)
			if err != nil {
				return fmt.Errorf("failed to resolve working directory: %w", err)
			}
		} else {
			workingDir, err = scope.CurrentWorkingDir()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
		}

		// Get prompts directory based on scope
		// For project scope with custom working dir, use prompts from that directory
		var promptsDir string
		if runWorkingDir != "" && GetScope() == scope.ScopeProject {
			promptsDir = filepath.Join(workingDir, "swarm", "prompts")
		} else {
			promptsDir, err = GetPromptsDir()
			if err != nil {
				return fmt.Errorf("failed to get prompts directory: %w", err)
			}
		}

		// Load or select prompt
		var promptContent string
		var promptName string

		// Count how many prompt sources were specified
		// Note: --stdin can be combined with --prompt, but not with --prompt-file or --prompt-string
		specifiedCount := 0
		if runPrompt != "" {
			specifiedCount++
		}
		if runPromptFile != "" {
			specifiedCount++
		}
		if runPromptString != "" {
			specifiedCount++
		}

		// --stdin can only combine with --prompt
		if runStdin && (runPromptFile != "" || runPromptString != "") {
			return fmt.Errorf("--stdin cannot be combined with --prompt-file or --prompt-string")
		}

		if specifiedCount > 1 {
			return fmt.Errorf("only one of --prompt, --prompt-file, or --prompt-string can be specified")
		}

		// Handle stdin input
		var stdinContent string
		if runStdin {
			// For detached child, use content passed from parent
			if runInternalDetached && runInternalStdin != "" {
				stdinContent = runInternalStdin
			} else {
				// Check if stdin has data
				if !prompt.IsStdinPiped() {
					return fmt.Errorf("--stdin specified but no input piped (use a pipe or redirect)")
				}
				var err error
				stdinContent, err = prompt.LoadPromptFromStdin()
				if err != nil {
					return fmt.Errorf("failed to read prompt from stdin: %w", err)
				}
			}
		}

		switch {
		case runPromptFile != "":
			// Load from arbitrary file path
			promptName = runPromptFile
			promptContent, err = prompt.LoadPromptFromFile(runPromptFile)
			if err != nil {
				return fmt.Errorf("failed to load prompt file: %w", err)
			}
		case runPromptString != "":
			// Use direct string
			promptName = "<string>"
			promptContent = prompt.WrapPromptString(runPromptString)
		case runStdin && runPrompt != "":
			// Combine stdin with named prompt
			promptName = runPrompt + "+stdin"
			basePrompt, err := prompt.LoadPrompt(promptsDir, runPrompt)
			if err != nil {
				return fmt.Errorf("failed to load prompt: %w", err)
			}
			promptContent = prompt.CombinePrompts(basePrompt, stdinContent)
		case runStdin:
			// Use stdin content directly
			promptName = "<stdin>"
			promptContent = stdinContent
		case runPrompt != "":
			// Load from prompts directory
			promptName = runPrompt
			promptContent, err = prompt.LoadPrompt(promptsDir, runPrompt)
			if err != nil {
				return fmt.Errorf("failed to load prompt: %w", err)
			}
		default:
			// Interactive selection not allowed in detached mode
			if runDetach {
				return fmt.Errorf("prompt must be specified when using detached mode (-d)")
			}
			promptName, promptContent, err = prompt.SelectPrompt(promptsDir)
			if err != nil {
				return fmt.Errorf("failed to select prompt: %w", err)
			}
		}

		// Generate task ID early so it can be injected into prompt
		// If running as detached child, use the task ID passed from parent
		taskID := runInternalTaskID
		if taskID == "" {
			taskID = state.GenerateID()
		}

		// Inject task ID into prompt content
		promptContent = prompt.InjectTaskID(promptContent, taskID)

		// Determine effective model (CLI flag overrides config)
		effectiveModel := appConfig.Model
		if cmd.Flags().Changed("model") {
			effectiveModel = runModel
		}

		// Default name to prompt name if not specified
		effectiveName := runName
		if effectiveName == "" {
			effectiveName = promptName
		}

		// Determine effective iterations (CLI flag overrides config default of 1)
		effectiveIterations := 1
		if cmd.Flags().Changed("iterations") {
			effectiveIterations = runIterations
		}

		// Parse and expand environment variables
		// If running as detached child, use the env vars passed from parent
		var expandedEnv []string
		var envNames []string
		envSource := runEnv
		if runInternalDetached && len(runInternalEnv) > 0 {
			// Detached child: env vars are already expanded by parent
			expandedEnv = runInternalEnv
			for _, e := range expandedEnv {
				if idx := strings.Index(e, "="); idx > 0 {
					envNames = append(envNames, e[:idx])
				}
			}
		} else if len(envSource) > 0 {
			expandedEnv = make([]string, 0, len(envSource))
			for _, e := range envSource {
				if strings.Contains(e, "=") {
					// KEY=VALUE format - use as-is
					expandedEnv = append(expandedEnv, e)
					if idx := strings.Index(e, "="); idx > 0 {
						envNames = append(envNames, e[:idx])
					}
				} else {
					// KEY format - look up from environment
					if val, ok := os.LookupEnv(e); ok {
						expandedEnv = append(expandedEnv, fmt.Sprintf("%s=%s", e, val))
						envNames = append(envNames, e)
					} else {
						return fmt.Errorf("environment variable %s not set", e)
					}
				}
			}
		}

		// Parse timeout durations
		// For detached child, use internal flags; otherwise use CLI flags or config
		var totalTimeout, iterTimeout time.Duration
		effectiveTimeout := runTimeout
		effectiveIterTimeout := runIterTimeout

		if runInternalDetached {
			// Detached child: use values passed from parent
			if runInternalTimeout != "" {
				effectiveTimeout = runInternalTimeout
			}
			if runInternalIterTimeout != "" {
				effectiveIterTimeout = runInternalIterTimeout
			}
		} else {
			// Apply config defaults if CLI flags not specified
			if effectiveTimeout == "" && appConfig.Timeout != "" {
				effectiveTimeout = appConfig.Timeout
			}
			if effectiveIterTimeout == "" && appConfig.IterTimeout != "" {
				effectiveIterTimeout = appConfig.IterTimeout
			}
		}

		if effectiveTimeout != "" {
			var err error
			totalTimeout, err = time.ParseDuration(effectiveTimeout)
			if err != nil {
				return fmt.Errorf("invalid timeout format %q: %w", effectiveTimeout, err)
			}
			if totalTimeout < 0 {
				return fmt.Errorf("timeout cannot be negative: %s", effectiveTimeout)
			}
		}
		if effectiveIterTimeout != "" {
			var err error
			iterTimeout, err = time.ParseDuration(effectiveIterTimeout)
			if err != nil {
				return fmt.Errorf("invalid iter-timeout format %q: %w", effectiveIterTimeout, err)
			}
			if iterTimeout < 0 {
				return fmt.Errorf("iter-timeout cannot be negative: %s", effectiveIterTimeout)
			}
		}

		// Handle detached mode
		if runDetach && !runInternalDetached {
			// Use pre-generated task ID for log file
			logFile, err := detach.LogFilePath(taskID)
			if err != nil {
				return fmt.Errorf("failed to create log file path: %w", err)
			}

			// Build args for the detached process
			detachedArgs := []string{"run", "--_internal-detached", "--_internal-task-id", taskID}
			if globalFlag {
				detachedArgs = append(detachedArgs, "--global")
			}
			if runModel != "" {
				detachedArgs = append(detachedArgs, "--model", runModel)
			}
			if runPrompt != "" {
				detachedArgs = append(detachedArgs, "--prompt", runPrompt)
			}
			if runPromptFile != "" {
				detachedArgs = append(detachedArgs, "--prompt-file", runPromptFile)
			}
			if runPromptString != "" {
				detachedArgs = append(detachedArgs, "--prompt-string", runPromptString)
			}
			// Pass stdin content to child (already read in parent)
			if runStdin && stdinContent != "" {
				detachedArgs = append(detachedArgs, "--stdin", "--_internal-stdin", stdinContent)
			}
			if cmd.Flags().Changed("iterations") {
				detachedArgs = append(detachedArgs, "--iterations", strconv.Itoa(runIterations))
			}
			if runName != "" {
				detachedArgs = append(detachedArgs, "--name", runName)
			}
			// Pass expanded env vars to child (already expanded in parent)
			for _, e := range expandedEnv {
				detachedArgs = append(detachedArgs, "--_internal-env", e)
			}
			// Pass timeout values to child
			if effectiveTimeout != "" {
				detachedArgs = append(detachedArgs, "--_internal-timeout", effectiveTimeout)
			}
			if effectiveIterTimeout != "" {
				detachedArgs = append(detachedArgs, "--_internal-iter-timeout", effectiveIterTimeout)
			}
			// Pass working dir to child if specified (use resolved absolute path)
			if runWorkingDir != "" {
				detachedArgs = append(detachedArgs, "--working-dir", workingDir)
			}

			// Start detached process
			pid, err := detach.StartDetached(detachedArgs, logFile, workingDir)
			if err != nil {
				return fmt.Errorf("failed to start detached process: %w", err)
			}

			// Register agent state
			mgr, err := state.NewManagerWithScope(GetScope(), workingDir)
			if err != nil {
				return fmt.Errorf("failed to initialize state manager: %w", err)
			}

			// Calculate timeout_at if total timeout is set
			var timeoutAt *time.Time
			if totalTimeout > 0 {
				t := time.Now().Add(totalTimeout)
				timeoutAt = &t
			}

			agentState := &state.AgentState{
				ID:          taskID,
				Name:        effectiveName,
				PID:         pid,
				Prompt:      promptName,
				Model:       effectiveModel,
				StartedAt:   time.Now(),
				Iterations:  effectiveIterations,
				CurrentIter: 0,
				Status:      "running",
				LogFile:     logFile,
				WorkingDir:  workingDir,
				EnvNames:    envNames,
				TimeoutAt:   timeoutAt,
			}

			if err := mgr.Register(agentState); err != nil {
				return fmt.Errorf("failed to register agent: %w", err)
			}

			fmt.Printf("Started detached agent: %s (PID: %d)\n", taskID, pid)
			fmt.Printf("Name: %s\n", agentState.Name)
			fmt.Printf("Iterations: %d\n", effectiveIterations)
			if totalTimeout > 0 {
				fmt.Printf("Timeout: %v\n", totalTimeout)
			}
			if iterTimeout > 0 {
				fmt.Printf("Iteration timeout: %v\n", iterTimeout)
			}
			fmt.Printf("Log file: %s\n", logFile)
			return nil
		}

		// For single iteration, run directly without state management overhead
		if effectiveIterations == 1 {
			fmt.Printf("Running agent with prompt: %s, model: %s\n", promptName, effectiveModel)

			// Use iter-timeout for single iteration, or total timeout if only that is set
			singleIterTimeout := iterTimeout
			if singleIterTimeout == 0 && totalTimeout > 0 {
				singleIterTimeout = totalTimeout
			}

			cfg := agent.Config{
				Model:   effectiveModel,
				Prompt:  promptContent,
				Command: appConfig.Command,
				Env:     expandedEnv,
				Timeout: singleIterTimeout,
			}

			runner := agent.NewRunner(cfg)
			err := runner.Run(os.Stdout)
			if err != nil && strings.Contains(err.Error(), "timed out") {
				fmt.Printf("\n[swarm] %v\n", err)
				os.Exit(124) // Exit code 124 matches GNU timeout convention
			}
			return err
		}

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), workingDir)
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		var agentState *state.AgentState
		if runInternalDetached {
			// Detached child: retrieve existing state registered by parent
			agentState, err = mgr.Get(taskID)
			if err != nil {
				return fmt.Errorf("failed to get agent state: %w", err)
			}
		} else {
			// Calculate timeout_at if total timeout is set
			var timeoutAt *time.Time
			if totalTimeout > 0 {
				t := time.Now().Add(totalTimeout)
				timeoutAt = &t
			}

			// Register this agent with working directory
			agentState = &state.AgentState{
				ID:          taskID,
				Name:        effectiveName,
				PID:         os.Getpid(),
				Prompt:      promptName,
				Model:       effectiveModel,
				StartedAt:   time.Now(),
				Iterations:  effectiveIterations,
				CurrentIter: 0,
				Status:      "running",
				WorkingDir:  workingDir,
				EnvNames:    envNames,
				TimeoutAt:   timeoutAt,
			}

			if err := mgr.Register(agentState); err != nil {
				return fmt.Errorf("failed to register agent: %w", err)
			}
		}

		// Multi-iteration mode with state management
		fmt.Printf("Starting agent '%s' with prompt: %s, model: %s, iterations: %d\n", agentState.Name, promptName, effectiveModel, effectiveIterations)
		if totalTimeout > 0 {
			fmt.Printf("Total timeout: %v\n", totalTimeout)
		}
		if iterTimeout > 0 {
			fmt.Printf("Iteration timeout: %v\n", iterTimeout)
		}

		// Set up total timeout context
		var timeoutCtx context.Context
		var timeoutCancel context.CancelFunc
		if totalTimeout > 0 {
			timeoutCtx, timeoutCancel = context.WithTimeout(context.Background(), totalTimeout)
			defer timeoutCancel()
		} else {
			timeoutCtx = context.Background()
		}

		// Track if we timed out for proper exit code
		timedOut := false

		// Ensure cleanup on exit
		defer func() {
			if timedOut {
				agentState.TimeoutReason = "total"
			}
			agentState.Status = "terminated"
			now := time.Now()
			agentState.TerminatedAt = &now
			if agentState.ExitReason == "" {
				agentState.ExitReason = "completed"
			}
			_ = mgr.Update(agentState)
			if timedOut {
				os.Exit(124) // Exit code 124 matches GNU timeout convention
			}
		}()

		// Handle signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Run iterations
		for i := 1; i <= agentState.Iterations; i++ {
			// Check for total timeout before starting iteration
			select {
			case <-timeoutCtx.Done():
				fmt.Println("\n[swarm] Total timeout reached, stopping")
				timedOut = true
				return nil
			default:
				// Continue
			}
			// Check for control signals from state
			currentState, err := mgr.Get(agentState.ID)
			if err == nil && currentState != nil {
				// Update iterations if changed
				if currentState.Iterations != agentState.Iterations {
					agentState.Iterations = currentState.Iterations
					fmt.Printf("\n[swarm] Iterations updated to %d\n", agentState.Iterations)
				}

				// Update model if changed
				if currentState.Model != agentState.Model {
					agentState.Model = currentState.Model
					fmt.Printf("\n[swarm] Model updated to %s\n", agentState.Model)
				}

				// Check for termination
				if currentState.TerminateMode == "immediate" {
					fmt.Println("\n[swarm] Received immediate termination signal")
					agentState.ExitReason = "killed"
					return nil
				}
				if currentState.TerminateMode == "after_iteration" && i > 1 {
					fmt.Println("\n[swarm] Terminating after iteration as requested")
					agentState.ExitReason = "killed"
					return nil
				}

				// Check for pause state and wait while paused
				if currentState.Paused {
					fmt.Println("\n[swarm] Agent paused, waiting for resume...")
					agentState.Paused = true
					now := time.Now()
					agentState.PausedAt = &now
					_ = mgr.Update(agentState)

					for currentState.Paused && currentState.Status == "running" {
						time.Sleep(1 * time.Second)
						currentState, err = mgr.Get(agentState.ID)
						if err != nil {
							break
						}
						// Allow termination while paused
						if currentState.TerminateMode != "" {
							if currentState.TerminateMode == "immediate" {
								fmt.Println("\n[swarm] Received immediate termination signal")
								agentState.ExitReason = "killed"
								return nil
							}
							break
						}
					}

					if !currentState.Paused {
						fmt.Println("\n[swarm] Agent resumed")
						agentState.Paused = false
						agentState.PausedAt = nil
						_ = mgr.Update(agentState)
					}
				}
			}

			// Update current iteration
			agentState.CurrentIter = i
			_ = mgr.Update(agentState)

			fmt.Printf("\n[swarm] === Iteration %d/%d ===\n", i, agentState.Iterations)

			// Create agent config with per-iteration timeout
			cfg := agent.Config{
				Model:   agentState.Model,
				Prompt:  promptContent,
				Command: appConfig.Command,
				Env:     expandedEnv,
				Timeout: iterTimeout,
			}

			// Run agent - errors should NOT stop the run (including iteration timeouts)
			runner := agent.NewRunner(cfg)
			if err := runner.RunWithContext(timeoutCtx, os.Stdout); err != nil {
				agentState.FailedIters++
				agentState.LastError = err.Error()
				if strings.Contains(err.Error(), "timed out") {
					fmt.Printf("\n[swarm] Iteration %d timed out after %v (continuing)\n", i, iterTimeout)
					// Record that this iteration timed out
					agentState.TimeoutReason = "iteration"
					_ = mgr.Update(agentState)
					// Reset timeout reason after recording (will be set to "total" if total timeout hit)
					agentState.TimeoutReason = ""
				} else {
					fmt.Printf("\n[swarm] Agent error (continuing): %v\n", err)
				}
			} else {
				agentState.SuccessfulIters++
			}
			_ = mgr.Update(agentState)

			// Check for signals and total timeout
			select {
			case sig := <-sigChan:
				fmt.Printf("\n[swarm] Received signal %v, stopping\n", sig)
				agentState.ExitReason = "signal"
				return nil
			case <-timeoutCtx.Done():
				fmt.Println("\n[swarm] Total timeout reached, stopping")
				timedOut = true
				return nil
			default:
				// Continue
			}
		}

		fmt.Printf("\n[swarm] Run completed (%d iterations)\n", agentState.Iterations)
		return nil
	},
}

func init() {
	runCmd.Flags().StringVarP(&runModel, "model", "m", "", "Model to use for the agent (overrides config)")
	runCmd.Flags().StringVarP(&runPrompt, "prompt", "p", "", "Prompt name (from prompts directory)")
	runCmd.Flags().StringVarP(&runPromptFile, "prompt-file", "f", "", "Path to prompt file")
	runCmd.Flags().StringVarP(&runPromptString, "prompt-string", "s", "", "Prompt string (direct text)")
	runCmd.Flags().BoolVarP(&runStdin, "stdin", "i", false, "Read prompt content from stdin")
	runCmd.Flags().IntVarP(&runIterations, "iterations", "n", 1, "Number of iterations to run (default: 1)")
	runCmd.Flags().StringVarP(&runName, "name", "N", "", "Name for the agent (for easier reference)")
	runCmd.Flags().BoolVarP(&runDetach, "detach", "d", false, "Run in detached mode (background)")
	runCmd.Flags().StringArrayVarP(&runEnv, "env", "e", nil, "Set environment variables (KEY=VALUE or KEY to pass from shell)")
	runCmd.Flags().StringVar(&runTimeout, "timeout", "", "Total timeout for run (e.g., 30m, 2h)")
	runCmd.Flags().StringVar(&runIterTimeout, "iter-timeout", "", "Timeout per iteration (e.g., 10m)")
	runCmd.Flags().BoolVar(&runInternalDetached, "_internal-detached", false, "Internal flag for detached execution")
	runCmd.Flags().MarkHidden("_internal-detached")
	runCmd.Flags().StringVar(&runInternalTaskID, "_internal-task-id", "", "Internal flag for passing task ID to detached child")
	runCmd.Flags().MarkHidden("_internal-task-id")
	runCmd.Flags().StringVar(&runInternalStdin, "_internal-stdin", "", "Internal flag for passing stdin content to detached child")
	runCmd.Flags().MarkHidden("_internal-stdin")
	runCmd.Flags().StringArrayVar(&runInternalEnv, "_internal-env", nil, "Internal flag for passing env vars to detached child")
	runCmd.Flags().MarkHidden("_internal-env")
	runCmd.Flags().StringVar(&runInternalTimeout, "_internal-timeout", "", "Internal flag for passing timeout to detached child")
	runCmd.Flags().MarkHidden("_internal-timeout")
	runCmd.Flags().StringVar(&runInternalIterTimeout, "_internal-iter-timeout", "", "Internal flag for passing iter-timeout to detached child")
	runCmd.Flags().MarkHidden("_internal-iter-timeout")
	runCmd.Flags().StringVarP(&runWorkingDir, "working-dir", "C", "", "Run agent in specified directory")
}
