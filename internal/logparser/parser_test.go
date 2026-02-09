package logparser

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewParser(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	if p == nil {
		t.Fatal("NewParser returned nil")
	}
	if p.out != &buf {
		t.Error("Parser output not set correctly")
	}
}

func TestProcessLineEmptyLine(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	p.ProcessLine("")
	p.ProcessLine("   ")
	p.ProcessLine("\n")
	p.ProcessLine("\t")

	if buf.Len() != 0 {
		t.Errorf("Empty lines should not produce output, got: %q", buf.String())
	}
}

func TestProcessLineInvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Invalid JSON should output raw line
	p.ProcessLine("not json at all")
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "not json at all") {
		t.Errorf("Invalid JSON should output raw line, got: %q", output)
	}
}

func TestProcessLineSystemInit(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	event := LogEvent{
		Type:      "system",
		Subtype:   "init",
		Model:     "opus-4.5",
		Cwd:       "/home/user/project",
		SessionID: "abc123",
	}
	jsonLine, _ := json.Marshal(event)

	p.ProcessLine(string(jsonLine))
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "System init") {
		t.Errorf("Should contain 'System init', got: %q", output)
	}
	if !strings.Contains(output, "model=opus-4.5") {
		t.Errorf("Should contain model, got: %q", output)
	}
	if !strings.Contains(output, "cwd=/home/user/project") {
		t.Errorf("Should contain cwd, got: %q", output)
	}
	if !strings.Contains(output, "session=abc123") {
		t.Errorf("Should contain session, got: %q", output)
	}
}

func TestProcessLineThinking(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	event := LogEvent{
		Type: "thinking",
		Text: "Let me think about this...",
	}
	jsonLine, _ := json.Marshal(event)

	p.ProcessLine(string(jsonLine))
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "thinking") {
		t.Errorf("Should contain thinking header, got: %q", output)
	}
	if !strings.Contains(output, "Let me think about this...") {
		t.Errorf("Should contain thinking text, got: %q", output)
	}
}

func TestProcessLineAssistantMessage(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	event := LogEvent{
		Type: "assistant",
		Message: &Message{
			Role: "assistant",
			Content: []ContentItem{
				{Type: "text", Text: "Here is my response"},
			},
		},
	}
	jsonLine, _ := json.Marshal(event)

	p.ProcessLine(string(jsonLine))
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "assistant") {
		t.Errorf("Should contain assistant header, got: %q", output)
	}
	if !strings.Contains(output, "Here is my response") {
		t.Errorf("Should contain message text, got: %q", output)
	}
}

func TestProcessLineUserMessage(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	event := LogEvent{
		Type: "user",
		Message: &Message{
			Role: "user",
			Content: []ContentItem{
				{Type: "text", Text: "User input here"},
			},
		},
	}
	jsonLine, _ := json.Marshal(event)

	p.ProcessLine(string(jsonLine))
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "user") {
		t.Errorf("Should contain user header, got: %q", output)
	}
	if !strings.Contains(output, "User input here") {
		t.Errorf("Should contain message text, got: %q", output)
	}
}

func TestProcessLineToolCallShell(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Test shell tool call
	eventJSON := `{
		"type": "tool_call",
		"tool_call": {
			"shellToolCall": {
				"args": {
					"command": "ls -la"
				}
			}
		}
	}`

	p.ProcessLine(eventJSON)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Shell") {
		t.Errorf("Should contain 'Shell', got: %q", output)
	}
	if !strings.Contains(output, "ls -la") {
		t.Errorf("Should contain command, got: %q", output)
	}
}

func TestProcessLineToolCallRead(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	eventJSON := `{
		"type": "tool_call",
		"tool_call": {
			"readToolCall": {
				"args": {
					"path": "/path/to/file.txt"
				}
			}
		}
	}`

	p.ProcessLine(eventJSON)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Read file") {
		t.Errorf("Should contain 'Read file', got: %q", output)
	}
	if !strings.Contains(output, "/path/to/file.txt") {
		t.Errorf("Should contain file path, got: %q", output)
	}
}

func TestProcessLineToolCallLs(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	eventJSON := `{
		"type": "tool_call",
		"tool_call": {
			"lsToolCall": {
				"args": {
					"path": "/home/user"
				}
			}
		}
	}`

	p.ProcessLine(eventJSON)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "List dir") {
		t.Errorf("Should contain 'List dir', got: %q", output)
	}
	if !strings.Contains(output, "/home/user") {
		t.Errorf("Should contain path, got: %q", output)
	}
}

func TestProcessLineToolCallWrite(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	eventJSON := `{
		"type": "tool_call",
		"tool_call": {
			"writeToolCall": {
				"args": {
					"path": "/path/to/file.txt",
					"content": "file content"
				}
			}
		}
	}`

	p.ProcessLine(eventJSON)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Write file") {
		t.Errorf("Should contain 'Write file', got: %q", output)
	}
}

