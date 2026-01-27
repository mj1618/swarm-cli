package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Integration tests for prompt loading

func TestLoadPromptWithSpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "backticks",
			content: "Use `code` blocks like this: ```go\nfunc main() {}\n```",
		},
		{
			name:    "quotes",
			content: `Say "hello" and 'goodbye'`,
		},
		{
			name:    "xml-like-tags",
			content: "<thinking>This looks like XML</thinking>",
		},
		{
			name:    "special-symbols",
			content: "Math: α + β = γ, Currency: $100 €50 £30",
		},
		{
			name:    "newlines-tabs",
			content: "Line 1\n\tIndented\n\t\tDouble indented\nBack to start",
		},
		{
			name:    "urls",
			content: "Visit https://example.com/path?query=value&other=123",
		},
		{
			name:    "shell-commands",
			content: "Run: `ls -la | grep test && echo $HOME`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := tt.name + ".md"
			err := os.WriteFile(filepath.Join(tempDir, filename), []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result, err := LoadPrompt(tempDir, tt.name)
			if err != nil {
				t.Fatalf("LoadPrompt failed: %v", err)
			}

			// Original content should be preserved
			if !strings.Contains(result, strings.TrimSpace(tt.content)) {
				t.Errorf("Content not preserved for %s", tt.name)
			}
		})
	}
}

func TestLoadPromptLargeFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a large prompt (100KB+)
	var content strings.Builder
	content.WriteString("# Large Prompt Test\n\n")
	for i := 0; i < 1000; i++ {
		content.WriteString("This is line number ")
		content.WriteString(string(rune('0' + i%10)))
		content.WriteString(" of the large prompt file.\n")
		content.WriteString("It contains various instructions and text.\n")
		content.WriteString("\n")
	}

	err := os.WriteFile(filepath.Join(tempDir, "large.md"), []byte(content.String()), 0644)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	result, err := LoadPrompt(tempDir, "large")
	if err != nil {
		t.Fatalf("LoadPrompt failed for large file: %v", err)
	}

	// Should contain the content
	if !strings.HasPrefix(result, "# Large Prompt Test") {
		t.Error("Large prompt should start with content")
	}
}

func TestLoadPromptEmptyFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create an empty file
	err := os.WriteFile(filepath.Join(tempDir, "empty.md"), []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	result, err := LoadPrompt(tempDir, "empty")
	if err != nil {
		t.Fatalf("LoadPrompt should handle empty file: %v", err)
	}

	// Empty file should result in empty string
	if result != "" {
		t.Errorf("Empty prompt should be empty string, got %q", result)
	}
}

func TestLoadPromptWhitespaceOnlyFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a whitespace-only file
	err := os.WriteFile(filepath.Join(tempDir, "whitespace.md"), []byte("   \n\n\t\t\n   "), 0644)
	if err != nil {
		t.Fatalf("Failed to create whitespace file: %v", err)
	}

	result, err := LoadPrompt(tempDir, "whitespace")
	if err != nil {
		t.Fatalf("LoadPrompt should handle whitespace-only file: %v", err)
	}

	// Content should be trimmed to empty string
	if result != "" {
		t.Errorf("Whitespace-only prompt should be empty string, got %q", result)
	}
}

func TestListPromptsWithManyFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create many prompt files
	numFiles := 50
	for i := 0; i < numFiles; i++ {
		filename := filepath.Join(tempDir, "prompt-"+string(rune('a'+i%26))+string(rune('0'+i/26))+".md")
		err := os.WriteFile(filename, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %d: %v", i, err)
		}
	}

	prompts, err := ListPrompts(tempDir)
	if err != nil {
		t.Fatalf("ListPrompts failed: %v", err)
	}

	if len(prompts) != numFiles {
		t.Errorf("Expected %d prompts, got %d", numFiles, len(prompts))
	}
}

func TestLoadPromptFromFileDifferentExtensions(t *testing.T) {
	tempDir := t.TempDir()

	// Create files with different extensions
	files := map[string]string{
		"prompt.txt":  "Text file content",
		"prompt.yaml": "yaml: content",
		"prompt":      "No extension content",
	}

	for name, content := range files {
		path := filepath.Join(tempDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}

		result, err := LoadPromptFromFile(path)
		if err != nil {
			t.Fatalf("LoadPromptFromFile failed for %s: %v", name, err)
		}

		if !strings.Contains(result, content) {
			t.Errorf("Content not preserved for %s", name)
		}
	}
}

func TestWrapPromptPreservesStructure(t *testing.T) {
	content := `# Main Title

## Subsection 1
- Item A
- Item B
- Item C

## Subsection 2
1. First
2. Second
3. Third

### Code Example
` + "```go\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n```"

	result := WrapPromptString(content)

	// All markdown structures should be preserved
	checks := []string{
		"# Main Title",
		"## Subsection 1",
		"- Item A",
		"1. First",
		"### Code Example",
		"func main()",
	}

	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("Structure not preserved: missing %q", check)
		}
	}
}

func TestLoadPromptCaseHandling(t *testing.T) {
	tempDir := t.TempDir()

	// Create a prompt with mixed case filename
	err := os.WriteFile(filepath.Join(tempDir, "MyPrompt.md"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Should work with exact case
	_, err = LoadPrompt(tempDir, "MyPrompt")
	if err != nil {
		t.Errorf("LoadPrompt should work with exact case: %v", err)
	}

	// With extension
	_, err = LoadPrompt(tempDir, "MyPrompt.md")
	if err != nil {
		t.Errorf("LoadPrompt should work with extension: %v", err)
	}
}

func TestLoadPromptFromFileAbsolutePath(t *testing.T) {
	tempDir := t.TempDir()
	absPath := filepath.Join(tempDir, "absolute-test.txt")

	content := "Absolute path test content"
	err := os.WriteFile(absPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	result, err := LoadPromptFromFile(absPath)
	if err != nil {
		t.Fatalf("LoadPromptFromFile failed with absolute path: %v", err)
	}

	if !strings.Contains(result, content) {
		t.Error("Content not loaded correctly from absolute path")
	}
}
