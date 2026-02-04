package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	replayIterations int
	replayModel      string
	replayName       string
	replayDetach     bool
	replayNoDetach   bool
	replayDryRun     bool
)

var replayCmd = &cobra.Command{
	Use:   "replay [task-id-or-name]",
	Short: "Re-run a previous agent with the same configuration",
	Long: `Re-run a previous agent using its saved configuration.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent

By default, the replay inherits the original agent's:
  - Prompt
  - Model
  - Iteration count
  - Detached mode (if log file exists)

Use flags to override any of these settings. The new agent gets
a unique ID and a name based on the original (e.g., "my-agent-replay-1").`,
	Example: `  # Replay agent by ID
  swarm replay abc123

  # Replay agent by name
  swarm replay my-agent

  # Replay most recent agent
  swarm replay @last
  swarm replay _

  # Override iterations
  swarm replay my-agent -n 20

  # Override model
  swarm replay my-agent -m claude-sonnet-4-20250514

  # Give it a custom name
  swarm replay my-agent -N retry-attempt

  # Force detached mode
  swarm replay my-agent -d

  # Force foreground mode
  swarm replay my-agent --no-detach

  # See what would run without executing
  swarm replay my-agent --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentIdentifier := args[0]

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		agent, err := ResolveAgentIdentifier(mgr, agentIdentifier)
		if err != nil {
			return fmt.Errorf("agent not found: %w", err)
		}

		// Check if prompt is replayable
		if agent.Prompt == "<file>" || agent.Prompt == "<string>" || agent.Prompt == "<stdin>" {
			return fmt.Errorf("cannot replay agent with prompt source %q - use 'swarm run' directly with the original prompt", agent.Prompt)
		}

		// Handle combined stdin prompts (like "coder+stdin")
		if strings.HasSuffix(agent.Prompt, "+stdin") {
			return fmt.Errorf("cannot replay agent with stdin-combined prompt %q - use 'swarm run' directly with the original prompt", agent.Prompt)
		}

		// Determine configuration (original values with overrides)
		prompt := agent.Prompt
		model := agent.Model
		iterations := agent.Iterations
		detached := agent.LogFile != "" // Was detached if it has a log file

		// Apply overrides
		if cmd.Flags().Changed("iterations") {
			iterations = replayIterations
		}
		if cmd.Flags().Changed("model") {
			model = replayModel
		}
		if replayDetach {
			detached = true
		}
		if replayNoDetach {
			detached = false
		}

		// Check for conflicting flags
		if replayDetach && replayNoDetach {
			return fmt.Errorf("cannot use both --detach and --no-detach")
		}

		// Generate name for replay
		name := replayName
		if name == "" {
			baseName := agent.Name
			if baseName == "" {
				baseName = agent.ID
			}
			name = generateReplayName(mgr, baseName)
		}

		// Build the command args for display
		runArgs := []string{"run"}
		runArgs = append(runArgs, "-p", prompt)
		runArgs = append(runArgs, "-m", model)
		runArgs = append(runArgs, "-n", strconv.Itoa(iterations))
		runArgs = append(runArgs, "-N", name)
		if detached {
			runArgs = append(runArgs, "-d")
		}
		if globalFlag {
			runArgs = append(runArgs, "-g")
		}

		// Dry run mode
		if replayDryRun {
			agentName := agent.Name
			if agentName == "" {
				agentName = agent.ID
			}
			fmt.Printf("Would replay agent: %s (%s)\n", agentName, agent.ID)
			fmt.Printf("Command: swarm %s\n", formatReplayArgs(runArgs))
			return nil
		}

		// Show replay info
		agentName := agent.Name
		if agentName == "" {
			agentName = agent.ID
		}
		fmt.Printf("Replaying agent: %s (%s)\n", agentName, agent.ID)
		fmt.Println("Original configuration:")
		fmt.Printf("  Prompt:     %s\n", agent.Prompt)
		fmt.Printf("  Model:      %s\n", agent.Model)
		if agent.Iterations == 0 {
			fmt.Printf("  Iterations: unlimited\n")
		} else {
			fmt.Printf("  Iterations: %d\n", agent.Iterations)
		}
		fmt.Printf("  Detached:   %v\n", agent.LogFile != "")

		// Show overrides if any
		hasOverrides := cmd.Flags().Changed("iterations") || cmd.Flags().Changed("model") ||
			replayDetach || replayNoDetach || replayName != ""
		if hasOverrides {
			fmt.Println("\nOverrides applied:")
			if cmd.Flags().Changed("iterations") {
				if iterations == 0 {
					fmt.Printf("  Iterations: unlimited\n")
				} else {
					fmt.Printf("  Iterations: %d\n", iterations)
				}
			}
			if cmd.Flags().Changed("model") {
				fmt.Printf("  Model:      %s\n", model)
			}
			if replayDetach || replayNoDetach {
				fmt.Printf("  Detached:   %v\n", detached)
			}
			if replayName != "" {
				fmt.Printf("  Name:       %s\n", name)
			}
		}
		fmt.Println()

		// Execute the run command by setting the run flags
		runPrompt = prompt
		runModel = model
		runIterations = iterations
		runName = name
		runDetach = detached

		// Clear any other run flags that might have been set
		runPromptFile = ""
		runPromptString = ""
		runStdin = false
		runForever = false
		runWorkingDir = ""
		runEnv = nil
		runTimeout = ""
		runIterTimeout = ""
		runOnComplete = ""

		return runCmd.RunE(cmd, []string{})
	},
}

// generateReplayName creates a unique replay name based on the original
func generateReplayName(mgr *state.Manager, baseName string) string {
	// Get all agents to check for name conflicts
	agents, err := mgr.List(false)
	if err != nil {
		return baseName + "-replay"
	}

	// Check if a name is in use by a running agent
	nameInUse := func(name string) bool {
		for _, a := range agents {
			if a.Name == name && a.Status == "running" {
				return true
			}
		}
		return false
	}

	// Try base replay name first
	replayName := baseName + "-replay"
	if !nameInUse(replayName) {
		return replayName
	}

	// Find the next available number suffix
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-replay-%d", baseName, i)
		if !nameInUse(candidate) {
			return candidate
		}
	}
}

// formatReplayArgs formats args for display
func formatReplayArgs(args []string) string {
	var parts []string
	for _, arg := range args {
		// Quote args with spaces
		if strings.Contains(arg, " ") {
			parts = append(parts, fmt.Sprintf("%q", arg))
		} else {
			parts = append(parts, arg)
		}
	}
	return strings.Join(parts, " ")
}

func init() {
	replayCmd.Flags().IntVarP(&replayIterations, "iterations", "n", 0, "Override iteration count")
	replayCmd.Flags().StringVarP(&replayModel, "model", "m", "", "Override model")
	replayCmd.Flags().StringVarP(&replayName, "name", "N", "", "Set name for the replayed agent")
	replayCmd.Flags().BoolVarP(&replayDetach, "detach", "d", false, "Run in detached mode")
	replayCmd.Flags().BoolVar(&replayNoDetach, "no-detach", false, "Run in foreground mode")
	replayCmd.Flags().BoolVar(&replayDryRun, "dry-run", false, "Show what would be run without executing")

	// Add dynamic completion for model flag
	replayCmd.RegisterFlagCompletionFunc("model", completeModelName)
}
