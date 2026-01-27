package logparser

import (
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
