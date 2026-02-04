package cmd

import (
	"fmt"

	"github.com/mj1618/swarm-cli/internal/prompt"
	"github.com/spf13/cobra"
)

var promptsCmd = &cobra.Command{
	Use:   "prompts",
	Short: "Manage prompt files",
	Long: `Manage prompt files used by agents.

Prompts are markdown files stored in:
  - Project: ./swarm/prompts/
  - Global:  ~/.swarm/prompts/

When called without a subcommand, lists available prompts.`,
	Example: `  # List prompts in current project
  swarm prompts

  # List global prompts
  swarm prompts -g

  # Show content of a prompt
  swarm prompts show coder`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to list behavior for backward compatibility
		return runPromptsList()
	},
}

var promptsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List available prompt files",
	Long: `List all available prompt files from the prompts directory.

By default, shows prompts from the project directory (./swarm/prompts/).
Use --global to show prompts from the global directory (~/.swarm/prompts/).`,
	Example: `  # List prompts in current project
  swarm prompts list

  # List global prompts
  swarm prompts list -g`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPromptsList()
	},
}

func runPromptsList() error {
	promptsDir, err := GetPromptsDir()
	if err != nil {
		return fmt.Errorf("failed to get prompts directory: %w", err)
	}

	prompts, err := prompt.ListPrompts(promptsDir)
	if err != nil {
		return err
	}

	if len(prompts) == 0 {
		fmt.Printf("No prompts found in %s\n", promptsDir)
		return nil
	}

	fmt.Printf("Available prompts (%s):\n", promptsDir)
	for _, p := range prompts {
		fmt.Printf("  %s\n", p)
	}

	return nil
}

func init() {
	promptsCmd.AddCommand(promptsListCmd)
}
