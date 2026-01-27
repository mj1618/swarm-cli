# Add `swarm models` command for listing available models

## Problem

Currently, users have no way to discover available models from within the swarm CLI. They must:

1. Know which backend they're using
2. Leave swarm and run backend-specific commands (`agent --list-models` for Cursor)
3. Or just guess/remember model names (`opus`, `sonnet` for Claude Code)

This creates friction when:
- Setting up swarm for the first time
- Switching between backends
- Trying to use a specific model variant

**Current workflow:**
```bash
# What models can I use?
swarm config show           # Shows "model = opus" but not what's available
agent --list-models         # Have to know the backend command
claude --help               # Doesn't list models clearly

# Then hope you typed the model name correctly
swarm run -p task -m claude-sonnet-4  # Is this right? No validation until it fails
```

**Desired workflow:**
```bash
swarm models                # Shows all available models with current default highlighted
swarm run -p task -m sonnet # Confident in model name
```

## Solution

Add a `swarm models` command that lists available models for the current backend, with the default model highlighted.

### Proposed API

```bash
# List available models (uses current backend from config)
swarm models

# Output as JSON (for scripting)
swarm models --format json

# Show only the default model
swarm models --default
```

### Example output

For Cursor backend:
```
$ swarm models
Available models (cursor backend):

  claude-opus-4-20250514
  claude-sonnet-4-20250514
  gpt-5.2
  gpt-5.2-codex
  gemini-3-pro
  gemini-3-flash
* opus-4.5-thinking (default)
  sonnet-4.5-thinking
  opus-4.5
  sonnet-4.5
  grok

Use 'swarm run -m <model>' to use a specific model.
```

For Claude Code backend:
```
$ swarm models
Available models (claude-code backend):

* opus (default)
  sonnet

Use 'swarm run -m <model>' to use a specific model.
```

JSON format:
```json
{
  "backend": "cursor",
  "default": "opus-4.5-thinking",
  "models": [
    "claude-opus-4-20250514",
    "claude-sonnet-4-20250514",
    "gpt-5.2",
    "opus-4.5-thinking",
    "sonnet-4.5-thinking"
  ]
}
```

## Files to create/change

- Create `cmd/models.go` - new command implementation
- May update `internal/agent/config.go` - add model listing function per backend

## Implementation details

### cmd/models.go

```go
package cmd

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"

    "github.com/fatih/color"
    "github.com/spf13/cobra"
)

var (
    modelsFormat  string
    modelsDefault bool
)

var modelsCmd = &cobra.Command{
    Use:   "models",
    Short: "List available models for the current backend",
    Long: `List available models for the configured agent backend.

The default model (from config) is highlighted with an asterisk (*).`,
    Example: `  # List all available models
  swarm models

  # Output as JSON
  swarm models --format json

  # Show only the default model
  swarm models --default`,
    RunE: func(cmd *cobra.Command, args []string) error {
        backend := appConfig.Backend
        defaultModel := appConfig.Model

        // Show only default if requested
        if modelsDefault {
            fmt.Println(defaultModel)
            return nil
        }

        // Get models based on backend
        var models []string
        var err error

        switch backend {
        case "cursor":
            models, err = getCursorModels()
        case "claude-code":
            models = getClaudeCodeModels()
        default:
            return fmt.Errorf("unknown backend: %s", backend)
        }

        if err != nil {
            return fmt.Errorf("failed to get models: %w", err)
        }

        // JSON output
        if modelsFormat == "json" {
            output := map[string]interface{}{
                "backend": backend,
                "default": defaultModel,
                "models":  models,
            }
            data, err := json.MarshalIndent(output, "", "  ")
            if err != nil {
                return err
            }
            fmt.Println(string(data))
            return nil
        }

        // Table output
        fmt.Printf("Available models (%s backend):\n\n", backend)

        defaultColor := color.New(color.FgGreen, color.Bold)
        for _, model := range models {
            if model == defaultModel {
                defaultColor.Printf("* %s (default)\n", model)
            } else {
                fmt.Printf("  %s\n", model)
            }
        }

        fmt.Println("\nUse 'swarm run -m <model>' to use a specific model.")
        return nil
    },
}

// getCursorModels retrieves available models from the Cursor agent CLI
func getCursorModels() ([]string, error) {
    cmd := exec.Command("agent", "--list-models")
    output, err := cmd.Output()
    if err != nil {
        // Fall back to known models if command fails
        return []string{
            "opus-4.5-thinking",
            "sonnet-4.5-thinking",
            "opus-4.5",
            "sonnet-4.5",
            "claude-opus-4-20250514",
            "claude-sonnet-4-20250514",
            "gpt-5.2",
            "gpt-5.2-codex",
            "gemini-3-pro",
            "gemini-3-flash",
            "grok",
        }, nil
    }

    // Parse output (one model per line)
    lines := strings.Split(strings.TrimSpace(string(output)), "\n")
    var models []string
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line != "" && !strings.HasPrefix(line, "#") {
            models = append(models, line)
        }
    }
    return models, nil
}

// getClaudeCodeModels returns the known models for Claude Code CLI
func getClaudeCodeModels() []string {
    // Claude Code has a fixed set of models
    return []string{
        "opus",
        "sonnet",
    }
}

func init() {
    modelsCmd.Flags().StringVar(&modelsFormat, "format", "", "Output format: json or table (default)")
    modelsCmd.Flags().BoolVar(&modelsDefault, "default", false, "Show only the default model")
    rootCmd.AddCommand(modelsCmd)
}
```

