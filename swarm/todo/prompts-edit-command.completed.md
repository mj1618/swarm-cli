# Add `swarm prompts edit` command for quick prompt editing

## Completion Notes (Agent cd59a862)

**Status**: Completed

**Implementation**:
- Created `cmd/prompts_edit.go` with the full edit subcommand implementation
- Created `cmd/prompts_edit_test.go` with tests for the `resolveEditor` function
- The `GetPromptPath` function already existed in `internal/prompt/loader.go`, no changes needed there

**Features implemented**:
- `swarm prompts edit <name>` opens the prompt in the user's editor
- `--editor` / `-e` flag to override the editor
- `--create` flag to create new prompts if they don't exist
- `--global` / `-g` flag support (inherited from parent command)
- Editor resolution: `--editor` flag > `$VISUAL` > `$EDITOR` > fallback (vim/vi/nano on Unix, notepad on Windows)

**Testing**:
- All unit tests pass
- Manual testing confirmed: help text, error messages, --create flag, --editor flag

---

## Problem

Currently, when users want to edit a prompt file, they need to:

1. Run `swarm prompts show <name> --path` to find the file location
2. Manually open the file in their preferred editor

This is cumbersome, especially when iterating on prompts. Other CLI tools (like `kubectl edit`, `git config --edit`, `crontab -e`) provide built-in edit commands that open files in the user's `$EDITOR`.

Users who frequently modify prompts would benefit from a streamlined workflow:

```bash
# Current workflow (2 steps)
swarm prompts show coder --path
# Output: File: /Users/me/project/swarm/prompts/coder.md
vim /Users/me/project/swarm/prompts/coder.md

# Proposed workflow (1 step)
swarm prompts edit coder
```

## Solution

Add a `swarm prompts edit <name>` command that opens the specified prompt file in the user's preferred editor.

### Proposed API

```bash
# Edit a project prompt (opens in $EDITOR)
swarm prompts edit coder

# Edit a global prompt
swarm prompts edit -g shared-task

# Specify editor explicitly (overrides $EDITOR)
swarm prompts edit coder --editor vim
swarm prompts edit coder -e code

# Create a new prompt if it doesn't exist
swarm prompts edit new-task --create
```

### Editor resolution order

The command determines which editor to use in this order:

1. `--editor` / `-e` flag (if specified)
2. `$VISUAL` environment variable
3. `$EDITOR` environment variable
4. Fallback to common editors: `vim`, `vi`, `nano`, `notepad` (platform-dependent)

### Behavior

1. If the prompt exists, open it in the editor
2. If the prompt doesn't exist and `--create` is specified, create an empty file and open it
3. If the prompt doesn't exist and `--create` is not specified, show an error with a hint
4. Wait for the editor to close before returning control to the terminal

## Files to create/change

- `cmd/prompts_edit.go` (new) - New edit subcommand
- `internal/prompt/loader.go` - Add `GetPromptPath` function if not already exported

## Implementation details

### cmd/prompts_edit.go

```go
package cmd

import (
    "fmt"
    "os"
    "os/exec"
    "runtime"

    "github.com/mj1618/swarm-cli/internal/prompt"
    "github.com/spf13/cobra"
)

var (
    promptsEditEditor string
    promptsEditCreate bool
)

var promptsEditCmd = &cobra.Command{
    Use:   "edit <name>",
    Short: "Edit a prompt file in your preferred editor",
    Long: `Open a prompt file in your preferred text editor.

The editor is determined by:
1. --editor flag (if specified)
2. $VISUAL environment variable
3. $EDITOR environment variable
4. Fallback: vim, vi, nano (Unix) or notepad (Windows)

