package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mj1618/swarm-cli/internal/agent"
	"github.com/mj1618/swarm-cli/internal/detach"
	"github.com/mj1618/swarm-cli/internal/prompt"
	"github.com/mj1618/swarm-cli/internal/runner"
	"github.com/mj1618/swarm-cli/internal/scope"
	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	cloneName       string
	cloneIterations int
	cloneForever    bool
	cloneModel      string
	cloneDetach     bool
	cloneForeground bool
	cloneSameDir    bool
	cloneDryRun     bool
	cloneEnv        []string
	cloneOnComplete string
)

var cloneCmd = &cobra.Command{
	Use:   "clone [task-id-or-name]",
	Short: "Clone an agent's configuration to start a new agent",
	Long: `Clone an existing agent's configuration to start a new agent.

This is useful for:
  - Re-running a completed agent
  - Running multiple agents with the same prompt
  - Running with slight configuration changes

The source agent can be running or terminated.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent

By default, the cloned agent runs in the current directory. Use --same-dir
to run in the source agent's original directory.`,
	Example: `  # Clone an agent to run it again
  swarm clone abc123

  # Clone by name
  swarm clone my-agent

  # Clone the most recent agent
  swarm clone @last
  swarm clone _

  # Clone with a new name
  swarm clone my-agent --name my-agent-v2

  # Clone with different iterations
  swarm clone abc123 -n 50

  # Clone with different model
  swarm clone abc123 -m claude-sonnet-4-20250514

  # Clone in background
  swarm clone abc123 -d

  # Clone in same directory as source
  swarm clone abc123 --same-dir

  # See what command would be run without executing
  swarm clone abc123 --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceIdentifier := args[0]

		// Get current working directory
		currentDir, err := scope.CurrentWorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), currentDir)
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// Find the source agent
		source, err := ResolveAgentIdentifier(mgr, sourceIdentifier)
		if err != nil {
			return fmt.Errorf("source agent not found: %w", err)
		}

		// Determine configuration (source values with overrides)
		promptName := source.Prompt
		effectiveModel := source.Model
		effectiveIterations := source.Iterations
		effectiveName := ""

		// Determine if source was detached (had a log file means it was detached)
		sourceWasDetached := source.LogFile != ""
		effectiveDetach := sourceWasDetached

		// Apply overrides
		if cmd.Flags().Changed("iterations") {
			effectiveIterations = cloneIterations
		}
		if cloneForever {
			effectiveIterations = 0
		}
		if cmd.Flags().Changed("model") {
			effectiveModel = cloneModel
		}
		if cmd.Flags().Changed("name") {
			effectiveName = cloneName
		} else if source.Name != "" {
			// Auto-generate name from source: "name" -> "name-clone"
			effectiveName = source.Name + "-clone"
		}

		// Handle detach/foreground flags
		if cloneForeground {
			effectiveDetach = false
		} else if cmd.Flags().Changed("detach") {
			effectiveDetach = cloneDetach
		}

		// Validate that --forever and explicit -n (with value > 0) aren't both specified
		if cloneForever && cmd.Flags().Changed("iterations") && cloneIterations > 0 {
			return fmt.Errorf("cannot use --forever with --iterations (use -n 0 for unlimited)")
		}

		// Determine working directory
		effectiveWorkingDir := currentDir
		if cloneSameDir && source.WorkingDir != "" {
			effectiveWorkingDir = source.WorkingDir
		}

		// Build the equivalent run command for dry-run
		var cmdParts []string
		cmdParts = append(cmdParts, "swarm", "run")

		// Determine how to pass the prompt
		if promptName == "<string>" {
			return fmt.Errorf("cannot clone agent with inline string prompt (prompt content not stored)")
		} else if strings.Contains(promptName, "/") {
			cmdParts = append(cmdParts, "-f", promptName)
		} else if strings.HasSuffix(promptName, "+stdin") {
			return fmt.Errorf("cannot clone agent with stdin-combined prompt (prompt content not stored)")
		} else if promptName == "<stdin>" {
			return fmt.Errorf("cannot clone agent with stdin prompt (prompt content not stored)")
		} else {
			cmdParts = append(cmdParts, "-p", promptName)
		}

		if effectiveIterations == 0 {
			cmdParts = append(cmdParts, "--forever")
		} else {
			cmdParts = append(cmdParts, "-n", fmt.Sprintf("%d", effectiveIterations))
		}
		cmdParts = append(cmdParts, "-m", effectiveModel)
		if effectiveName != "" {
			cmdParts = append(cmdParts, "--name", effectiveName)
		}
		if effectiveDetach {
			cmdParts = append(cmdParts, "-d")
		}
		if cloneSameDir && source.WorkingDir != "" {
			cmdParts = append(cmdParts, "-C", source.WorkingDir)
		}

		if cloneDryRun {
			fmt.Println(strings.Join(cmdParts, " "))
			return nil
		}

		// Get prompts directory based on scope and working directory
		var promptsDir string
		if cloneSameDir && source.WorkingDir != "" && GetScope() == scope.ScopeProject {
			promptsDir = source.WorkingDir + "/swarm/prompts"
		} else {
			promptsDir, err = GetPromptsDir()
			if err != nil {
				return fmt.Errorf("failed to get prompts directory: %w", err)
			}
		}

		// Load the prompt content
		var promptContent string
		if strings.Contains(promptName, "/") {
			// File path - load from file
			promptContent, err = prompt.LoadPromptFromFile(promptName)
			if err != nil {
				return fmt.Errorf("failed to load prompt file: %w", err)
			}
		} else {
			// Named prompt - load from prompts directory
			promptContent, err = prompt.LoadPrompt(promptsDir, promptName)
			if err != nil {
				return fmt.Errorf("failed to load prompt: %w", err)
			}
		}

		// Generate task ID early so it can be injected into prompt
		taskID := state.GenerateID()

		// Inject task ID into prompt content
		promptContent = prompt.InjectTaskID(promptContent, taskID)

		// Parse and expand environment variables
		var expandedEnv []string
		var envNames []string
		if len(cloneEnv) > 0 {
			expandedEnv = make([]string, 0, len(cloneEnv))
			for _, e := range cloneEnv {
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

		// Handle detached mode
		if effectiveDetach {
			logFile, err := detach.LogFilePath(taskID)
			if err != nil {
				return fmt.Errorf("failed to create log file path: %w", err)
			}

			// Build args for the detached process
			detachedArgs := []string{"run", "--_internal-detached", "--_internal-task-id", taskID}
			if globalFlag {
				detachedArgs = append(detachedArgs, "--global")
			}
			detachedArgs = append(detachedArgs, "--model", effectiveModel)

			// Determine how to pass the prompt
			if strings.Contains(promptName, "/") {
				detachedArgs = append(detachedArgs, "--prompt-file", promptName)
			} else {
				detachedArgs = append(detachedArgs, "--prompt", promptName)
			}

			if effectiveIterations == 0 {
				detachedArgs = append(detachedArgs, "--forever")
			} else {
				detachedArgs = append(detachedArgs, "--iterations", strconv.Itoa(effectiveIterations))
			}
			if effectiveName != "" {
				detachedArgs = append(detachedArgs, "--name", effectiveName)
			}
			// Pass expanded env vars to child
			for _, e := range expandedEnv {
				detachedArgs = append(detachedArgs, "--_internal-env", e)
			}
			// Pass on-complete hook to child
			if cloneOnComplete != "" {
				detachedArgs = append(detachedArgs, "--_internal-on-complete", cloneOnComplete)
			}

			// Start detached process
			pid, err := detach.StartDetached(detachedArgs, logFile, effectiveWorkingDir)
			if err != nil {
				return fmt.Errorf("failed to start detached process: %w", err)
			}

			// Register agent state
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
				WorkingDir:  effectiveWorkingDir,
				EnvNames:    envNames,
				OnComplete:  cloneOnComplete,
			}

			if err := mgr.Register(agentState); err != nil {
				return fmt.Errorf("failed to register agent: %w", err)
			}

			fmt.Printf("Cloned agent %s as: %s (PID: %d)\n", source.ID, taskID, pid)
			fmt.Printf("Name: %s\n", agentState.Name)
			if effectiveIterations == 0 {
				fmt.Println("Iterations: unlimited")
			} else {
				fmt.Printf("Iterations: %d\n", effectiveIterations)
			}
			fmt.Printf("Log file: %s\n", logFile)
			return nil
		}

		// Warning if running forever in foreground
		if effectiveIterations == 0 {
			fmt.Println("Warning: Running forever in foreground. Press Ctrl+C to stop.")
		}

		// For single iteration, run with state tracking
		if effectiveIterations == 1 {
			// Register single-iteration agent in state
			agentState := &state.AgentState{
				ID:          taskID,
				Name:        effectiveName,
				PID:         os.Getpid(),
				Prompt:      promptName,
				Model:       effectiveModel,
				StartedAt:   time.Now(),
				Iterations:  1,
				CurrentIter: 1,
				Status:      "running",
				WorkingDir:  effectiveWorkingDir,
				EnvNames:    envNames,
				OnComplete:  cloneOnComplete,
			}

			if err := mgr.Register(agentState); err != nil {
				return fmt.Errorf("failed to register agent: %w", err)
			}

			// Ensure cleanup on exit
			defer func() {
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
			}()

			fmt.Printf("Cloning agent %s with prompt: %s, model: %s\n", source.ID, promptName, effectiveModel)

			cfg := agent.Config{
				Model:   effectiveModel,
				Prompt:  promptContent,
				Command: appConfig.Command,
				Env:     expandedEnv,
			}

			agentRunner := agent.NewRunner(cfg)
			err = agentRunner.Run(os.Stdout)
			if err != nil {
				agentState.FailedIters = 1
				agentState.LastError = err.Error()
				return err
			}
			agentState.SuccessfulIters = 1
			return nil
		}

		// Register this agent with working directory
		agentState := &state.AgentState{
			ID:          taskID,
			Name:        effectiveName,
			PID:         os.Getpid(),
			Prompt:      promptName,
			Model:       effectiveModel,
			StartedAt:   time.Now(),
			Iterations:  effectiveIterations,
			CurrentIter: 0,
			Status:      "running",
			WorkingDir:  effectiveWorkingDir,
			EnvNames:    envNames,
			OnComplete:  cloneOnComplete,
		}

		if err := mgr.Register(agentState); err != nil {
			return fmt.Errorf("failed to register agent: %w", err)
		}

		// Multi-iteration mode with state management
		if effectiveIterations == 0 {
			fmt.Printf("Cloning agent %s as '%s' with prompt: %s, model: %s, iterations: unlimited\n", source.ID, agentState.Name, promptName, effectiveModel)
		} else {
			fmt.Printf("Cloning agent %s as '%s' with prompt: %s, model: %s, iterations: %d\n", source.ID, agentState.Name, promptName, effectiveModel, effectiveIterations)
		}

		// Run the multi-iteration loop
		loopCfg := runner.LoopConfig{
			Manager:           mgr,
			AgentState:        agentState,
			PromptContent:     promptContent,
			Command:           appConfig.Command,
			Config:            appConfig,
			Env:               expandedEnv,
			Output:            os.Stdout,
			StartingIteration: 1,
		}

		_, err = runner.RunLoop(loopCfg)
		return err
	},
}

func init() {
	cloneCmd.Flags().StringVarP(&cloneName, "name", "N", "", "Name for the cloned agent")
	cloneCmd.Flags().IntVarP(&cloneIterations, "iterations", "n", 0, "Override iteration count")
	cloneCmd.Flags().BoolVarP(&cloneForever, "forever", "F", false, "Run indefinitely until manually stopped")
	cloneCmd.Flags().StringVarP(&cloneModel, "model", "m", "", "Override model")
	cloneCmd.Flags().BoolVarP(&cloneDetach, "detach", "d", false, "Run in detached mode")
	cloneCmd.Flags().BoolVar(&cloneForeground, "foreground", false, "Run in foreground mode (overrides source mode)")
	cloneCmd.Flags().BoolVar(&cloneSameDir, "same-dir", false, "Run in same directory as source agent")
	cloneCmd.Flags().BoolVar(&cloneDryRun, "dry-run", false, "Print equivalent run command without executing")
	cloneCmd.Flags().StringArrayVarP(&cloneEnv, "env", "e", nil, "Set environment variables (KEY=VALUE or KEY to pass from shell)")
	cloneCmd.Flags().StringVar(&cloneOnComplete, "on-complete", "", "Command to run when agent completes")

	// Add dynamic completion for agent identifier and model flag
	cloneCmd.ValidArgsFunction = completeAgentIdentifier
	cloneCmd.RegisterFlagCompletionFunc("model", completeModelName)
}
