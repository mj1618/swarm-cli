package logparser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/color"
)

// Parser processes JSONL log lines and pretty-prints them.
// It is designed to never panic or return errors that would terminate the agent.
type Parser struct {
	out        io.Writer
	openRun    *openRun
	lastHeader string
}

type openRun struct {
	kind   string // "assistant", "user", "thinking"
	lastCh string
}

// LogEvent represents a parsed log line from the agent.
type LogEvent struct {
	Type        string                 `json:"type"`
	Subtype     string                 `json:"subtype"`
	TimestampMs int64                  `json:"timestamp_ms"`
	Message     *Message               `json:"message"`
	Text        string                 `json:"text"`
	Model       string                 `json:"model"`
	Cwd         string                 `json:"cwd"`
	SessionID   string                 `json:"session_id"`
	ToolCall    map[string]interface{} `json:"tool_call"`
	Result      string                 `json:"result"`
	DurationMs  int64                  `json:"duration_ms"`
	// Usage fields (may be present in API response events)
	Usage       *Usage                 `json:"usage,omitempty"`
	// Alternative usage locations in different API formats
	InputTokens  *int64                `json:"input_tokens,omitempty"`
	OutputTokens *int64                `json:"output_tokens,omitempty"`
}

// Usage represents token usage from an API response.
type Usage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	TotalTokens  int64 `json:"total_tokens"`
	// Alternative field names used by some APIs
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
}

// UsageStats holds accumulated usage statistics.
type UsageStats struct {
	InputTokens  int64
	OutputTokens int64
	CurrentTask  string
}

// Message represents a user or assistant message.
type Message struct {
	Role    string        `json:"role"`
	Content []ContentItem `json:"content"`
}

// ContentItem represents a content item in a message.
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewParser creates a new log parser that writes to the given output.
func NewParser(out io.Writer) *Parser {
	return &Parser{
		out: out,
	}
}

// UsageCallback is called when usage stats are updated.
type UsageCallback func(stats UsageStats)

// StreamingParser extends Parser to track usage stats and emit callbacks.
type StreamingParser struct {
	*Parser
	stats         UsageStats
	onUsageUpdate UsageCallback
}

// NewStreamingParser creates a parser that tracks usage and calls the callback on updates.
func NewStreamingParser(out io.Writer, onUsageUpdate UsageCallback) *StreamingParser {
	return &StreamingParser{
		Parser:        NewParser(out),
		onUsageUpdate: onUsageUpdate,
	}
}

// ProcessLine processes a line and updates usage stats.
func (sp *StreamingParser) ProcessLine(line string) {
	// Try to extract usage before normal processing
	sp.extractUsage(line)
	sp.Parser.ProcessLine(line)
}

// Stats returns the current usage statistics.
func (sp *StreamingParser) Stats() UsageStats {
	return sp.stats
}

// extractUsage extracts token usage and current task from a log line.
func (sp *StreamingParser) extractUsage(line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	var event LogEvent
	if err := json.Unmarshal([]byte(trimmed), &event); err != nil {
		return
	}

	updated := false

	// Extract usage from various possible locations
	if event.Usage != nil {
		inputTokens := event.Usage.InputTokens
		if inputTokens == 0 {
			inputTokens = event.Usage.PromptTokens
		}
		outputTokens := event.Usage.OutputTokens
		if outputTokens == 0 {
			outputTokens = event.Usage.CompletionTokens
		}
		if inputTokens > 0 || outputTokens > 0 {
			sp.stats.InputTokens += inputTokens
			sp.stats.OutputTokens += outputTokens
			updated = true
		}
	}

	// Check for direct token fields
	if event.InputTokens != nil && *event.InputTokens > 0 {
		sp.stats.InputTokens += *event.InputTokens
		updated = true
	}
	if event.OutputTokens != nil && *event.OutputTokens > 0 {
		sp.stats.OutputTokens += *event.OutputTokens
		updated = true
	}

	// Update current task based on event type
	taskUpdated := sp.updateCurrentTask(&event)
	if taskUpdated {
		updated = true
	}

	// Emit callback if anything changed
	if updated && sp.onUsageUpdate != nil {
		sp.onUsageUpdate(sp.stats)
	}
}

