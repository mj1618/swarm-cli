package cmd

import (
	"fmt"
	"os"

	"github.com/matt/swarm-cli/internal/prompt"
	"github.com/matt/swarm-cli/internal/scope"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for swarm-cli.

To load completions:

Bash:
  $ source <(swarm completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ swarm completion bash > /etc/bash_completion.d/swarm
  # macOS:
  $ swarm completion bash > $(brew --prefix)/etc/bash_completion.d/swarm

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ swarm completion zsh > "${fpath[1]}/_swarm"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ swarm completion fish | source

  # To load completions for each session, execute once:
  $ swarm completion fish > ~/.config/fish/completions/swarm.fish

PowerShell:
  PS> swarm completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> swarm completion powershell > swarm.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

// completeAgentIdentifier provides dynamic completion for agent IDs and names.
// Used by commands that take an agent identifier as an argument.
func completeAgentIdentifier(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get state manager - use global scope to show all agents for discovery
	mgr, err := state.NewManagerWithScope(scope.ScopeGlobal, "")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Include all agents (running and terminated) for broader completion
	agents, err := mgr.List(false)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string

	// Add special identifiers
	completions = append(completions, "@last\tMost recently started agent")
	completions = append(completions, "_\tMost recently started agent")

	for _, agent := range agents {
		// Add ID with description
		desc := fmt.Sprintf("%s (%s)", agent.Prompt, agent.Status)
		if agent.Paused {
			desc = fmt.Sprintf("%s (paused)", agent.Prompt)
		}
		completions = append(completions, fmt.Sprintf("%s\t%s", agent.ID, desc))

		// Add name if present
		if agent.Name != "" {
			completions = append(completions, fmt.Sprintf("%s\t%s", agent.Name, desc))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completeRunningAgentIdentifier provides dynamic completion for running agents only.
// Used by commands that only operate on running agents (kill, stop, etc.).
func completeRunningAgentIdentifier(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	mgr, err := state.NewManagerWithScope(scope.ScopeGlobal, "")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Only running agents
	agents, err := mgr.List(true)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string

	// Add special identifiers
	completions = append(completions, "@last\tMost recently started agent")
	completions = append(completions, "_\tMost recently started agent")

	for _, agent := range agents {
		desc := fmt.Sprintf("%s - iter %d/%d", agent.Prompt, agent.CurrentIter, agent.Iterations)
		if agent.Paused {
			desc = fmt.Sprintf("%s (paused)", agent.Prompt)
		}
		completions = append(completions, fmt.Sprintf("%s\t%s", agent.ID, desc))
		if agent.Name != "" {
			completions = append(completions, fmt.Sprintf("%s\t%s", agent.Name, desc))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

// completePromptName provides dynamic completion for prompt names.
func completePromptName(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Try to get prompts directory based on current scope
	promptsDir, err := scope.ScopeProject.PromptsDir()
	if err != nil {
		// Fall back to global prompts
		promptsDir, err = scope.ScopeGlobal.PromptsDir()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	prompts, err := prompt.ListPrompts(promptsDir)
	if err != nil {
		// Try global prompts as fallback
		promptsDir, err = scope.ScopeGlobal.PromptsDir()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		prompts, err = prompt.ListPrompts(promptsDir)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}

	return prompts, cobra.ShellCompDirectiveNoFileComp
}

// completeModelName provides dynamic completion for model names.
func completeModelName(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Common model aliases and full names
	models := []string{
		"opus\tClaude Opus (latest)",
		"sonnet\tClaude Sonnet (latest)",
		"claude-opus-4-20250514\tClaude Opus 4",
		"claude-sonnet-4-20250514\tClaude Sonnet 4",
	}
	return models, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
