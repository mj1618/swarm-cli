package cmd

import (
	"fmt"
	"os"
	"os/signal"
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
	restartModel      string
	restartIterations int
	restartName       string
	restartDetach     bool
)

var restartCmd = &cobra.Command{
	Use:   "restart [agent-id-or-name]",
	Short: "Restart a terminated agent",
	Long: `Restart a terminated agent with its original configuration.

The agent can be specified by its ID or name.

If the original name is taken by a running agent, a number suffix (-2, -3, etc.)
will be appended automatically to make the name unique.

You can optionally override the model, iterations, or name.`,
	Example: `  # Restart by ID
  swarm restart abc123

  # Restart by name
  swarm restart my-agent

  # Restart in detached mode
  swarm restart my-agent -d

  # Override iterations
  swarm restart my-agent -n 20

  # Override model
  swarm restart my-agent -m claude-sonnet-4-20250514

  # Override name
  swarm restart my-agent -N new-name`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentIdentifier := args[0]

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

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), workingDir)
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// Find the agent to restart
		oldAgent, err := mgr.GetByNameOrID(agentIdentifier)
		if err != nil {
			return fmt.Errorf("agent not found: %w", err)
		}

		// Validate agent is terminated
		if oldAgent.Status != "terminated" {
			return fmt.Errorf("agent is not terminated (status: %s), use 'swarm kill' first", oldAgent.Status)
		}

		// Load the prompt content
		var promptContent string
		promptName := oldAgent.Prompt

		// Determine prompt type and load accordingly
		if promptName == "<string>" {
			return fmt.Errorf("cannot restart agent with inline string prompt (prompt content not stored)")
		} else if strings.Contains(promptName, "/") {
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

		// Determine effective values (use overrides if provided, else original)
		effectiveModel := oldAgent.Model
		if cmd.Flags().Changed("model") {
			effectiveModel = restartModel
		}

		effectiveIterations := oldAgent.Iterations
		if cmd.Flags().Changed("iterations") {
			effectiveIterations = restartIterations
		}

		effectiveName := oldAgent.Name
		if cmd.Flags().Changed("name") {
			effectiveName = restartName
		}

		// Use original working directory for the restarted agent
		effectiveWorkingDir := oldAgent.WorkingDir

		// Handle detached mode
		if restartDetach {
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
			detachedArgs = append(detachedArgs, "--model", effectiveModel)

			// Determine how to pass the prompt
			if strings.Contains(promptName, "/") {
				detachedArgs = append(detachedArgs, "--prompt-file", promptName)
			} else {
				detachedArgs = append(detachedArgs, "--prompt", promptName)
			}

			detachedArgs = append(detachedArgs, "--iterations", strconv.Itoa(effectiveIterations))
			if effectiveName != "" {
				detachedArgs = append(detachedArgs, "--name", effectiveName)
			}

			// Start detached process
			pid, err := detach.StartDetached(detachedArgs, logFile, effectiveWorkingDir)
			if err != nil {
				return fmt.Errorf("failed to start detached process: %w", err)
			}

			// Register agent state
			agentState := &state.AgentState{
				ID:          agentID,
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
			}

			if err := mgr.Register(agentState); err != nil {
				return fmt.Errorf("failed to register agent: %w", err)
			}

			fmt.Printf("Restarted agent as detached: %s (PID: %d)\n", agentID, pid)
			fmt.Printf("Name: %s\n", agentState.Name)
			fmt.Printf("Iterations: %d\n", effectiveIterations)
			fmt.Printf("Log file: %s\n", logFile)
			return nil
		}

		// For single iteration, run directly without state management overhead
		if effectiveIterations == 1 {
			fmt.Printf("Restarting agent with prompt: %s, model: %s\n", promptName, effectiveModel)

			cfg := agent.Config{
				Model:   effectiveModel,
				Prompt:  promptContent,
				Command: appConfig.Command,
			}

			runner := agent.NewRunner(cfg)
			return runner.Run(os.Stdout)
		}

		// Register this agent with working directory
		agentState := &state.AgentState{
			ID:          state.GenerateID(),
			Name:        effectiveName,
			PID:         os.Getpid(),
			Prompt:      promptName,
			Model:       effectiveModel,
			StartedAt:   time.Now(),
			Iterations:  effectiveIterations,
			CurrentIter: 0,
			Status:      "running",
			WorkingDir:  effectiveWorkingDir,
		}

		if err := mgr.Register(agentState); err != nil {
			return fmt.Errorf("failed to register agent: %w", err)
		}

		// Multi-iteration mode with state management
		fmt.Printf("Restarting agent '%s' with prompt: %s, model: %s, iterations: %d\n", agentState.Name, promptName, effectiveModel, effectiveIterations)

		// Ensure cleanup on exit
		defer func() {
			agentState.Status = "terminated"
			_ = mgr.Update(agentState)
		}()

		// Handle signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Run iterations
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

				// Check for pause state and wait while paused
				if currentState.Paused {
					fmt.Println("\n[swarm] Agent paused, waiting for resume...")
					agentState.Paused = true
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
								return nil
							}
							break
						}
					}

					if !currentState.Paused {
						fmt.Println("\n[swarm] Agent resumed")
						agentState.Paused = false
						_ = mgr.Update(agentState)
					}
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

			// Run agent - errors should NOT stop the run
			runner := agent.NewRunner(cfg)
			if err := runner.Run(os.Stdout); err != nil {
				fmt.Printf("\n[swarm] Agent error (continuing): %v\n", err)
			}

			// Check for signals
			select {
			case sig := <-sigChan:
				fmt.Printf("\n[swarm] Received signal %v, stopping\n", sig)
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
	restartCmd.Flags().StringVarP(&restartModel, "model", "m", "", "Model to use (overrides original)")
	restartCmd.Flags().IntVarP(&restartIterations, "iterations", "n", 0, "Number of iterations (overrides original)")
	restartCmd.Flags().StringVarP(&restartName, "name", "N", "", "Name for the agent (overrides original)")
	restartCmd.Flags().BoolVarP(&restartDetach, "detach", "d", false, "Run in detached mode (background)")
}