// updateCurrentTask updates the current task based on the event.
func (sp *StreamingParser) updateCurrentTask(event *LogEvent) bool {
	var newTask string

	switch event.Type {
	case "tool_call":
		newTask = sp.summarizeToolCallForTask(event)
	case "assistant":
		// Only update if we have meaningful content
		if event.Message != nil {
			text := sp.pickTextFromContent(event.Message.Content)
			if len(text) > 50 {
				text = text[:47] + "..."
			}
			if text != "" {
				newTask = "Thinking: " + text
			}
		}
	case "system":
		if event.Subtype == "init" {
			newTask = "Initializing..."
		}
	case "result":
		if event.Subtype != "" {
			newTask = "Result: " + event.Subtype
		}
	}

	if newTask != "" && newTask != sp.stats.CurrentTask {
		sp.stats.CurrentTask = newTask
		return true
	}
	return false
}

// summarizeToolCallForTask creates a short summary for the current task display.
func (sp *StreamingParser) summarizeToolCallForTask(event *LogEvent) string {
	if event.ToolCall == nil {
		return "Tool call"
	}

	var toolName string
	var inner map[string]interface{}
	for k, v := range event.ToolCall {
		toolName = k
		if m, ok := v.(map[string]interface{}); ok {
			inner = m
		}
		break
	}

	if toolName == "" {
		return "Tool call"
	}

	var args map[string]interface{}
	if inner != nil {
		if a, ok := inner["args"].(map[string]interface{}); ok {
			args = a
		}
	}

	switch toolName {
	case "shellToolCall":
		if cmd := sp.getStringArg(args, "command", "simpleCommand"); cmd != "" {
			cmd = sp.asSingleLine(cmd)
			if len(cmd) > 40 {
				cmd = cmd[:37] + "..."
			}
			return "Shell: " + cmd
		}
		return "Shell"
	case "lsToolCall":
		if path := sp.getStringArg(args, "path"); path != "" {
			return "List: " + sp.truncatePath(path)
		}
		return "List dir"
	case "readToolCall":
		if path := sp.getStringArg(args, "file_path", "path"); path != "" {
			return "Read: " + sp.truncatePath(path)
		}
		return "Read file"
	case "writeToolCall":
		if path := sp.getStringArg(args, "file_path", "path"); path != "" {
			return "Write: " + sp.truncatePath(path)
		}
		return "Write file"
	case "applyPatchToolCall":
		return "Apply patch"
	case "searchToolCall", "grepToolCall":
		return "Search"
	case "webSearchToolCall":
		return "Web search"
	}

	// Clean up tool name
	name := strings.TrimSuffix(toolName, "ToolCall")
	name = strings.TrimSuffix(name, "Call")
	return name
}

// truncatePath shortens a file path for display.
func (sp *StreamingParser) truncatePath(path string) string {
	// Get just the filename if path is too long
	if len(path) > 30 {
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			filename := parts[len(parts)-1]
			if len(filename) <= 30 {
				return filename
			}
			return filename[:27] + "..."
		}
	}
	return path
}

// ProcessLine processes a single log line.
// It never returns an error - on parse failure, it outputs the raw line.
func (p *Parser) ProcessLine(line string) {
	defer func() {
		// Recover from any panics - just output raw line
		if r := recover(); r != nil {
			p.safeWrite(line + "\n\n")
		}
	}()

	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	var event LogEvent
	if err := json.Unmarshal([]byte(trimmed), &event); err != nil {
		// Not valid JSON - output raw
		p.flushRun()
		p.safeWrite(trimmed + "\n\n")
		return
	}

	header := p.fmtHeader(&event)

	// Merge consecutive assistant/user message fragments
	if (event.Type == "assistant" || event.Type == "user") && event.Message != nil {
		role := event.Message.Role
		if role == "" {
			role = event.Type
		}
		text := p.pickRawTextFromContent(event.Message.Content)
		p.startOrAppendRun(role, fmt.Sprintf("[%s]", role), text)
		return
	}

	// Merge thinking deltas
	if event.Type == "thinking" {
		text := p.sanitizeSingleLine(event.Text)
		p.startOrAppendRun("thinking", "[thinking]", text)
		return
	}

	// Non-mergeable event: flush and print
	p.flushRun()
	p.maybePrintHeader(header)
	p.safeWrite(p.bodyFor(&event) + "\n\n")
}

