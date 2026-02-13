# Add stdin prompt input support

## Completion Notes (Agent cd59a862)

Implemented stdin prompt input support. Changes made:

### Files Modified:
- `internal/prompt/loader.go` - Added `LoadPromptFromStdin()`, `IsStdinPiped()`, and `CombinePrompts()` functions
- `internal/prompt/loader_test.go` - Added tests for `CombinePrompts()` function
- `cmd/run.go` - Added `--stdin`/`-i` flag with full support including:
  - Reading prompt content from stdin
  - Combining stdin with named prompts using `{{STDIN}}` placeholder or appending
  - Proper error handling for no input, empty input, and conflicting flags
  - Detached mode support (stdin content passed via `--_internal-stdin` flag to child process)

### Testing:
- All unit tests pass
- Manual testing confirmed:
  - `echo "test" | swarm run --stdin` - works correctly
  - `echo "" | swarm run --stdin` - returns "stdin is empty" error
  - `swarm run --stdin < /dev/null` - returns "no input piped" error
  - `--stdin --prompt-string "x"` - returns "cannot be combined" error

---

## Problem

Currently, swarm provides three ways to specify a prompt:
1. `--prompt/-p` - Named prompt from the prompts directory
2. `--prompt-file/-f` - Load prompt from a file path
3. `--prompt-string/-s` - Inline prompt string

However, there's no way to pipe content into the prompt from stdin. This limits the ability to use swarm in powerful command-line workflows such as:

```bash
# These workflows are NOT currently possible:
cat README.md | swarm run -p reviewer
git diff | swarm run -p code-reviewer
echo "Fix the bug in auth.go" | swarm run
curl https://example.com/spec.md | swarm run -p implementer
```

Users wanting to dynamically generate prompts or pass file contents must use workarounds:
- Save to a temp file and use `--prompt-file`
- Use command substitution with `--prompt-string "$(cat file.txt)"` (fails with large content or special characters)

## Solution

Add stdin support with a `--stdin` flag (or `-i`) that reads the prompt content from stdin.

### Proposed API

```bash
# Read entire prompt from stdin
echo "Review this code for bugs" | swarm run --stdin

# Pipe file contents as prompt
cat requirements.md | swarm run --stdin

# Combine with git workflows
git diff | swarm run --stdin -m claude-sonnet-4-20250514

# Use with process substitution
swarm run --stdin < <(echo "Task: "; cat spec.md)

# Pipe command output
grep -r "TODO" src/ | swarm run --stdin -p todo-fixer
```

### Combining stdin with prompt templates

When `--stdin` is used alongside `--prompt/-p`, the stdin content could be appended to the named prompt (or injected at a placeholder):

```bash
# stdin content appended to the 'reviewer' prompt
cat code.go | swarm run --stdin -p reviewer

# Or with a placeholder in the prompt file:
# reviewer.md: "Review this code:\n\n{{STDIN}}\n\nFocus on security issues."
```

### Auto-detection (optional enhancement)

Optionally, swarm could auto-detect piped input when no prompt is specified:

```bash
# If stdin has content and no prompt specified, use stdin
echo "Fix the typo in README.md" | swarm run
```

However, this should be opt-in or behind a flag to avoid confusion.

## Files to create/change

- `cmd/run.go` - Add `--stdin` flag and stdin reading logic
- `internal/prompt/loader.go` - Add `LoadPromptFromStdin()` function

## Implementation details

### cmd/run.go changes

```go
var runStdin bool

// In RunE, add stdin handling in the prompt selection switch:
case runStdin:
    // Check if stdin has data
    stat, _ := os.Stdin.Stat()
    if (stat.Mode() & os.ModeCharDevice) != 0 {
        return fmt.Errorf("--stdin specified but no input piped")
    }
    
    promptContent, err = prompt.LoadPromptFromStdin()
    if err != nil {
        return fmt.Errorf("failed to read prompt from stdin: %w", err)
    }
    promptName = "<stdin>"
    
    // If a named prompt is also specified, combine them
    if runPrompt != "" {
        basePrompt, err := prompt.LoadPrompt(promptsDir, runPrompt)
        if err != nil {
            return fmt.Errorf("failed to load prompt: %w", err)
        }
        promptContent = prompt.CombinePrompts(basePrompt, promptContent)
        promptName = runPrompt + "+stdin"
    }

// In init():
runCmd.Flags().BoolVarP(&runStdin, "stdin", "i", false, "Read prompt content from stdin")
```

