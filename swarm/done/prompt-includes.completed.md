# Add prompt include/import directive for prompt composition

## Completed by cd59a862

### Implementation Summary

Created the following files:
- `internal/prompt/include.go` - Core include processing with `ProcessIncludes()`, `ValidateIncludes()`, and `ExtractIncludes()` functions
- `internal/prompt/include_test.go` - Comprehensive tests for include functionality
- `cmd/prompts_check.go` - New `swarm prompts check` subcommand

Modified:
- `internal/prompt/loader.go` - Integrated include processing into `LoadPrompt()` and `LoadPromptFromFile()`, added `LoadPromptRawExpanded()`
- `cmd/prompts_show.go` - Added `--expand/-e` flag

Added example files:
- `swarm/prompts/common/rules.md` - Example shared rules file
- `swarm/prompts/test-includes.md` - Example prompt using includes

### Features Implemented
- `{{include: path}}` directive for including files
- Auto `.md` extension if not specified
- Nested includes with circular dependency detection
- Max depth limit (10 levels)
- Binary file detection
- Whitespace preservation
- `swarm prompts show --expand` to view expanded prompts
- `swarm prompts check` to validate all prompts

### Not Implemented (Out of Scope)
- Configuration in `.swarm.toml` (hardcoded defaults are sufficient for now)
- Path traversal warnings

---

## Problem

When managing multiple prompts for different agents, users often have common instructions that should be shared:

1. **Common rules**: Project-specific coding standards, tool usage guidelines, or safety rules
2. **Shared context**: Repository structure explanations, API documentation, or architecture notes
3. **Reusable templates**: Standard task structures, output formats, or checklists

Currently, users must either:
- Duplicate content across prompts (maintenance nightmare)
- Use very long monolithic prompts
- Manually concatenate prompts with shell scripting

Example of the problem:
```bash
# Every prompt repeats the same 50 lines of project rules
# prompts/feature-a.md - 200 lines (50 common + 150 specific)
# prompts/feature-b.md - 180 lines (50 common + 130 specific)
# prompts/bugfix.md    - 170 lines (50 common + 120 specific)

# When rules change, must update all three files
```

## Solution

Add an `{{include: path}}` directive that allows prompts to include content from other files.

### Proposed Syntax

```markdown
# My Task Prompt

{{include: common/rules.md}}

## Specific Task

Now do the specific work for this task...

{{include: common/output-format.md}}
```

### Include Path Resolution

1. Paths are relative to the prompts directory (e.g., `./swarm/prompts/`)
2. Absolute paths are supported but discouraged
3. File extension is optional (`.md` assumed if not specified)
4. Nested includes are supported (with cycle detection)

### Example Structure

```
swarm/
  prompts/
    common/
      rules.md           # Shared coding rules
      safety.md          # Safety guidelines
      output-format.md   # Standard output format
    tasks/
      feature.md         # {{include: common/rules.md}} + feature task
      bugfix.md          # {{include: common/rules.md}} + bugfix task
      review.md          # {{include: common/rules.md}} + review task
```

### Directive Behavior

| Directive | Behavior |
|-----------|----------|
| `{{include: file.md}}` | Include file content at this location |
| `{{include: dir/file.md}}` | Include from subdirectory |
| `{{include: ../other/file.md}}` | Include from parent directory |
| `{{include: file}}` | Auto-add `.md` extension |

### Error Handling

- **File not found**: Error with helpful message showing searched paths
- **Circular include**: Error listing the cycle (e.g., "a.md → b.md → a.md")
- **Invalid path**: Error with valid path examples
- **Max depth exceeded**: Error if nesting exceeds 10 levels (configurable)

## Files to create/change

- Modify `internal/prompt/loader.go` - add include processing logic
- Create `internal/prompt/include.go` - include directive parser and resolver
- Add tests in `internal/prompt/include_test.go`

## Implementation details

### internal/prompt/include.go

