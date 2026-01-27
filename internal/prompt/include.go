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
func ProcessIncludes(content string, baseDir string) (string, error) {
	return processIncludesInternal(content, baseDir, 0, make(map[string]bool))
}

// processIncludesInternal is the recursive implementation of ProcessIncludes.
func processIncludesInternal(content string, baseDir string, depth int, seen map[string]bool) (string, error) {
	if depth > maxIncludeDepth {
		return "", fmt.Errorf("maximum include depth (%d) exceeded - check for circular includes", maxIncludeDepth)
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
		pathMatch := strings.TrimSpace(content[match[2]:match[3]])

		// Resolve the include path
		includePath, err := resolveIncludePath(pathMatch, baseDir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve include %q: %w", pathMatch, err)
		}

		// Check for circular includes
		absPath, err := filepath.Abs(includePath)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path for %q: %w", includePath, err)
		}

		if seen[absPath] {
			return "", fmt.Errorf("circular include detected: %s (appears twice in include chain)", absPath)
		}
		seen[absPath] = true

		// Read the included file
		includeContent, err := os.ReadFile(includePath)
		if err != nil {
			return "", fmt.Errorf("failed to read include file %q: %w", includePath, err)
		}

		// Check if content appears to be binary
		if isBinaryContent(includeContent) {
			return "", fmt.Errorf("include file %q appears to be binary", includePath)
		}

		// Recursively process includes in the included content
		includeDir := filepath.Dir(includePath)
		processed, err := processIncludesInternal(string(includeContent), includeDir, depth+1, seen)
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
	// Trim any whitespace
	includePath = strings.TrimSpace(includePath)

	// Handle absolute paths
	if filepath.IsAbs(includePath) {
		if _, err := os.Stat(includePath); os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", includePath)
		}
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

// isBinaryContent checks if content appears to be binary (contains null bytes).
func isBinaryContent(content []byte) bool {
	// Check first 8KB for null bytes
	checkLen := len(content)
	if checkLen > 8192 {
		checkLen = 8192
	}
	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

// ValidateIncludes validates all include directives in a prompt can be resolved.
// Returns a list of included files if successful, or an error if any include fails.
func ValidateIncludes(content string, baseDir string) ([]string, error) {
	return validateIncludesInternal(content, baseDir, 0, make(map[string]bool))
}

// validateIncludesInternal is the recursive implementation of ValidateIncludes.
func validateIncludesInternal(content string, baseDir string, depth int, seen map[string]bool) ([]string, error) {
	if depth > maxIncludeDepth {
		return nil, fmt.Errorf("maximum include depth (%d) exceeded - check for circular includes", maxIncludeDepth)
	}

	// Find all include directives
	matches := includeRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	var included []string

	for _, match := range matches {
		pathMatch := strings.TrimSpace(match[1])

		// Resolve the include path
		includePath, err := resolveIncludePath(pathMatch, baseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve include %q: %w", pathMatch, err)
		}

		// Check for circular includes
		absPath, err := filepath.Abs(includePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %q: %w", includePath, err)
		}

		if seen[absPath] {
			return nil, fmt.Errorf("circular include detected: %s (appears twice in include chain)", absPath)
		}
		seen[absPath] = true

		// Add to included list (relative path for display)
		relPath, err := filepath.Rel(baseDir, includePath)
		if err != nil {
			relPath = includePath
		}
		included = append(included, relPath)

		// Read and validate nested includes
		includeContent, err := os.ReadFile(includePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read include file %q: %w", includePath, err)
		}

		// Check if content appears to be binary
		if isBinaryContent(includeContent) {
			return nil, fmt.Errorf("include file %q appears to be binary", includePath)
		}

		// Recursively validate includes in the included content
		includeDir := filepath.Dir(includePath)
		nestedIncludes, err := validateIncludesInternal(string(includeContent), includeDir, depth+1, seen)
		if err != nil {
			return nil, fmt.Errorf("error validating includes in %q: %w", includePath, err)
		}
		included = append(included, nestedIncludes...)

		// Remove from seen after processing
		delete(seen, absPath)
	}

	return included, nil
}

// ExtractIncludes extracts all include paths from content without resolving them.
// This is useful for quick inspection without file system access.
func ExtractIncludes(content string) []string {
	matches := includeRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	var includes []string
	for _, match := range matches {
		includes = append(includes, strings.TrimSpace(match[1]))
	}
	return includes
}
