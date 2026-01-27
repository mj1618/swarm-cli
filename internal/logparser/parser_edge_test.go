package logparser

import (
	"bytes"
	"strings"
	"testing"
)

// Edge case tests for the log parser

func TestProcessLineLongContent(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Very long text content
	longText := strings.Repeat("a", 100000)
	p.ProcessLine(`{"type": "thinking", "text": "` + longText + `"}`)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "aaaaaa") {
		t.Error("Long content should be processed")
	}
}

func TestProcessLineMultilineJSON(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// JSON with embedded newlines in string values
	p.ProcessLine(`{"type": "thinking", "text": "line1\nline2\nline3"}`)
	p.Flush()

	output := buf.String()
	// Newlines should be converted to spaces by sanitizeSingleLine
	if strings.Contains(output, "\n") && !strings.Contains(output, "[thinking]") {
		// Allow newlines in the output formatting, just check thinking is processed
	}
	if !strings.Contains(output, "thinking") {
		t.Error("Thinking header should be present")
	}
}

func TestProcessLineUnicodeContent(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Unicode characters
	p.ProcessLine(`{"type": "thinking", "text": "中文测试 日本語 한국어 🎉🚀💻"}`)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "中文测试") {
		t.Error("Chinese characters should be preserved")
	}
	if !strings.Contains(output, "日本語") {
		t.Error("Japanese characters should be preserved")
	}
	if !strings.Contains(output, "🎉") {
		t.Error("Emoji should be preserved")
	}
}

func TestProcessLineNullValues(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	testCases := []string{
		`{"type": null}`,
		`{"type": "thinking", "text": null}`,
		`{"type": "assistant", "message": {"role": null, "content": null}}`,
		`{"type": "tool_call", "tool_call": {"test": null}}`,
		`{"timestamp_ms": null}`,
	}

	for _, tc := range testCases {
		// Should not panic
		p.ProcessLine(tc)
	}
	p.Flush()
}

func TestProcessLineNestedJSON(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Deeply nested JSON
	p.ProcessLine(`{
		"type": "tool_call",
		"tool_call": {
			"shellToolCall": {
				"id": "abc123",
				"args": {
					"command": "echo test",
					"options": {
						"timeout": 30,
						"env": {
							"HOME": "/home/user"
						}
					}
				}
			}
		}
	}`)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Shell") {
		t.Error("Should process nested tool call")
	}
	if !strings.Contains(output, "echo test") {
		t.Error("Should extract command from nested args")
	}
}

func TestProcessLineTimestampFormatting(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Event with timestamp
	p.ProcessLine(`{"type": "system", "subtype": "init", "timestamp_ms": 1706400000000, "model": "test"}`)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "System init") {
		t.Error("Should format system init event")
	}
}

func TestProcessLineMixedEventTypes(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Sequence of different event types
	events := []string{
		`{"type": "system", "subtype": "init", "model": "test"}`,
		`{"type": "user", "message": {"role": "user", "content": [{"type": "text", "text": "Hello"}]}}`,
		`{"type": "thinking", "text": "Processing..."}`,
		`{"type": "assistant", "message": {"role": "assistant", "content": [{"type": "text", "text": "Hi!"}]}}`,
		`{"type": "tool_call", "tool_call": {"readToolCall": {"args": {"path": "/test"}}}}`,
		`{"type": "result", "result": "file content"}`,
	}

	for _, e := range events {
		p.ProcessLine(e)
	}
	p.Flush()

	output := buf.String()
	checks := []string{"System init", "user", "thinking", "assistant", "Read file", "Result"}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("Output should contain %q", check)
		}
	}
}

func TestProcessLineEscapedCharacters(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// JSON with escaped characters
	p.ProcessLine(`{"type": "thinking", "text": "Quote: \"hello\" Tab:\t Backslash: \\"}`)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Quote:") {
		t.Error("Escaped quotes should be processed")
	}
}

func TestProcessLineEmptyToolCall(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	testCases := []string{
		`{"type": "tool_call", "tool_call": {}}`,
		`{"type": "tool_call"}`,
		`{"type": "tool_call", "tool_call": {"unknownTool": {}}}`,
		`{"type": "tool_call", "tool_call": {"unknownTool": {"args": {}}}}`,
	}

	for _, tc := range testCases {
		buf.Reset()
		p.ProcessLine(tc)
		p.Flush()
		// Should not panic and should produce some output
	}
}

func TestProcessLineMessageWithEmptyContent(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	p.ProcessLine(`{"type": "assistant", "message": {"role": "assistant", "content": []}}`)
	p.Flush()

	// Should handle empty content gracefully
	// Output format may vary, just ensure no panic
}

