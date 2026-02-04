# Add `swarm prompts show` command to view prompt contents

## Problem

The `swarm prompts` command lists available prompts by name, but there's no way to view the content of a prompt from the CLI. Users have to manually navigate to `./swarm/prompts/` or `~/.swarm/prompts/` and open the file to see what a prompt contains.

This is a common workflow:
1. User runs `swarm prompts` to see available prompts
2. User wants to check what "coder" or "planner" actually does
3. User has to `cat ./swarm/prompts/coder.md` manually

The swarm CLI follows Docker-like conventions (per PLAN.md), and Docker has `docker inspect` for viewing details. Adding `swarm prompts show <name>` would complete the prompts workflow.

## Solution

Add a `show` subcommand to `prompts` that displays the content of a named prompt.

### Proposed API

```bash
# Show content of a prompt from the project prompts directory
swarm prompts show coder

# Show content of a global prompt
swarm prompts show -g shared-task

# Show with path information
swarm prompts show coder --path
# Output includes: "File: ./swarm/prompts/coder.md" header

# Pipe-friendly raw output (no decorations)
swarm prompts show coder --raw
```

### Default output

```
═══════════════════════════════════════════════════════════════════════════════
Prompt: coder
File: ./swarm/prompts/coder.md
═══════════════════════════════════════════════════════════════════════════════

# Coder Prompt

You are a coding agent. Your task is to:

1. Read the task file
2. Implement the required changes
3. Run tests to verify

...
```

### Raw output (--raw)

Just the prompt content with no header, suitable for piping:

```
# Coder Prompt

You are a coding agent...
```

## Files to create/change

- Modify `cmd/prompts.go` - convert to parent command with subcommands
- Create `cmd/prompts_list.go` - move list functionality here (or keep inline)
- Create `cmd/prompts_show.go` - new show subcommand
- Modify `internal/prompt/loader.go` - add `GetPromptPath()` helper if needed

## Implementation details

### cmd/prompts.go (updated to be parent command)

```go
package cmd

import (
	"github.com/spf13/cobra"
)

var promptsCmd = &cobra.Command{
	Use:   "prompts",
	Short: "Manage prompt files",
	Long: `Manage prompt files used by agents.

Prompts are markdown files stored in:
  - Project: ./swarm/prompts/
  - Global:  ~/.swarm/prompts/`,
}

func init() {
	// Subcommands added in their respective files
}
```

### cmd/prompts_list.go

```go
package cmd

import (
	"fmt"

	"github.com/mj1618/swarm-cli/internal/prompt"
	"github.com/spf13/cobra"
)

var promptsListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List available prompt files",
	Long: `List all available prompt files from the prompts directory.

By default, shows prompts from the project directory (./swarm/prompts/).
Use --global to show prompts from the global directory (~/.swarm/prompts/).`,
	Example: `  # List prompts in current project
  swarm prompts list

  # List global prompts
  swarm prompts list -g`,
	RunE: func(cmd *cobra.Command, args []string) error {
		promptsDir, err := GetPromptsDir()
		if err != nil {
			return fmt.Errorf("failed to get prompts directory: %w", err)
		}

		prompts, err := prompt.ListPrompts(promptsDir)
		if err != nil {
			return err
		}

		if len(prompts) == 0 {
			fmt.Printf("No prompts found in %s\n", promptsDir)
			return nil
		}

		fmt.Printf("Available prompts (%s):\n", promptsDir)
		for _, p := range prompts {
			fmt.Printf("  %s\n", p)
		}

		return nil
	},
}

func init() {
	promptsCmd.AddCommand(promptsListCmd)
}
```

### cmd/prompts_show.go

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/mj1618/swarm-cli/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	promptsShowRaw  bool
	promptsShowPath bool
)

var promptsShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show the content of a prompt",
	Long: `Display the content of a prompt file.

By default, shows prompts from the project directory (./swarm/prompts/).
Use --global to show a prompt from the global directory (~/.swarm/prompts/).`,
	Example: `  # Show a project prompt
  swarm prompts show coder

  # Show a global prompt
  swarm prompts show -g shared-task

  # Show with file path header
  swarm prompts show coder --path

  # Raw output (no decorations, for piping)
  swarm prompts show coder --raw`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		promptName := args[0]

		promptsDir, err := GetPromptsDir()
		if err != nil {
			return fmt.Errorf("failed to get prompts directory: %w", err)
		}

		// Load the prompt content
		content, err := prompt.LoadPrompt(promptsDir, promptName)
		if err != nil {
			return fmt.Errorf("failed to load prompt '%s': %w", promptName, err)
		}

		// Construct the file path for display
		promptPath := filepath.Join(promptsDir, promptName+".md")

		// Raw mode: just output content
		if promptsShowRaw {
			fmt.Print(content)
			// Ensure trailing newline
			if !strings.HasSuffix(content, "\n") {
				fmt.Println()
			}
			return nil
		}

		// Formatted output with header
		bold := color.New(color.Bold)
		dim := color.New(color.Faint)

		separator := strings.Repeat("═", 79)
		bold.Println(separator)
		fmt.Printf("Prompt: ")
		bold.Println(promptName)
		if promptsShowPath {
			fmt.Printf("File: ")
			dim.Println(promptPath)
		}
		bold.Println(separator)
		fmt.Println()
		fmt.Print(content)
		if !strings.HasSuffix(content, "\n") {
			fmt.Println()
		}

		return nil
	},
}

