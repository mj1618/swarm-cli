package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/mj1618/swarm-cli/internal/prompt"
	"github.com/spf13/cobra"
)

var promptsCheckCmd = &cobra.Command{
	Use:   "check [name...]",
	Short: "Validate prompt files and their includes",
	Long: `Validate that all {{include: path}} directives in prompts can be resolved.

If no prompt names are provided, validates all prompts in the directory.
Shows which includes each prompt uses and reports any errors.`,
	Example: `  # Check all prompts
  swarm prompts check

  # Check specific prompts
  swarm prompts check coder planner

  # Check global prompts
  swarm prompts check -g`,
	RunE: func(cmd *cobra.Command, args []string) error {
		promptsDir, err := GetPromptsDir()
		if err != nil {
			return fmt.Errorf("failed to get prompts directory: %w", err)
		}

		// Get list of prompts to check
		var promptNames []string
		if len(args) > 0 {
			promptNames = args
		} else {
			// Get all prompts
			promptNames, err = prompt.ListPrompts(promptsDir)
			if err != nil {
				return fmt.Errorf("failed to list prompts: %w", err)
			}
		}

		if len(promptNames) == 0 {
			fmt.Printf("No prompts found in %s\n", promptsDir)
			return nil
		}

		// Colors for output
		green := color.New(color.FgGreen)
		red := color.New(color.FgRed)
		dim := color.New(color.Faint)

		var hasErrors bool

		for _, name := range promptNames {
			// Add .md extension if not present
			filename := name
			if !strings.HasSuffix(filename, ".md") {
				filename = filename + ".md"
			}

			path := filepath.Join(promptsDir, filename)

			// Read the prompt content
			content, err := os.ReadFile(path)
			if err != nil {
				red.Printf("✗ %s: ", name)
				fmt.Printf("failed to read: %v\n", err)
				hasErrors = true
				continue
			}

			// Validate includes
			includes, err := prompt.ValidateIncludes(string(content), promptsDir)
			if err != nil {
				red.Printf("✗ %s: ", name)
				fmt.Printf("%v\n", err)
				hasErrors = true
				continue
			}

			// Success
			green.Printf("✓ %s", name)
			if len(includes) > 0 {
				dim.Printf(" (includes: %s)", strings.Join(includes, ", "))
			}
			fmt.Println()
		}

		if hasErrors {
			return fmt.Errorf("some prompts have errors")
		}

		return nil
	},
}

func init() {
	promptsCmd.AddCommand(promptsCheckCmd)
}