func TestProcessLineToolCallApplyPatch(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	eventJSON := `{
		"type": "tool_call",
		"tool_call": {
			"applyPatchToolCall": {
				"args": {}
			}
		}
	}`

	p.ProcessLine(eventJSON)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Apply patch") {
		t.Errorf("Should contain 'Apply patch', got: %q", output)
	}
}

func TestProcessLineResult(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	event := LogEvent{
		Type:       "result",
		Subtype:    "success",
		Result:     "Operation completed successfully",
		DurationMs: 1500,
	}
	jsonLine, _ := json.Marshal(event)

	p.ProcessLine(string(jsonLine))
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Result") {
		t.Errorf("Should contain 'Result', got: %q", output)
	}
	if !strings.Contains(output, "success") {
		t.Errorf("Should contain subtype, got: %q", output)
	}
	if !strings.Contains(output, "1500ms") {
		t.Errorf("Should contain duration, got: %q", output)
	}
	if !strings.Contains(output, "Operation completed successfully") {
		t.Errorf("Should contain result text, got: %q", output)
	}
}

func TestProcessLineResultEmpty(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	event := LogEvent{
		Type:   "result",
		Result: "",
	}
	jsonLine, _ := json.Marshal(event)

	p.ProcessLine(string(jsonLine))
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "(empty)") {
		t.Errorf("Empty result should show '(empty)', got: %q", output)
	}
}

func TestProcessLineMergeConsecutiveThinking(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Multiple thinking events should be merged
	events := []LogEvent{
		{Type: "thinking", Text: "First thought "},
		{Type: "thinking", Text: "second thought "},
		{Type: "thinking", Text: "third thought."},
	}

	for _, e := range events {
		jsonLine, _ := json.Marshal(e)
		p.ProcessLine(string(jsonLine))
	}
	p.Flush()

	output := buf.String()
	// Should only have one [thinking] header
	headerCount := strings.Count(output, "[thinking]")
	if headerCount != 1 {
		t.Errorf("Should have exactly 1 thinking header, got %d in: %q", headerCount, output)
	}

	// Should contain all text
	if !strings.Contains(output, "First thought") {
		t.Error("Missing first thought")
	}
	if !strings.Contains(output, "second thought") {
		t.Error("Missing second thought")
	}
	if !strings.Contains(output, "third thought") {
		t.Error("Missing third thought")
	}
}

func TestProcessLineMergeConsecutiveAssistant(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Multiple assistant messages should be merged
	events := []LogEvent{
		{Type: "assistant", Message: &Message{Role: "assistant", Content: []ContentItem{{Text: "Part 1 "}}}},
		{Type: "assistant", Message: &Message{Role: "assistant", Content: []ContentItem{{Text: "Part 2 "}}}},
		{Type: "assistant", Message: &Message{Role: "assistant", Content: []ContentItem{{Text: "Part 3"}}}},
	}

	for _, e := range events {
		jsonLine, _ := json.Marshal(e)
		p.ProcessLine(string(jsonLine))
	}
	p.Flush()

	output := buf.String()
	// Should have merged content
	if !strings.Contains(output, "Part 1") {
		t.Error("Missing Part 1")
	}
	if !strings.Contains(output, "Part 2") {
		t.Error("Missing Part 2")
	}
	if !strings.Contains(output, "Part 3") {
		t.Error("Missing Part 3")
	}
}

func TestProcessLineFlushOnTypeChange(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Thinking followed by tool call should flush
	p.ProcessLine(`{"type": "thinking", "text": "Thinking..."}`)
	p.ProcessLine(`{"type": "tool_call", "tool_call": {"shellToolCall": {"args": {"command": "ls"}}}}`)
	p.Flush()

	output := buf.String()
	// Both should appear
	if !strings.Contains(output, "thinking") {
		t.Error("Should contain thinking")
	}
	if !strings.Contains(output, "Shell") {
		t.Error("Should contain Shell")
	}
}

func TestParserNeverPanics(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Various malformed inputs that might cause panics
	testCases := []string{
		`{}`,
		`{"type": null}`,
		`{"type": "tool_call", "tool_call": null}`,
		`{"type": "tool_call", "tool_call": {}}`,
		`{"type": "assistant", "message": null}`,
		`{"type": "assistant", "message": {"content": null}}`,
		`{"type": "assistant", "message": {"content": []}}`,
		`{"message": {"role": "test", "content": [{"text": ""}]}}`,
		`{"type": "unknown_type"}`,
		`{invalid json`,
		`null`,
		`[]`,
		`"string"`,
		`12345`,
		`true`,
	}

	for _, tc := range testCases {
		// Should not panic
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Parser panicked on input %q: %v", tc, r)
				}
			}()
			p.ProcessLine(tc)
		}()
	}

	p.Flush()
}

