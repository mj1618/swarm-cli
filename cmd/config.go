package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mj1618/swarm-cli/internal/config"
	"github.com/spf13/cobra"
)

var configGlobal bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage swarm configuration",
	Long:  `View and manage swarm-cli configuration files.`,
	Example: `  # Show current configuration
  swarm config show

  # Switch to claude-code backend
  swarm config set-backend claude-code`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display the merged configuration",
	Long:  `Display the effective configuration after merging global and project configs.`,
	Example: `  # Show effective configuration
  swarm config show`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		fmt.Println("# Effective configuration (merged from all sources)")
		fmt.Println()
		fmt.Print(cfg.ToTOML())
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config file locations",
	Long:  `Display the paths to global and project configuration files.`,
	Example: `  # Show config file paths
  swarm config path`,
	RunE: func(cmd *cobra.Command, args []string) error {
		globalPath, err := config.GlobalConfigPath()
		if err != nil {
			globalPath = fmt.Sprintf("<error: %v>", err)
		}

		projectPath := config.ProjectConfigPath()

		// Check existence
		globalExists := "not found"
		if _, err := os.Stat(globalPath); err == nil {
			globalExists = "exists"
		}

		projectExists := "not found"
		if _, err := os.Stat(projectPath); err == nil {
			projectExists = "exists"
		}

		fmt.Println("Configuration file locations:")
		fmt.Println()
		fmt.Printf("  Global:  %s (%s)\n", globalPath, globalExists)
		fmt.Printf("  Project: %s (%s)\n", projectPath, projectExists)
		fmt.Println()
		fmt.Println("Priority: CLI flags > project config > global config > defaults")
		return nil
	},
}

