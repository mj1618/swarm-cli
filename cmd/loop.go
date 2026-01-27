package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
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
	loopModel            string
	loopPrompt           string
	loopPromptFile       string
	loopPromptString     string
	loopIterations       int
	loopName             string
	loopDetach           bool
	loopInternalDetached bool
)

var loopCmd = &cobra.Command{
	Use:   "loop",
	Short: "Run an agent in a loop",
	Long:  `Run an agent repeatedly for a specified number of iterations.`,
	Example: `  # Run 10 iterations with a named prompt
  swarm loop -p my-prompt -n 10

  # Run with a custom name for easy reference
  swarm loop -p my-prompt -n 5 -N my-agent

  # Run in background
  swarm loop -p my-prompt -n 20 -d

  # Run with a specific model
  swarm loop -p my-prompt -n 10 -m claude-sonnet-4-20250514`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get prompts directory based on scope
		promptsDir, err := GetPromptsDir()
		if err != nil {
			return fmt.Errorf("failed to get prompts directory: %w", err)
		}

		// Get current working directory for state tracking
		workingDir, err := scope.CurrentWorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Load or select prompt
		var promptContent string
		var promptName string

		// Count how many prompt sources were specified
		specifiedCount := 0
		if loopPrompt != "" {
			specifiedCount++
		}
		if loopPromptFile != "" {
			specifiedCount++
		}
		if loopPromptString != "" {
			specifiedCount++
		}

		if specifiedCount > 1 {
			return fmt.Errorf("only one of --prompt, --prompt-file, or --prompt-string can be specified")
		}

		switch {
		case loopPromptFile != "":
			// Load from arbitrary file path
			promptName = loopPromptFile
			promptContent, err = prompt.LoadPromptFromFile(loopPromptFile)
			if err != nil {
				return fmt.Errorf("failed to load prompt file: %w", err)
			}
		case loopPromptString != "":
			// Use direct string
			promptName = "<string>"
			promptContent = prompt.WrapPromptString(loopPromptString)
		case loopPrompt != "":
			// Load from prompts directory
			promptName = loopPrompt
			promptContent, err = prompt.LoadPrompt(promptsDir, loopPrompt)
			if err != nil {
				return fmt.Errorf("failed to load prompt: %w", err)
			}
		default:
			// Interactive selection not allowed in detached mode
			if loopDetach {
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
			effectiveModel = loopModel
		}

		// Determine effective iterations (CLI flag overrides config)
		effectiveIterations := appConfig.Iterations
		if cmd.Flags().Changed("iterations") {
			effectiveIterations = loopIterations
		}

		// Handle detached mode
		if loopDetach && !loopInternalDetached {
			// Generate agent ID and log file
			agentID := state.GenerateID()
			logFile, err := detach.LogFilePath(agentID)
			if err != nil {
				return fmt.Errorf("failed to create log file path: %w", err)
			}

			// Build args for the detached process
			detachedArgs := []string{"loop", "--_internal-detached"}
			if globalFlag {
				detachedArgs = append(detachedArgs, "--global")
			}
			if loopModel != "" {
				detachedArgs = append(detachedArgs, "--model", loopModel)
			}
			if loopPrompt != "" {
				detachedArgs = append(detachedArgs, "--prompt", loopPrompt)
			}
			if loopPromptFile != "" {
				detachedArgs = append(detachedArgs, "--prompt-file", loopPromptFile)
			}
			if loopPromptString != "" {
				detachedArgs = append(detachedArgs, "--prompt-string", loopPromptString)
			}
			if loopIterations > 0 {
				detachedArgs = append(detachedArgs, "--iterations", strconv.Itoa(loopIterations))
			}
			if loopName != "" {
				detachedArgs = append(detachedArgs, "--name", loopName)
			}

			// Start detached process
			pid, err := detach.StartDetached(detachedArgs, logFile, workingDir)
			if err != nil {
				return fmt.Errorf("failed to start detached process: %w", err)
			}

			// Register agent state (the detached process will update it)
			mgr, err := state.NewManagerWithScope(GetScope(), workingDir)
			if err != nil {
				return fmt.Errorf("failed to initialize state manager: %w", err)
			}

			agentState := &state.AgentState{
				ID:          agentID,
				Name:        loopName,
				PID:         pid,
				Prompt:      promptName,
				Model:       effectiveModel,
				StartedAt:   time.Now(),
				Iterations:  effectiveIterations,
				CurrentIter: 0,
				Status:      "running",
				LogFile:     logFile,
				WorkingDir:  workingDir,
			}

			if err := mgr.Register(agentState); err != nil {
				return fmt.Errorf("failed to register agent: %w", err)
			}

			fmt.Printf("Started detached loop: %s (PID: %d)\n", agentID, pid)
			if loopName != "" {
				fmt.Printf("Name: %s\n", loopName)
			}
			fmt.Printf("Iterations: %d\n", effectiveIterations)
			fmt.Printf("Log file: %s\n", logFile)
			return nil
		}

		if loopName != "" {
			fmt.Printf("Starting loop '%s' with prompt: %s, model: %s, iterations: %d\n", loopName, promptName, effectiveModel, effectiveIterations)
		} else {
			fmt.Printf("Starting loop with prompt: %s, model: %s, iterations: %d\n", promptName, effectiveModel, effectiveIterations)
		}

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), workingDir)
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// Register this agent loop with working directory
		agentState := &state.AgentState{
			ID:          state.GenerateID(),
			Name:        loopName,
			PID:         os.Getpid(),
			Prompt:      promptName,
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

		// Ensure cleanup on exit
		defer func() {
			agentState.Status = "terminated"
			_ = mgr.Update(agentState)
		}()

		// Handle signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Run loop
		for i := 1; i <= agentState.Iterations; i++ {
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
					return nil
				}
				if currentState.TerminateMode == "after_iteration" && i > 1 {
					fmt.Println("\n[swarm] Terminating after iteration as requested")
					return nil
				}
			}

			// Update current iteration
			agentState.CurrentIter = i
			_ = mgr.Update(agentState)

			fmt.Printf("\n[swarm] === Iteration %d/%d ===\n", i, agentState.Iterations)

			// Create agent config
			cfg := agent.Config{
				Model:   agentState.Model,
				Prompt:  promptContent,
				Command: appConfig.Command,
			}

			// Run agent - errors should NOT stop the loop
			runner := agent.NewRunner(cfg)
			if err := runner.Run(os.Stdout); err != nil {
				fmt.Printf("\n[swarm] Agent error (continuing): %v\n", err)
			}

			// Check for signals
			select {
			case sig := <-sigChan:
				fmt.Printf("\n[swarm] Received signal %v, stopping loop\n", sig)
				return nil
			default:
				// Continue
			}
		}

		fmt.Printf("\n[swarm] Loop completed (%d iterations)\n", agentState.Iterations)
		return nil
	},
}

func init() {
	loopCmd.Flags().StringVarP(&loopModel, "model", "m", "", "Model to use for the agent (overrides config)")
	loopCmd.Flags().StringVarP(&loopPrompt, "prompt", "p", "", "Prompt name (from prompts directory)")
	loopCmd.Flags().StringVarP(&loopPromptFile, "prompt-file", "f", "", "Path to prompt file")
	loopCmd.Flags().StringVarP(&loopPromptString, "prompt-string", "s", "", "Prompt string (direct text)")
	loopCmd.Flags().IntVarP(&loopIterations, "iterations", "n", 0, "Number of iterations to run (overrides config)")
	loopCmd.Flags().StringVarP(&loopName, "name", "N", "", "Name for the agent (for easier reference)")
	loopCmd.Flags().BoolVarP(&loopDetach, "detach", "d", false, "Run in detached mode (background)")
	loopCmd.Flags().BoolVar(&loopInternalDetached, "_internal-detached", false, "Internal flag for detached execution")
	loopCmd.Flags().MarkHidden("_internal-detached")
}
