package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/matt/swarm-cli/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	promptsShowRaw  bool
	promptsShowPath bool
)

var promptsShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show the content of a prompt",
	Long: `Display the content of a prompt file.

By default, shows prompts from the project directory (./swarm/prompts/).
Use --global to show a prompt from the global directory (~/.swarm/prompts/).`,
	Example: `  # Show a project prompt
  swarm prompts show coder

  # Show a global prompt
  swarm prompts show -g shared-task

  # Show with file path header
  swarm prompts show coder --path

  # Raw output (no decorations, for piping)
  swarm prompts show coder --raw`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		promptName := args[0]

		promptsDir, err := GetPromptsDir()
		if err != nil {
			return fmt.Errorf("failed to get prompts directory: %w", err)
		}

		// Load the raw prompt content
		content, err := prompt.LoadPromptRaw(promptsDir, promptName)
		if err != nil {
			return fmt.Errorf("failed to load prompt '%s': %w", promptName, err)
		}

		// Get the file path for display
		promptPath := prompt.GetPromptPath(promptsDir, promptName)

		// Raw mode: just output content
		if promptsShowRaw {
			fmt.Print(content)
			// Ensure trailing newline
			if !strings.HasSuffix(content, "\n") {
				fmt.Println()
			}
			return nil
		}

		// Formatted output with header
		bold := color.New(color.Bold)
		dim := color.New(color.Faint)

		separator := strings.Repeat("═", 79)
		bold.Println(separator)
		fmt.Printf("Prompt: ")
		bold.Println(promptName)
		if promptsShowPath {
			fmt.Printf("File: ")
			dim.Println(promptPath)
		}
		bold.Println(separator)
		fmt.Println()
		fmt.Print(content)
		if !strings.HasSuffix(content, "\n") {
			fmt.Println()
		}

		return nil
	},
}

func init() {
	promptsShowCmd.Flags().BoolVar(&promptsShowRaw, "raw", false, "Output raw content without formatting")
	promptsShowCmd.Flags().BoolVar(&promptsShowPath, "path", false, "Show the file path in the header")
	promptsCmd.AddCommand(promptsShowCmd)
}
