package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/matt/swarm-cli/internal/agent"
	"github.com/matt/swarm-cli/internal/detach"
	"github.com/matt/swarm-cli/internal/prompt"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	runModel            string
	runPrompt           string
	runPromptFile       string
	runPromptString     string
	runDetach           bool
	runInternalDetached bool
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a single agent",
	Long:  `Run a single agent with a specified prompt and model.`,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		// Load or select prompt
		var promptContent string
		var promptName string

		// Count how many prompt sources were specified
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

		if specifiedCount > 1 {
			return fmt.Errorf("only one of --prompt, --prompt-file, or --prompt-string can be specified")
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

		// Determine effective model (CLI flag overrides config)
		effectiveModel := appConfig.Model
		if cmd.Flags().Changed("model") {
			effectiveModel = runModel
		}

		// Handle detached mode
		if runDetach && !runInternalDetached {
			// Generate agent ID and log file
			agentID := state.GenerateID()
			logFile, err := detach.LogFilePath(agentID)
			if err != nil {
				return fmt.Errorf("failed to create log file path: %w", err)
			}

			// Build args for the detached process
			detachedArgs := []string{"run", "--_internal-detached"}
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

			agentState := &state.AgentState{
				ID:         agentID,
				PID:        pid,
				Prompt:     promptName,
				Model:      effectiveModel,
				StartedAt:  time.Now(),
				Iterations: 1,
				Status:     "running",
				LogFile:    logFile,
				WorkingDir: workingDir,
			}

			if err := mgr.Register(agentState); err != nil {
				return fmt.Errorf("failed to register agent: %w", err)
			}

			fmt.Printf("Started detached agent: %s (PID: %d)\n", agentID, pid)
			fmt.Printf("Log file: %s\n", logFile)
			return nil
		}

		fmt.Printf("Running agent with prompt: %s, model: %s\n", promptName, effectiveModel)

		// Create and run agent
		cfg := agent.Config{
			Model:   effectiveModel,
			Prompt:  promptContent,
			Command: appConfig.Command,
		}

		runner := agent.NewRunner(cfg)
		return runner.Run(os.Stdout)
	},
}

func init() {
	runCmd.Flags().StringVarP(&runModel, "model", "m", "", "Model to use for the agent (overrides config)")
	runCmd.Flags().StringVarP(&runPrompt, "prompt", "p", "", "Prompt name (from prompts directory)")
	runCmd.Flags().StringVarP(&runPromptFile, "prompt-file", "f", "", "Path to prompt file")
	runCmd.Flags().StringVarP(&runPromptString, "prompt-string", "s", "", "Prompt string (direct text)")
	runCmd.Flags().BoolVarP(&runDetach, "detach", "d", false, "Run in detached mode (background)")
	runCmd.Flags().BoolVar(&runInternalDetached, "_internal-detached", false, "Internal flag for detached execution")
	runCmd.Flags().MarkHidden("_internal-detached")
}
