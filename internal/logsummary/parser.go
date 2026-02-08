package logsummary

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/mj1618/swarm-cli/internal/state"
)

// Summary contains parsed summary information from agent logs.
type Summary struct {
	DurationSeconds     int64 `json:"duration_seconds"`
	IterationsCompleted int   `json:"iterations_completed"`

	AvgIterationSeconds   int64 `json:"avg_iteration_seconds,omitempty"`
	FastestIteration      int64 `json:"fastest_iteration_seconds,omitempty"`
	FastestIterationNum   int   `json:"fastest_iteration_num,omitempty"`
	SlowestIteration      int64 `json:"slowest_iteration_seconds,omitempty"`
	SlowestIterationNum   int   `json:"slowest_iteration_num,omitempty"`

	FilesCreated  int `json:"files_created"`
	FilesModified int `json:"files_modified"`
	FilesDeleted  int `json:"files_deleted"`
	ToolCalls     int `json:"tool_calls"`

	Errors []LogError `json:"errors,omitempty"`
	Events []LogEvent `json:"events,omitempty"`

	LastAction string `json:"last_action,omitempty"`
}

// LogError represents an error found in logs.
type LogError struct {
	Iteration int    `json:"iteration"`
	Message   string `json:"message"`
}

// LogEvent represents a notable event from logs.
type LogEvent struct {
	Iteration int    `json:"iteration"`
	Type      string `json:"type"`
	Message   string `json:"message"`
}

// logEntry represents a parsed JSON log entry.
type logEntry struct {
	Type        string                 `json:"type"`
	Subtype     string                 `json:"subtype"`
	TimestampMs int64                  `json:"timestamp_ms"`
	Text        string                 `json:"text"`
	Result      string                 `json:"result"`
	ToolCall    map[string]interface{} `json:"tool_call"`
	Message     *logMessage            `json:"message"`
}

type logMessage struct {
	Role    string        `json:"role"`
	Content []contentItem `json:"content"`
}

type contentItem struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

// Patterns for parsing logs
var (
	iterationStartPattern = regexp.MustCompile(`\[swarm\] === Iteration (\d+)/(\d+) ===`)
	errorPattern          = regexp.MustCompile(`(?i)\b(error|failed|exception|panic)\b`)
)

