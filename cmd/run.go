package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/matt/swarm-cli/internal/agent"
	"github.com/matt/swarm-cli/internal/detach"
	"github.com/matt/swarm-cli/internal/label"
	"github.com/matt/swarm-cli/internal/prompt"
	"github.com/matt/swarm-cli/internal/runner"
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
	runForever             bool
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
	runInternalStartIter   int
	runOnComplete          string
	runInternalOnComplete  string
	runLabels              []string
	runInternalLabels      []string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run an agent",
	Long: `Run an agent with a specified prompt and model.

By default, runs a single iteration. Use -n to run multiple iterations.
When running multiple iterations, agent failures do not stop the run.

Labels can be attached to agents for categorization and filtering using the
--label (-l) flag. Labels are key-value pairs in the format key=value.`,
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
  swarm run -p frontend -C ./frontend -d

  # Run with labels for categorization
  swarm run -p task -l team=frontend -l priority=high

  # Run with multiple labels
  swarm run -p task -l env=staging -l ticket=PROJ-123 -d`,
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
		// 0 means unlimited (forever mode)
		effectiveIterations := 1
		if runForever {
			effectiveIterations = 0
		} else if cmd.Flags().Changed("iterations") {
			effectiveIterations = runIterations
		}

		// Validate that --forever and explicit -n (with value > 0) aren't both specified
		if runForever && cmd.Flags().Changed("iterations") && runIterations > 0 {
			return fmt.Errorf("cannot use --forever with --iterations (use -n 0 for unlimited)")
		}

		// Warning if running forever in foreground
		if effectiveIterations == 0 && !runDetach {
			fmt.Println("Warning: Running forever in foreground. Press Ctrl+C to stop.")
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

		// Determine effective on-complete hook
		// For detached child, use value passed from parent
		effectiveOnComplete := runOnComplete
		if runInternalDetached && runInternalOnComplete != "" {
			effectiveOnComplete = runInternalOnComplete
		}

		// Parse labels
		// For detached child, use labels passed from parent
		var labels map[string]string
		labelSource := runLabels
		if runInternalDetached && len(runInternalLabels) > 0 {
			labelSource = runInternalLabels
		}
		if len(labelSource) > 0 {
			var err error
			labels, err = label.ParseMultiple(labelSource)
			if err != nil {
				return fmt.Errorf("invalid label: %w", err)
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
			if runForever {
				detachedArgs = append(detachedArgs, "--forever")
			} else if cmd.Flags().Changed("iterations") {
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
			// Pass on-complete hook to child
			if runOnComplete != "" {
				detachedArgs = append(detachedArgs, "--_internal-on-complete", runOnComplete)
			}
			// Pass labels to child
			for _, l := range runLabels {
				detachedArgs = append(detachedArgs, "--_internal-label", l)
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
				Labels:      labels,
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
				OnComplete:  runOnComplete,
			}

			if err := mgr.Register(agentState); err != nil {
				return fmt.Errorf("failed to register agent: %w", err)
			}

			fmt.Printf("Started detached agent: %s (PID: %d)\n", taskID, pid)
			fmt.Printf("Name: %s\n", agentState.Name)
			if effectiveIterations == 0 {
				fmt.Println("Iterations: unlimited")
			} else {
				fmt.Printf("Iterations: %d\n", effectiveIterations)
			}
			if totalTimeout > 0 {
				fmt.Printf("Timeout: %v\n", totalTimeout)
			}
			if iterTimeout > 0 {
				fmt.Printf("Iteration timeout: %v\n", iterTimeout)
			}
			fmt.Printf("Log file: %s\n", logFile)
			return nil
		}

		// For single iteration, run with state tracking but simpler flow (no loop/pause/signal handling)
		if effectiveIterations == 1 {
			// Create state manager with scope
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

			// Register single-iteration agent in state
			agentState := &state.AgentState{
				ID:          taskID,
				Name:        effectiveName,
				Labels:      labels,
				PID:         os.Getpid(),
				Prompt:      promptName,
				Model:       effectiveModel,
				StartedAt:   time.Now(),
				Iterations:  1,
				CurrentIter: 1,
				Status:      "running",
				WorkingDir:  workingDir,
				EnvNames:    envNames,
				TimeoutAt:   timeoutAt,
				OnComplete:  effectiveOnComplete,
			}

			if err := mgr.Register(agentState); err != nil {
				return fmt.Errorf("failed to register agent: %w", err)
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

				// Execute on-complete hook
				if agentState.OnComplete != "" {
					if err := agent.ExecuteOnCompleteHook(agentState); err != nil {
						fmt.Printf("[swarm] Warning: on-complete hook failed: %v\n", err)
					}
				}

				if timedOut {
					os.Exit(124) // Exit code 124 matches GNU timeout convention
				}
			}()

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
			err = runner.Run(os.Stdout)
			if err != nil {
				agentState.FailedIters = 1
				agentState.LastError = err.Error()
				if strings.Contains(err.Error(), "timed out") {
					timedOut = true
					fmt.Printf("\n[swarm] %v\n", err)
					return nil // Let defer handle the exit
				}
				return err
			}
			agentState.SuccessfulIters = 1
			return nil
		}

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), workingDir)
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		var agentState *state.AgentState
		// Calculate starting iteration (usually 1, unless passed via internal flag for --continue)
		startingIteration := 1
		if runInternalStartIter > 0 {
			startingIteration = runInternalStartIter
		}

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
				Labels:      labels,
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
				OnComplete:  effectiveOnComplete,
			}

			if err := mgr.Register(agentState); err != nil {
				return fmt.Errorf("failed to register agent: %w", err)
			}
		}

		// Multi-iteration mode with state management
		if effectiveIterations == 0 {
			fmt.Printf("Starting agent '%s' with prompt: %s, model: %s, iterations: unlimited\n", agentState.Name, promptName, effectiveModel)
		} else {
			fmt.Printf("Starting agent '%s' with prompt: %s, model: %s, iterations: %d\n", agentState.Name, promptName, effectiveModel, effectiveIterations)
		}
		if totalTimeout > 0 {
			fmt.Printf("Total timeout: %v\n", totalTimeout)
		}
		if iterTimeout > 0 {
			fmt.Printf("Iteration timeout: %v\n", iterTimeout)
		}

		// Run the multi-iteration loop
		loopCfg := runner.LoopConfig{
			Manager:           mgr,
			AgentState:        agentState,
			PromptContent:     promptContent,
			Command:           appConfig.Command,
			Env:               expandedEnv,
			Output:            os.Stdout,
			StartingIteration: startingIteration,
			TotalTimeout:      totalTimeout,
			IterTimeout:       iterTimeout,
		}

		result, err := runner.RunLoop(loopCfg)
		if err != nil {
			return err
		}

		// Exit with timeout code if timed out
		if result.TimedOut {
			os.Exit(124) // Exit code 124 matches GNU timeout convention
		}

		return nil
	},
}

func init() {
	runCmd.Flags().StringVarP(&runModel, "model", "m", "", "Model to use for the agent (overrides config)")
	runCmd.Flags().StringVarP(&runPrompt, "prompt", "p", "", "Prompt name (from prompts directory)")
	runCmd.Flags().StringVarP(&runPromptFile, "prompt-file", "f", "", "Path to prompt file")
	runCmd.Flags().StringVarP(&runPromptString, "prompt-string", "s", "", "Prompt string (direct text)")
	runCmd.Flags().BoolVarP(&runStdin, "stdin", "i", false, "Read prompt content from stdin")
	runCmd.Flags().IntVarP(&runIterations, "iterations", "n", 1, "Number of iterations to run (0 = unlimited, default: 1)")
	runCmd.Flags().BoolVarP(&runForever, "forever", "F", false, "Run indefinitely until manually stopped")
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
	runCmd.Flags().IntVar(&runInternalStartIter, "_internal-start-iter", 0, "Internal flag for passing start iteration to detached child")
	runCmd.Flags().MarkHidden("_internal-start-iter")
	runCmd.Flags().StringVarP(&runWorkingDir, "working-dir", "C", "", "Run agent in specified directory")
	runCmd.Flags().StringVar(&runOnComplete, "on-complete", "", "Command to run when agent completes")
	runCmd.Flags().StringVar(&runInternalOnComplete, "_internal-on-complete", "", "Internal flag for passing on-complete to detached child")
	runCmd.Flags().MarkHidden("_internal-on-complete")
	runCmd.Flags().StringArrayVarP(&runLabels, "label", "l", nil, "Label to attach (key=value format, can be repeated)")
	runCmd.Flags().StringArrayVar(&runInternalLabels, "_internal-label", nil, "Internal flag for passing labels to detached child")
	runCmd.Flags().MarkHidden("_internal-label")

	// Add dynamic completion for prompt and model flags
	runCmd.RegisterFlagCompletionFunc("prompt", completePromptName)
	runCmd.RegisterFlagCompletionFunc("model", completeModelName)
}
