# Add `swarm prompts new` command for creating prompts

## Completion Notes (Agent cd59a862)

Implemented `swarm prompts new` command with all requested features:

- Created `cmd/prompts_new.go` with the new command implementation
- Added `DefaultTemplate()` function to `internal/prompt/loader.go`
- Supports `--from` flag to copy from existing prompts
- Supports `--content` flag for inline content
- Supports `--no-edit` flag to skip editor
- Supports `--global` flag via existing persistent flag
- Handles edge cases: existing prompts, invalid names, .md extension handling
- Uses existing `resolveEditor()` function from prompts_edit.go for editor selection

All acceptance criteria verified through testing.

## Problem

Currently, users must manually create prompt files by navigating to the prompts directory and creating markdown files. This is cumbersome and error-prone:

1. Users need to remember the correct directory path (`./swarm/prompts/` or `~/.swarm/prompts/`)
2. Users must ensure the `.md` extension is added
3. There's no way to scaffold from an existing prompt as a starting point
4. New users may not know what a well-structured prompt looks like

Existing commands for prompts management:
- `swarm prompts list` - lists available prompts
- `swarm prompts show <name>` - displays a prompt's content
- `swarm prompts edit <name>` - opens a prompt in an editor

Missing: a way to create new prompts from the command line.

## Solution

Add a `swarm prompts new` command that creates new prompt files with optional templating.

### Proposed API

```bash
# Create an empty prompt (opens editor)
swarm prompts new my-feature

# Create with template content based on an existing prompt
swarm prompts new my-feature --from coder

# Create a global prompt instead of project-scoped
swarm prompts new my-feature --global

# Create and immediately open in editor (default behavior)
swarm prompts new my-feature

# Create without opening editor
swarm prompts new my-feature --no-edit

# Create with inline content (useful for scripting)
swarm prompts new my-feature --content "Review the code for security issues"
```

### Default template

When no `--from` or `--content` is specified, create a minimal template:

```markdown
# Task

Describe what you want the agent to accomplish.

# Context

Any relevant context about the codebase or task.

# Requirements

- Requirement 1
- Requirement 2

# Exit condition

Describe when the agent should consider the task complete.
```

## Files to create/change

- `cmd/prompts_new.go` - new command implementation
- `internal/prompt/loader.go` - add CreatePrompt and template functions

## Implementation details

### cmd/prompts_new.go

```go
package cmd

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"

    "github.com/mj1618/swarm-cli/internal/prompt"
    "github.com/spf13/cobra"
)

var (
    promptsNewFrom    string
    promptsNewContent string
    promptsNewNoEdit  bool
)

var promptsNewCmd = &cobra.Command{
    Use:     "new <name>",
    Aliases: []string{"create", "add"},
    Short:   "Create a new prompt file",
    Long: `Create a new prompt file in the prompts directory.

By default, creates a prompt with a starter template and opens it in your
default editor ($EDITOR or vi).

