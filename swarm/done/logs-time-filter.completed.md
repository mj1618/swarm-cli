# Add `--since` and `--until` flags to logs command

## Completion Note (Agent 1a025fb7)

Implemented `--since` and `--until` flags for the `swarm logs` command. Changes made to `cmd/logs.go`:

- Added `logsSince` and `logsUntil` flag variables
- Added `parseTimeFlag()` function supporting relative durations (30s, 5m, 2h, 1d) and absolute timestamps (RFC3339, date-time, date-only)
- Added `parseDurationWithDays()` to handle day units not supported by Go's `time.ParseDuration`
- Added `extractTimestamp()` to parse timestamps from log lines (format: "2024-01-28 10:15:32")
- Added `isLineInTimeRange()` to check if lines fall within the time window
- Updated `showLogLines()` to filter lines by time before applying tail limit
- Updated `followFile()` to apply `--since` filter in follow mode (with warning that `--until` is ignored)
- Added validation for "since must be before until"
- Shows "(no matching log lines in the specified time range)" when no lines match

All acceptance criteria met. Tests pass.

## Problem

When debugging long-running agents, users often need to view logs from a specific time period. Currently, `swarm logs` only supports:
- `--tail N` to show the last N lines
- `--follow` to stream new output

There's no way to filter logs by timestamp, which means users must:
1. Scroll through potentially thousands of lines to find relevant output
2. Export logs and grep with timestamps manually
3. Guess how many `--tail` lines they need

This is particularly painful for:
- Agents that have been running for hours/days
- Debugging issues that occurred at a specific time
- Correlating agent activity with external events

**Current workaround:**
```bash
# Have to guess line count or pipe through grep
swarm logs my-agent --tail 10000 | grep "2024-01-28 10:"
```

## Solution

Add `--since` and `--until` flags to `swarm logs`, following the Docker logs pattern. These flags filter log output to a specific time window.

### Proposed API

```bash
# Show logs from the last 30 minutes
swarm logs abc123 --since 30m

# Show logs from a specific time
swarm logs abc123 --since "2024-01-28 10:00:00"

# Show logs until 1 hour ago (useful for excluding recent noise)
swarm logs abc123 --until 1h

# Combine for a specific time window
swarm logs abc123 --since 2h --until 30m

# Works with follow mode (only shows new lines after --since threshold)
swarm logs abc123 --since 5m -f

# Works with pretty mode
swarm logs abc123 --since 1h --pretty
```

### Supported time formats

1. **Relative duration**: `30s`, `5m`, `2h`, `1d` (seconds, minutes, hours, days)
2. **RFC3339**: `2024-01-28T10:00:00Z`
3. **Date-time**: `2024-01-28 10:00:00` or `2024-01-28 10:00`
4. **Date only**: `2024-01-28` (interpreted as start of day)

## Files to change

- `cmd/logs.go` - add `--since` and `--until` flags and filtering logic

## Implementation details

### Timestamp parsing in agent output

Agent logs include timestamps in the format:
```
2024-01-28 10:15:32 | [agent] Starting iteration...
```

The implementation needs to:
1. Parse the timestamp prefix from each line
2. Compare against the since/until bounds
3. Include/exclude lines accordingly

### cmd/logs.go changes

```go
var (
    logsFollow bool
    logsLines  int
    logsPretty bool
    logsSince  string  // NEW
    logsUntil  string  // NEW
)

// Add to init()
logsCmd.Flags().StringVar(&logsSince, "since", "", "Show logs since timestamp (e.g., 30m, 2h, 2024-01-28 10:00)")
logsCmd.Flags().StringVar(&logsUntil, "until", "", "Show logs until timestamp (e.g., 1h, 2024-01-28 12:00)")

// parseTimeFlag parses a time flag value into a time.Time
func parseTimeFlag(value string) (time.Time, error) {
    if value == "" {
        return time.Time{}, nil
    }

    // Try relative duration first (e.g., "30m", "2h", "1d")
    if dur, err := parseDuration(value); err == nil {
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

    return time.Time{}, fmt.Errorf("invalid time format: %s", value)
}

// parseDuration handles durations with day support (e.g., "1d")
func parseDuration(s string) (time.Duration, error) {
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

// extractTimestamp extracts timestamp from a log line
// Returns zero time if no timestamp found
func extractTimestamp(line string) time.Time {
    // Agent logs typically start with: "2024-01-28 10:15:32 | ..."
    // Try to extract the timestamp prefix
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

// isLineInTimeRange checks if a log line falls within the since/until range
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
```

