package cmd

import (
	"testing"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		max      int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "short",
			max:      10,
			expected: "short",
		},
		{
			name:     "string equal to max",
			input:    "exactly10!",
			max:      10,
			expected: "exactly10!",
		},
		{
			name:     "string longer than max",
			input:    "this is a very long string",
			max:      10,
			expected: "this is...",
		},
		{
			name:     "empty string",
			input:    "",
			max:      10,
			expected: "",
		},
		{
			name:     "truncate to 3 chars leaves only ...",
			input:    "hello",
			max:      3,
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.max)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q",
					tt.input, tt.max, result, tt.expected)
			}
		})
	}
}

func TestAttachCmdExists(t *testing.T) {
	// Verify the attach command is registered
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "attach" {
			found = true
			break
		}
	}
	if !found {
		t.Error("attach command not found in root command")
	}
}

func TestAttachCmdFlags(t *testing.T) {
	// Verify the flags exist
	flags := []string{"no-interactive", "tail"}
	for _, flag := range flags {
		f := attachCmd.Flags().Lookup(flag)
		if f == nil {
			t.Errorf("flag --%s not found on attach command", flag)
		}
	}
}

func TestAttachCmdRequiresArg(t *testing.T) {
	// Verify that exactly 1 argument is required
	if attachCmd.Args == nil {
		t.Error("attach command should have Args validation")
	}
}
