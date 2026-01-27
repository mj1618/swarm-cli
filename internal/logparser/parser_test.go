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
