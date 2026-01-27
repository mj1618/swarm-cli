package cmd

import (
	"testing"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no escape codes",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "single escape code",
			input:    "\x1b[2Khello",
			expected: "hello",
		},
		{
			name:     "multiple escape codes",
			input:    "\x1b[2K\x1b[Gmodel - Description",
			expected: "model - Description",
		},
		{
			name:     "cursor movement codes",
			input:    "\x1b[2K\x1b[1A\x1b[2K\x1b[GAvailable models",
			expected: "Available models",
		},
		{
			name:     "color codes",
			input:    "\x1b[31mred text\x1b[0m",
			expected: "red text",
		},
		{
			name:     "mixed content",
			input:    "before\x1b[2Kmiddle\x1b[0mafter",
			expected: "beforemiddleafter",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.expected {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetClaudeCodeModels(t *testing.T) {
	models := getClaudeCodeModels()

	if len(models) == 0 {
		t.Error("getClaudeCodeModels() returned empty list")
	}

	// Check that opus and sonnet are present
	hasOpus := false
	hasSonnet := false
	for _, m := range models {
		if m.ID == "opus" {
			hasOpus = true
		}
		if m.ID == "sonnet" {
			hasSonnet = true
		}
	}

	if !hasOpus {
		t.Error("getClaudeCodeModels() missing 'opus' model")
	}
	if !hasSonnet {
		t.Error("getClaudeCodeModels() missing 'sonnet' model")
	}
}

func TestGetFallbackCursorModels(t *testing.T) {
	models := getFallbackCursorModels()

	if len(models) == 0 {
		t.Error("getFallbackCursorModels() returned empty list")
	}

	// Check that some expected models are present
	hasOpus := false
	hasSonnet := false
	for _, m := range models {
		if m.ID == "opus-4.5-thinking" {
			hasOpus = true
		}
		if m.ID == "sonnet-4.5-thinking" {
			hasSonnet = true
		}
	}

	if !hasOpus {
		t.Error("getFallbackCursorModels() missing 'opus-4.5-thinking' model")
	}
	if !hasSonnet {
		t.Error("getFallbackCursorModels() missing 'sonnet-4.5-thinking' model")
	}
}
