package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/mj1618n/go-isatty"
	"github.com/mj1618/swarm-cli/internal/process"
	"github.com/mj1618/swarm-cli/internal/scope"
	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var killAllGraceful bool
var killAllForce bool

var killAllCmd = &cobra.Command{
	Use:   "kill-all",
	Short: "Terminate all running and paused agents",
	Long: `Terminate all running and paused agents immediately or gracefully.

By default, agents are terminated immediately using SIGKILL. Use --graceful to allow
each agent's current iteration to complete before terminating.

The command will prompt for confirmation before terminating agents. Use --force to
skip the confirmation prompt.

The command operates on agents in the current project directory by default.
Use --global to terminate all agents across all projects.`,
	Example: `  # Terminate all agents immediately (with confirmation)
  swarm kill-all

  # Terminate without confirmation
  swarm kill-all --force

  # Graceful termination (wait for current iterations)
  swarm kill-all --graceful

  # Terminate all agents globally
  swarm kill-all --global`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// List all agents (including paused ones)
		allAgents, err := mgr.List(false)
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		// Filter for running agents (which includes paused agents since they have status "running")
		var agents []*state.AgentState
		for _, agent := range allAgents {
			if agent.Status == "running" {
				agents = append(agents, agent)
			}
		}

		if len(agents) == 0 {
			fmt.Println("No running or paused agents found")
			return nil
		}

		// Show confirmation unless --force is used
		if !killAllForce {
			scopeStr := "in this project"
			if GetScope() == scope.ScopeGlobal {
				scopeStr = "globally (all projects)"
			}

			fmt.Printf("This will terminate %d agent(s) %s", len(agents), scopeStr)

			// List agents if small number (5 or fewer)
			if len(agents) <= 5 {
				fmt.Println(":")
				for _, agent := range agents {
					name := agent.ID
					if agent.Name != "" {
						name = fmt.Sprintf("%s (%s)", agent.Name, agent.ID)
					}
					fmt.Printf("  - %s\n", name)
				}
			} else {
				fmt.Println(".")
			}

			// Check if stdin is a terminal (interactive mode)
			if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
				fmt.Println("Non-interactive mode detected. Use --force to skip confirmation.")
				fmt.Println("Aborted.")
				return nil
			}

			fmt.Print("Are you sure? [y/N] ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		// Use atomic method for control field to avoid race conditions
		count := 0
		for _, agent := range agents {
			if killAllGraceful {
				// Graceful termination: wait for current iteration to complete
				if err := mgr.SetTerminateMode(agent.ID, "after_iteration"); err != nil {
					fmt.Printf("Warning: failed to update agent %s: %v\n", agent.ID, err)
					continue
				}
			} else {
				// Immediate termination using SIGKILL
				if err := mgr.SetTerminateMode(agent.ID, "immediate"); err != nil {
					fmt.Printf("Warning: failed to update agent %s: %v\n", agent.ID, err)
					continue
				}

				// Force kill the process immediately (SIGKILL on Unix)
				if err := process.ForceKill(agent.PID); err != nil {
					fmt.Printf("Warning: could not kill process %d: %v\n", agent.PID, err)
				}
			}
			count++
		}

		if killAllGraceful {
			fmt.Printf("%d agent(s) will terminate after current iteration\n", count)
		} else {
			fmt.Printf("Killed %d agent(s)\n", count)
		}
		return nil
	},
}

func init() {
	killAllCmd.Flags().BoolVarP(&killAllGraceful, "graceful", "G", false, "Terminate after current iteration completes")
	killAllCmd.Flags().BoolVarP(&killAllForce, "force", "f", false, "Skip confirmation prompt")
}
