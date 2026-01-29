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
	expectedCommands := []string{"run", "list", "inspect", "update"}
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
	flags := []string{"model", "prompt", "prompt-file", "prompt-string", "iterations", "name"}
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
		"n": "iterations",
		"N": "name",
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

	// Iterations default should be 1
	iterFlag := cmd.Flags().Lookup("iterations")
	if iterFlag.DefValue != "1" {
		t.Errorf("iterations default value should be '1', got '%s'", iterFlag.DefValue)
	}
}

func TestUpdateCommandFlags(t *testing.T) {
	// Find the update command (which has "control" as an alias)
	var cmd *Command
	for _, c := range rootCmd.Commands() {
		if strings.HasPrefix(c.Use, "update") {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("update command not found")
	}

	// Check flags exist
	flags := []string{"model", "iterations", "terminate", "terminate-after"}
	for _, flagName := range flags {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("update command missing flag '%s'", flagName)
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
			t.Errorf("update command missing short flag '-%s'", short)
		} else if flag.Name != long {
			t.Errorf("short flag '-%s' should map to '%s', got '%s'", short, long, flag.Name)
		}
	}

	// Check that "control" is an alias
	hasControlAlias := false
	for _, alias := range cmd.Aliases {
		if alias == "control" {
			hasControlAlias = true
			break
		}
	}
	if !hasControlAlias {
		t.Error("update command should have 'control' as an alias")
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

func TestInspectCommand(t *testing.T) {
	// Find the inspect command (which has "view" as an alias)
	var cmd *Command
	for _, c := range rootCmd.Commands() {
		if strings.HasPrefix(c.Use, "inspect") {
			cmd = c
			break
		}
	}

	if cmd == nil {
		t.Fatal("inspect command not found")
	}

	// Inspect requires exactly 1 argument (task-id)
	if cmd.Use != "inspect [task-id-or-name]" {
		t.Errorf("inspect command Use should be 'inspect [task-id-or-name]', got '%s'", cmd.Use)
	}

	// Check that "view" is an alias
	hasViewAlias := false
	for _, alias := range cmd.Aliases {
		if alias == "view" {
			hasViewAlias = true
			break
		}
	}
	if !hasViewAlias {
		t.Error("inspect command should have 'view' as an alias")
	}
}

func TestCommandHelp(t *testing.T) {
	commands := []string{"run", "list", "update"}

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
