package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/mj1618/swarm-cli/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	promptsEditEditor string
	promptsEditCreate bool
)

var promptsEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a prompt file in your preferred editor",
	Long: `Open a prompt file in your preferred text editor.

The editor is determined by:
1. --editor flag (if specified)
2. $VISUAL environment variable
3. $EDITOR environment variable
4. Fallback: vim, vi, nano (Unix) or notepad (Windows)

By default, shows prompts from the project directory (./swarm/prompts/).
Use --global to edit a prompt from the global directory (~/.swarm/prompts/).`,
	Example: `  # Edit a project prompt
  swarm prompts edit coder

  # Edit a global prompt
  swarm prompts edit -g shared-task

  # Use a specific editor
  swarm prompts edit coder --editor code
  swarm prompts edit coder -e vim

  # Create a new prompt if it doesn't exist
  swarm prompts edit new-task --create`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		promptName := args[0]

		promptsDir, err := GetPromptsDir()
		if err != nil {
			return fmt.Errorf("failed to get prompts directory: %w", err)
		}

		promptPath := prompt.GetPromptPath(promptsDir, promptName)

		// Check if file exists
		if _, err := os.Stat(promptPath); os.IsNotExist(err) {
			if !promptsEditCreate {
				return fmt.Errorf("prompt '%s' not found at %s\n\nUse --create to create a new prompt", promptName, promptPath)
			}

			// Create the prompts directory if needed
			if err := os.MkdirAll(promptsDir, 0755); err != nil {
				return fmt.Errorf("failed to create prompts directory: %w", err)
			}

			// Create empty file
			if err := os.WriteFile(promptPath, []byte(""), 0644); err != nil {
				return fmt.Errorf("failed to create prompt file: %w", err)
			}
			fmt.Printf("Created new prompt: %s\n", promptPath)
		}

		// Determine editor
		editor := resolveEditor(promptsEditEditor)
		if editor == "" {
			return fmt.Errorf("no editor found. Set $EDITOR or use --editor flag")
		}

		// Open editor
		editorCmd := exec.Command(editor, promptPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		if err := editorCmd.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		return nil
	},
}

// resolveEditor determines which editor to use.
func resolveEditor(override string) string {
	// 1. Command-line override
	if override != "" {
		return override
	}

	// 2. $VISUAL
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}

	// 3. $EDITOR
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	// 4. Platform-specific fallbacks
	if runtime.GOOS == "windows" {
		return "notepad"
	}

	// Try common Unix editors
	for _, editor := range []string{"vim", "vi", "nano"} {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}

	return ""
}

func init() {
	promptsEditCmd.Flags().StringVarP(&promptsEditEditor, "editor", "e", "", "Editor to use (overrides $EDITOR)")
	promptsEditCmd.Flags().BoolVar(&promptsEditCreate, "create", false, "Create the prompt file if it doesn't exist")
	promptsCmd.AddCommand(promptsEditCmd)
}
