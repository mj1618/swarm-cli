package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/matt/swarm-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	modelsFormat  string
	modelsDefault bool
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models for the current backend",
	Long: `List available models for the configured agent backend.

The default model (from config) is highlighted with an asterisk (*).

For the cursor backend, models are fetched from the 'agent --list-models' command.
For the claude-code backend, a fixed set of known models is shown.`,
	Example: `  # List all available models
  swarm models

  # Output as JSON
  swarm models --format json

  # Show only the default model
  swarm models --default`,
	RunE: func(cmd *cobra.Command, args []string) error {
		backend := appConfig.Backend
		defaultModel := appConfig.Model

		// Show only default if requested
		if modelsDefault {
			fmt.Println(defaultModel)
			return nil
		}

		// Get models based on backend
		var models []ModelInfo
		var err error

		switch backend {
		case config.BackendCursor:
			models, err = getCursorModels()
		case config.BackendClaudeCode:
			models = getClaudeCodeModels()
		default:
			return fmt.Errorf("unknown backend: %s", backend)
		}

		if err != nil {
			return fmt.Errorf("failed to get models: %w", err)
		}

		// JSON output
		if modelsFormat == "json" {
			modelNames := make([]string, len(models))
			for i, m := range models {
				modelNames[i] = m.ID
			}
			output := map[string]interface{}{
				"backend": backend,
				"default": defaultModel,
				"models":  modelNames,
			}
			data, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		}

		// Table output
		fmt.Printf("Available models (%s backend):\n\n", backend)

		defaultColor := color.New(color.FgGreen, color.Bold)
		for _, model := range models {
			if model.ID == defaultModel {
				if model.Description != "" {
					defaultColor.Printf("* %s - %s (default)\n", model.ID, model.Description)
				} else {
					defaultColor.Printf("* %s (default)\n", model.ID)
				}
			} else {
				if model.Description != "" {
					fmt.Printf("  %s - %s\n", model.ID, model.Description)
				} else {
					fmt.Printf("  %s\n", model.ID)
				}
			}
		}

		fmt.Println("\nUse 'swarm run -m <model>' to use a specific model.")
		return nil
	},
}

// ModelInfo represents a model with its ID and optional description.
type ModelInfo struct {
	ID          string
	Description string
}

// getCursorModels retrieves available models from the Cursor agent CLI.
func getCursorModels() ([]ModelInfo, error) {
	cmd := exec.Command("agent", "--list-models")
	output, err := cmd.Output()
	if err != nil {
		// Fall back to known models if command fails
		return getFallbackCursorModels(), nil
	}

	// Strip ANSI escape codes from output
	cleanOutput := stripANSI(string(output))

	// Parse output (format: "model-id - Description" or "model-id - Description  (current, default)")
	lines := strings.Split(cleanOutput, "\n")
	var models []ModelInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines, loading messages, headers, and tips
		if line == "" ||
			line == "Available models" ||
			strings.HasPrefix(line, "Loading") ||
			strings.HasPrefix(line, "Tip:") {
			continue
		}

		// Parse "model-id - Description" format
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) >= 1 {
			id := strings.TrimSpace(parts[0])
			if id == "" {
				continue
			}

			description := ""
			if len(parts) == 2 {
				// Remove "(current, default)" suffix if present
				desc := parts[1]
				desc = strings.TrimSuffix(desc, "(current, default)")
				desc = strings.TrimSuffix(desc, "(current)")
				desc = strings.TrimSuffix(desc, "(default)")
				description = strings.TrimSpace(desc)
			}

			models = append(models, ModelInfo{
				ID:          id,
				Description: description,
			})
		}
	}

	if len(models) == 0 {
		// If parsing failed, return fallback
		return getFallbackCursorModels(), nil
	}

	return models, nil
}

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	// Simple state machine to remove ANSI escape sequences
	var result strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			// ANSI sequences end with a letter
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}

// getFallbackCursorModels returns a hardcoded list of known Cursor models.
func getFallbackCursorModels() []ModelInfo {
	return []ModelInfo{
		{ID: "opus-4.5-thinking", Description: "Claude 4.5 Opus (Thinking)"},
		{ID: "sonnet-4.5-thinking", Description: "Claude 4.5 Sonnet (Thinking)"},
		{ID: "opus-4.5", Description: "Claude 4.5 Opus"},
		{ID: "sonnet-4.5", Description: "Claude 4.5 Sonnet"},
		{ID: "claude-opus-4-20250514", Description: "Claude 4 Opus"},
		{ID: "claude-sonnet-4-20250514", Description: "Claude 4 Sonnet"},
		{ID: "gpt-5.2", Description: "GPT-5.2"},
		{ID: "gpt-5.2-codex", Description: "GPT-5.2 Codex"},
		{ID: "gemini-3-pro", Description: "Gemini 3 Pro"},
		{ID: "gemini-3-flash", Description: "Gemini 3 Flash"},
		{ID: "grok", Description: "Grok"},
	}
}

// getClaudeCodeModels returns the known models for Claude Code CLI.
func getClaudeCodeModels() []ModelInfo {
	// Claude Code has a fixed set of models
	return []ModelInfo{
		{ID: "opus", Description: "Claude 4 Opus"},
		{ID: "sonnet", Description: "Claude 4 Sonnet"},
	}
}

func init() {
	modelsCmd.Flags().StringVar(&modelsFormat, "format", "", "Output format: json or table (default)")
	modelsCmd.Flags().BoolVar(&modelsDefault, "default", false, "Show only the default model")
	rootCmd.AddCommand(modelsCmd)
}