// Flush ensures any buffered content is written.
func (p *Parser) Flush() {
	p.flushRun()
}

func (p *Parser) safeWrite(s string) {
	// Never let write errors propagate
	_, _ = p.out.Write([]byte(s))
}

func (p *Parser) flushRun() {
	if p.openRun == nil {
		return
	}
	p.safeWrite("\n\n")
	p.openRun = nil
}

func (p *Parser) maybePrintHeader(header string) {
	if header == "" {
		return
	}
	if header == p.lastHeader {
		return
	}
	headerColor := color.New(color.FgCyan, color.Bold)
	headerColor.Fprint(p.out, header+"\n")
	p.lastHeader = header
}

func (p *Parser) startOrAppendRun(kind, header, fragment string) {
	if fragment == "" {
		return
	}

	if p.openRun == nil || p.openRun.kind != kind {
		p.flushRun()
		p.maybePrintHeader(header)
		p.openRun = &openRun{kind: kind}
	}

	p.safeWrite(fragment)
	if len(fragment) > 0 {
		p.openRun.lastCh = string(fragment[len(fragment)-1])
	}
}

func (p *Parser) fmtHeader(event *LogEvent) string {
	var pieces []string
	if event.Type != "" {
		pieces = append(pieces, event.Type)
	}
	if event.Subtype != "" {
		pieces = append(pieces, event.Subtype)
	}
	if len(pieces) > 0 {
		return fmt.Sprintf("[%s]", strings.Join(pieces, " / "))
	}
	return ""
}

func (p *Parser) fmtPrefix(event *LogEvent) string {
	var pieces []string

	// Format timestamp
	if event.TimestampMs > 0 {
		t := time.UnixMilli(event.TimestampMs)
		pieces = append(pieces, t.Format(time.RFC3339))
	}

	if event.Type != "" {
		pieces = append(pieces, event.Type)
	}
	if event.Subtype != "" {
		pieces = append(pieces, event.Subtype)
	}

	if len(pieces) > 0 {
		return fmt.Sprintf("[%s] ", strings.Join(pieces, " / "))
	}
	return ""
}

func (p *Parser) bodyFor(event *LogEvent) string {
	// System init
	if event.Type == "system" && event.Subtype == "init" {
		var bits []string
		if event.Model != "" {
			bits = append(bits, fmt.Sprintf("model=%s", event.Model))
		}
		if event.Cwd != "" {
			bits = append(bits, fmt.Sprintf("cwd=%s", event.Cwd))
		}
		if event.SessionID != "" {
			bits = append(bits, fmt.Sprintf("session=%s", event.SessionID))
		}
		if len(bits) > 0 {
			return fmt.Sprintf("System init (%s)", strings.Join(bits, ", "))
		}
		return "System init"
	}

	// Thinking
	if event.Type == "thinking" {
		text := p.asSingleLine(event.Text)
		if text == "" {
			return "(thinking)"
		}
		return text
	}

	// User/assistant messages
	if (event.Type == "user" || event.Type == "assistant") && event.Message != nil {
		text := p.pickTextFromContent(event.Message.Content)
		if text == "" {
			return "(no text)"
		}
		return text
	}

	// Tool call
	if event.Type == "tool_call" {
		return p.summarizeToolCall(event)
	}

	// Result
	if event.Type == "result" {
		var bits []string
		if event.Subtype != "" {
			bits = append(bits, event.Subtype)
		}
		if event.DurationMs > 0 {
			bits = append(bits, fmt.Sprintf("%dms", event.DurationMs))
		}
		msg := p.asSingleLine(event.Result)
		if msg == "" {
			msg = "(empty)"
		}
		if len(bits) > 0 {
			return fmt.Sprintf("Result (%s): %s", strings.Join(bits, ", "), msg)
		}
		return fmt.Sprintf("Result: %s", msg)
	}

	// Fallback
	if event.Type != "" {
		return fmt.Sprintf("%s event", event.Type)
	}
	return "(unknown event)"
}