By default, shows prompts from the project directory (./swarm/prompts/).
Use --global to edit a prompt from the global directory (~/.swarm/prompts/).`,
    Example: `  # Edit a project prompt
  swarm prompts edit coder

  # Edit a global prompt
  swarm prompts edit -g shared-task

  # Use a specific editor
  swarm prompts edit coder --editor code
  swarm prompts edit coder -e vim

  # Create a new prompt if it doesn't exist
  swarm prompts edit new-task --create`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        promptName := args[0]

        promptsDir, err := GetPromptsDir()
        if err != nil {
            return fmt.Errorf("failed to get prompts directory: %w", err)
        }

        promptPath := prompt.GetPromptPath(promptsDir, promptName)

        // Check if file exists
        if _, err := os.Stat(promptPath); os.IsNotExist(err) {
            if !promptsEditCreate {
                return fmt.Errorf("prompt '%s' not found at %s\n\nUse --create to create a new prompt", promptName, promptPath)
            }
            
            // Create the prompts directory if needed
            if err := os.MkdirAll(promptsDir, 0755); err != nil {
                return fmt.Errorf("failed to create prompts directory: %w", err)
            }
            
            // Create empty file
            if err := os.WriteFile(promptPath, []byte(""), 0644); err != nil {
                return fmt.Errorf("failed to create prompt file: %w", err)
            }
            fmt.Printf("Created new prompt: %s\n", promptPath)
        }

        // Determine editor
        editor := resolveEditor(promptsEditEditor)
        if editor == "" {
            return fmt.Errorf("no editor found. Set $EDITOR or use --editor flag")
        }

        // Open editor
        editorCmd := exec.Command(editor, promptPath)
        editorCmd.Stdin = os.Stdin
        editorCmd.Stdout = os.Stdout
        editorCmd.Stderr = os.Stderr

        if err := editorCmd.Run(); err != nil {
            return fmt.Errorf("editor exited with error: %w", err)
        }

        return nil
    },
}

// resolveEditor determines which editor to use.
func resolveEditor(override string) string {
    // 1. Command-line override
    if override != "" {
        return override
    }

    // 2. $VISUAL
    if editor := os.Getenv("VISUAL"); editor != "" {
        return editor
    }

    // 3. $EDITOR
    if editor := os.Getenv("EDITOR"); editor != "" {
        return editor
    }

    // 4. Platform-specific fallbacks
    if runtime.GOOS == "windows" {
        return "notepad"
    }

    // Try common Unix editors
    for _, editor := range []string{"vim", "vi", "nano"} {
        if _, err := exec.LookPath(editor); err == nil {
            return editor
        }
    }

    return ""
}

func init() {
    promptsEditCmd.Flags().StringVarP(&promptsEditEditor, "editor", "e", "", "Editor to use (overrides $EDITOR)")
    promptsEditCmd.Flags().BoolVar(&promptsEditCreate, "create", false, "Create the prompt file if it doesn't exist")
    promptsCmd.AddCommand(promptsEditCmd)
}
```

## Edge cases

1. **Editor not found**: Clear error message telling user to set `$EDITOR` or use `--editor` flag.

2. **Prompt doesn't exist**: By default, show error with hint about `--create`. With `--create`, create an empty file and open it.

3. **Prompts directory doesn't exist**: When using `--create`, automatically create the directory structure.

4. **Editor exits with error**: Report the error but don't treat it as fatal (user might have just quit without saving).

5. **Editor requires flags**: Some editors (like VS Code) need `--wait` flag to block. Users can specify this via `--editor "code --wait"`.

6. **Relative vs absolute paths**: Always pass absolute paths to the editor to avoid confusion.

7. **Windows compatibility**: Use `notepad` as fallback on Windows, handle path separators correctly.

## Tests to add

Add test file `cmd/prompts_edit_test.go`:

```go
package cmd

import (
    "os"
    "testing"
)

func TestResolveEditor(t *testing.T) {
    tests := []struct {
        name     string
        override string
        visual   string
        editor   string
        want     string
    }{
        {
            name:     "override takes precedence",
            override: "custom-editor",
            visual:   "visual-editor",
            editor:   "default-editor",
            want:     "custom-editor",
        },
        {
            name:   "VISUAL over EDITOR",
            visual: "visual-editor",
            editor: "default-editor",
            want:   "visual-editor",
        },
        {
            name:   "EDITOR as fallback",
            editor: "default-editor",
            want:   "default-editor",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Save and restore environment
            oldVisual := os.Getenv("VISUAL")
            oldEditor := os.Getenv("EDITOR")
            defer func() {
                os.Setenv("VISUAL", oldVisual)
                os.Setenv("EDITOR", oldEditor)
            }()

            os.Setenv("VISUAL", tt.visual)
            os.Setenv("EDITOR", tt.editor)

            got := resolveEditor(tt.override)
            // Note: on systems without vim/vi/nano, fallback won't work
            // so we only check when we expect a specific result
            if tt.want != "" && got != tt.want {
                t.Errorf("resolveEditor() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Use cases

### Quick iteration on prompts

```bash
# Edit, save, run - fast feedback loop
swarm prompts edit coder
swarm run -p coder -n 1
# Not quite right? Edit again
swarm prompts edit coder
```

### Create new prompts from scratch

```bash
# Create and edit a new prompt
swarm prompts edit my-new-task --create
```

### Use VS Code with waiting

```bash
# For GUI editors that don't block by default
swarm prompts edit coder -e "code --wait"
```

### Global prompt management

```bash
# Edit shared prompts used across projects
swarm prompts edit -g company-standards
```

## Acceptance criteria

- `swarm prompts edit coder` opens the prompt in the user's editor
- `swarm prompts edit -g shared` opens global prompts
- `--editor vim` overrides environment variables
- `--create` creates new prompts if they don't exist
- Clear error message when prompt not found (without `--create`)
- Clear error message when no editor is configured
- Works on Linux, macOS, and Windows
- Editor blocks until user closes the file
- Environment variables `$VISUAL` and `$EDITOR` are respected in correct order
