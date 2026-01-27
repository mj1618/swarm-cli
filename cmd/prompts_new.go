package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/matt/swarm-cli/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	promptsNewFrom    string
	promptsNewContent string
	promptsNewNoEdit  bool
)

var promptsNewCmd = &cobra.Command{
	Use:     "new <name>",
	Aliases: []string{"create", "add"},
	Short:   "Create a new prompt file",
	Long: `Create a new prompt file in the prompts directory.

By default, creates a prompt with a starter template and opens it in your
default editor ($EDITOR or vi).

Use --from to copy an existing prompt as a starting point.
Use --content to create with specific content (useful for scripting).
Use --no-edit to create without opening the editor.

By default, creates prompts in the project directory (./swarm/prompts/).
Use --global to create in the global directory (~/.swarm/prompts/).`,
	Example: `  # Create a new prompt and open in editor
  swarm prompts new my-feature

  # Create based on an existing prompt
  swarm prompts new my-feature --from coder

  # Create a global prompt
  swarm prompts new my-helper --global

  # Create with specific content
  swarm prompts new quick-fix --content "Fix any linting errors in the codebase"

  # Create without opening editor
  swarm prompts new my-feature --no-edit`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Validate name (no special characters, no .md extension needed)
		if strings.ContainsAny(name, "/\\:*?\"<>|") {
			return fmt.Errorf("invalid prompt name: contains special characters")
		}

		// Get prompts directory based on scope
		promptsDir, err := GetPromptsDir()
		if err != nil {
			return fmt.Errorf("failed to get prompts directory: %w", err)
		}

		// Ensure prompts directory exists
		if err := os.MkdirAll(promptsDir, 0755); err != nil {
			return fmt.Errorf("failed to create prompts directory: %w", err)
		}

		// Build file path
		filename := name
		if !strings.HasSuffix(filename, ".md") {
			filename = filename + ".md"
		}
		filePath := filepath.Join(promptsDir, filename)

		// Check if file already exists
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("prompt %q already exists; use 'swarm prompts edit %s' to modify it", name, name)
		}

		// Determine content
		var content string
		switch {
		case promptsNewContent != "":
			content = promptsNewContent
		case promptsNewFrom != "":
			// Load existing prompt as template
			existingContent, err := prompt.LoadPromptRaw(promptsDir, promptsNewFrom)
			if err != nil {
				// Try global prompts if not found in project
				globalDir, globErr := getGlobalPromptsDir()
				if globErr == nil {
					existingContent, err = prompt.LoadPromptRaw(globalDir, promptsNewFrom)
				}
				if err != nil {
					return fmt.Errorf("template prompt %q not found: %w", promptsNewFrom, err)
				}
			}
			content = existingContent
		default:
			content = prompt.DefaultTemplate()
		}

		// Write the file
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create prompt file: %w", err)
		}

		fmt.Printf("Created prompt: %s\n", filePath)

		// Open in editor unless --no-edit
		if !promptsNewNoEdit {
			editor := resolveEditor("")
			if editor == "" {
				fmt.Println("Note: no editor found. Set $EDITOR to edit the prompt.")
				return nil
			}

			editorCmd := exec.Command(editor, filePath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr

			if err := editorCmd.Run(); err != nil {
				fmt.Printf("Note: could not open editor (%v)\n", err)
			}
		}

		return nil
	},
}

func getGlobalPromptsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".swarm", "prompts"), nil
}

func init() {
	promptsNewCmd.Flags().StringVar(&promptsNewFrom, "from", "", "Copy content from an existing prompt")
	promptsNewCmd.Flags().StringVar(&promptsNewContent, "content", "", "Initial content for the prompt")
	promptsNewCmd.Flags().BoolVar(&promptsNewNoEdit, "no-edit", false, "Don't open the editor after creating")

	promptsCmd.AddCommand(promptsNewCmd)
}