func (p *Parser) summarizeToolCall(event *LogEvent) string {
	if event.ToolCall == nil {
		return "Tool call"
	}

	// Find the tool name (first key in tool_call)
	var toolName string
	var inner map[string]interface{}
	for k, v := range event.ToolCall {
		toolName = k
		if m, ok := v.(map[string]interface{}); ok {
			inner = m
		}
		break
	}

	if toolName == "" {
		return "Tool call"
	}

	// Get args
	var args map[string]interface{}
	if inner != nil {
		if a, ok := inner["args"].(map[string]interface{}); ok {
			args = a
		}
	}

	switch toolName {
	case "shellToolCall":
		if cmd := p.getStringArg(args, "command", "simpleCommand"); cmd != "" {
			return fmt.Sprintf("Shell: %s", p.asSingleLine(cmd))
		}
		return "Shell: (command)"

	case "lsToolCall":
		if path := p.getStringArg(args, "path"); path != "" {
			return fmt.Sprintf("List dir: %s", p.asSingleLine(path))
		}
		return "List dir"

	case "readToolCall":
		if path := p.getStringArg(args, "file_path", "path"); path != "" {
			return fmt.Sprintf("Read file: %s", p.asSingleLine(path))
		}
		return "Read file"

	case "writeToolCall":
		return "Write file"

	case "applyPatchToolCall":
		return "Apply patch"
	}

	// Fallback: show tool name
	return toolName
}

func (p *Parser) getStringArg(args map[string]interface{}, keys ...string) string {
	if args == nil {
		return ""
	}
	for _, key := range keys {
		if v, ok := args[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func (p *Parser) pickTextFromContent(content []ContentItem) string {
	var parts []string
	for _, item := range content {
		if item.Text != "" {
			parts = append(parts, item.Text)
		}
	}
	return p.asSingleLine(strings.Join(parts, "\n"))
}

func (p *Parser) pickRawTextFromContent(content []ContentItem) string {
	var out strings.Builder
	for _, item := range content {
		if item.Text != "" {
			out.WriteString(item.Text)
		}
	}
	return p.sanitizeSingleLine(out.String())
}

var (
	newlineRe    = regexp.MustCompile(`\r?\n`)
	whitespaceRe = regexp.MustCompile(`\s+`)
)

func (p *Parser) asSingleLine(s string) string {
	s = newlineRe.ReplaceAllString(s, " ")
	s = whitespaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func (p *Parser) sanitizeSingleLine(s string) string {
	// Keep it single-line but don't trim/collapse spaces
	return newlineRe.ReplaceAllString(s, " ")
}

// ParseEvent parses a single log line and returns the event.
// Returns nil if the line is not valid JSON.
func ParseEvent(line string) *LogEvent {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	var event LogEvent
	if err := json.Unmarshal([]byte(trimmed), &event); err != nil {
		return nil
	}
	return &event
}

// ScanLogFile reads a log file and returns accumulated usage stats.
// This is useful for getting stats from an existing log file.
func ScanLogFile(reader io.Reader) UsageStats {
	sp := NewStreamingParser(io.Discard, nil)
	
	scanner := newLineScanner(reader)
	for scanner.Scan() {
		sp.extractUsage(scanner.Text())
	}
	
	return sp.stats
}

// newLineScanner creates a scanner with a larger buffer for long lines.
func newLineScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	return scanner
}
