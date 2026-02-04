package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mj1618/swarm-cli/internal/logparser"
	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	logsFollow        bool
	logsLines         int
	logsPretty        bool
	logsSince         string
	logsUntil         string
	logsGrep          []string // grep patterns (regex)
	logsGrepInvert    bool     // invert match (show non-matching lines)
	logsGrepCase      bool     // case-sensitive matching
	logsContext       int      // context lines (-C)
	logsContextBefore int      // lines before match (-B)
	logsContextAfter  int      // lines after match (-A)
)

var logsCmd = &cobra.Command{
	Use:     "logs [task-id-or-name]",
	Aliases: []string{"tail"},
	Short:   "View the output of a running or completed agent",
	Long: `View the log output of a detached agent.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent

Use -f to follow the output in real-time, or --tail to specify the number
of lines to show.

Use --since and --until to filter logs by timestamp. Supported formats:
- Relative duration: 30s, 5m, 2h, 1d
- RFC3339: 2024-01-28T10:00:00Z
- Date-time: 2024-01-28 10:00:00 or 2024-01-28 10:00
- Date only: 2024-01-28 (interpreted as start of day)

Use --grep to filter log lines by pattern (regex). The pattern is case-insensitive
by default. Use --case-sensitive for case-sensitive matching. Multiple --grep
flags can be specified to match any of the patterns (OR logic).`,
	Example: `  # Show last 50 lines of agent abc123
  swarm logs abc123

  # Follow output of agent named "myagent"
  swarm logs myagent -f

  # Follow logs of the most recent agent
  swarm logs @last -f
  swarm logs _ -f

  # Show last 100 lines
  swarm logs abc123 --tail 100

  # Show logs from the last 30 minutes
  swarm logs abc123 --since 30m

  # Show logs from a specific time window
  swarm logs abc123 --since 2h --until 30m

  # Show logs since a specific date
  swarm logs abc123 --since "2024-01-28 10:00:00"

  # Filter logs by pattern (case-insensitive)
  swarm logs abc123 --grep error

  # Case-sensitive grep
  swarm logs abc123 --grep Error --case-sensitive

  # Regex pattern
  swarm logs abc123 --grep "tool_use.*Read"

  # Show context around matches
  swarm logs abc123 --grep error -C 3
  swarm logs abc123 --grep error -B 2 -A 5

  # Invert match (show non-matching lines)
  swarm logs abc123 --grep "^\[swarm\]" --invert

  # Multiple patterns (OR logic)
  swarm logs abc123 --grep error --grep warning

  # Combine with other flags
  swarm logs abc123 --grep error --since 30m --pretty`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentIdentifier := args[0]

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		agent, err := ResolveAgentIdentifier(mgr, agentIdentifier)
		if err != nil {
			return fmt.Errorf("agent not found: %w", err)
		}

		if agent.LogFile == "" {
			return fmt.Errorf("agent %s was not started in detached mode (no log file)", agentIdentifier)
		}

		// Check if log file exists
		if _, err := os.Stat(agent.LogFile); os.IsNotExist(err) {
			return fmt.Errorf("log file not found: %s", agent.LogFile)
		}

		// Parse time flags
		var sinceTime, untilTime time.Time
		if logsSince != "" {
			sinceTime, err = ParseTimeFlag(logsSince)
			if err != nil {
				return fmt.Errorf("invalid --since format: %w", err)
			}
		}
		if logsUntil != "" {
			untilTime, err = ParseTimeFlag(logsUntil)
			if err != nil {
				return fmt.Errorf("invalid --until format: %w", err)
			}
		}

		// Validate since is before until
		if !sinceTime.IsZero() && !untilTime.IsZero() && sinceTime.After(untilTime) {
			return fmt.Errorf("--since time must be before --until time")
		}

		// Compile grep patterns
		var grepPatterns []*regexp.Regexp
		for _, pattern := range logsGrep {
			flags := ""
			if !logsGrepCase {
				flags = "(?i)"
			}
			re, err := regexp.Compile(flags + pattern)
			if err != nil {
				return fmt.Errorf("invalid grep pattern %q: %w", pattern, err)
			}
			grepPatterns = append(grepPatterns, re)
		}

		// Calculate context lines (explicit -B/-A override -C)
		contextBefore := logsContext
		contextAfter := logsContext
		if logsContextBefore > 0 {
			contextBefore = logsContextBefore
		}
		if logsContextAfter > 0 {
			contextAfter = logsContextAfter
		}

		if logsFollow {
			// Warn if --until is used with --follow
			if logsUntil != "" {
				fmt.Println("Warning: --until is ignored when using --follow")
				untilTime = time.Time{}
			}
			// Warn if context is used with --follow
			if contextBefore > 0 || contextAfter > 0 {
				fmt.Println("Warning: context flags (-C/-B/-A) are ignored when using --follow")
				contextBefore = 0
				contextAfter = 0
			}
			return followFile(agent.LogFile, sinceTime, untilTime, grepPatterns, logsGrepInvert)
		}

		return showLogLines(agent.LogFile, logsLines, nil, sinceTime, untilTime, grepPatterns, logsGrepInvert, contextBefore, contextAfter)
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow the output in real-time")
	logsCmd.Flags().IntVar(&logsLines, "tail", 50, "Number of lines to show from the end of the logs")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "Number of lines to show (alias for --tail)")
	logsCmd.Flags().MarkHidden("lines") // Keep -n working but prefer --tail in docs
	logsCmd.Flags().BoolVarP(&logsPretty, "pretty", "P", false, "Pretty-print log output with colors and formatting")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "Show logs since timestamp (e.g., 30m, 2h, 2024-01-28 10:00)")
	logsCmd.Flags().StringVar(&logsUntil, "until", "", "Show logs until timestamp (e.g., 1h, 2024-01-28 12:00)")
	logsCmd.Flags().StringArrayVar(&logsGrep, "grep", nil, "Filter lines matching pattern (regex, case-insensitive by default)")
	logsCmd.Flags().BoolVar(&logsGrepInvert, "invert", false, "Invert match (show non-matching lines)")
	logsCmd.Flags().BoolVar(&logsGrepCase, "case-sensitive", false, "Make grep pattern case-sensitive")
	logsCmd.Flags().IntVarP(&logsContext, "context", "C", 0, "Show N lines of context around matches")
	logsCmd.Flags().IntVarP(&logsContextBefore, "before", "B", 0, "Show N lines before each match")
	logsCmd.Flags().IntVarP(&logsContextAfter, "after", "A", 0, "Show N lines after each match")
	rootCmd.AddCommand(logsCmd)

	// Add dynamic completion for agent identifier
	logsCmd.ValidArgsFunction = completeAgentIdentifier
}

