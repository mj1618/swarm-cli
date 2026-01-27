package cmd

import (
	"testing"
)

func TestCloneCommandFlags(t *testing.T) {
	// Test that --name flag exists
	nameFlag := cloneCmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Error("Expected --name flag to exist")
	}
	if nameFlag.Shorthand != "N" {
		t.Errorf("Expected --name shorthand to be 'N', got '%s'", nameFlag.Shorthand)
	}

	// Test that --iterations flag exists
	iterFlag := cloneCmd.Flags().Lookup("iterations")
	if iterFlag == nil {
		t.Error("Expected --iterations flag to exist")
	}
	if iterFlag.Shorthand != "n" {
		t.Errorf("Expected --iterations shorthand to be 'n', got '%s'", iterFlag.Shorthand)
	}

	// Test that --model flag exists
	modelFlag := cloneCmd.Flags().Lookup("model")
	if modelFlag == nil {
		t.Error("Expected --model flag to exist")
	}
	if modelFlag.Shorthand != "m" {
		t.Errorf("Expected --model shorthand to be 'm', got '%s'", modelFlag.Shorthand)
	}

	// Test that --detach flag exists
	detachFlag := cloneCmd.Flags().Lookup("detach")
	if detachFlag == nil {
		t.Error("Expected --detach flag to exist")
	}
	if detachFlag.Shorthand != "d" {
		t.Errorf("Expected --detach shorthand to be 'd', got '%s'", detachFlag.Shorthand)
	}

	// Test that --foreground flag exists
	fgFlag := cloneCmd.Flags().Lookup("foreground")
	if fgFlag == nil {
		t.Error("Expected --foreground flag to exist")
	}

	// Test that --same-dir flag exists
	sameDirFlag := cloneCmd.Flags().Lookup("same-dir")
	if sameDirFlag == nil {
		t.Error("Expected --same-dir flag to exist")
	}

	// Test that --dry-run flag exists
	dryRunFlag := cloneCmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Error("Expected --dry-run flag to exist")
	}

	// Test that --forever flag exists
	foreverFlag := cloneCmd.Flags().Lookup("forever")
	if foreverFlag == nil {
		t.Error("Expected --forever flag to exist")
	}
	if foreverFlag.Shorthand != "F" {
		t.Errorf("Expected --forever shorthand to be 'F', got '%s'", foreverFlag.Shorthand)
	}

	// Test that --env flag exists
	envFlag := cloneCmd.Flags().Lookup("env")
	if envFlag == nil {
		t.Error("Expected --env flag to exist")
	}
	if envFlag.Shorthand != "e" {
		t.Errorf("Expected --env shorthand to be 'e', got '%s'", envFlag.Shorthand)
	}

	// Test that --on-complete flag exists
	onCompleteFlag := cloneCmd.Flags().Lookup("on-complete")
	if onCompleteFlag == nil {
		t.Error("Expected --on-complete flag to exist")
	}
}

func TestCloneCommandUsage(t *testing.T) {
	// Test that command usage is correct
	if cloneCmd.Use != "clone [agent-id-or-name]" {
		t.Errorf("Expected Use to be 'clone [agent-id-or-name]', got '%s'", cloneCmd.Use)
	}

	// Test that short description is set
	if cloneCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}

	// Test that long description is set
	if cloneCmd.Long == "" {
		t.Error("Expected Long description to be set")
	}

	// Check examples are set
	if cloneCmd.Example == "" {
		t.Error("Expected Example to be set")
	}
}

func TestCloneCommandArgs(t *testing.T) {
	// clone requires exactly 1 argument
	if cloneCmd.Args == nil {
		t.Error("Expected Args to be set")
	}

	// Test that args validation rejects no arguments
	err := cloneCmd.Args(cloneCmd, []string{})
	if err == nil {
		t.Error("Expected error when passing no arguments to clone")
	}

	// Test that args validation accepts exactly one argument
	err = cloneCmd.Args(cloneCmd, []string{"agent-id"})
	if err != nil {
		t.Errorf("Expected no error with one argument, got: %v", err)
	}

	// Test that args validation rejects too many arguments
	err = cloneCmd.Args(cloneCmd, []string{"agent-id", "extra"})
	if err == nil {
		t.Error("Expected error when passing too many arguments to clone")
	}
}
