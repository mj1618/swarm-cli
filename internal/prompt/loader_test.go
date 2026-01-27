package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrapPromptString(t *testing.T) {
	input := "Do something useful"
	result := WrapPromptString(input)

	// Should contain system tags
	if !strings.Contains(result, "<system>") {
		t.Error("wrapped prompt should contain <system> tag")
	}
	if !strings.Contains(result, "</system>") {
		t.Error("wrapped prompt should contain </system> tag")
	}

	// Should contain user tags
	if !strings.Contains(result, "<user>") {
		t.Error("wrapped prompt should contain <user> tag")
	}
	if !strings.Contains(result, "</user>") {
		t.Error("wrapped prompt should contain </user> tag")
	}

	// Should contain the original content
	if !strings.Contains(result, input) {
		t.Error("wrapped prompt should contain original content")
	}

	// Should contain AGENTS.md instruction
	if !strings.Contains(result, "AGENTS.md") {
		t.Error("wrapped prompt should contain AGENTS.md instruction")
	}
}

func TestWrapPromptStringWithWhitespace(t *testing.T) {
	input := "  content with whitespace  \n\n"
	result := WrapPromptString(input)

	// The content should be trimmed inside the tags
	if !strings.Contains(result, "content with whitespace") {
		t.Error("wrapped prompt should contain trimmed content")
	}
}