var configSetBackendCmd = &cobra.Command{
	Use:   "set-backend [cursor|claude-code|codex]",
	Short: "Switch the agent backend",
	Long: `Switch between different agent CLI backends.

Available backends:
  cursor      - Cursor's agent CLI (uses stream-json output with log parsing)
  claude-code - Anthropic's Claude Code CLI (uses stream-json output with log parsing)
  codex       - OpenAI's Codex CLI (uses JSONL output with log parsing)

This command updates the config file with the appropriate preset for the chosen backend.
By default, updates the project config (swarm/swarm.toml). Use --global to update the global config.`,
	Example: `  # Use Cursor backend
  swarm config set-backend cursor

  # Use Claude Code backend
  swarm config set-backend claude-code

  # Use Codex backend
  swarm config set-backend codex

  # Update global config instead of project
  swarm config set-backend claude-code --global`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: config.ValidBackends(),
	RunE: func(cmd *cobra.Command, args []string) error {
		backend := strings.ToLower(args[0])

		// Validate backend
		validBackends := config.ValidBackends()
		valid := false
		for _, v := range validBackends {
			if v == backend {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid backend %q, valid options: %s", backend, strings.Join(validBackends, ", "))
		}

		// Determine config path
		var configPath string
		var err error
		if configGlobal {
			configPath, err = config.GlobalConfigPath()
			if err != nil {
				return fmt.Errorf("failed to determine global config path: %w", err)
			}
		} else {
			configPath = config.ProjectConfigPath()
		}

		// Load existing config or start with defaults
		cfg := config.DefaultConfig()
		if _, err := os.Stat(configPath); err == nil {
			// Config file exists, load it first to preserve other settings
			loadedCfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load existing config: %w", err)
			}
			cfg = loadedCfg
		}

		// Apply the new backend
		if err := cfg.SetBackend(backend); err != nil {
			return err
		}

		// Create parent directory if needed
		dir := filepath.Dir(configPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write updated config
		if err := os.WriteFile(configPath, []byte(cfg.ToTOML()), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("Backend switched to %q\n", backend)
		fmt.Printf("Updated config: %s\n", configPath)
		return nil
	},
}

var configSystemPromptFile bool

// resolveConfigPath returns the path of the config file to mutate (project by
// default, global if useGlobal is true).
func resolveConfigPath(useGlobal bool) (string, error) {
	if useGlobal {
		return config.GlobalConfigPath()
	}
	return config.ProjectConfigPath(), nil
}

// loadOrDefaultConfig returns the merged effective config if any config file
// exists on disk, or a fresh DefaultConfig() otherwise. Used by `set-*`
// subcommands so writes preserve existing settings.
func loadOrDefaultConfig(configPath string) (*config.Config, error) {
	cfg := config.DefaultConfig()
	if _, err := os.Stat(configPath); err == nil {
		loadedCfg, err := config.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load existing config: %w", err)
		}
		cfg = loadedCfg
	}
	return cfg, nil
}

// writeConfig writes cfg as TOML to configPath, creating parent dirs as needed.
func writeConfig(cfg *config.Config, configPath string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	if err := os.WriteFile(configPath, []byte(cfg.ToTOML()), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

// PersistSystemPrompt updates the configured system prompt and writes the
// change to the appropriate config file (project unless useGlobal is true).
// It is exported via the package for use by the `swarm run` command, which
// allows users to set + persist a system prompt in the same step.
func PersistSystemPrompt(content string, useGlobal bool) (string, error) {
	configPath, err := resolveConfigPath(useGlobal)
	if err != nil {
		return "", fmt.Errorf("failed to determine config path: %w", err)
	}
	cfg, err := loadOrDefaultConfig(configPath)
	if err != nil {
		return "", err
	}
	cfg.SystemPrompt = content
	if err := writeConfig(cfg, configPath); err != nil {
		return "", err
	}
	return configPath, nil
}

// readSystemPromptInput resolves the user-supplied system prompt value into a
// string. When fromFile is true the value is treated as a file path; otherwise
// it's treated as raw text. Empty input is rejected so users don't accidentally
// clear the configured prompt — `swarm config remove-system-prompt` is the
// explicit clear path.
func readSystemPromptInput(value string, fromFile bool) (string, error) {
	if fromFile {
		if value == "" {
			return "", fmt.Errorf("--file requires a path")
		}
		data, err := os.ReadFile(value)
		if err != nil {
			return "", fmt.Errorf("failed to read system prompt file %s: %w", value, err)
		}
		content := strings.TrimRight(string(data), "\n")
		if content == "" {
			return "", fmt.Errorf("system prompt file %s is empty", value)
		}
		return content, nil
	}
	if value == "" {
		return "", fmt.Errorf("system prompt content cannot be empty (use `swarm config remove-system-prompt` to clear)")
	}
	return value, nil
}

var configSetSystemPromptCmd = &cobra.Command{
	Use:   "set-system-prompt [text]",
	Short: "Set the custom system prompt for claude-code runs",
	Long: `Set the custom system prompt that swarm passes to the agent via the
` + "`--system-prompt`" + ` flag (currently only honored by the claude-code backend).

The prompt can be supplied either as inline text or read from a file with --file.
The value is persisted to the project config (swarm/swarm.toml) by default;
pass --global to update the global config instead.

To remove the configured system prompt later, run:
  swarm config remove-system-prompt`,
	Example: `  # Set inline text
  swarm config set-system-prompt "You are a senior code reviewer. Be terse."

  # Read from a file
  swarm config set-system-prompt --file ./system-prompt.md

  # Persist globally
  swarm config set-system-prompt "Always cite sources." --global`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var raw string
		if len(args) == 1 {
			raw = args[0]
		}
		if configSystemPromptFile && raw == "" {
			return fmt.Errorf("--file requires a path argument")
		}
		if !configSystemPromptFile && raw == "" {
			return fmt.Errorf("system prompt text is required (or pass --file <path>)")
		}
		content, err := readSystemPromptInput(raw, configSystemPromptFile)
		if err != nil {
			return err
		}
		path, err := PersistSystemPrompt(content, configGlobal)
		if err != nil {
			return err
		}
		fmt.Printf("Custom system prompt updated (%d chars)\n", len(content))
		fmt.Printf("Updated config: %s\n", path)
		return nil
	},
}

var configRemoveSystemPromptCmd = &cobra.Command{
	Use:     "remove-system-prompt",
	Aliases: []string{"unset-system-prompt", "clear-system-prompt"},
	Short:   "Remove the custom system prompt",
	Long: `Clear any previously configured custom system prompt. After running this,
agent invocations will no longer include the ` + "`--system-prompt`" + ` flag.

By default updates the project config (swarm/swarm.toml); pass --global to
update the global config instead.`,
	Example: `  swarm config remove-system-prompt
  swarm config remove-system-prompt --global`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := PersistSystemPrompt("", configGlobal)
		if err != nil {
			return err
		}
		fmt.Println("Custom system prompt removed")
		fmt.Printf("Updated config: %s\n", path)
		return nil
	},
}

var configSetModelCmd = &cobra.Command{
	Use:   "set-model [model]",
	Short: "Set the default model",
	Long: `Set the default model for agent runs.

The model is used when no --model flag is specified on the run command.
By default, updates the project config (swarm/swarm.toml). Use --global to update the global config.

Note: Model names are not validated - different backends support different models,
and the backend CLI will report an error if the model is invalid.`,
	Example: `  # Set default model for project
  swarm config set-model opus

  # Set model in global config
  swarm config set-model claude-sonnet-4-20250514 --global`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		model := strings.TrimSpace(args[0])

		// Validate that model is not empty
		if model == "" {
			return fmt.Errorf("model cannot be empty")
		}

		// Determine config path
		var configPath string
		var err error
		if configGlobal {
			configPath, err = config.GlobalConfigPath()
			if err != nil {
				return fmt.Errorf("failed to determine global config path: %w", err)
			}
		} else {
			configPath = config.ProjectConfigPath()
		}

		// Load existing config or start with defaults
		cfg := config.DefaultConfig()
		if _, err := os.Stat(configPath); err == nil {
			// Config file exists, load it first to preserve other settings
			loadedCfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load existing config: %w", err)
			}
			cfg = loadedCfg
		}

		// Update the model
		cfg.Model = model

		// Create parent directory if needed
		dir := filepath.Dir(configPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Write updated config
		if err := os.WriteFile(configPath, []byte(cfg.ToTOML()), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("Default model set to %q\n", model)
		fmt.Printf("Updated config: %s\n", configPath)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configSetBackendCmd)
	configCmd.AddCommand(configSetModelCmd)
	configCmd.AddCommand(configSetSystemPromptCmd)
	configCmd.AddCommand(configRemoveSystemPromptCmd)

	configSetBackendCmd.Flags().BoolVarP(&configGlobal, "global", "g", false, "Update global config instead of project config")
	configSetModelCmd.Flags().BoolVarP(&configGlobal, "global", "g", false, "Update global config instead of project config")
	configSetSystemPromptCmd.Flags().BoolVarP(&configGlobal, "global", "g", false, "Update global config instead of project config")
	configSetSystemPromptCmd.Flags().BoolVarP(&configSystemPromptFile, "file", "f", false, "Treat the positional argument as a path to a file whose contents become the system prompt")
	configRemoveSystemPromptCmd.Flags().BoolVarP(&configGlobal, "global", "g", false, "Update global config instead of project config")

	rootCmd.AddCommand(configCmd)
}