// ParseTimeFlag parses a time flag value into a time.Time.
// It supports relative durations (e.g., "30m", "2h", "1d") and absolute timestamps.
func ParseTimeFlag(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	// Try relative duration first (e.g., "30m", "2h", "1d")
	if dur, err := ParseDurationWithDays(value); err == nil {
		return time.Now().Add(-dur), nil
	}

	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}

	// Try common formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, format := range formats {
		if t, err := time.ParseInLocation(format, value, time.Local); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized time format: %s (use 30m, 2h, 1d, or 2024-01-28 10:00:00)", value)
}

// ParseDurationWithDays handles durations with day support (e.g., "1d").
// Standard time.ParseDuration doesn't support 'd' for days.
func ParseDurationWithDays(s string) (time.Duration, error) {
	// Handle days specially since time.ParseDuration doesn't support 'd'
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

// ExtractTimestamp extracts timestamp from a log line.
// Returns zero time if no timestamp found.
// Agent logs typically start with: "2024-01-28 10:15:32 | ..."
func ExtractTimestamp(line string) time.Time {
	if len(line) < 19 {
		return time.Time{}
	}

	// Try parsing first 19 chars as timestamp
	t, err := time.ParseInLocation("2006-01-02 15:04:05", line[:19], time.Local)
	if err == nil {
		return t
	}

	return time.Time{}
}

// IsLineInTimeRange checks if a log line falls within the since/until range.
// Lines without timestamps are included by default (they're likely continuations).
func IsLineInTimeRange(line string, since, until time.Time) bool {
	ts := ExtractTimestamp(line)
	if ts.IsZero() {
		// Lines without timestamps are included if we're in an active range
		// (This handles continuation lines and non-timestamped output)
		return true
	}

	if !since.IsZero() && ts.Before(since) {
		return false
	}
	if !until.IsZero() && ts.After(until) {
		return false
	}
	return true
}

// MatchesGrep returns true if the line matches any of the grep patterns.
// If invert is true, returns true if the line matches NONE of the patterns.
// If patterns is empty, returns true (no filter).
func MatchesGrep(line string, patterns []*regexp.Regexp, invert bool) bool {
	if len(patterns) == 0 {
		return true // No filter, include all
	}

	for _, re := range patterns {
		if re.MatchString(line) {
			return !invert
		}
	}
	return invert
}

// showLogLines shows the last n lines of a file.
// If parser is provided, lines are processed through it for pretty-printing.
// If parser is nil and logsPretty is true, a new parser is created and flushed.
// If since/until are non-zero, only lines within the time range are shown.
// If grepPatterns is non-empty, only lines matching the patterns are shown.
// If invert is true, shows lines NOT matching the patterns.
// contextBefore/contextAfter add context lines around matches (like grep -B/-A).
func showLogLines(filepath string, n int, parser *logparser.Parser, since, until time.Time, grepPatterns []*regexp.Regexp, invert bool, contextBefore, contextAfter int) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		fmt.Println("(log file is empty)")
		return nil
	}

	hasTimeFilter := !since.IsZero() || !until.IsZero()
	hasGrepFilter := len(grepPatterns) > 0
	hasContext := contextBefore > 0 || contextAfter > 0

	// Read the file and collect lines
	scanner := bufio.NewScanner(file)

	// Use a larger buffer for potentially long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// For grep with context, we need to track all lines and their match status
	type lineWithMatch struct {
		text    string
		matches bool
	}
	var allLines []lineWithMatch

	for scanner.Scan() {
		line := scanner.Text()

		// Apply time filter if specified
		if hasTimeFilter && !IsLineInTimeRange(line, since, until) {
			continue
		}

		if hasGrepFilter {
			matches := MatchesGrep(line, grepPatterns, invert)
			allLines = append(allLines, lineWithMatch{text: line, matches: matches})
		} else {
			allLines = append(allLines, lineWithMatch{text: line, matches: true})
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log file: %w", err)
	}

	// Apply grep filter with optional context
	var filtered []string
	if hasGrepFilter && hasContext {
		// Mark lines to include based on proximity to matches
		include := make([]bool, len(allLines))
		for i, l := range allLines {
			if l.matches {
				// Include this line and context
				start := i - contextBefore
				if start < 0 {
					start = 0
				}
				end := i + contextAfter + 1
				if end > len(allLines) {
					end = len(allLines)
				}
				for j := start; j < end; j++ {
					include[j] = true
				}
			}
		}

		// Collect included lines, adding separators between non-adjacent groups
		lastIncluded := -2 // Track last included index for separator logic
		for i, l := range allLines {
			if include[i] {
				// Add separator if there's a gap (non-adjacent)
				if lastIncluded >= 0 && i > lastIncluded+1 {
					filtered = append(filtered, "--")
				}
				filtered = append(filtered, l.text)
				lastIncluded = i
			}
		}
	} else {
		// Simple filter without context
		for _, l := range allLines {
			if l.matches {
				filtered = append(filtered, l.text)
			}
		}
	}

	// Keep last n lines
	if len(filtered) > n {
		filtered = filtered[len(filtered)-n:]
	}

	if len(filtered) == 0 {
		if hasTimeFilter || hasGrepFilter {
			fmt.Println("(no matching log lines)")
		}
		return nil
	}

	// Print the lines
	if logsPretty {
		ownParser := parser == nil
		if ownParser {
			parser = logparser.NewParser(os.Stdout)
		}
		for _, line := range filtered {
			// Don't pretty-print the separator
			if line == "--" {
				fmt.Println("--")
			} else {
				parser.ProcessLine(line)
			}
		}
		if ownParser {
			parser.Flush()
		}
	} else {
		for _, line := range filtered {
			fmt.Println(line)
		}
	}

	return nil
}

