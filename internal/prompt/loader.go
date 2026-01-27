package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ListPrompts returns all available prompt names from the prompts directory.
func ListPrompts(promptsDir string) ([]string, error) {
	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("prompts directory not found: %s", promptsDir)
		}
		return nil, err
	}

	var prompts []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".md") {
			// Remove .md extension for display
			prompts = append(prompts, strings.TrimSuffix(name, ".md"))
		}
	}

	return prompts, nil
}

// LoadPrompt loads a prompt file and wraps it with system/user tags.
func LoadPrompt(promptsDir, name string) (string, error) {
	// Add .md extension if not present
	filename := name
	if !strings.HasSuffix(filename, ".md") {
		filename = filename + ".md"
	}

	path := filepath.Join(promptsDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("prompt not found: %s", name)
		}
		return "", err
	}

	// Wrap prompt with system/user tags
	wrapped := wrapPrompt(string(content))
	return wrapped, nil
}

// LoadPromptFromFile loads a prompt from an arbitrary file path and wraps it with system/user tags.
func LoadPromptFromFile(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("prompt file not found: %s", filePath)
		}
		return "", err
	}

	// Wrap prompt with system/user tags
	wrapped := wrapPrompt(string(content))
	return wrapped, nil
}

// WrapPromptString wraps a raw prompt string with system/user tags.
func WrapPromptString(content string) string {
	return wrapPrompt(content)
}

// wrapPrompt processes the prompt content (trims whitespace).
func wrapPrompt(content string) string {
	return strings.TrimSpace(content)
}

// LoadPromptRaw loads a prompt file without any processing.
// Returns the raw file content as-is, suitable for display.
func LoadPromptRaw(promptsDir, name string) (string, error) {
	// Add .md extension if not present
	filename := name
	if !strings.HasSuffix(filename, ".md") {
		filename = filename + ".md"
	}

	path := filepath.Join(promptsDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("prompt not found: %s", name)
		}
		return "", err
	}

	return string(content), nil
}

// GetPromptPath returns the full path to a prompt file.
func GetPromptPath(promptsDir, name string) string {
	filename := name
	if !strings.HasSuffix(filename, ".md") {
		filename = filename + ".md"
	}
	return filepath.Join(promptsDir, filename)
}

// DefaultTemplate returns a starter template for new prompts.
func DefaultTemplate() string {
	return `# Task

Describe what you want the agent to accomplish.

# Context

Any relevant context about the codebase or task.

# Requirements

- Requirement 1
- Requirement 2

# Exit condition

Describe when the agent should consider the task complete.
`
}

// InjectTaskID injects the task ID at the beginning of the prompt content.
func InjectTaskID(promptContent, taskID string) string {
	taskIDLine := fmt.Sprintf("Your Swarm Task ID is %s.", taskID)
	return taskIDLine + "\n\n" + promptContent
}

// LoadPromptFromStdin reads prompt content from stdin.
func LoadPromptFromStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	var builder strings.Builder

	for {
		line, err := reader.ReadString('\n')
		builder.WriteString(line)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
	}

	content := strings.TrimSpace(builder.String())
	if content == "" {
		return "", fmt.Errorf("stdin is empty")
	}

	return WrapPromptString(content), nil
}

// IsStdinPiped returns true if stdin has piped input (not a terminal).
func IsStdinPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// CombinePrompts combines a base prompt with additional content.
// If the base prompt contains {{STDIN}}, it's replaced. Otherwise, content is appended.
func CombinePrompts(base, additional string) string {
	const placeholder = "{{STDIN}}"
	if strings.Contains(base, placeholder) {
		return strings.Replace(base, placeholder, additional, 1)
	}
	return base + "\n\n---\n\n" + additional
}

// SelectPrompt presents an interactive prompt selection and returns the selected prompt.
func SelectPrompt(promptsDir string) (name string, content string, err error) {
	prompts, err := ListPrompts(promptsDir)
	if err != nil {
		return "", "", err
	}

	if len(prompts) == 0 {
		return "", "", fmt.Errorf("no prompts found in %s", promptsDir)
	}

	// Display available prompts
	fmt.Println("Available prompts:")
	for i, p := range prompts {
		fmt.Printf("  %d. %s\n", i+1, p)
	}
	fmt.Println()

	// Read user selection
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Select a prompt (number or name): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", "", fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)

		// Try parsing as number
		var selection int
		if _, err := fmt.Sscanf(input, "%d", &selection); err == nil {
			if selection >= 1 && selection <= len(prompts) {
				name = prompts[selection-1]
				content, err = LoadPrompt(promptsDir, name)
				return name, content, err
			}
			fmt.Printf("Invalid selection. Please enter 1-%d.\n", len(prompts))
			continue
		}

		// Try matching by name
		for _, p := range prompts {
			if strings.EqualFold(p, input) {
				content, err = LoadPrompt(promptsDir, p)
				return p, content, err
			}
		}

		fmt.Println("Prompt not found. Please try again.")
	}
}
