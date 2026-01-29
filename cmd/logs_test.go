package cmd

import (
	"regexp"
	"testing"
	"time"
)

func TestMatchesGrep(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		patterns []string
		invert   bool
		expected bool
	}{
		{
			name:     "empty patterns matches everything",
			line:     "some log line",
			patterns: nil,
			invert:   false,
			expected: true,
		},
		{
			name:     "single pattern match",
			line:     "error: something went wrong",
			patterns: []string{"error"},
			invert:   false,
			expected: true,
		},
		{
			name:     "single pattern no match",
			line:     "info: all good",
			patterns: []string{"error"},
			invert:   false,
			expected: false,
		},
		{
			name:     "case insensitive match",
			line:     "ERROR: something went wrong",
			patterns: []string{"(?i)error"},
			invert:   false,
			expected: true,
		},
		{
			name:     "multiple patterns OR logic - first matches",
			line:     "error: critical failure",
			patterns: []string{"error", "warning"},
			invert:   false,
			expected: true,
		},
		{
			name:     "multiple patterns OR logic - second matches",
			line:     "warning: disk space low",
			patterns: []string{"error", "warning"},
			invert:   false,
			expected: true,
		},
		{
			name:     "multiple patterns OR logic - none match",
			line:     "info: process started",
			patterns: []string{"error", "warning"},
			invert:   false,
			expected: false,
		},
		{
			name:     "invert - line matches pattern",
			line:     "[swarm] status update",
			patterns: []string{"\\[swarm\\]"},
			invert:   true,
			expected: false,
		},
		{
			name:     "invert - line does not match pattern",
			line:     "regular log line",
			patterns: []string{"\\[swarm\\]"},
			invert:   true,
			expected: true,
		},
		{
			name:     "regex pattern",
			line:     "tool_use: Read { path: \"src/main.go\" }",
			patterns: []string{"tool_use.*Read"},
			invert:   false,
			expected: true,
		},
		{
			name:     "regex pattern no match",
			line:     "tool_use: Write { path: \"src/main.go\" }",
			patterns: []string{"tool_use.*Read"},
			invert:   false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compile patterns
			var patterns []*regexp.Regexp
			for _, p := range tt.patterns {
				re, err := regexp.Compile(p)
				if err != nil {
					t.Fatalf("failed to compile pattern %q: %v", p, err)
				}
				patterns = append(patterns, re)
			}

			result := MatchesGrep(tt.line, patterns, tt.invert)
			if result != tt.expected {
				t.Errorf("MatchesGrep(%q, %v, %v) = %v, want %v",
					tt.line, tt.patterns, tt.invert, result, tt.expected)
			}
		})
	}
}

func TestParseTimeFlag(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expectError bool
	}{
		{
			name:        "empty value",
			value:       "",
			expectError: false,
		},
		{
			name:        "relative duration minutes",
			value:       "30m",
			expectError: false,
		},
		{
			name:        "relative duration hours",
			value:       "2h",
			expectError: false,
		},
		{
			name:        "relative duration days",
			value:       "1d",
			expectError: false,
		},
		{
			name:        "RFC3339",
			value:       "2024-01-28T10:00:00Z",
			expectError: false,
		},
		{
			name:        "date-time with seconds",
			value:       "2024-01-28 10:00:00",
			expectError: false,
		},
		{
			name:        "date-time without seconds",
			value:       "2024-01-28 10:00",
			expectError: false,
		},
		{
			name:        "date only",
			value:       "2024-01-28",
			expectError: false,
		},
		{
			name:        "invalid format",
			value:       "not-a-date",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTimeFlag(tt.value)
			if tt.expectError && err == nil {
				t.Errorf("ParseTimeFlag(%q) expected error, got nil", tt.value)
			}
			if !tt.expectError && err != nil {
				t.Errorf("ParseTimeFlag(%q) unexpected error: %v", tt.value, err)
			}
		})
	}
}

func TestIsLineInTimeRange(t *testing.T) {
	// Test line with valid timestamp
	lineWithTimestamp := "2024-01-28 10:15:32 | some log content"

	// Parse some test times
	beforeLine, _ := ParseTimeFlag("2024-01-28 10:00:00")
	afterLine, _ := ParseTimeFlag("2024-01-28 10:30:00")
	wayAfterLine, _ := ParseTimeFlag("2024-01-28 11:00:00")

	tests := []struct {
		name     string
		line     string
		since    string
		until    string
		expected bool
	}{
		{
			name:     "no filter",
			line:     lineWithTimestamp,
			since:    "",
			until:    "",
			expected: true,
		},
		{
			name:     "since before line timestamp",
			line:     lineWithTimestamp,
			since:    "2024-01-28 10:00:00",
			until:    "",
			expected: true,
		},
		{
			name:     "since after line timestamp",
			line:     lineWithTimestamp,
			since:    "2024-01-28 10:30:00",
			until:    "",
			expected: false,
		},
		{
			name:     "until after line timestamp",
			line:     lineWithTimestamp,
			since:    "",
			until:    "2024-01-28 10:30:00",
			expected: true,
		},
		{
			name:     "until before line timestamp",
			line:     lineWithTimestamp,
			since:    "",
			until:    "2024-01-28 10:00:00",
			expected: false,
		},
		{
			name:     "line in range",
			line:     lineWithTimestamp,
			since:    "2024-01-28 10:00:00",
			until:    "2024-01-28 10:30:00",
			expected: true,
		},
		{
			name:     "line without timestamp included",
			line:     "  continuation line",
			since:    "2024-01-28 10:00:00",
			until:    "2024-01-28 10:30:00",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var since, until = time.Time{}, time.Time{}
			if tt.since != "" {
				since, _ = ParseTimeFlag(tt.since)
			}
			if tt.until != "" {
				until, _ = ParseTimeFlag(tt.until)
			}

			result := IsLineInTimeRange(tt.line, since, until)
			if result != tt.expected {
				t.Errorf("IsLineInTimeRange(%q, %v, %v) = %v, want %v",
					tt.line, tt.since, tt.until, result, tt.expected)
			}
		})
	}

	// Suppress unused variable warnings
	_ = beforeLine
	_ = afterLine
	_ = wayAfterLine
}

func TestExtractTimestamp(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		expectValid  bool
	}{
		{
			name:        "valid timestamp",
			line:        "2024-01-28 10:15:32 | some content",
			expectValid: true,
		},
		{
			name:        "no timestamp",
			line:        "some content without timestamp",
			expectValid: false,
		},
		{
			name:        "short line",
			line:        "short",
			expectValid: false,
		},
		{
			name:        "empty line",
			line:        "",
			expectValid: false,
		},
		{
			name:        "invalid timestamp format",
			line:        "28-01-2024 10:15:32 | wrong format",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := ExtractTimestamp(tt.line)
			if tt.expectValid && ts.IsZero() {
				t.Errorf("ExtractTimestamp(%q) expected valid time, got zero", tt.line)
			}
			if !tt.expectValid && !ts.IsZero() {
				t.Errorf("ExtractTimestamp(%q) expected zero time, got %v", tt.line, ts)
			}
		})
	}
}
