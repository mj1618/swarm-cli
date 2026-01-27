package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	// Reset the command for testing
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})

	// Check basic properties
	if rootCmd.Use != "swarm" {
		t.Errorf("Root command Use should be 'swarm', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short != "Swarm CLI - Manage AI agents" {
		t.Errorf("Root command Short description mismatch")
	}

	// Verify subcommands are registered
	expectedCommands := []string{"run", "loop", "list", "view", "control"}
	for _, name := range expectedCommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == name || strings.HasPrefix(cmd.Use, name+" ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand '%s' not found", name)
		}
	}
}

func TestExecute(t *testing.T) {
	// Test that Execute returns without error when no args (shows help)
	rootCmd.SetArgs([]string{})
	var out bytes.Buffer
	rootCmd.SetOut(&out)

	// Execute should not return an error for no args (shows usage)
	err := Execute()
	if err != nil {
		t.Errorf("Execute() with no args returned error: %v", err)
	}
}

func TestRunCommandFlags(t *testing.T) {
	// Find the run command
	var cmd *Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "run" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("run command not found")
	}

	// Check flags exist
	flags := []string{"model", "prompt", "prompt-file", "prompt-string"}
	for _, flagName := range flags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("run command missing flag '%s'", flagName)
		}
	}

	// Check short flags
	shortFlags := map[string]string{
		"m": "model",
		"p": "prompt",
		"f": "prompt-file",
		"s": "prompt-string",
	}
	for short, long := range shortFlags {
		flag := cmd.Flags().ShorthandLookup(short)
		if flag == nil {
			t.Errorf("run command missing short flag '-%s'", short)
		} else if flag.Name != long {
			t.Errorf("short flag '-%s' should map to '%s', got '%s'", short, long, flag.Name)
		}
	}

	// Model default is now empty (comes from config at runtime)
	modelFlag := cmd.Flags().Lookup("model")
	if modelFlag.DefValue != "" {
		t.Errorf("model default value should be '' (loaded from config), got '%s'", modelFlag.DefValue)
	}
}

func TestLoopCommandFlags(t *testing.T) {
	// Find the loop command
	var cmd *Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "loop" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("loop command not found")
	}

	// Check flags exist
	flags := []string{"model", "prompt", "prompt-file", "prompt-string", "iterations"}
	for _, flagName := range flags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("loop command missing flag '%s'", flagName)
		}
	}

	// Check short flags
	shortFlags := map[string]string{
		"m": "model",
		"p": "prompt",
		"f": "prompt-file",
		"s": "prompt-string",
		"n": "iterations",
	}
	for short, long := range shortFlags {
		flag := cmd.Flags().ShorthandLookup(short)
		if flag == nil {
			t.Errorf("loop command missing short flag '-%s'", short)
		} else if flag.Name != long {
			t.Errorf("short flag '-%s' should map to '%s', got '%s'", short, long, flag.Name)
		}
	}

	// Defaults are now empty/0 (loaded from config at runtime)
	modelFlag := cmd.Flags().Lookup("model")
	if modelFlag.DefValue != "" {
		t.Errorf("model default value should be '' (loaded from config), got '%s'", modelFlag.DefValue)
	}

	iterFlag := cmd.Flags().Lookup("iterations")
	if iterFlag.DefValue != "0" {
		t.Errorf("iterations default value should be '0' (loaded from config), got '%s'", iterFlag.DefValue)
	}
}

func TestControlCommandFlags(t *testing.T) {
	// Find the control command
	var cmd *Command
	for _, c := range rootCmd.Commands() {
		if strings.HasPrefix(c.Use, "control") {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("control command not found")
	}

	// Check flags exist
	flags := []string{"model", "iterations", "terminate", "terminate-after"}
	for _, flagName := range flags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("control command missing flag '%s'", flagName)
		}
	}

	// Check short flags
	shortFlags := map[string]string{
		"m": "model",
		"n": "iterations",
	}
	for short, long := range shortFlags {
		flag := cmd.Flags().ShorthandLookup(short)
		if flag == nil {
			t.Errorf("control command missing short flag '-%s'", short)
		} else if flag.Name != long {
			t.Errorf("short flag '-%s' should map to '%s', got '%s'", short, long, flag.Name)
		}
	}
}

func TestListCommand(t *testing.T) {
	// Find the list command
	var cmd *Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "list" {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("list command not found")
	}

	if cmd.Short != "List running agents" {
		t.Errorf("list command short description mismatch")
	}
}

func TestViewCommand(t *testing.T) {
	// Find the view command
	var cmd *Command
	for _, c := range rootCmd.Commands() {
		if strings.HasPrefix(c.Use, "view") {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("view command not found")
	}

	// View requires exactly 1 argument (agent-id)
	if cmd.Use != "view [agent-id-or-name]" {
		t.Errorf("view command Use should be 'view [agent-id-or-name]', got '%s'", cmd.Use)
	}
}

func TestCommandHelp(t *testing.T) {
	commands := []string{"run", "loop", "list", "control"}

	for _, cmdName := range commands {
		var cmd *Command
		for _, c := range rootCmd.Commands() {
			if strings.HasPrefix(c.Use, cmdName) {
				cmd = c
				break
			}
		}

		if cmd == nil {
			t.Errorf("command '%s' not found", cmdName)
			continue
		}

		// Each command should have a short description
		if cmd.Short == "" {
			t.Errorf("command '%s' missing Short description", cmdName)
		}

		// Each command should have a long description
		if cmd.Long == "" {
			t.Errorf("command '%s' missing Long description", cmdName)
		}
	}
}

// Type alias for cleaner test code
type Command = cobra.Command

func TestRootCommandHasNoRunE(t *testing.T) {
	// Root command should not have RunE - it just shows help
	if rootCmd.RunE != nil {
		t.Error("Root command should not have RunE")
	}
	if rootCmd.Run != nil {
		t.Error("Root command should not have Run")
	}
}

func TestSubcommandsHaveRunE(t *testing.T) {
	// All subcommands should have RunE (except parent commands like 'config')
	for _, cmd := range rootCmd.Commands() {
		// Skip help command which is added by cobra
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			continue
		}
		// Skip parent commands that have subcommands
		if cmd.HasSubCommands() {
			continue
		}
		if cmd.RunE == nil {
			t.Errorf("subcommand '%s' should have RunE", cmd.Name())
		}
	}
}
