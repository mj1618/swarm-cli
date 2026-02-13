# Add `--grep` flag to filter log content by pattern

## Completion Notes

**Completed by agent cd59a862 on 2026-01-28**

Implemented the `--grep` flag for filtering log output by regex patterns. Changes made:

1. **cmd/logs.go**: Added new flags and filtering logic:
   - `--grep` flag (StringArray) for regex patterns (case-insensitive by default)
   - `--invert` flag to show non-matching lines
   - `--case-sensitive` flag for case-sensitive matching
   - `-C/--context` flag for context lines around matches
   - `-B/--before` flag for lines before matches
   - `-A/--after` flag for lines after matches

2. **cmd/logs_test.go**: Added comprehensive tests for:
   - `matchesGrep()` function with various pattern combinations
   - `parseTimeFlag()` function
   - `isLineInTimeRange()` function
   - `extractTimestamp()` function

Note: The `-g` shorthand was not used for `--grep` because it conflicts with the global `--global/-g` flag.

All acceptance criteria met:
- Pattern matching case-insensitive by default ✓
- `--case-sensitive` flag works ✓
- `--invert` flag works ✓
- Context flags (`-C`, `-B`, `-A`) work ✓
- Multiple `--grep` flags use OR logic ✓
- Works with `--pretty` (colors preserved) ✓
- Works with `--since`/`--until` ✓
- Works with `--follow` (context disabled with warning) ✓
- Invalid regex produces clear error messages ✓
- Context lines separated with `--` when not adjacent ✓

---

## Problem

When viewing agent logs, users often want to find specific content:

1. **Error investigation**: Find all error messages in a long log
2. **Tool usage**: See which files the agent modified
3. **API calls**: Track specific function or tool invocations
4. **Progress tracking**: Find iteration markers or status messages

Currently, users must pipe through external `grep`, which:
- Loses the pretty-printing colors and formatting (`--pretty`)
- Requires reading/streaming the entire log first
- Makes it awkward to combine with other flags like `--follow`

```bash
# Current awkward workflow:
swarm logs my-agent --tail 1000 | grep -i error           # Loses colors
swarm logs my-agent -P | grep error                       # Still loses colors
swarm logs my-agent --since 1h | grep "tool_use" | less   # Loses context
```

## Solution

Add a `--grep` flag that filters log lines by a regex pattern, applied before pretty-printing so colors are preserved.

### Proposed API

```bash
# Find errors in logs (case-insensitive by default)
swarm logs my-agent --grep error

# Case-sensitive search
swarm logs my-agent --grep Error --case-sensitive

# Regex pattern
swarm logs my-agent --grep "tool_use.*Read"

# Combine with other flags
swarm logs my-agent --grep error --pretty           # Pretty-print matching lines
swarm logs my-agent --grep error --since 30m        # Errors in last 30 minutes
swarm logs my-agent --grep "iteration" -f           # Follow and filter

# Show context around matches (like grep -C)
swarm logs my-agent --grep error -C 3               # 3 lines before and after
swarm logs my-agent --grep error -B 2 -A 5          # 2 before, 5 after

# Invert match (show non-matching lines)
swarm logs my-agent --grep "^\[swarm\]" --invert    # Hide swarm status lines

# Multiple patterns (OR logic)
swarm logs my-agent --grep error --grep warning     # Match either
```

### Output examples

**Basic grep:**
```
$ swarm logs my-agent --grep error --tail 100
2024-01-28 10:15:32 | Error: file not found: config.yaml
2024-01-28 10:18:45 | [swarm] Agent error (continuing): exit status 1
2024-01-28 10:22:10 | TypeError: Cannot read property 'map' of undefined
```

**With context:**
```
$ swarm logs my-agent --grep "tool_use" -C 1 --tail 50
2024-01-28 10:15:30 | Processing file changes...
2024-01-28 10:15:31 | tool_use: Read { path: "src/index.ts" }
2024-01-28 10:15:32 | File contents loaded successfully
--
2024-01-28 10:15:45 | Applying fix...
2024-01-28 10:15:46 | tool_use: StrReplace { path: "src/index.ts" }
2024-01-28 10:15:47 | Changes applied
```

**With pretty-print preserved:**
```
$ swarm logs my-agent --grep "iteration" -P
[swarm] === Iteration 1/10 ===
[swarm] === Iteration 2/10 ===
[swarm] === Iteration 3/10 ===
```

## Files to create/change

- `cmd/logs.go` - Add `--grep` flag and filtering logic

## Implementation details

### cmd/logs.go changes

```go
var (
    logsFollow        bool
    logsLines         int
    logsPretty        bool
    logsSince         string
    logsUntil         string
    logsGrep          []string  // NEW: support multiple patterns
    logsGrepInvert    bool      // NEW
    logsGrepCase      bool      // NEW: case-sensitive (default false)
    logsContext       int       // NEW: -C context lines
    logsContextBefore int       // NEW: -B lines before
    logsContextAfter  int       // NEW: -A lines after
)

// In RunE:
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

// Pass to showLogLines and followFile
return showLogLines(agent.LogFile, logsLines, nil, sinceTime, untilTime, grepPatterns, logsGrepInvert, contextBefore, contextAfter)

// In init():
logsCmd.Flags().StringArrayVarP(&logsGrep, "grep", "g", nil, "Filter lines matching pattern (regex, case-insensitive by default)")
logsCmd.Flags().BoolVar(&logsGrepInvert, "invert", false, "Invert match (show non-matching lines)")
logsCmd.Flags().BoolVar(&logsGrepCase, "case-sensitive", false, "Make grep pattern case-sensitive")
logsCmd.Flags().IntVarP(&logsContext, "context", "C", 0, "Show N lines of context around matches")
logsCmd.Flags().IntVarP(&logsContextBefore, "before", "B", 0, "Show N lines before each match")
logsCmd.Flags().IntVarP(&logsContextAfter, "after", "A", 0, "Show N lines after each match")
```

