package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/matt/swarm-cli/internal/config"
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
	Use:   "set-backend [cursor|claude-code]",
	Short: "Switch the agent backend",
	Long: `Switch between different agent CLI backends.

Available backends:
  cursor      - Cursor's agent CLI (uses stream-json output with log parsing)
  claude-code - Anthropic's Claude Code CLI (uses direct text streaming)

This command updates the config file with the appropriate preset for the chosen backend.
By default, updates the project config (swarm/.swarm.toml). Use --global to update the global config.`,
	Example: `  # Use Cursor backend
  swarm config set-backend cursor

  # Use Claude Code backend
  swarm config set-backend claude-code

  # Update global config instead of project
  swarm config set-backend claude-code --global`,
	Args: cobra.ExactArgs(1),
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

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configSetBackendCmd)

	configSetBackendCmd.Flags().BoolVarP(&configGlobal, "global", "g", false, "Update global config instead of project config")

	rootCmd.AddCommand(configCmd)
}
