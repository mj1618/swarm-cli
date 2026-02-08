package cmd

import (
	"fmt"

	"github.com/mj1618/swarm-cli/internal/config"
	"github.com/mj1618/swarm-cli/internal/scope"
	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/mj1618/swarm-cli/internal/version"
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
  - Run agents with custom prompts (single or multiple iterations)
  - List and manage running agents
  - View agent logs and status

By default, operations are scoped to the current project directory.
Use --global to operate across all projects.`,
	Example: `  # Run an agent with a prompt from the prompts directory
  swarm run -p my-prompt

  # Run an agent for 10 iterations in the background
  swarm run -p my-prompt -n 10 -d

  # List all running agents
  swarm list

  # Inspect details of an agent
  swarm inspect abc123`,
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

	// Set version for --version flag
	rootCmd.Version = version.Version

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(composeStopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(killCmd)
	rootCmd.AddCommand(killAllCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(stopAllCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(startAllCmd)
	rootCmd.AddCommand(promptsCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(summaryCmd)
	rootCmd.AddCommand(pauseAllCmd)
	rootCmd.AddCommand(resumeAllCmd)
	rootCmd.AddCommand(replayCmd)
	rootCmd.AddCommand(cloneCmd)
	rootCmd.AddCommand(topCmd)
	rootCmd.AddCommand(serveCmd)
}

// GetScope returns the current scope (project or global).
func GetScope() scope.Scope {
	return appScope
}

// GetPromptsDir returns the prompts directory based on current scope.
func GetPromptsDir() (string, error) {
	return appScope.PromptsDir()
}

// IsLastIdentifier returns true if the identifier refers to the most recent agent.
func IsLastIdentifier(identifier string) bool {
	return identifier == "@last" || identifier == "_"
}

// ResolveAgentIdentifier resolves an agent identifier to an AgentState.
// Handles special identifiers like "@last" and "_" which refer to the most recently started agent.
func ResolveAgentIdentifier(mgr *state.Manager, identifier string) (*state.AgentState, error) {
	if IsLastIdentifier(identifier) {
		agent, err := mgr.GetLast()
		if err != nil {
			return nil, fmt.Errorf("no recent agent found: %w", err)
		}
		return agent, nil
	}
	return mgr.GetByNameOrID(identifier)
}
