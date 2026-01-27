package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/matt/swarm-cli/internal/logparser"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	logsFollow bool
	logsLines  int
	logsPretty bool
	logsSince  string
	logsUntil  string
)

var logsCmd = &cobra.Command{
	Use:     "logs [agent-id-or-name]",
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
- Date only: 2024-01-28 (interpreted as start of day)`,
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
  swarm logs abc123 --since "2024-01-28 10:00:00"`,
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
			sinceTime, err = parseTimeFlag(logsSince)
			if err != nil {
				return fmt.Errorf("invalid --since format: %w", err)
			}
		}
		if logsUntil != "" {
			untilTime, err = parseTimeFlag(logsUntil)
			if err != nil {
				return fmt.Errorf("invalid --until format: %w", err)
			}
		}

		// Validate since is before until
		if !sinceTime.IsZero() && !untilTime.IsZero() && sinceTime.After(untilTime) {
			return fmt.Errorf("--since time must be before --until time")
		}

		if logsFollow {
			// Warn if --until is used with --follow
			if logsUntil != "" {
				fmt.Println("Warning: --until is ignored when using --follow")
				untilTime = time.Time{}
			}
			return followFile(agent.LogFile, sinceTime, untilTime)
		}

		return showLogLines(agent.LogFile, logsLines, nil, sinceTime, untilTime)
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
	rootCmd.AddCommand(logsCmd)
}

// parseTimeFlag parses a time flag value into a time.Time.
// It supports relative durations (e.g., "30m", "2h", "1d") and absolute timestamps.
func parseTimeFlag(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	// Try relative duration first (e.g., "30m", "2h", "1d")
	if dur, err := parseDurationWithDays(value); err == nil {
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

// parseDurationWithDays handles durations with day support (e.g., "1d").
// Standard time.ParseDuration doesn't support 'd' for days.
func parseDurationWithDays(s string) (time.Duration, error) {
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

// extractTimestamp extracts timestamp from a log line.
// Returns zero time if no timestamp found.
// Agent logs typically start with: "2024-01-28 10:15:32 | ..."
func extractTimestamp(line string) time.Time {
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

// isLineInTimeRange checks if a log line falls within the since/until range.
// Lines without timestamps are included by default (they're likely continuations).
func isLineInTimeRange(line string, since, until time.Time) bool {
	ts := extractTimestamp(line)
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

// showLogLines shows the last n lines of a file.
// If parser is provided, lines are processed through it for pretty-printing.
// If parser is nil and logsPretty is true, a new parser is created and flushed.
// If since/until are non-zero, only lines within the time range are shown.
func showLogLines(filepath string, n int, parser *logparser.Parser, since, until time.Time) error {
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

	// Read the file and collect lines
	lines := make([]string, 0, n)
	scanner := bufio.NewScanner(file)

	// Use a larger buffer for potentially long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// Apply time filter if specified
		if hasTimeFilter && !isLineInTimeRange(line, since, until) {
			continue
		}

		lines = append(lines, line)
		if len(lines) > n {
			lines = lines[1:] // Keep only last n lines
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log file: %w", err)
	}

	if len(lines) == 0 && hasTimeFilter {
		fmt.Println("(no matching log lines in the specified time range)")
		return nil
	}

	// Print the lines
	if logsPretty {
		ownParser := parser == nil
		if ownParser {
			parser = logparser.NewParser(os.Stdout)
		}
		for _, line := range lines {
			parser.ProcessLine(line)
		}
		if ownParser {
			parser.Flush()
		}
	} else {
		for _, line := range lines {
			fmt.Println(line)
		}
	}

	return nil
}

// followFile follows a file in real-time.
// If since is non-zero, only shows lines with timestamps after that time.
// The until parameter is ignored in follow mode (warning already shown to user).
func followFile(filepath string, since, until time.Time) error {
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

	// First, show last few lines for context (with time filter applied)
	if err := showLogLines(filepath, logsLines, parser, since, until); err != nil {
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
		if !since.IsZero() && !isLineInTimeRange(line, since, time.Time{}) {
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
