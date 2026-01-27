package cmd

import (
	"fmt"

	"github.com/matt/swarm-cli/internal/config"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/spf13/cobra"
)

// appConfig holds the loaded configuration (global + project merged)
var appConfig *config.Config

// globalFlag indicates if --global was specified
var globalFlag bool

// appScope holds the current scope (project or global)
var appScope scope.Scope

var rootCmd = &cobra.Command{
	Use:   "swarm",
	Short: "Swarm CLI - Manage AI agents",
	Long: `Swarm CLI is a tool for running and managing AI agents.

It allows you to:
  - Run single agents with custom prompts
  - Run agents in a loop for multiple iterations
  - List and manage running agents
  - View agent logs and status

By default, operations are scoped to the current project directory.
Use --global to operate across all projects.`,
	Example: `  # Run an agent with a prompt from the prompts directory
  swarm run -p my-prompt

  # Run an agent loop in the background
  swarm loop -p my-prompt -n 10 -d

  # List all running agents
  swarm list

  # View details of an agent
  swarm view abc123`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set scope based on global flag
		if globalFlag {
			appScope = scope.ScopeGlobal
		} else {
			appScope = scope.ScopeProject
		}

		// Skip config loading for config subcommand (it handles its own loading)
		if cmd.Name() == "config" || (cmd.Parent() != nil && cmd.Parent().Name() == "config") {
			return nil
		}

		var err error
		appConfig, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add global flag as persistent (available to all subcommands)
	rootCmd.PersistentFlags().BoolVarP(&globalFlag, "global", "g", false, "Operate globally instead of project-scoped")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(loopCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(viewCmd)
	rootCmd.AddCommand(controlCmd)
}

// GetScope returns the current scope (project or global).
func GetScope() scope.Scope {
	return appScope
}

// GetPromptsDir returns the prompts directory based on current scope.
func GetPromptsDir() (string, error) {
	return appScope.PromptsDir()
}