```go
package prompt

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"
)

var includeRegex = regexp.MustCompile(`\{\{include:\s*([^}]+)\}\}`)

const maxIncludeDepth = 10

// ProcessIncludes recursively processes include directives in prompt content.
// baseDir is the directory containing the prompt file (for relative path resolution).
// Returns the processed content with all includes expanded.
func ProcessIncludes(content string, baseDir string, depth int, seen map[string]bool) (string, error) {
    if depth > maxIncludeDepth {
        return "", fmt.Errorf("maximum include depth (%d) exceeded - check for circular includes", maxIncludeDepth)
    }

    if seen == nil {
        seen = make(map[string]bool)
    }

    // Find all include directives
    matches := includeRegex.FindAllStringSubmatchIndex(content, -1)
    if len(matches) == 0 {
        return content, nil
    }

    // Process includes from end to start (to preserve indices)
    result := content
    for i := len(matches) - 1; i >= 0; i-- {
        match := matches[i]
        fullMatch := content[match[0]:match[1]]
        pathMatch := strings.TrimSpace(content[match[2]:match[3]])

        // Resolve the include path
        includePath, err := resolveIncludePath(pathMatch, baseDir)
        if err != nil {
            return "", fmt.Errorf("failed to resolve include %q: %w", pathMatch, err)
        }

        // Check for circular includes
        absPath, _ := filepath.Abs(includePath)
        if seen[absPath] {
            cycle := formatCycle(seen, absPath)
            return "", fmt.Errorf("circular include detected: %s", cycle)
        }
        seen[absPath] = true

        // Read the included file
        includeContent, err := os.ReadFile(includePath)
        if err != nil {
            return "", fmt.Errorf("failed to read include file %q: %w", includePath, err)
        }

        // Recursively process includes in the included content
        includeDir := filepath.Dir(includePath)
        processed, err := ProcessIncludes(string(includeContent), includeDir, depth+1, seen)
        if err != nil {
            return "", fmt.Errorf("error processing includes in %q: %w", includePath, err)
        }

        // Remove from seen after processing (allows same file in different branches)
        delete(seen, absPath)

        // Replace the include directive with the processed content
        result = result[:match[0]] + processed + result[match[1]:]
    }

    return result, nil
}

// resolveIncludePath resolves an include path relative to the base directory.
func resolveIncludePath(includePath, baseDir string) (string, error) {
    // Handle absolute paths
    if filepath.IsAbs(includePath) {
        return includePath, nil
    }

    // Add .md extension if not present
    if !strings.HasSuffix(includePath, ".md") {
        includePath = includePath + ".md"
    }

    // Resolve relative to base directory
    resolved := filepath.Join(baseDir, includePath)

    // Check if file exists
    if _, err := os.Stat(resolved); os.IsNotExist(err) {
        return "", fmt.Errorf("file not found: %s (looked in %s)", includePath, baseDir)
    }

    return resolved, nil
}

// formatCycle creates a readable cycle string for error messages.
func formatCycle(seen map[string]bool, cycleStart string) string {
    // Simple representation - in production could track order
    return cycleStart + " (appears twice in include chain)"
}
```

### internal/prompt/loader.go changes

```go
// In LoadPrompt function, after reading the file:
func LoadPrompt(dir, name string) (string, error) {
    filePath := filepath.Join(dir, name+".md")
    
    content, err := os.ReadFile(filePath)
    if err != nil {
        // ... existing error handling ...
    }

    // Process include directives
    processed, err := ProcessIncludes(string(content), dir, 0, nil)
    if err != nil {
        return "", fmt.Errorf("failed to process includes in prompt %q: %w", name, err)
    }

    return processed, nil
}

// Similar changes to LoadPromptFromFile
```

### Example prompts

**swarm/prompts/common/rules.md**:
```markdown
## Project Rules

- Follow the existing code style
- Write tests for new functionality
- Do not modify files outside the scope of the task
- Always run `npm test` before completing
```