## Use cases

### First-time setup

```bash
# User just installed swarm, wants to know what models are available
swarm models
# Sees list, picks one
swarm config set-model sonnet-4.5-thinking
```

### Scripting

```bash
# Get default model for a script
DEFAULT=$(swarm models --default)
echo "Using model: $DEFAULT"

# Get all models as JSON
swarm models --format json | jq '.models[]'
```

### Model validation (future enhancement)

The model list could be used to validate model names in `swarm run`:

```bash
swarm run -p task -m invalid-model
# Error: unknown model "invalid-model". Run 'swarm models' to see available models.
```

### Backend comparison

```bash
# Check Cursor models
swarm config set-backend cursor
swarm models

# Check Claude Code models  
swarm config set-backend claude-code
swarm models
```

## Edge cases

1. **Backend CLI not installed**: If `agent --list-models` fails, fall back to a hardcoded list of known models with a warning.

2. **Unknown backend**: Return error with suggestion to run `swarm config set-backend`.

3. **Empty model list**: Show helpful message suggesting the backend may not be configured correctly.

4. **Default model not in list**: Still show it as default (may be a custom/beta model).

5. **Network issues for remote model lists**: If a backend requires network to list models (none currently do), handle gracefully with timeout.

## Future enhancements (out of scope)

1. **Model info**: `swarm models --info opus` to show details about a specific model (capabilities, pricing tier, etc.)

2. **Model aliases**: Support short names that map to full model names (e.g., `opus` -> `claude-opus-4-20250514`)

3. **Model validation**: Warn or error when using an unknown model name in `swarm run`

4. **Backend auto-detection**: If no backend configured, try to detect which CLIs are installed

## Acceptance criteria

- `swarm models` lists available models for the current backend
- Default model is highlighted with `*` and "(default)" label
- `--format json` outputs valid JSON with backend, default, and models
- `--default` outputs only the default model name (for scripting)
- Works with both `cursor` and `claude-code` backends
- Gracefully handles missing backend CLI with fallback model list
- Help text explains how to use models with `swarm run`

---

## Completion Notes (Agent cd59a862)

### Implementation Summary

Created `cmd/models.go` implementing the `swarm models` command:

1. **Basic functionality**: Lists available models for the current backend
2. **Default model highlighting**: Shows default model with `*` prefix and "(default)" label in green
3. **JSON output**: `--format json` outputs valid JSON with backend, default, and models array
4. **Default-only mode**: `--default` outputs only the default model name for scripting
5. **Cursor backend**: Fetches models from `agent --list-models` command, with fallback to hardcoded list
6. **Claude Code backend**: Returns fixed list of known models (opus, sonnet)

Also created `cmd/models_test.go` with tests for:
- ANSI escape code stripping
- Claude Code model list
- Fallback Cursor model list

### Files Created
- `cmd/models.go` - Main command implementation
- `cmd/models_test.go` - Unit tests

### Acceptance Criteria Met
- [x] `swarm models` lists available models for the current backend
- [x] Default model is highlighted with `*` and "(default)" label
- [x] `--format json` outputs valid JSON with backend, default, and models
- [x] `--default` outputs only the default model name (for scripting)
- [x] Works with both `cursor` and `claude-code` backends
- [x] Gracefully handles missing backend CLI with fallback model list
- [x] Help text explains how to use models with `swarm run`
