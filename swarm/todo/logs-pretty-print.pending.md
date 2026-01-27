# Add pretty-print formatting to logs command

## Problem

The `cmd/logs.go` command displays raw log output directly from the log file. However, there's already a fully-featured `internal/logparser` package that can parse JSONL agent logs and pretty-print them with:

- Colored headers for event types
- Merged consecutive message fragments (assistant, user, thinking)
- Summarized tool calls (Shell, Read file, List dir, etc.)
- Formatted system init events
- Graceful fallback to raw output on parse errors

Currently, when a user runs `swarm logs myagent`, they see raw JSONL lines like:

```json
{"type":"system","subtype":"init","model":"claude-opus-4-20250514","cwd":"/path","session_id":"abc"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hello"}]}}
```

Instead of the formatted output the logparser provides:

```
[system / init]
System init (model=claude-opus-4-20250514, cwd=/path, session=abc)

[assistant]
Hello
```

## Solution

Add a `--pretty` / `-P` flag to the `logs` command that pipes log lines through the `logparser.Parser` before displaying them. This should:

1. Add `--pretty` flag (defaults to `false` to preserve backward compatibility)
2. In `showLogLines()`, when pretty mode is enabled, process each line through `logparser.NewParser(os.Stdout).ProcessLine(line)` and call `Flush()` at the end
3. In `followFile()`, when pretty mode is enabled, process each new line through the parser
4. Maintain a single `logparser.Parser` instance for the duration of the command to properly merge consecutive fragments

## Files to change

- `cmd/logs.go` -- add `--pretty` flag and integrate `logparser.Parser`

## Implementation details

```go
var logsPretty bool

// In init():
logsCmd.Flags().BoolVarP(&logsPretty, "pretty", "P", false, "Pretty-print log output")

// In showLogLines(), wrap the print loop:
if logsPretty {
    parser := logparser.NewParser(os.Stdout)
    for _, line := range lines {
        parser.ProcessLine(line)
    }
    parser.Flush()
} else {
    for _, line := range lines {
        fmt.Println(line)
    }
}

// In followFile(), similar pattern for real-time processing
```

## Acceptance criteria

- `swarm logs abc123` continues to show raw log lines (default behavior unchanged)
- `swarm logs abc123 --pretty` shows formatted, colorized output
- `swarm logs abc123 -Pf` (follow with pretty) works correctly, merging streaming fragments
- Pretty mode gracefully falls back to raw lines if JSON parsing fails (per logparser design)
- Parser is flushed properly when command exits or follow mode is interrupted