**swarm/prompts/common/output.md**:
```markdown
## Output Format

When complete, provide:
1. Summary of changes made
2. Files modified
3. Any concerns or follow-up tasks
```

**swarm/prompts/feature.md**:
```markdown
# Feature Implementation

{{include: common/rules.md}}

## Task

Implement the feature described below...

{{include: common/output.md}}
```

When loaded, this expands to the full combined prompt.

## Use cases

### Shared coding standards

```markdown
# prompts/common/standards.md
- Use TypeScript strict mode
- No `any` types
- All functions must have JSDoc comments

# prompts/add-feature.md
{{include: common/standards.md}}

Add the following feature...
```

### Environment-specific rules

```markdown
# prompts/common/prod-rules.md
- Never delete production data
- All changes must be backward compatible
- Log all database modifications

# prompts/deploy-fix.md
{{include: common/prod-rules.md}}

Fix the following production issue...
```

### Task templates with variable sections

```markdown
# prompts/common/task-header.md
Your Swarm Task ID is {{SWARM_TASK_ID}}.

# prompts/common/task-footer.md
Exit when the task is complete.

# prompts/any-task.md
{{include: common/task-header.md}}

[Specific task content here]

{{include: common/task-footer.md}}
```

### Layered includes

```markdown
# prompts/common/base.md
You are a helpful coding assistant.

# prompts/common/typescript.md
{{include: common/base.md}}
You specialize in TypeScript and React.

# prompts/react-feature.md
{{include: common/typescript.md}}
Implement the following React component...
```

## Edge cases

1. **File not found**: Clear error message with the resolved path that was searched.

2. **Circular includes**: Detected and reported with the cycle path:
   ```
   Error: circular include detected: a.md → b.md → a.md
   ```

3. **Max depth exceeded**: Error after 10 levels of nesting (configurable).

4. **Empty include**: If included file is empty, just inserts nothing (valid use case for conditional content).

5. **Include at start/end of file**: Works correctly, no extra whitespace added.

6. **Multiple includes of same file**: Allowed (same content included multiple times, useful for repeated sections).

7. **Include in inline prompt string**: Not supported for `-s` flag (security consideration - arbitrary file access). Only works with prompt files.

8. **Binary/non-text files**: Error if file appears to be binary.

9. **Whitespace handling**: Preserves original whitespace; included content is inserted exactly as-is.

10. **Path traversal**: Allowed but logs a warning if path goes above prompts directory.

## Validation command

Add a `swarm prompts check` subcommand to validate prompts:

```bash
# Validate all prompts (check includes resolve correctly)
swarm prompts check

# Validate specific prompt
swarm prompts check feature.md

# Show expanded prompt (with includes resolved)
swarm prompts show feature.md --expand
```

Output example:
```
$ swarm prompts check
✓ feature.md (includes: common/rules.md, common/output.md)
✓ bugfix.md (includes: common/rules.md)
✗ broken.md: include not found: common/missing.md
```

## Configuration

Add optional configuration in `.swarm.toml`:

```toml
[prompts]
# Custom include syntax (default: {{include: path}})
include_syntax = "{{include: %s}}"

# Maximum include depth (default: 10)
max_include_depth = 10

# Warn on path traversal above prompts dir (default: true)
warn_path_traversal = true
```

## Acceptance criteria

- `{{include: file.md}}` directive includes file content at that position
- Relative paths resolve from the including file's directory
- Nested includes work up to 10 levels deep
- Circular includes are detected and produce a clear error
- Missing include files produce a clear error with the searched path
- `swarm prompts show <name> --expand` shows the fully expanded prompt
- `swarm prompts check` validates all prompts can be loaded
- Include processing works for both named prompts and prompt files
- Inline prompt strings (`-s` flag) do not process includes (security)
- Whitespace is preserved correctly during include expansion
