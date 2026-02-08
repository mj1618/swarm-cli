package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var outputRegex = regexp.MustCompile(`\{\{output:\s*([^}]+)\}\}`)

// ProcessOutputDirectives replaces {{output:task_name}} directives with the
// contents of the corresponding task output file from the pipeline output directory.
// If outputDir is empty (not running in a pipeline), missing-output placeholders are used.
func ProcessOutputDirectives(content, outputDir string) (string, error) {
	matches := outputRegex.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	// Process from end to start to preserve indices
	result := content
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		taskName := strings.TrimSpace(content[match[2]:match[3]])

		var replacement string
		if outputDir == "" {
			replacement = fmt.Sprintf("(No output available from task %q — not running in a pipeline)", taskName)
		} else {
			outputPath := filepath.Join(outputDir, taskName+".txt")
			data, err := os.ReadFile(outputPath)
			if err != nil {
				if os.IsNotExist(err) {
					replacement = fmt.Sprintf("(No output available from task %q)", taskName)
				} else {
					return "", fmt.Errorf("failed to read output for task %q: %w", taskName, err)
				}
			} else {
				replacement = fmt.Sprintf("--- Output from task %q ---\n%s\n--- End output from task %q ---", taskName, strings.TrimRight(string(data), "\n"), taskName)
			}
		}

		result = result[:match[0]] + replacement + result[match[1]:]
	}

	return result, nil
}