// followFile follows a file in real-time.
// If since is non-zero, only shows lines with timestamps after that time.
// The until parameter is ignored in follow mode (warning already shown to user).
// If grepPatterns is non-empty, only lines matching the patterns are shown.
// Context flags are not supported in follow mode (warning already shown to user).
func followFile(filepath string, since, until time.Time, grepPatterns []*regexp.Regexp, invert bool) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Create parser if pretty mode is enabled - used for both initial lines and follow
	var parser *logparser.Parser
	if logsPretty {
		parser = logparser.NewParser(os.Stdout)
	}

	// First, show last few lines for context (with time and grep filter applied, no context lines in follow mode)
	if err := showLogLines(filepath, logsLines, parser, since, until, grepPatterns, invert, 0, 0); err != nil {
		return err
	}

	// Seek to end of file
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}

	fmt.Println("\n--- Following log (Ctrl+C to stop) ---")

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// No new data, wait a bit
				time.Sleep(100 * time.Millisecond)
				continue
			}
			// Flush parser before returning error
			if parser != nil {
				parser.Flush()
			}
			return fmt.Errorf("error reading log file: %w", err)
		}

		// Apply time filter for follow mode (only --since matters, --until is ignored)
		if !since.IsZero() && !IsLineInTimeRange(line, since, time.Time{}) {
			continue
		}

		// Apply grep filter
		if !MatchesGrep(line, grepPatterns, invert) {
			continue
		}

		if logsPretty && parser != nil {
			// Process through parser (strips the trailing newline itself)
			parser.ProcessLine(line)
		} else {
			// Print without extra newline since ReadString includes the \n
			fmt.Print(line)
		}
	}
}