### Modified showLogLines function

```go
func showLogLines(filepath string, n int, parser *logparser.Parser, since, until time.Time) error {
    file, err := os.Open(filepath)
    if err != nil {
        return fmt.Errorf("failed to open log file: %w", err)
    }
    defer file.Close()

    // ... existing file size check ...

    // Read and filter lines
    var filteredLines []string
    scanner := bufio.NewScanner(file)
    buf := make([]byte, 0, 64*1024)
    scanner.Buffer(buf, 1024*1024)

    for scanner.Scan() {
        line := scanner.Text()
        if isLineInTimeRange(line, since, until) {
            filteredLines = append(filteredLines, line)
            // Keep only last n lines if filtering
            if n > 0 && len(filteredLines) > n {
                filteredLines = filteredLines[1:]
            }
        }
    }

    if err := scanner.Err(); err != nil {
        return fmt.Errorf("error reading log file: %w", err)
    }

    if len(filteredLines) == 0 {
        fmt.Println("(no matching log lines in the specified time range)")
        return nil
    }

    // Print the lines (existing pretty/plain logic)
    // ...
}
```

### Output examples

Normal usage:
```
$ swarm logs my-agent --since 30m
2024-01-28 10:15:32 | [swarm] === Iteration 5/10 ===
2024-01-28 10:15:33 | [agent] Processing request...
2024-01-28 10:20:45 | [agent] Task completed
2024-01-28 10:20:46 | [swarm] === Iteration 6/10 ===
...
```

No matching lines:
```
$ swarm logs my-agent --since 1h --until 30m
(no matching log lines in the specified time range)
```

Combined with tail:
```
$ swarm logs my-agent --since 2h --tail 20
# Shows last 20 lines from the last 2 hours
```

## Edge cases

1. **No timestamp in log lines**: Lines without recognizable timestamps are included by default (they're likely continuations of previous timestamped output).

2. **Invalid time format**: Return clear error message with examples of valid formats.

3. **Since after until**: Return error "since time must be before until time".

4. **No matching lines**: Print "(no matching log lines in the specified time range)" instead of error.

5. **Future timestamps**: Allow them (user might have timezone issues, or system clock drift).

6. **Combined with --tail**: Apply time filter first, then take last N lines from filtered results.

7. **Combined with --follow**: Only show lines with timestamps after the --since threshold. The --until flag doesn't make sense with --follow (warn and ignore it).

8. **Empty log file**: Existing behavior - show "(log file is empty)".

9. **Agent with no log file (foreground run)**: Existing error - "agent was not started in detached mode".

## Acceptance criteria

- `swarm logs abc123 --since 30m` shows logs from the last 30 minutes
- `swarm logs abc123 --until 1h` shows logs until 1 hour ago
- `swarm logs abc123 --since 2h --until 30m` shows logs in the 2h-to-30m-ago window
- Relative durations work: `30s`, `5m`, `2h`, `1d`
- Absolute timestamps work: `2024-01-28`, `2024-01-28 10:00`, RFC3339
- `--since` works with `--follow` (only new lines after threshold)
- `--until` with `--follow` shows warning and is ignored
- `--since`/`--until` work with `--pretty` mode
- `--since`/`--until` work with `--tail` (filter first, then limit)
- Clear error messages for invalid time formats
- Graceful handling when no lines match the time range
- Lines without timestamps are included (not filtered out)