### Filtering logic

```go
// matchesGrep returns true if the line matches any of the grep patterns.
// If invert is true, returns true if the line matches NONE of the patterns.
func matchesGrep(line string, patterns []*regexp.Regexp, invert bool) bool {
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

// showLogLines with grep support
func showLogLines(filepath string, n int, parser *logparser.Parser, since, until time.Time, grepPatterns []*regexp.Regexp, invert bool, beforeCtx, afterCtx int) error {
    // ... existing file open logic ...
    
    // For context support, we need a sliding window
    type lineWithMatch struct {
        text    string
        matches bool
    }
    
    var allLines []lineWithMatch
    for scanner.Scan() {
        line := scanner.Text()
        
        // Apply time filter first
        if hasTimeFilter && !isLineInTimeRange(line, since, until) {
            continue
        }
        
        matches := matchesGrep(line, grepPatterns, invert)
        allLines = append(allLines, lineWithMatch{text: line, matches: matches})
    }
    
    // If no grep pattern or no context, simple filter
    if len(grepPatterns) == 0 || (beforeCtx == 0 && afterCtx == 0) {
        var filtered []string
        for _, l := range allLines {
            if l.matches {
                filtered = append(filtered, l.text)
            }
        }
        // Keep last n lines
        if len(filtered) > n {
            filtered = filtered[len(filtered)-n:]
        }
        // Print...
        return nil
    }
    
    // Context-aware filtering
    // Mark lines to include based on proximity to matches
    include := make([]bool, len(allLines))
    for i, l := range allLines {
        if l.matches {
            // Include this line and context
            start := max(0, i-beforeCtx)
            end := min(len(allLines), i+afterCtx+1)
            for j := start; j < end; j++ {
                include[j] = true
            }
        }
    }
    
    // Collect included lines
    var filtered []string
    for i, l := range allLines {
        if include[i] {
            filtered = append(filtered, l.text)
        }
    }
    
    // Keep last n
    if len(filtered) > n {
        filtered = filtered[len(filtered)-n:]
    }
    
    // Print...
}
```

### Follow mode with grep

```go
func followFile(filepath string, since, until time.Time, grepPatterns []*regexp.Regexp, invert bool) error {
    // ... existing setup ...
    
    // In the read loop:
    line, err := reader.ReadString('\n')
    if err != nil {
        // ... existing error handling ...
    }
    
    // Apply time filter
    if !since.IsZero() && !isLineInTimeRange(line, since, time.Time{}) {
        continue
    }
    
    // Apply grep filter (no context in follow mode - would require buffering)
    if !matchesGrep(line, grepPatterns, invert) {
        continue
    }
    
    // Print matching line...
}
```

## Edge cases

1. **Invalid regex**: Return clear error message with the invalid pattern
2. **Empty pattern**: Treat as "match all" (no filter)
3. **Multiple `--grep` flags**: OR logic (match any pattern)
4. **Grep with `--pretty`**: Apply grep before pretty-printing so colors are preserved
5. **Grep with context in follow mode**: Disable context (would require complex buffering), show warning
6. **Grep with `--tail 0`**: Show all matching lines from entire log
7. **Context overlaps**: When two matches are close, merge their context windows
8. **Binary content in logs**: Regexp handles this gracefully (no special handling needed)

## Testing scenarios

```bash
# Setup: Have an agent with logs containing various content

# Basic grep
swarm logs my-agent --grep error --tail 100        # Find "error" (case-insensitive)
swarm logs my-agent --grep Error --case-sensitive  # Case-sensitive

# Regex patterns
swarm logs my-agent --grep "iteration \d+/\d+"     # Match iteration progress
swarm logs my-agent --grep "^2024-01-28 10:"       # Lines from specific hour

# Context
swarm logs my-agent --grep error -C 2              # 2 lines context each side
swarm logs my-agent --grep error -B 5              # 5 lines before only

# Combined filters
swarm logs my-agent --grep error --since 1h        # Errors in last hour
swarm logs my-agent --grep tool_use --pretty       # Pretty-printed tool usage

# Follow mode
swarm logs my-agent --grep error -f                # Follow and filter
swarm logs my-agent --grep "iteration" -f -P       # Follow iteration progress

# Invert
swarm logs my-agent --grep "^\[swarm\]" --invert   # Hide swarm messages

# Multiple patterns
swarm logs my-agent --grep error --grep warning    # Match either
```

## Acceptance criteria

- `swarm logs my-agent --grep pattern` filters output to matching lines
- Pattern matching is case-insensitive by default
- `--case-sensitive` flag enables case-sensitive matching
- `--invert` flag shows non-matching lines
- `-C N`, `-B N`, `-A N` flags add context lines around matches
- Multiple `--grep` flags use OR logic
- Works with `--pretty` flag (colors preserved)
- Works with `--since` and `--until` (applied in sequence: time filter, then grep)
- Works with `--follow` mode (no context support in follow)
- Invalid regex patterns produce clear error messages
- Context lines are separated with `--` when not adjacent