// Parse reads agent logs and generates a summary.
func Parse(agent *state.AgentState) (*Summary, error) {
	now := time.Now()
	endTime := now
	if agent.TerminatedAt != nil {
		endTime = *agent.TerminatedAt
	}

	summary := &Summary{
		DurationSeconds:     int64(endTime.Sub(agent.StartedAt).Seconds()),
		IterationsCompleted: agent.CurrentIter,
	}

	if agent.LogFile == "" {
		return summary, nil
	}

	file, err := os.Open(agent.LogFile)
	if err != nil {
		if os.IsNotExist(err) {
			return summary, nil
		}
		return summary, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	currentIteration := 0
	var iterationStartTime int64 // timestamp in ms
	var iterationDurations []int64
	filesCreated := make(map[string]bool)
	filesModified := make(map[string]bool)
	filesDeleted := make(map[string]bool)
	var lastLine string
	var lastTimestamp int64
	seenErrors := make(map[string]bool) // Deduplicate errors

	scanner := bufio.NewScanner(file)
	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		lastLine = line

		// Check for iteration markers (plain text lines)
		if matches := iterationStartPattern.FindStringSubmatch(line); len(matches) > 0 {
			// Record duration of previous iteration if we have timestamps
			if currentIteration > 0 && iterationStartTime > 0 && lastTimestamp > iterationStartTime {
				dur := (lastTimestamp - iterationStartTime) / 1000 // convert to seconds
				if dur > 0 {
					iterationDurations = append(iterationDurations, dur)
				}
			}
			currentIteration++
			iterationStartTime = lastTimestamp // Will be updated by next JSON line
			continue
		}

		// Try to parse as JSON
		var entry logEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Not JSON - check for error patterns in raw text
			if currentIteration > 0 && errorPattern.MatchString(line) {
				errKey := fmt.Sprintf("%d:%s", currentIteration, truncateString(line, 50))
				if !seenErrors[errKey] {
					seenErrors[errKey] = true
					summary.Errors = append(summary.Errors, LogError{
						Iteration: currentIteration,
						Message:   truncateString(line, 100),
					})
				}
			}
			continue
		}

		// Update timestamp tracking
		if entry.TimestampMs > 0 {
			lastTimestamp = entry.TimestampMs
			if iterationStartTime == 0 || (currentIteration > 0 && iterationStartTime < entry.TimestampMs-1000) {
				// Update iteration start time with first timestamp after iteration marker
				if iterationStartTime == 0 {
					iterationStartTime = entry.TimestampMs
				}
			}
		}

		// Count tool calls (Cursor format)
		if entry.Type == "tool_call" && entry.ToolCall != nil {
			summary.ToolCalls++

			// Track file operations from tool calls
			for toolName, toolData := range entry.ToolCall {
				if args := extractArgs(toolData); args != nil {
					switch toolName {
					case "writeToolCall", "Write":
						if path := getStringFromMap(args, "path", "file_path"); path != "" {
							filesCreated[path] = true
						}
					case "StrReplace", "strReplaceToolCall", "Edit":
						if path := getStringFromMap(args, "path", "file_path"); path != "" {
							filesModified[path] = true
						}
					case "Delete", "deleteToolCall":
						if path := getStringFromMap(args, "path", "file_path"); path != "" {
							filesDeleted[path] = true
						}
					}
				}
			}
		}

		// Count tool calls from Claude Code assistant messages with tool_use content blocks
		if entry.Type == "assistant" && entry.Message != nil {
			for _, item := range entry.Message.Content {
				if item.Type == "tool_use" {
					summary.ToolCalls++
					switch item.Name {
					case "Write":
						if path := getStringFromMap(item.Input, "file_path"); path != "" {
							filesCreated[path] = true
						}
					case "Edit":
						if path := getStringFromMap(item.Input, "file_path"); path != "" {
							filesModified[path] = true
						}
					}
				}
			}
		}

		// Track errors from result entries
		if entry.Type == "result" && entry.Subtype == "error" {
			errMsg := entry.Result
			if errMsg == "" {
				errMsg = "Unknown error"
			}
			errKey := fmt.Sprintf("%d:%s", currentIteration, truncateString(errMsg, 50))
			if !seenErrors[errKey] && currentIteration > 0 {
				seenErrors[errKey] = true
				summary.Errors = append(summary.Errors, LogError{
					Iteration: currentIteration,
					Message:   truncateString(errMsg, 100),
				})
			}
		}

		// Track key events (git commits, test runs, etc.)
		if entry.Type == "tool_call" && entry.ToolCall != nil {
			for toolName, toolData := range entry.ToolCall {
				if toolName == "shellToolCall" || toolName == "Shell" {
					if args := extractArgs(toolData); args != nil {
						cmd := getStringFromMap(args, "command", "simpleCommand")
						trackShellCommand(cmd, currentIteration, summary)
					}
				}
			}
		}

		// Track key events from Claude Code assistant messages with Bash tool_use
		if entry.Type == "assistant" && entry.Message != nil {
			for _, item := range entry.Message.Content {
				if item.Type == "tool_use" && item.Name == "Bash" {
					cmd := getStringFromMap(item.Input, "command")
					trackShellCommand(cmd, currentIteration, summary)
				}
			}
		}
	}

	// Handle last iteration duration
	if currentIteration > 0 && iterationStartTime > 0 && lastTimestamp > iterationStartTime {
		dur := (lastTimestamp - iterationStartTime) / 1000
		if dur > 0 {
			iterationDurations = append(iterationDurations, dur)
		}
	}

	// Calculate file counts (deduplicated)
	summary.FilesCreated = len(filesCreated)
	summary.FilesModified = len(filesModified)
	summary.FilesDeleted = len(filesDeleted)

	// Calculate iteration statistics
	if len(iterationDurations) > 0 {
		var total int64
		summary.FastestIteration = iterationDurations[0]
		summary.FastestIterationNum = 1
		summary.SlowestIteration = iterationDurations[0]
		summary.SlowestIterationNum = 1

		for i, dur := range iterationDurations {
			total += dur
			if dur < summary.FastestIteration {
				summary.FastestIteration = dur
				summary.FastestIterationNum = i + 1
			}
			if dur > summary.SlowestIteration {
				summary.SlowestIteration = dur
				summary.SlowestIterationNum = i + 1
			}
		}
		summary.AvgIterationSeconds = total / int64(len(iterationDurations))
	}

	// Extract last action from final log lines
	if lastLine != "" {
		// Try to extract meaningful last action
		var entry logEntry
		if err := json.Unmarshal([]byte(lastLine), &entry); err == nil {
			if entry.Type == "assistant" && entry.Message != nil {
				for _, item := range entry.Message.Content {
					if item.Text != "" {
						summary.LastAction = truncateString(item.Text, 80)
						break
					}
				}
			} else if entry.Type == "result" && entry.Result != "" {
				summary.LastAction = truncateString(entry.Result, 80)
			}
		}
		if summary.LastAction == "" {
			summary.LastAction = truncateString(lastLine, 80)
		}
	}

	// Limit errors and events to prevent memory issues
	if len(summary.Errors) > 50 {
		summary.Errors = summary.Errors[:50]
	}
	if len(summary.Events) > 50 {
		summary.Events = summary.Events[:50]
	}

	return summary, nil
}

func trackShellCommand(cmd string, currentIteration int, summary *Summary) {
	if cmd == "" {
		return
	}
	if strings.Contains(cmd, "git commit") {
		summary.Events = append(summary.Events, LogEvent{
			Iteration: currentIteration,
			Type:      "commit",
			Message:   truncateString(cmd, 80),
		})
	} else if strings.Contains(cmd, "npm test") || strings.Contains(cmd, "go test") || strings.Contains(cmd, "pytest") {
		summary.Events = append(summary.Events, LogEvent{
			Iteration: currentIteration,
			Type:      "test",
			Message:   truncateString(cmd, 80),
		})
	}
}

func extractArgs(toolData interface{}) map[string]interface{} {
	if m, ok := toolData.(map[string]interface{}); ok {
		if args, ok := m["args"].(map[string]interface{}); ok {
			return args
		}
		// Some tool calls have args at the top level
		return m
	}
	return nil
}

func getStringFromMap(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

func truncateString(s string, max int) string {
	// Clean up the string first
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")

	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// Formatting helpers

// FormatDuration formats the total duration as a human-readable string.
func (s *Summary) FormatDuration() string {
	return formatDuration(time.Duration(s.DurationSeconds) * time.Second)
}

// FormatAvgIteration formats the average iteration duration.
func (s *Summary) FormatAvgIteration() string {
	return formatDuration(time.Duration(s.AvgIterationSeconds) * time.Second)
}

// FormatFastestIteration formats the fastest iteration duration.
func (s *Summary) FormatFastestIteration() string {
	return formatDuration(time.Duration(s.FastestIteration) * time.Second)
}

// FormatSlowestIteration formats the slowest iteration duration.
func (s *Summary) FormatSlowestIteration() string {
	return formatDuration(time.Duration(s.SlowestIteration) * time.Second)
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	sec := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, sec)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, sec)
	}
	return fmt.Sprintf("%ds", sec)
}