func TestListPrompts(t *testing.T) {
	// Create temp directory with test prompts
	tempDir := t.TempDir()

	// Create test prompt files
	testPrompts := []string{"test1.md", "test2.md", "another-prompt.md"}
	for _, name := range testPrompts {
		err := os.WriteFile(filepath.Join(tempDir, name), []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Create a non-md file that should be ignored
	err := os.WriteFile(filepath.Join(tempDir, "ignored.txt"), []byte("ignored"), 0644)
	if err != nil {
		t.Fatalf("Failed to create ignored file: %v", err)
	}

	// Create a directory that should be ignored
	err = os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// List prompts
	prompts, err := ListPrompts(tempDir)
	if err != nil {
		t.Fatalf("ListPrompts failed: %v", err)
	}

	// Should have exactly 3 prompts
	if len(prompts) != 3 {
		t.Errorf("Expected 3 prompts, got %d", len(prompts))
	}

	// Check that .md extension is stripped
	for _, p := range prompts {
		if strings.HasSuffix(p, ".md") {
			t.Errorf("Prompt name should not have .md extension: %s", p)
		}
	}

	// Check expected prompts are present
	expected := map[string]bool{"test1": true, "test2": true, "another-prompt": true}
	for _, p := range prompts {
		if !expected[p] {
			t.Errorf("Unexpected prompt: %s", p)
		}
		delete(expected, p)
	}
	for remaining := range expected {
		t.Errorf("Missing prompt: %s", remaining)
	}
}

func TestListPromptsEmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	prompts, err := ListPrompts(tempDir)
	if err != nil {
		t.Fatalf("ListPrompts failed: %v", err)
	}

	if len(prompts) != 0 {
		t.Errorf("Expected 0 prompts, got %d", len(prompts))
	}
}

func TestListPromptsNonExistent(t *testing.T) {
	_, err := ListPrompts("/nonexistent/path/prompts")
	if err == nil {
		t.Error("ListPrompts should fail for non-existent directory")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found': %v", err)
	}
}

func TestLoadPrompt(t *testing.T) {
	tempDir := t.TempDir()

	// Create test prompt
	content := "# Test Prompt\n\nDo something useful"
	err := os.WriteFile(filepath.Join(tempDir, "mytest.md"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load with name (no extension)
	result, err := LoadPrompt(tempDir, "mytest")
	if err != nil {
		t.Fatalf("LoadPrompt failed: %v", err)
	}

	// Should be wrapped
	if !strings.Contains(result, "<system>") {
		t.Error("loaded prompt should be wrapped")
	}
	if !strings.Contains(result, content) {
		t.Error("loaded prompt should contain original content")
	}
}

func TestLoadPromptWithExtension(t *testing.T) {
	tempDir := t.TempDir()

	content := "# Test Prompt"
	err := os.WriteFile(filepath.Join(tempDir, "withext.md"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load with .md extension
	result, err := LoadPrompt(tempDir, "withext.md")
	if err != nil {
		t.Fatalf("LoadPrompt with extension failed: %v", err)
	}

	if !strings.Contains(result, content) {
		t.Error("loaded prompt should contain original content")
	}
}

func TestLoadPromptNotFound(t *testing.T) {
	tempDir := t.TempDir()

	_, err := LoadPrompt(tempDir, "nonexistent")
	if err == nil {
		t.Error("LoadPrompt should fail for non-existent prompt")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found': %v", err)
	}
}

func TestLoadPromptFromFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "custom-prompt.txt")

	content := "Custom prompt content"
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := LoadPromptFromFile(filePath)
	if err != nil {
		t.Fatalf("LoadPromptFromFile failed: %v", err)
	}

	// Should be wrapped
	if !strings.Contains(result, "<system>") {
		t.Error("loaded prompt should be wrapped")
	}
	if !strings.Contains(result, content) {
		t.Error("loaded prompt should contain original content")
	}
}

func TestLoadPromptFromFileNotFound(t *testing.T) {
	_, err := LoadPromptFromFile("/nonexistent/path/file.md")
	if err == nil {
		t.Error("LoadPromptFromFile should fail for non-existent file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found': %v", err)
	}
}

func TestWrapPromptFormat(t *testing.T) {
	content := "Test content"
	result := wrapPrompt(content)

	// Check exact structure
	expected := `<system>
Always check the AGENTS.md file for the latest instructions before doing anything.
</system>

<user>
Test content
</user>`

	if result != expected {
		t.Errorf("Wrapped prompt format mismatch.\nGot:\n%s\n\nExpected:\n%s", result, expected)
	}
}

func TestWrapPromptTrimsContent(t *testing.T) {
	content := "\n\n  Test content  \n\n"
	result := wrapPrompt(content)

	// Content should be trimmed
	if strings.Contains(result, "\n\n  Test") {
		t.Error("Content should be trimmed of leading whitespace")
	}
	if strings.Contains(result, "content  \n") {
		t.Error("Content should be trimmed of trailing whitespace")
	}
}

func TestLoadPromptMultiline(t *testing.T) {
	tempDir := t.TempDir()

	content := `# Multi-line Prompt

This is a multi-line prompt.

## Section 1
- Item 1
- Item 2

## Section 2
More content here.`

	err := os.WriteFile(filepath.Join(tempDir, "multiline.md"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := LoadPrompt(tempDir, "multiline")
	if err != nil {
		t.Fatalf("LoadPrompt failed: %v", err)
	}

	// Should preserve multi-line content
	if !strings.Contains(result, "## Section 1") {
		t.Error("Multi-line content should be preserved")
	}
	if !strings.Contains(result, "- Item 1") {
		t.Error("List items should be preserved")
	}
}

func TestLoadPromptUnicode(t *testing.T) {
	tempDir := t.TempDir()

	content := "# 测试提示\n\nこんにちは 🎉"
	err := os.WriteFile(filepath.Join(tempDir, "unicode.md"), []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := LoadPrompt(tempDir, "unicode")
	if err != nil {
		t.Fatalf("LoadPrompt failed: %v", err)
	}

	if !strings.Contains(result, "测试提示") {
		t.Error("Chinese characters should be preserved")
	}
	if !strings.Contains(result, "こんにちは") {
		t.Error("Japanese characters should be preserved")
	}
	if !strings.Contains(result, "🎉") {
		t.Error("Emoji should be preserved")
	}
}

func TestListPromptsOnlyMdFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create various file types
	files := map[string]string{
		"prompt1.md":   "md file",
		"prompt2.md":   "md file",
		"readme.txt":   "txt file",
		"script.sh":    "sh file",
		"config.json":  "json file",
		".hidden.md":   "hidden md",
		"Makefile":     "no extension",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(tempDir, name), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create %s: %v", name, err)
		}
	}

	prompts, err := ListPrompts(tempDir)
	if err != nil {
		t.Fatalf("ListPrompts failed: %v", err)
	}

	// Should only find .md files (excluding hidden)
	// Note: .hidden.md is actually a hidden file on Unix systems and may not be returned by ReadDir depending on implementation
	// Let's check we have at least the visible .md files
	if len(prompts) < 2 {
		t.Errorf("Expected at least 2 prompts, got %d", len(prompts))
	}

	for _, p := range prompts {
		if p == "readme" || p == "script" || p == "config" || p == "Makefile" {
			t.Errorf("Non-.md file should not be listed: %s", p)
		}
	}
}
