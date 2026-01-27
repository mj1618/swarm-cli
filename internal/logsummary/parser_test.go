package logsummary

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matt/swarm-cli/internal/state"
)

func TestParseEmptyLogFile(t *testing.T) {
	// Create temp log file
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logFile, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	agent := &state.AgentState{
		ID:          "test123",
		StartedAt:   time.Now().Add(-10 * time.Minute),
		CurrentIter: 5,
		LogFile:     logFile,
	}

	summary, err := Parse(agent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if summary.IterationsCompleted != 5 {
		t.Errorf("Expected 5 iterations, got %d", summary.IterationsCompleted)
	}
}

func TestParseNoLogFile(t *testing.T) {
	agent := &state.AgentState{
		ID:          "test123",
		StartedAt:   time.Now().Add(-10 * time.Minute),
		CurrentIter: 3,
		LogFile:     "",
	}

	summary, err := Parse(agent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if summary.IterationsCompleted != 3 {
		t.Errorf("Expected 3 iterations, got %d", summary.IterationsCompleted)
	}
}

func TestParseMissingLogFile(t *testing.T) {
	agent := &state.AgentState{
		ID:          "test123",
		StartedAt:   time.Now().Add(-10 * time.Minute),
		CurrentIter: 3,
		LogFile:     "/nonexistent/path/to/log.log",
	}

	summary, err := Parse(agent)
	if err != nil {
		t.Fatalf("Parse should not fail for missing file: %v", err)
	}

	if summary.IterationsCompleted != 3 {
		t.Errorf("Expected 3 iterations, got %d", summary.IterationsCompleted)
	}
}

func TestParseToolCalls(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Create log with tool calls
	logs := []string{
		`[swarm] === Iteration 1/10 ===`,
		`{"type":"tool_call","timestamp_ms":1706450100000,"tool_call":{"writeToolCall":{"args":{"path":"/tmp/file1.txt"}}}}`,
		`{"type":"tool_call","timestamp_ms":1706450110000,"tool_call":{"writeToolCall":{"args":{"path":"/tmp/file2.txt"}}}}`,
		`{"type":"tool_call","timestamp_ms":1706450120000,"tool_call":{"StrReplace":{"args":{"path":"/tmp/existing.txt"}}}}`,
		`{"type":"tool_call","timestamp_ms":1706450130000,"tool_call":{"shellToolCall":{"args":{"command":"git status"}}}}`,
	}

	content := ""
	for _, line := range logs {
		content += line + "\n"
	}
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	agent := &state.AgentState{
		ID:          "test123",
		StartedAt:   time.Now().Add(-10 * time.Minute),
		CurrentIter: 1,
		LogFile:     logFile,
	}

	summary, err := Parse(agent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if summary.ToolCalls != 4 {
		t.Errorf("Expected 4 tool calls, got %d", summary.ToolCalls)
	}

	if summary.FilesCreated != 2 {
		t.Errorf("Expected 2 files created, got %d", summary.FilesCreated)
	}

	if summary.FilesModified != 1 {
		t.Errorf("Expected 1 file modified, got %d", summary.FilesModified)
	}
}

func TestParseErrors(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logs := []string{
		`[swarm] === Iteration 1/10 ===`,
		`{"type":"result","subtype":"error","timestamp_ms":1706450100000,"result":"Permission denied"}`,
		`[swarm] === Iteration 2/10 ===`,
		`{"type":"result","subtype":"success","timestamp_ms":1706450200000,"result":"OK"}`,
		`[swarm] === Iteration 3/10 ===`,
		`{"type":"result","subtype":"error","timestamp_ms":1706450300000,"result":"Connection failed"}`,
	}

	content := ""
	for _, line := range logs {
		content += line + "\n"
	}
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	agent := &state.AgentState{
		ID:          "test123",
		StartedAt:   time.Now().Add(-10 * time.Minute),
		CurrentIter: 3,
		LogFile:     logFile,
	}

	summary, err := Parse(agent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(summary.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(summary.Errors))
	}

	if len(summary.Errors) > 0 && summary.Errors[0].Iteration != 1 {
		t.Errorf("Expected first error on iteration 1, got %d", summary.Errors[0].Iteration)
	}
}

func TestParseGitCommitEvents(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logs := []string{
		`[swarm] === Iteration 1/10 ===`,
		`{"type":"tool_call","timestamp_ms":1706450100000,"tool_call":{"shellToolCall":{"args":{"command":"git commit -m \"Add feature\""}}}}`,
		`[swarm] === Iteration 2/10 ===`,
		`{"type":"tool_call","timestamp_ms":1706450200000,"tool_call":{"shellToolCall":{"args":{"command":"go test ./..."}}}}`,
	}

	content := ""
	for _, line := range logs {
		content += line + "\n"
	}
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	agent := &state.AgentState{
		ID:          "test123",
		StartedAt:   time.Now().Add(-10 * time.Minute),
		CurrentIter: 2,
		LogFile:     logFile,
	}

	summary, err := Parse(agent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(summary.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(summary.Events))
	}

	foundCommit := false
	foundTest := false
	for _, event := range summary.Events {
		if event.Type == "commit" {
			foundCommit = true
		}
		if event.Type == "test" {
			foundTest = true
		}
	}

	if !foundCommit {
		t.Error("Expected to find commit event")
	}
	if !foundTest {
		t.Error("Expected to find test event")
	}
}

func TestParseDuration(t *testing.T) {
	startTime := time.Now().Add(-1*time.Hour - 30*time.Minute - 45*time.Second)
	agent := &state.AgentState{
		ID:          "test123",
		StartedAt:   startTime,
		CurrentIter: 10,
		LogFile:     "",
	}

	summary, err := Parse(agent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Duration should be approximately 1h30m45s
	expected := int64(1*3600 + 30*60 + 45)
	tolerance := int64(2) // Allow 2 seconds tolerance

	if summary.DurationSeconds < expected-tolerance || summary.DurationSeconds > expected+tolerance {
		t.Errorf("Expected duration ~%d seconds, got %d", expected, summary.DurationSeconds)
	}
}

func TestParseTerminatedAgent(t *testing.T) {
	startTime := time.Now().Add(-2 * time.Hour)
	terminatedAt := startTime.Add(1 * time.Hour)

	agent := &state.AgentState{
		ID:           "test123",
		StartedAt:    startTime,
		TerminatedAt: &terminatedAt,
		CurrentIter:  10,
		LogFile:      "",
	}

	summary, err := Parse(agent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Duration should be 1 hour (terminated - started)
	expected := int64(3600)
	tolerance := int64(2)

	if summary.DurationSeconds < expected-tolerance || summary.DurationSeconds > expected+tolerance {
		t.Errorf("Expected duration ~%d seconds, got %d", expected, summary.DurationSeconds)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int64
		expected string
	}{
		{30, "30s"},
		{90, "1m 30s"},
		{3661, "1h 1m 1s"},
		{7200, "2h 0m 0s"},
		{0, "0s"},
	}

	for _, tc := range tests {
		summary := &Summary{DurationSeconds: tc.seconds}
		result := summary.FormatDuration()
		if result != tc.expected {
			t.Errorf("FormatDuration(%d) = %q, want %q", tc.seconds, result, tc.expected)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		max      int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is..."},
		{"exact", 5, "exact"},
		{"with\nnewlines", 20, "with newlines"},
		{"  spaces  ", 20, "spaces"},
	}

	for _, tc := range tests {
		result := truncateString(tc.input, tc.max)
		if result != tc.expected {
			t.Errorf("truncateString(%q, %d) = %q, want %q", tc.input, tc.max, result, tc.expected)
		}
	}
}

func TestExtractArgs(t *testing.T) {
	// Test with nested args
	data := map[string]interface{}{
		"args": map[string]interface{}{
			"path": "/tmp/test.txt",
		},
	}
	args := extractArgs(data)
	if args == nil {
		t.Fatal("Expected args to be extracted")
	}
	if args["path"] != "/tmp/test.txt" {
		t.Errorf("Expected path to be /tmp/test.txt, got %v", args["path"])
	}

	// Test with nil
	if extractArgs(nil) != nil {
		t.Error("Expected nil for nil input")
	}

	// Test with non-map
	if extractArgs("string") != nil {
		t.Error("Expected nil for non-map input")
	}
}

func TestParseLastAction(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Create a log ending with an assistant message
	lastMsg := logEntry{
		Type:        "assistant",
		TimestampMs: 1706450200000,
		Message: &logMessage{
			Role: "assistant",
			Content: []contentItem{
				{Type: "text", Text: "I have completed the task successfully."},
			},
		},
	}
	lastMsgJSON, _ := json.Marshal(lastMsg)

	logs := []string{
		`[swarm] === Iteration 1/1 ===`,
		`{"type":"tool_call","timestamp_ms":1706450100000,"tool_call":{"shellToolCall":{"args":{"command":"ls"}}}}`,
		string(lastMsgJSON),
	}

	content := ""
	for _, line := range logs {
		content += line + "\n"
	}
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	agent := &state.AgentState{
		ID:          "test123",
		StartedAt:   time.Now().Add(-10 * time.Minute),
		CurrentIter: 1,
		LogFile:     logFile,
	}

	summary, err := Parse(agent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if summary.LastAction != "I have completed the task successfully." {
		t.Errorf("Expected last action to be the assistant message, got %q", summary.LastAction)
	}
}

func TestParseDeduplicatesErrors(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Same error repeated multiple times
	logs := []string{
		`[swarm] === Iteration 1/10 ===`,
		`{"type":"result","subtype":"error","timestamp_ms":1706450100000,"result":"Permission denied"}`,
		`{"type":"result","subtype":"error","timestamp_ms":1706450110000,"result":"Permission denied"}`,
		`{"type":"result","subtype":"error","timestamp_ms":1706450120000,"result":"Permission denied"}`,
	}

	content := ""
	for _, line := range logs {
		content += line + "\n"
	}
	if err := os.WriteFile(logFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	agent := &state.AgentState{
		ID:          "test123",
		StartedAt:   time.Now().Add(-10 * time.Minute),
		CurrentIter: 1,
		LogFile:     logFile,
	}

	summary, err := Parse(agent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should only have one error due to deduplication
	if len(summary.Errors) != 1 {
		t.Errorf("Expected 1 deduplicated error, got %d", len(summary.Errors))
	}
}