func init() {
	promptsShowCmd.Flags().BoolVar(&promptsShowRaw, "raw", false, "Output raw content without formatting")
	promptsShowCmd.Flags().BoolVar(&promptsShowPath, "path", false, "Show the file path in the header")
	promptsCmd.AddCommand(promptsShowCmd)
}
```

### Backward compatibility

To maintain backward compatibility, `swarm prompts` (with no subcommand) should still list prompts. This can be done by setting `list` as the default action:

```go
// In cmd/prompts.go
var promptsCmd = &cobra.Command{
	Use:   "prompts",
	Short: "Manage prompt files",
	Long:  `...`,
	// Run list by default when no subcommand given
	Run: func(cmd *cobra.Command, args []string) {
		promptsListCmd.Run(cmd, args)
	},
}
```

Or alternatively, use Cobra's `SetDefaultCommand` or check if subcommand was provided.

## Edge cases

1. **Prompt not found**: Return clear error message with the path that was searched.
   ```
   Error: failed to load prompt 'nonexistent': prompt file not found: ./swarm/prompts/nonexistent.md
   ```

2. **Empty prompt file**: Display the header but show "(empty file)" or just blank content.

3. **Prompt name with special characters**: Handle prompts like `my-task` or `feature_123` correctly.

4. **Very long prompts**: No pagination - just output the full content. Users can pipe to `less` if needed.

5. **Binary or corrupted file**: The existing `LoadPrompt` function handles this - just output whatever is read.

6. **No prompts directory**: Error message should indicate whether it's project or global scope that's missing.

7. **Backward compatibility**: `swarm prompts` without subcommand should still work and list prompts (delegates to `swarm prompts list`).

## Testing

### Manual testing

```bash
# Create a test prompt
mkdir -p ./swarm/prompts
echo "# Test Prompt\n\nThis is a test." > ./swarm/prompts/test-prompt.md

# Test show command
swarm prompts show test-prompt
swarm prompts show test-prompt --path
swarm prompts show test-prompt --raw
swarm prompts show test-prompt --raw | wc -l

# Test error case
swarm prompts show nonexistent

# Test backward compatibility
swarm prompts  # should still list prompts
```

### Unit tests

Add `cmd/prompts_show_test.go`:

```go
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptsShowCommand(t *testing.T) {
	// Create temp prompts directory
	tmpDir := t.TempDir()
	promptsDir := filepath.Join(tmpDir, "swarm", "prompts")
	os.MkdirAll(promptsDir, 0755)

	// Create test prompt
	testContent := "# Test\n\nHello world"
	os.WriteFile(filepath.Join(promptsDir, "test.md"), []byte(testContent), 0644)

	// Test basic show (would need to mock GetPromptsDir or use integration test)
	// This is a placeholder for the actual test implementation
}

func TestPromptsShowRawOutput(t *testing.T) {
	// Test that --raw produces clean output without headers
}

func TestPromptsShowNotFound(t *testing.T) {
	// Test error handling for missing prompts
}
```

## Acceptance criteria

- `swarm prompts show <name>` displays the content of the named prompt
- `swarm prompts show <name> --path` includes the file path in the output header
- `swarm prompts show <name> --raw` outputs only the content with no formatting
- `swarm prompts show -g <name>` shows a global prompt
- Clear error message when prompt doesn't exist
- `swarm prompts` (no subcommand) still lists prompts for backward compatibility
- `swarm prompts list` works as explicit list command
- Works with both project and global prompts directories

---

## Completion Notes (Agent 118d3fa6)

**Completed on:** 2026-01-28

**Files created:**
- `cmd/prompts_show.go` - Show subcommand implementation with --raw and --path flags

**Files modified:**
- `cmd/prompts.go` - Converted to parent command with backward compatibility (swarm prompts still lists prompts)
- `internal/prompt/loader.go` - Added `LoadPromptRaw()` and `GetPromptPath()` helper functions

**All acceptance criteria met:**
- `swarm prompts show <name>` displays prompt content with formatted header
- `swarm prompts show <name> --path` includes file path in header
- `swarm prompts show <name> --raw` outputs raw content for piping
- `swarm prompts show -g <name>` shows global prompt (uses existing --global flag)
- Clear error message when prompt doesn't exist
- `swarm prompts` (no subcommand) still lists prompts for backward compatibility
- `swarm prompts list` and `swarm prompts ls` work as explicit list commands
- All existing tests pass