func TestAsSingleLine(t *testing.T) {
	p := &Parser{}

	tests := []struct {
		input    string
		expected string
	}{
		{"simple text", "simple text"},
		{"line1\nline2", "line1 line2"},
		{"line1\r\nline2", "line1 line2"},
		{"  spaces  ", "spaces"},
		{"multiple   spaces", "multiple spaces"},
		{"\t\ttabs\t\t", "tabs"},
		{"", ""},
	}

	for _, tt := range tests {
		result := p.asSingleLine(tt.input)
		if result != tt.expected {
			t.Errorf("asSingleLine(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeSingleLine(t *testing.T) {
	p := &Parser{}

	tests := []struct {
		input    string
		expected string
	}{
		{"simple text", "simple text"},
		{"line1\nline2", "line1 line2"},
		{"line1\r\nline2", "line1 line2"},
		{"  spaces preserved  ", "  spaces preserved  "},
	}

	for _, tt := range tests {
		result := p.sanitizeSingleLine(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeSingleLine(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestFmtHeader(t *testing.T) {
	p := &Parser{}

	tests := []struct {
		event    LogEvent
		expected string
	}{
		{LogEvent{Type: "assistant"}, "[assistant]"},
		{LogEvent{Type: "tool_call", Subtype: "shell"}, "[tool_call / shell]"},
		{LogEvent{Type: "", Subtype: ""}, ""},
		{LogEvent{Subtype: "only_subtype"}, "[only_subtype]"},
	}

	for _, tt := range tests {
		result := p.fmtHeader(&tt.event)
		if result != tt.expected {
			t.Errorf("fmtHeader(%+v) = %q, want %q", tt.event, result, tt.expected)
		}
	}
}

func TestPickTextFromContent(t *testing.T) {
	p := &Parser{}

	tests := []struct {
		content  []ContentItem
		expected string
	}{
		{[]ContentItem{{Text: "single"}}, "single"},
		{[]ContentItem{{Text: "first"}, {Text: "second"}}, "first second"},
		{[]ContentItem{{Text: ""}}, ""},
		{[]ContentItem{}, ""},
		{nil, ""},
	}

	for _, tt := range tests {
		result := p.pickTextFromContent(tt.content)
		if result != tt.expected {
			t.Errorf("pickTextFromContent(%+v) = %q, want %q", tt.content, result, tt.expected)
		}
	}
}

func TestGetStringArg(t *testing.T) {
	p := &Parser{}

	args := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"num":  123,
	}

	// Single key lookup
	if result := p.getStringArg(args, "key1"); result != "value1" {
		t.Errorf("getStringArg(key1) = %q, want 'value1'", result)
	}

	// Multiple key fallback
	if result := p.getStringArg(args, "missing", "key2"); result != "value2" {
		t.Errorf("getStringArg(missing, key2) = %q, want 'value2'", result)
	}

	// Non-string value
	if result := p.getStringArg(args, "num"); result != "" {
		t.Errorf("getStringArg(num) = %q, want ''", result)
	}

	// Missing key
	if result := p.getStringArg(args, "missing"); result != "" {
		t.Errorf("getStringArg(missing) = %q, want ''", result)
	}

	// Nil args
	if result := p.getStringArg(nil, "key1"); result != "" {
		t.Errorf("getStringArg(nil, key1) = %q, want ''", result)
	}
}

func TestSummarizeToolCallUnknown(t *testing.T) {
	p := &Parser{}

	event := &LogEvent{
		Type: "tool_call",
		ToolCall: map[string]interface{}{
			"customToolCall": map[string]interface{}{
				"args": map[string]interface{}{},
			},
		},
	}

	result := p.summarizeToolCall(event)
	if result != "customToolCall" {
		t.Errorf("summarizeToolCall returned %q, want 'customToolCall'", result)
	}
}

func TestSummarizeToolCallEmpty(t *testing.T) {
	p := &Parser{}

	event := &LogEvent{
		Type:     "tool_call",
		ToolCall: nil,
	}

	result := p.summarizeToolCall(event)
	if result != "Tool call" {
		t.Errorf("summarizeToolCall returned %q, want 'Tool call'", result)
	}

	event2 := &LogEvent{
		Type:     "tool_call",
		ToolCall: map[string]interface{}{},
	}

	result2 := p.summarizeToolCall(event2)
	if result2 != "Tool call" {
		t.Errorf("summarizeToolCall returned %q, want 'Tool call'", result2)
	}
}

func TestBodyForUnknownType(t *testing.T) {
	p := &Parser{}

	event := &LogEvent{Type: "custom_type"}
	result := p.bodyFor(event)
	if result != "custom_type event" {
		t.Errorf("bodyFor returned %q, want 'custom_type event'", result)
	}

	event2 := &LogEvent{}
	result2 := p.bodyFor(event2)
	if result2 != "(unknown event)" {
		t.Errorf("bodyFor returned %q, want '(unknown event)'", result2)
	}
}

func TestFlush(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Start a run
	p.ProcessLine(`{"type": "thinking", "text": "thinking text"}`)

	// Flush should close the run
	p.Flush()

	// After flush, openRun should be nil
	if p.openRun != nil {
		t.Error("Flush should clear openRun")
	}
}

func TestHeaderDeduplication(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Same header type events back to back - header should only appear once
	p.ProcessLine(`{"type": "result", "result": "first"}`)
	p.ProcessLine(`{"type": "result", "result": "second"}`)
	p.Flush()

	output := buf.String()
	// The [result] header should appear for each non-mergeable event
	// (result events are not merged like thinking/assistant)
	// But this tests that the last header tracking works
	if len(output) == 0 {
		t.Error("Expected some output")
	}
}

func TestMultipleContentItems(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	event := LogEvent{
		Type: "assistant",
		Message: &Message{
			Role: "assistant",
			Content: []ContentItem{
				{Type: "text", Text: "First part"},
				{Type: "text", Text: "Second part"},
				{Type: "text", Text: "Third part"},
			},
		},
	}
	jsonLine, _ := json.Marshal(event)

	p.ProcessLine(string(jsonLine))
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "First part") {
		t.Error("Missing first part")
	}
	if !strings.Contains(output, "Second part") {
		t.Error("Missing second part")
	}
	if !strings.Contains(output, "Third part") {
		t.Error("Missing third part")
	}
}

// --- Claude Code stream-json format tests ---

func TestProcessLineClaudeCodeToolUseInAssistant(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Claude Code embeds tool_use blocks in assistant message content
	eventJSON := `{
		"type": "assistant",
		"message": {
			"role": "assistant",
			"content": [
				{"type": "text", "text": "Let me read that file."},
				{"type": "tool_use", "id": "tu_1", "name": "Read", "input": {"file_path": "/src/main.go"}}
			]
		}
	}`

	p.ProcessLine(eventJSON)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Let me read that file.") {
		t.Errorf("Should contain text content, got: %q", output)
	}
	if !strings.Contains(output, "Read file: /src/main.go") {
		t.Errorf("Should contain tool use summary, got: %q", output)
	}
	if !strings.Contains(output, "[tool_use]") {
		t.Errorf("Should contain [tool_use] header, got: %q", output)
	}
}

func TestProcessLineClaudeCodeBashToolUse(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	eventJSON := `{
		"type": "assistant",
		"message": {
			"role": "assistant",
			"content": [
				{"type": "tool_use", "id": "tu_2", "name": "Bash", "input": {"command": "go test ./..."}}
			]
		}
	}`

	p.ProcessLine(eventJSON)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Shell: go test ./...") {
		t.Errorf("Should contain shell command summary, got: %q", output)
	}
}

func TestProcessLineClaudeCodeWriteToolUse(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	eventJSON := `{
		"type": "assistant",
		"message": {
			"role": "assistant",
			"content": [
				{"type": "tool_use", "id": "tu_3", "name": "Write", "input": {"file_path": "/tmp/test.go", "content": "package main"}}
			]
		}
	}`

	p.ProcessLine(eventJSON)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Write file: /tmp/test.go") {
		t.Errorf("Should contain write file summary, got: %q", output)
	}
}

func TestProcessLineClaudeCodeEditToolUse(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	eventJSON := `{
		"type": "assistant",
		"message": {
			"role": "assistant",
			"content": [
				{"type": "tool_use", "id": "tu_4", "name": "Edit", "input": {"file_path": "/src/main.go", "old_string": "foo", "new_string": "bar"}}
			]
		}
	}`

	p.ProcessLine(eventJSON)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Edit file: /src/main.go") {
		t.Errorf("Should contain edit file summary, got: %q", output)
	}
}

func TestProcessLineClaudeCodeGlobGrepToolUse(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	eventJSON := `{
		"type": "assistant",
		"message": {
			"role": "assistant",
			"content": [
				{"type": "tool_use", "id": "tu_5", "name": "Glob", "input": {"pattern": "**/*.go"}},
				{"type": "tool_use", "id": "tu_6", "name": "Grep", "input": {"pattern": "func main"}}
			]
		}
	}`

	p.ProcessLine(eventJSON)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Glob: **/*.go") {
		t.Errorf("Should contain glob summary, got: %q", output)
	}
	if !strings.Contains(output, "Grep: func main") {
		t.Errorf("Should contain grep summary, got: %q", output)
	}
}

func TestProcessLineClaudeCodeTextOnlyAssistant(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Text-only assistant messages should still merge like before
	events := []string{
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "Part A "}]}}`,
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "Part B"}]}}`,
	}

	for _, e := range events {
		p.ProcessLine(e)
	}
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Part A") {
		t.Error("Missing Part A")
	}
	if !strings.Contains(output, "Part B") {
		t.Error("Missing Part B")
	}
	// Should have only one [assistant] header (merged)
	headerCount := strings.Count(output, "[assistant]")
	if headerCount != 1 {
		t.Errorf("Should have exactly 1 assistant header, got %d in: %q", headerCount, output)
	}
}

func TestProcessLineClaudeCodeToolResultEvent(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	eventJSON := `{"type": "tool_result", "content": "file contents here"}`

	p.ProcessLine(eventJSON)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Result: file contents here") {
		t.Errorf("Should contain tool result, got: %q", output)
	}
}

func TestProcessLineClaudeCodeStandaloneToolUse(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	eventJSON := `{"type": "tool_use", "tool_name": "Bash", "input": {"command": "ls -la"}}`

	p.ProcessLine(eventJSON)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Shell: ls -la") {
		t.Errorf("Should contain shell summary, got: %q", output)
	}
}

// --- StreamingParser tests: realtime claude-code output parsing ---

func TestStreamingParserUsageAccumulation(t *testing.T) {
	var buf bytes.Buffer
	var callbackCount int
	var lastStats UsageStats

	sp := NewStreamingParser(&buf, func(stats UsageStats) {
		callbackCount++
		lastStats = stats
	})

	// Simulate a claude-code session with multiple usage events
	lines := []string{
		`{"type": "system", "subtype": "init", "model": "opus", "cwd": "/tmp", "session_id": "s1"}`,
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "Let me help."}]}, "usage": {"input_tokens": 100, "output_tokens": 50}}`,
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "tool_use", "id": "tu_1", "name": "Read", "input": {"file_path": "/src/main.go"}}]}, "usage": {"input_tokens": 200, "output_tokens": 30}}`,
		`{"type": "tool_result", "content": "package main\nfunc main() {}", "usage": {"input_tokens": 150, "output_tokens": 80}}`,
	}

	for _, line := range lines {
		sp.ProcessLine(line)
	}
	sp.Flush()

	// Usage should accumulate across events
	stats := sp.Stats()
	if stats.InputTokens != 450 {
		t.Errorf("Expected 450 input tokens, got %d", stats.InputTokens)
	}
	if stats.OutputTokens != 160 {
		t.Errorf("Expected 160 output tokens, got %d", stats.OutputTokens)
	}

	// Callback should have been called for each usage update
	if callbackCount < 3 {
		t.Errorf("Expected at least 3 callback invocations, got %d", callbackCount)
	}

	// Last stats from callback should match
	if lastStats.InputTokens != stats.InputTokens {
		t.Errorf("Last callback stats mismatch: got %d, want %d", lastStats.InputTokens, stats.InputTokens)
	}
}

func TestStreamingParserCurrentTaskUpdates(t *testing.T) {
	var buf bytes.Buffer
	var taskHistory []string

	sp := NewStreamingParser(&buf, func(stats UsageStats) {
		taskHistory = append(taskHistory, stats.CurrentTask)
	})

	// Simulate a claude-code session with various tool events
	lines := []string{
		// System init
		`{"type": "system", "subtype": "init", "model": "opus", "cwd": "/project"}`,
		// Assistant thinking
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "I'll read the file and make changes to fix the bug in the authentication module."}]}, "usage": {"input_tokens": 100, "output_tokens": 50}}`,
		// Tool use: Read file
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "tool_use", "id": "tu_1", "name": "Read", "input": {"file_path": "/src/auth.go"}}]}, "usage": {"input_tokens": 50, "output_tokens": 20}}`,
		// Tool use: Bash command
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "tool_use", "id": "tu_2", "name": "Bash", "input": {"command": "go test ./internal/auth/..."}}]}, "usage": {"input_tokens": 50, "output_tokens": 20}}`,
		// Tool use: Edit file
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "tool_use", "id": "tu_3", "name": "Edit", "input": {"file_path": "/src/auth.go", "old_string": "foo", "new_string": "bar"}}]}, "usage": {"input_tokens": 50, "output_tokens": 20}}`,
		// Tool use: Grep
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "tool_use", "id": "tu_4", "name": "Grep", "input": {"pattern": "func Auth"}}]}, "usage": {"input_tokens": 50, "output_tokens": 20}}`,
		// Result
		`{"type": "result", "subtype": "success", "result": "Done", "duration_ms": 5000, "usage": {"input_tokens": 50, "output_tokens": 10}}`,
	}

	for _, line := range lines {
		sp.ProcessLine(line)
	}
	sp.Flush()

	// Verify task updates happened and contain expected summaries
	if len(taskHistory) == 0 {
		t.Fatal("Expected task history to be non-empty")
	}

	// Check that specific task summaries appeared in order
	expectedTasks := []string{
		"Initializing...",
		"Thinking:",
		"Read: /src/auth.go",
		"Shell: go test ./internal/auth/...",
		"Edit: /src/auth.go",
		"Search",
		"Result: success",
	}

	for _, expected := range expectedTasks {
		found := false
		for _, task := range taskHistory {
			if strings.Contains(task, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected task containing %q in history, got: %v", expected, taskHistory)
		}
	}
}

func TestStreamingParserRealtimeCallbackTiming(t *testing.T) {
	// Verify that callbacks fire after each line, not batched
	var buf bytes.Buffer
	callbackAfterLine := make(map[int]UsageStats)
	lineNum := 0

	sp := NewStreamingParser(&buf, func(stats UsageStats) {
		callbackAfterLine[lineNum] = stats
	})

	lines := []string{
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "Hello"}]}, "usage": {"input_tokens": 100, "output_tokens": 50}}`,
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "tool_use", "id": "tu_1", "name": "Bash", "input": {"command": "ls"}}]}, "usage": {"input_tokens": 200, "output_tokens": 100}}`,
		`{"type": "result", "subtype": "success", "result": "ok", "usage": {"input_tokens": 50, "output_tokens": 10}}`,
	}

	for i, line := range lines {
		lineNum = i
		sp.ProcessLine(line)
	}
	sp.Flush()

	// Line 0 should have triggered a callback with 100 input tokens
	if stats, ok := callbackAfterLine[0]; ok {
		if stats.InputTokens != 100 {
			t.Errorf("After line 0: expected 100 input tokens, got %d", stats.InputTokens)
		}
	} else {
		t.Error("Expected callback after line 0")
	}

	// Line 1 should have accumulated tokens
	if stats, ok := callbackAfterLine[1]; ok {
		if stats.InputTokens != 300 {
			t.Errorf("After line 1: expected 300 input tokens, got %d", stats.InputTokens)
		}
	} else {
		t.Error("Expected callback after line 1")
	}

	// Line 2 should have all tokens
	if stats, ok := callbackAfterLine[2]; ok {
		if stats.InputTokens != 350 {
			t.Errorf("After line 2: expected 350 input tokens, got %d", stats.InputTokens)
		}
	} else {
		t.Error("Expected callback after line 2")
	}
}

func TestStreamingParserDirectTokenFields(t *testing.T) {
	// Test that message.usage (assistant events) and top-level usage (result events) are extracted
	var buf bytes.Buffer
	sp := NewStreamingParser(&buf, nil)

	lines := []string{
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "hi"}], "usage": {"input_tokens": 500, "output_tokens": 200}}}`,
		`{"type": "result", "subtype": "success", "result": "done", "usage": {"input_tokens": 100, "output_tokens": 50}}`,
	}

	for _, line := range lines {
		sp.ProcessLine(line)
	}
	sp.Flush()

	stats := sp.Stats()
	if stats.InputTokens != 600 {
		t.Errorf("Expected 600 input tokens, got %d", stats.InputTokens)
	}
	if stats.OutputTokens != 250 {
		t.Errorf("Expected 250 output tokens, got %d", stats.OutputTokens)
	}
}

func TestStreamingParserCacheTokenFields(t *testing.T) {
	// Test that cache_read_input_tokens and cache_creation_input_tokens are included in input totals
	var buf bytes.Buffer
	sp := NewStreamingParser(&buf, nil)

	sp.ProcessLine(`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "hi"}], "usage": {"input_tokens": 3, "output_tokens": 7, "cache_read_input_tokens": 13931, "cache_creation_input_tokens": 6152}}}`)
	sp.Flush()

	stats := sp.Stats()
	expectedInput := int64(3 + 13931 + 6152)
	if stats.InputTokens != expectedInput {
		t.Errorf("Expected %d input tokens (including cache), got %d", expectedInput, stats.InputTokens)
	}
	if stats.OutputTokens != 7 {
		t.Errorf("Expected 7 output tokens, got %d", stats.OutputTokens)
	}
}

func TestStreamingParserTotalCostUSD(t *testing.T) {
	// Test that total_cost_usd from result events is captured
	var buf bytes.Buffer
	sp := NewStreamingParser(&buf, nil)

	sp.ProcessLine(`{"type": "result", "subtype": "success", "result": "done", "total_cost_usd": 0.0091211, "usage": {"input_tokens": 3, "output_tokens": 7, "cache_read_input_tokens": 13931}}`)
	sp.Flush()

	stats := sp.Stats()
	if stats.TotalCostUSD != 0.0091211 {
		t.Errorf("Expected TotalCostUSD 0.0091211, got %f", stats.TotalCostUSD)
	}
}

func TestStreamingParserPromptCompletionTokenFields(t *testing.T) {
	// Test alternative naming: prompt_tokens / completion_tokens
	var buf bytes.Buffer
	sp := NewStreamingParser(&buf, nil)

	sp.ProcessLine(`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "hi"}]}, "usage": {"prompt_tokens": 300, "completion_tokens": 150}}`)
	sp.Flush()

	stats := sp.Stats()
	if stats.InputTokens != 300 {
		t.Errorf("Expected 300 input tokens from prompt_tokens, got %d", stats.InputTokens)
	}
	if stats.OutputTokens != 150 {
		t.Errorf("Expected 150 output tokens from completion_tokens, got %d", stats.OutputTokens)
	}
}

func TestStreamingParserFullClaudeCodeSession(t *testing.T) {
	// End-to-end test simulating a realistic Claude Code streaming session
	var buf bytes.Buffer
	var finalStats UsageStats

	sp := NewStreamingParser(&buf, func(stats UsageStats) {
		finalStats = stats
	})

	// Simulate a realistic Claude Code session
	session := []string{
		// Init
		`{"type": "system", "subtype": "init", "model": "claude-opus-4-6", "cwd": "/Users/dev/project", "session_id": "sess_abc123", "timestamp_ms": 1700000000000}`,
		// Assistant starts thinking
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "I'll help you fix this bug. Let me first look at the relevant files."}]}, "usage": {"input_tokens": 1500, "output_tokens": 30}}`,
		// Read a file
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "tool_use", "id": "toolu_01", "name": "Read", "input": {"file_path": "/Users/dev/project/src/handler.go"}}]}, "usage": {"input_tokens": 200, "output_tokens": 15}}`,
		// Tool result
		`{"type": "tool_result", "content": "package handler\n\nfunc Handle(r *Request) error {\n\treturn nil\n}"}`,
		// Grep for related code
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "tool_use", "id": "toolu_02", "name": "Grep", "input": {"pattern": "Handle\\(", "path": "/Users/dev/project/src"}}]}, "usage": {"input_tokens": 3000, "output_tokens": 20}}`,
		// Tool result
		`{"type": "tool_result", "content": "src/handler.go:3:func Handle(r *Request) error {\nsrc/main.go:15:  handler.Handle(req)"}`,
		// Edit the file
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "I see the issue. The handler doesn't validate the request. Let me fix that."}, {"type": "tool_use", "id": "toolu_03", "name": "Edit", "input": {"file_path": "/Users/dev/project/src/handler.go", "old_string": "func Handle(r *Request) error {\n\treturn nil\n}", "new_string": "func Handle(r *Request) error {\n\tif r == nil {\n\t\treturn fmt.Errorf(\"nil request\")\n\t}\n\treturn nil\n}"}}]}, "usage": {"input_tokens": 3500, "output_tokens": 60}}`,
		// Run tests
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "tool_use", "id": "toolu_04", "name": "Bash", "input": {"command": "cd /Users/dev/project && go test ./..."}}]}, "usage": {"input_tokens": 4000, "output_tokens": 20}}`,
		// Tool result
		`{"type": "tool_result", "content": "ok\tproject/src\t0.003s"}`,
		// Final response
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "The fix has been applied and all tests pass."}]}, "usage": {"input_tokens": 4500, "output_tokens": 15}}`,
		// Result
		`{"type": "result", "subtype": "success", "result": "Task completed successfully", "duration_ms": 12500, "usage": {"input_tokens": 500, "output_tokens": 10}}`,
	}

	for _, line := range session {
		sp.ProcessLine(line)
	}
	sp.Flush()

	output := buf.String()

	// Verify key output elements are present
	checks := []struct {
		desc    string
		content string
	}{
		{"system init", "System init"},
		{"model info", "claude-opus-4-6"},
		{"assistant text", "fix this bug"},
		{"read tool use", "Read file:"},
		{"tool result", "Result:"},
		{"grep tool use", "Grep:"},
		{"edit tool use", "Edit file:"},
		{"bash tool use", "Shell: cd /Users/dev/project && go test"},
		{"final text", "tests pass"},
		{"result", "success"},
	}

	for _, c := range checks {
		if !strings.Contains(output, c.content) {
			t.Errorf("Output should contain %s (%q), got: %q", c.desc, c.content, output)
		}
	}

	// Verify usage stats accumulated
	if finalStats.InputTokens == 0 {
		t.Error("Expected non-zero input tokens")
	}
	if finalStats.OutputTokens == 0 {
		t.Error("Expected non-zero output tokens")
	}

	// Final task should reflect the result
	if !strings.Contains(finalStats.CurrentTask, "Result") {
		t.Errorf("Final task should contain 'Result', got: %q", finalStats.CurrentTask)
	}
}

func TestStreamingParserClaudeCodeStandaloneToolUseTask(t *testing.T) {
	// Test standalone tool_use events (not embedded in assistant message)
	var buf bytes.Buffer
	var lastTask string

	sp := NewStreamingParser(&buf, func(stats UsageStats) {
		lastTask = stats.CurrentTask
	})

	lines := []string{
		`{"type": "tool_use", "tool_name": "Read", "input": {"file_path": "/etc/hosts"}, "usage": {"input_tokens": 10, "output_tokens": 5}}`,
		`{"type": "tool_use", "name": "WebSearch", "input": {"query": "golang error handling"}, "usage": {"input_tokens": 10, "output_tokens": 5}}`,
		`{"type": "tool_use", "tool_name": "Task", "input": {}, "usage": {"input_tokens": 10, "output_tokens": 5}}`,
	}

	expectedTasks := []string{
		"Read:",
		"Web search",
		"Subagent",
	}

	for i, line := range lines {
		sp.ProcessLine(line)
		if !strings.Contains(lastTask, expectedTasks[i]) {
			t.Errorf("After line %d: expected task containing %q, got %q", i, expectedTasks[i], lastTask)
		}
	}
}

func TestStreamingParserNoCallbackOnNonUsageEvents(t *testing.T) {
	// Events without usage data should not trigger callback
	var buf bytes.Buffer
	callbackCount := 0

	sp := NewStreamingParser(&buf, func(stats UsageStats) {
		callbackCount++
	})

	// These events have no usage data and no task-changing info beyond init
	sp.ProcessLine(`{"type": "system", "subtype": "init", "model": "opus"}`)

	afterInit := callbackCount

	// Plain text events without usage should not trigger additional callbacks
	sp.ProcessLine(`{"type": "thinking", "text": "hmm"}`)
	sp.Flush()

	if callbackCount != afterInit {
		t.Errorf("Non-usage events should not trigger extra callbacks: got %d after init count %d", callbackCount, afterInit)
	}
}

func TestStreamingParserCursorFormatToolCall(t *testing.T) {
	// Verify the parser also handles Cursor-format tool_call events for task tracking
	var buf bytes.Buffer
	var lastTask string

	sp := NewStreamingParser(&buf, func(stats UsageStats) {
		lastTask = stats.CurrentTask
	})

	// Cursor-format shell tool call with usage
	sp.ProcessLine(`{"type": "tool_call", "tool_call": {"shellToolCall": {"args": {"command": "npm test"}}}, "usage": {"input_tokens": 100, "output_tokens": 20}}`)

	if !strings.Contains(lastTask, "Shell") || !strings.Contains(lastTask, "npm test") {
		t.Errorf("Expected task 'Shell: npm test', got: %q", lastTask)
	}

	// Cursor-format read tool call
	sp.ProcessLine(`{"type": "tool_call", "tool_call": {"readToolCall": {"args": {"file_path": "/very/long/path/to/deeply/nested/component/file.tsx"}}}, "usage": {"input_tokens": 50, "output_tokens": 10}}`)

	if !strings.Contains(lastTask, "Read:") {
		t.Errorf("Expected task containing 'Read:', got: %q", lastTask)
	}

	stats := sp.Stats()
	if stats.InputTokens != 150 {
		t.Errorf("Expected 150 input tokens, got %d", stats.InputTokens)
	}
}

func TestScanLogFileClaudeCodeFormat(t *testing.T) {
	// Test ScanLogFile with claude-code format events
	logContent := strings.Join([]string{
		`{"type": "system", "subtype": "init", "model": "opus"}`,
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "hello"}]}, "usage": {"input_tokens": 500, "output_tokens": 100}}`,
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "tool_use", "id": "tu_1", "name": "Bash", "input": {"command": "ls"}}]}, "usage": {"input_tokens": 800, "output_tokens": 50}}`,
		`{"type": "tool_result", "content": "file1.go\nfile2.go"}`,
		`{"type": "result", "subtype": "success", "result": "done", "usage": {"input_tokens": 200, "output_tokens": 30}}`,
	}, "\n")

	stats := ScanLogFile(strings.NewReader(logContent))

	if stats.InputTokens != 1500 {
		t.Errorf("Expected 1500 input tokens, got %d", stats.InputTokens)
	}
	if stats.OutputTokens != 180 {
		t.Errorf("Expected 180 output tokens, got %d", stats.OutputTokens)
	}
}

func TestParseEventClaudeCodeFormats(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantType string
		wantNil  bool
	}{
		{
			name:     "system init",
			line:     `{"type": "system", "subtype": "init", "model": "opus"}`,
			wantType: "system",
		},
		{
			name:     "assistant with tool_use",
			line:     `{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "tool_use", "name": "Read", "input": {"file_path": "/foo"}}]}}`,
			wantType: "assistant",
		},
		{
			name:     "tool_result",
			line:     `{"type": "tool_result", "content": "file contents"}`,
			wantType: "tool_result",
		},
		{
			name:     "standalone tool_use",
			line:     `{"type": "tool_use", "tool_name": "Bash", "input": {"command": "ls"}}`,
			wantType: "tool_use",
		},
		{
			name:     "result success",
			line:     `{"type": "result", "subtype": "success", "result": "done", "duration_ms": 5000}`,
			wantType: "result",
		},
		{
			name:    "empty line",
			line:    "",
			wantNil: true,
		},
		{
			name:    "invalid json",
			line:    "not json",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := ParseEvent(tt.line)
			if tt.wantNil {
				if event != nil {
					t.Errorf("Expected nil event, got type=%q", event.Type)
				}
				return
			}
			if event == nil {
				t.Fatal("Expected non-nil event")
			}
			if event.Type != tt.wantType {
				t.Errorf("Expected type %q, got %q", tt.wantType, event.Type)
			}
		})
	}
}