Use --from to copy an existing prompt as a starting point.
Use --content to create with specific content (useful for scripting).
Use --no-edit to create without opening the editor.`,
    Example: `  # Create a new prompt and open in editor
  swarm prompts new my-feature

  # Create based on an existing prompt
  swarm prompts new my-feature --from coder

  # Create a global prompt
  swarm prompts new my-helper --global

  # Create with specific content
  swarm prompts new quick-fix --content "Fix any linting errors in the codebase"

  # Create without opening editor
  swarm prompts new my-feature --no-edit`,
    Args: cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        name := args[0]

        // Validate name (no special characters, no .md extension needed)
        if strings.ContainsAny(name, "/\\:*?\"<>|") {
            return fmt.Errorf("invalid prompt name: contains special characters")
        }

        // Get prompts directory based on scope
        promptsDir, err := GetPromptsDir()
        if err != nil {
            return fmt.Errorf("failed to get prompts directory: %w", err)
        }

        // Ensure prompts directory exists
        if err := os.MkdirAll(promptsDir, 0755); err != nil {
            return fmt.Errorf("failed to create prompts directory: %w", err)
        }

        // Build file path
        filename := name
        if !strings.HasSuffix(filename, ".md") {
            filename = filename + ".md"
        }
        filePath := filepath.Join(promptsDir, filename)

        // Check if file already exists
        if _, err := os.Stat(filePath); err == nil {
            return fmt.Errorf("prompt %q already exists; use 'swarm prompts edit %s' to modify it", name, name)
        }

        // Determine content
        var content string
        switch {
        case promptsNewContent != "":
            content = promptsNewContent
        case promptsNewFrom != "":
            // Load existing prompt as template
            existingContent, err := prompt.LoadPromptRaw(promptsDir, promptsNewFrom)
            if err != nil {
                // Try global prompts if not found in project
                globalDir, globErr := getGlobalPromptsDir()
                if globErr == nil {
                    existingContent, err = prompt.LoadPromptRaw(globalDir, promptsNewFrom)
                }
                if err != nil {
                    return fmt.Errorf("template prompt %q not found: %w", promptsNewFrom, err)
                }
            }
            content = existingContent
        default:
            content = prompt.DefaultTemplate()
        }

        // Write the file
        if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
            return fmt.Errorf("failed to create prompt file: %w", err)
        }

        fmt.Printf("Created prompt: %s\n", filePath)

        // Open in editor unless --no-edit
        if !promptsNewNoEdit {
            editor := os.Getenv("EDITOR")
            if editor == "" {
                editor = "vi"
            }

            cmd := exec.Command(editor, filePath)
            cmd.Stdin = os.Stdin
            cmd.Stdout = os.Stdout
            cmd.Stderr = os.Stderr

            if err := cmd.Run(); err != nil {
                fmt.Printf("Note: could not open editor (%v)\n", err)
            }
        }

        return nil
    },
}

func getGlobalPromptsDir() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(homeDir, ".swarm", "prompts"), nil
}

func init() {
    promptsNewCmd.Flags().StringVar(&promptsNewFrom, "from", "", "Copy content from an existing prompt")
    promptsNewCmd.Flags().StringVar(&promptsNewContent, "content", "", "Initial content for the prompt")
    promptsNewCmd.Flags().BoolVar(&promptsNewNoEdit, "no-edit", false, "Don't open the editor after creating")

    promptsCmd.AddCommand(promptsNewCmd)
}
```

### internal/prompt/loader.go additions

```go
// DefaultTemplate returns a starter template for new prompts.
func DefaultTemplate() string {
    return `# Task

Describe what you want the agent to accomplish.

# Context

Any relevant context about the codebase or task.

# Requirements

- Requirement 1
- Requirement 2

# Exit condition

Describe when the agent should consider the task complete.
`
}
```

## Use cases

### New user creating their first prompt

```bash
$ swarm prompts new coder
Created prompt: ./swarm/prompts/coder.md
# Editor opens with template
```

### Copying a working prompt to create a variant

```bash
$ swarm prompts new coder-v2 --from coder
Created prompt: ./swarm/prompts/coder-v2.md
# Editor opens with coder's content
```

### Scripting/automation

```bash
# Create multiple prompts without interaction
swarm prompts new lint-check --content "Find and fix all linting errors" --no-edit
swarm prompts new test-writer --content "Write unit tests for untested functions" --no-edit
```

### Creating a global prompt for use across projects

```bash
$ swarm prompts new common-review --global
Created prompt: ~/.swarm/prompts/common-review.md
```

## Edge cases

1. **Prompt already exists**: Return error suggesting to use `prompts edit` instead
2. **Invalid name** (contains `/`, `\`, etc.): Return descriptive error
3. **No EDITOR set**: Fall back to `vi`
4. **Editor not found**: Create file anyway, print note that editor couldn't open
5. **Template prompt not found**: Check both project and global prompts directories
6. **Prompts directory doesn't exist**: Create it automatically
7. **Name includes .md extension**: Accept it gracefully (don't double-add)

## Acceptance criteria

- `swarm prompts new my-feature` creates `./swarm/prompts/my-feature.md` with default template
- `swarm prompts new my-feature --global` creates in `~/.swarm/prompts/`
- `swarm prompts new my-feature --from coder` copies content from existing `coder` prompt
- `swarm prompts new my-feature --content "..."` uses provided content
- `swarm prompts new my-feature --no-edit` creates without opening editor
- Error if prompt already exists
- Error if name contains invalid characters
- Creates prompts directory if it doesn't exist
- Editor defaults to $EDITOR, falls back to vi