### internal/prompt/loader.go additions

```go
import (
    "bufio"
    "io"
    "os"
    "strings"
)

// LoadPromptFromStdin reads prompt content from stdin.
func LoadPromptFromStdin() (string, error) {
    reader := bufio.NewReader(os.Stdin)
    var builder strings.Builder
    
    for {
        line, err := reader.ReadString('\n')
        builder.WriteString(line)
        if err != nil {
            if err == io.EOF {
                break
            }
            return "", err
        }
    }
    
    content := strings.TrimSpace(builder.String())
    if content == "" {
        return "", fmt.Errorf("stdin is empty")
    }
    
    return WrapPromptString(content), nil
}

// CombinePrompts combines a base prompt with additional content.
// If the base prompt contains {{STDIN}}, it's replaced. Otherwise, content is appended.
func CombinePrompts(base, additional string) string {
    const placeholder = "{{STDIN}}"
    if strings.Contains(base, placeholder) {
        return strings.Replace(base, placeholder, additional, 1)
    }
    return base + "\n\n---\n\n" + additional
}
```

### Flag conflict checking

```go
// Update the specifiedCount check:
if runStdin {
    specifiedCount++
}
// Or allow stdin to combine with --prompt but not with --prompt-file or --prompt-string
```

## Use cases

### Code review workflow

```bash
# Review staged changes
git diff --staged | swarm run --stdin -p reviewer

# Review a specific file
cat src/auth.go | swarm run --stdin -p security-reviewer
```

### Documentation generation

```bash
# Generate docs from code
cat src/api.go | swarm run --stdin -p doc-generator
```

### Bug fixing

```bash
# Pass error logs to a fixer agent
tail -100 app.log | grep ERROR | swarm run --stdin -p debugger
```

### Multi-file context

```bash
# Pass multiple files as context
cat src/*.go | swarm run --stdin -p refactorer
```

### Integration with other tools

```bash
# Pipe from clipboard (macOS)
pbpaste | swarm run --stdin -p helper

# Pipe from web
curl -s https://api.example.com/spec | swarm run --stdin -p implementer
```

### Using prompt templates with stdin

Create a prompt template with a placeholder:

```markdown
# reviewer.md
Review the following code:

{{STDIN}}

Focus on:
- Security vulnerabilities
- Performance issues
- Code style
```

Then use it:

```bash
cat buggy-code.go | swarm run --stdin -p reviewer
```

## Edge cases

1. **Empty stdin**: Return error "stdin is empty" - don't silently use an empty prompt.

2. **Large stdin**: Read entirely into memory. For extremely large inputs (>10MB), consider a warning or truncation (with `--max-stdin-size` option).

3. **Binary input**: Detect and reject binary content with a helpful error message.

4. **Stdin with detached mode**: This is tricky since the detached process won't have access to the original stdin. Solution: Read stdin content in parent process and pass via `--prompt-string` to child (or temp file for large content).

5. **Interactive terminal**: If stdin is a terminal (not piped), show error when `--stdin` is used: "No input piped. Use a pipe or redirect."

6. **Combined with prompt template**: When using both `--stdin` and `--prompt`, inject stdin content at `{{STDIN}}` placeholder if present, otherwise append.

7. **Timeout on stdin read**: Add a reasonable timeout (30 seconds) to prevent hanging if stdin is from a slow source.

## Dependencies

No new dependencies required.

## Acceptance criteria

- `echo "test" | swarm run --stdin` runs agent with "test" as prompt
- `cat file.md | swarm run --stdin` runs agent with file contents as prompt
- `swarm run --stdin` without piped input shows helpful error
- `--stdin` can be combined with `--prompt` to augment a named prompt
- `--stdin` cannot be combined with `--prompt-file` or `--prompt-string`
- Works correctly with detached mode (`-d`)
- Large inputs (up to reasonable limit) work correctly
- Stdin content is properly wrapped/formatted as a prompt
