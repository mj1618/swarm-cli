package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/matt/swarm-cli/internal/agent"
	"github.com/matt/swarm-cli/internal/detach"
	"github.com/matt/swarm-cli/internal/prompt"
	"github.com/matt/swarm-cli/internal/runner"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	restartModel         string
	restartIterations    int
	restartForever       bool
	restartName          string
	restartDetach        bool
	restartEnv           []string
	restartContinue      bool
	restartInternalStart int
	restartOnComplete    string
)

var restartCmd = &cobra.Command{
	Use:   "restart [agent-id-or-name]",
	Short: "Restart a terminated agent",
	Long: `Restart a terminated agent with its original configuration.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent

If the original name is taken by a running agent, a number suffix (-2, -3, etc.)
will be appended automatically to make the name unique.

You can optionally override the model, iterations, or name.

Use --continue to resume from where the agent left off instead of starting
from iteration 1.`,
	Example: `  # Restart by ID
  swarm restart abc123

  # Restart by name
  swarm restart my-agent

  # Restart the most recent agent
  swarm restart @last
  swarm restart _

  # Restart in detached mode
  swarm restart my-agent -d

  # Continue from last iteration (if agent was at 15/20, starts at 16/20)
  swarm restart my-agent --continue

  # Continue with more iterations
  swarm restart my-agent -c -n 30

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
		oldAgent, err := ResolveAgentIdentifier(mgr, agentIdentifier)
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
		if restartForever {
			effectiveIterations = 0
		} else if cmd.Flags().Changed("iterations") {
			effectiveIterations = restartIterations
		}

		// Validate that --forever and explicit -n (with value > 0) aren't both specified
		if restartForever && cmd.Flags().Changed("iterations") && restartIterations > 0 {
			return fmt.Errorf("cannot use --forever with --iterations (use -n 0 for unlimited)")
		}

		// Calculate starting iteration for --continue flag
		startingIteration := 1
		if restartContinue {
			startingIteration = oldAgent.CurrentIter + 1

			// Validate there are iterations remaining (unless unlimited)
			if effectiveIterations > 0 && startingIteration > effectiveIterations {
				return fmt.Errorf("agent already completed all %d iterations; use --iterations to add more", oldAgent.Iterations)
			}

			// If CurrentIter is 0, agent was terminated before completing first iteration
			if startingIteration <= 1 {
				startingIteration = 1
			}

			fmt.Printf("Continuing from iteration %d\n", startingIteration)
		}

		// For detached child process, use the internal start flag
		if restartInternalStart > 0 {
			startingIteration = restartInternalStart
		}

		effectiveName := oldAgent.Name
		if cmd.Flags().Changed("name") {
			effectiveName = restartName
		}

		// Use original working directory for the restarted agent
		effectiveWorkingDir := oldAgent.WorkingDir

		// Parse and expand environment variables (does not preserve original env vars)
		var expandedEnv []string
		var envNames []string
		if len(restartEnv) > 0 {
			expandedEnv = make([]string, 0, len(restartEnv))
			for _, e := range restartEnv {
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

			if effectiveIterations == 0 {
				detachedArgs = append(detachedArgs, "--forever")
			} else {
				detachedArgs = append(detachedArgs, "--iterations", strconv.Itoa(effectiveIterations))
			}
			if effectiveName != "" {
				detachedArgs = append(detachedArgs, "--name", effectiveName)
			}
			// Pass starting iteration if --continue was used
			if restartContinue {
				detachedArgs = append(detachedArgs, "--_internal-start-iter", strconv.Itoa(startingIteration))
			}
			// Pass expanded env vars to child
			for _, e := range expandedEnv {
				detachedArgs = append(detachedArgs, "--_internal-env", e)
			}
			// Pass on-complete hook to child
			if restartOnComplete != "" {
				detachedArgs = append(detachedArgs, "--_internal-on-complete", restartOnComplete)
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
				CurrentIter: startingIteration - 1, // Will be incremented to startingIteration in first loop
				Status:      "running",
				LogFile:     logFile,
				WorkingDir:  effectiveWorkingDir,
				EnvNames:    envNames,
				OnComplete:  restartOnComplete,
			}

			if err := mgr.Register(agentState); err != nil {
				return fmt.Errorf("failed to register agent: %w", err)
			}

			fmt.Printf("Restarted agent as detached: %s (PID: %d)\n", agentID, pid)
			fmt.Printf("Name: %s\n", agentState.Name)
			if effectiveIterations == 0 {
				fmt.Println("Iterations: unlimited")
			} else {
				if startingIteration > 1 {
					fmt.Printf("Iterations: %d (starting from %d)\n", effectiveIterations, startingIteration)
				} else {
					fmt.Printf("Iterations: %d\n", effectiveIterations)
				}
			}
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
				Env:     expandedEnv,
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
			CurrentIter: startingIteration - 1, // Will be incremented to startingIteration in first loop
			Status:      "running",
			WorkingDir:  effectiveWorkingDir,
			EnvNames:    envNames,
			OnComplete:  restartOnComplete,
		}

		if err := mgr.Register(agentState); err != nil {
			return fmt.Errorf("failed to register agent: %w", err)
		}

		// Multi-iteration mode with state management
		if effectiveIterations == 0 {
			fmt.Printf("Restarting agent '%s' with prompt: %s, model: %s, iterations: unlimited\n", agentState.Name, promptName, effectiveModel)
		} else {
			fmt.Printf("Restarting agent '%s' with prompt: %s, model: %s, iterations: %d\n", agentState.Name, promptName, effectiveModel, effectiveIterations)
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
		}

		_, err = runner.RunLoop(loopCfg)
		return err
	},
}

func init() {
	restartCmd.Flags().StringVarP(&restartModel, "model", "m", "", "Model to use (overrides original)")
	restartCmd.Flags().IntVarP(&restartIterations, "iterations", "n", 0, "Number of iterations (0 = unlimited, overrides original)")
	restartCmd.Flags().BoolVarP(&restartForever, "forever", "F", false, "Run indefinitely until manually stopped")
	restartCmd.Flags().StringVarP(&restartName, "name", "N", "", "Name for the agent (overrides original)")
	restartCmd.Flags().BoolVarP(&restartDetach, "detach", "d", false, "Run in detached mode (background)")
	restartCmd.Flags().StringArrayVarP(&restartEnv, "env", "e", nil, "Set environment variables (KEY=VALUE or KEY to pass from shell)")
	restartCmd.Flags().BoolVarP(&restartContinue, "continue", "c", false, "Continue from last iteration instead of starting from 1")
	restartCmd.Flags().IntVar(&restartInternalStart, "_internal-start-iter", 0, "Internal flag for passing start iteration to detached child")
	restartCmd.Flags().MarkHidden("_internal-start-iter")
	restartCmd.Flags().StringVar(&restartOnComplete, "on-complete", "", "Command to run when agent completes")

	// Add dynamic completion for agent identifier and model flag
	restartCmd.ValidArgsFunction = completeAgentIdentifier
	restartCmd.RegisterFlagCompletionFunc("model", completeModelName)
}
