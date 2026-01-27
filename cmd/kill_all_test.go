package cmd

import (
	"testing"
)

func TestKillAllCommandFlags(t *testing.T) {
	// Test that --force flag exists
	forceFlag := killAllCmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Error("Expected --force flag to exist")
	}
	if forceFlag.Shorthand != "f" {
		t.Errorf("Expected --force shorthand to be 'f', got '%s'", forceFlag.Shorthand)
	}
	if forceFlag.DefValue != "false" {
		t.Errorf("Expected --force default to be 'false', got '%s'", forceFlag.DefValue)
	}

	// Test that --graceful flag exists
	gracefulFlag := killAllCmd.Flags().Lookup("graceful")
	if gracefulFlag == nil {
		t.Error("Expected --graceful flag to exist")
	}
	if gracefulFlag.Shorthand != "G" {
		t.Errorf("Expected --graceful shorthand to be 'G', got '%s'", gracefulFlag.Shorthand)
	}
	if gracefulFlag.DefValue != "false" {
		t.Errorf("Expected --graceful default to be 'false', got '%s'", gracefulFlag.DefValue)
	}
}

func TestKillAllCommandUsage(t *testing.T) {
	// Test that command usage is correct
	if killAllCmd.Use != "kill-all" {
		t.Errorf("Expected Use to be 'kill-all', got '%s'", killAllCmd.Use)
	}

	// Test that long description mentions --force
	if killAllCmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Check examples include --force
	examples := killAllCmd.Example
	if examples == "" {
		t.Error("Expected Example to be set")
	}
}

func TestKillAllCommandArgs(t *testing.T) {
	// kill-all should accept no arguments
	if killAllCmd.Args == nil {
		t.Error("Expected Args to be set")
	}
	
	// Test that args validation rejects arguments
	err := killAllCmd.Args(killAllCmd, []string{"unexpected-arg"})
	if err == nil {
		t.Error("Expected error when passing arguments to kill-all")
	}
}