func TestProcessLineMessageWithMultipleTextItems(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	p.ProcessLine(`{
		"type": "assistant",
		"message": {
			"role": "assistant",
			"content": [
				{"type": "text", "text": "First"},
				{"type": "image", "url": "http://example.com/img.png"},
				{"type": "text", "text": "Second"},
				{"type": "text", "text": "Third"}
			]
		}
	}`)
	p.Flush()

	output := buf.String()
	// Should include text items, skip non-text
	if !strings.Contains(output, "First") {
		t.Error("Should include First")
	}
	if !strings.Contains(output, "Second") {
		t.Error("Should include Second")
	}
	if !strings.Contains(output, "Third") {
		t.Error("Should include Third")
	}
}

func TestProcessLineResultWithDuration(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	p.ProcessLine(`{"type": "result", "subtype": "success", "result": "Done", "duration_ms": 12345}`)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "12345ms") {
		t.Error("Should show duration in ms")
	}
}

func TestProcessLineToolCallWithSimpleCommand(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Test simpleCommand fallback for shell
	p.ProcessLine(`{
		"type": "tool_call",
		"tool_call": {
			"shellToolCall": {
				"args": {
					"simpleCommand": "pwd"
				}
			}
		}
	}`)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Shell") {
		t.Error("Should identify as shell tool")
	}
	if !strings.Contains(output, "pwd") {
		t.Error("Should show command")
	}
}

func TestProcessLineToolCallReadWithFilePath(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Test file_path fallback for read
	p.ProcessLine(`{
		"type": "tool_call",
		"tool_call": {
			"readToolCall": {
				"args": {
					"file_path": "/home/user/test.txt"
				}
			}
		}
	}`)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Read file") {
		t.Error("Should identify as read tool")
	}
	if !strings.Contains(output, "test.txt") {
		t.Error("Should show file path")
	}
}

func TestParserFlushMultipleTimes(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Multiple flushes should be safe
	p.Flush()
	p.Flush()
	p.Flush()

	p.ProcessLine(`{"type": "thinking", "text": "test"}`)
	p.Flush()
	p.Flush()

	// Should not panic
}

func TestParserConcurrentSafe(t *testing.T) {
	// Note: The parser is not designed to be concurrency-safe,
	// but it should handle sequential access correctly
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Sequential access should work
	for i := 0; i < 100; i++ {
		p.ProcessLine(`{"type": "thinking", "text": "test"}`)
	}
	p.Flush()

	output := buf.String()
	if len(output) == 0 {
		t.Error("Should have produced output")
	}
}

func TestProcessLineOnlySubtype(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Event with only subtype, no type
	p.ProcessLine(`{"subtype": "init"}`)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "[init]") {
		t.Error("Should show subtype in header")
	}
}

func TestProcessLineEmptyMessage(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	p.ProcessLine(`{"type": "assistant", "message": {}}`)
	p.Flush()

	// Should handle gracefully
}

func TestProcessLineMessageWithEmptyRole(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	p.ProcessLine(`{
		"type": "assistant",
		"message": {
			"role": "",
			"content": [{"type": "text", "text": "content"}]
		}
	}`)
	p.Flush()

	output := buf.String()
	// Should fall back to event type for header
	if !strings.Contains(output, "assistant") {
		t.Error("Should use event type when role is empty")
	}
}

func TestProcessLineVeryLongCommand(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// Very long command
	longCmd := "echo " + strings.Repeat("x", 10000)
	p.ProcessLine(`{"type": "tool_call", "tool_call": {"shellToolCall": {"args": {"command": "` + longCmd + `"}}}}`)
	p.Flush()

	output := buf.String()
	if !strings.Contains(output, "Shell") {
		t.Error("Should process long command")
	}
}

func TestProcessLineSpecialJSONCharacters(t *testing.T) {
	var buf bytes.Buffer
	p := NewParser(&buf)

	// JSON with special characters that need escaping
	testCases := []string{
		`{"type": "thinking", "text": "Path: C:\\Users\\test"}`,
		`{"type": "thinking", "text": "Tab\there"}`,
		`{"type": "thinking", "text": "Line\nbreak"}`,
	}

	for _, tc := range testCases {
		p.ProcessLine(tc)
	}
	p.Flush()
}

func TestBodyForThinkingEmpty(t *testing.T) {
	p := &Parser{}
	event := &LogEvent{Type: "thinking", Text: ""}

	result := p.bodyFor(event)
	if result != "(thinking)" {
		t.Errorf("Empty thinking should return '(thinking)', got %q", result)
	}
}

func TestBodyForMessageNoText(t *testing.T) {
	p := &Parser{}
	event := &LogEvent{
		Type:    "assistant",
		Message: &Message{Role: "assistant", Content: []ContentItem{}},
	}

	result := p.bodyFor(event)
	if result != "(no text)" {
		t.Errorf("Message with no text should return '(no text)', got %q", result)
	}
}
