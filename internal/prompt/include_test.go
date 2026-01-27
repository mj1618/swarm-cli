package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessIncludes_SimpleInclude(t *testing.T) {
	tempDir := t.TempDir()

	// Create included file
	err := os.WriteFile(filepath.Join(tempDir, "rules.md"), []byte("## Rules\n- Rule 1\n- Rule 2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create included file: %v", err)
	}

	// Create main prompt with include
	content := `# My Prompt

{{include: rules.md}}

## Task
Do something useful.`

	result, err := ProcessIncludes(content, tempDir)
	if err != nil {
		t.Fatalf("ProcessIncludes failed: %v", err)
	}

	// Should contain the included content
	if !strings.Contains(result, "## Rules") {
		t.Error("Result should contain included rules header")
	}
	if !strings.Contains(result, "- Rule 1") {
		t.Error("Result should contain included rule 1")
	}

	// Should contain the original content
	if !strings.Contains(result, "# My Prompt") {
		t.Error("Result should contain original header")
	}
	if !strings.Contains(result, "Do something useful") {
		t.Error("Result should contain original task")
	}

	// Should not contain the include directive
	if strings.Contains(result, "{{include:") {
		t.Error("Result should not contain include directive")
	}
}

func TestProcessIncludes_SubdirectoryInclude(t *testing.T) {
	tempDir := t.TempDir()

	// Create subdirectory and included file
	commonDir := filepath.Join(tempDir, "common")
	err := os.MkdirAll(commonDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	err = os.WriteFile(filepath.Join(commonDir, "header.md"), []byte("# Common Header"), 0644)
	if err != nil {
		t.Fatalf("Failed to create included file: %v", err)
	}

	content := `{{include: common/header.md}}

## Task
Specific task content.`

	result, err := ProcessIncludes(content, tempDir)
	if err != nil {
		t.Fatalf("ProcessIncludes failed: %v", err)
	}

	if !strings.Contains(result, "# Common Header") {
		t.Error("Result should contain included header from subdirectory")
	}
}

func TestProcessIncludes_NestedIncludes(t *testing.T) {
	tempDir := t.TempDir()

	// Create base.md
	err := os.WriteFile(filepath.Join(tempDir, "base.md"), []byte("Base content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create base.md: %v", err)
	}

	// Create intermediate.md that includes base.md
	err = os.WriteFile(filepath.Join(tempDir, "intermediate.md"), []byte("Intermediate\n\n{{include: base.md}}\n\nEnd intermediate"), 0644)
	if err != nil {
		t.Fatalf("Failed to create intermediate.md: %v", err)
	}

	// Main content includes intermediate.md
	content := "Top level\n\n{{include: intermediate.md}}\n\nEnd top"

	result, err := ProcessIncludes(content, tempDir)
	if err != nil {
		t.Fatalf("ProcessIncludes failed: %v", err)
	}

	// Should contain all levels
	if !strings.Contains(result, "Top level") {
		t.Error("Result should contain top level content")
	}
	if !strings.Contains(result, "Intermediate") {
		t.Error("Result should contain intermediate content")
	}
	if !strings.Contains(result, "Base content") {
		t.Error("Result should contain base content")
	}
}

func TestProcessIncludes_CircularInclude(t *testing.T) {
	tempDir := t.TempDir()

	// Create a.md that includes b.md
	err := os.WriteFile(filepath.Join(tempDir, "a.md"), []byte("A content\n{{include: b.md}}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create a.md: %v", err)
	}

	// Create b.md that includes a.md (circular)
	err = os.WriteFile(filepath.Join(tempDir, "b.md"), []byte("B content\n{{include: a.md}}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create b.md: %v", err)
	}

	content := "{{include: a.md}}"

	_, err = ProcessIncludes(content, tempDir)
	if err == nil {
		t.Fatal("ProcessIncludes should fail with circular include")
	}

	if !strings.Contains(err.Error(), "circular include") {
		t.Errorf("Error should mention circular include: %v", err)
	}
}

func TestProcessIncludes_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()

	content := "{{include: nonexistent.md}}"

	_, err := ProcessIncludes(content, tempDir)
	if err == nil {
		t.Fatal("ProcessIncludes should fail with nonexistent file")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention file not found: %v", err)
	}
}

func TestProcessIncludes_MaxDepthExceeded(t *testing.T) {
	tempDir := t.TempDir()

	// Create a chain of includes that exceeds max depth
	for i := 0; i <= maxIncludeDepth+1; i++ {
		var content string
		if i < maxIncludeDepth+1 {
			content = "{{include: level" + string(rune('a'+i+1)) + ".md}}"
		} else {
			content = "Final content"
		}
		filename := "level" + string(rune('a'+i)) + ".md"
		err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	content := "{{include: levela.md}}"

	_, err := ProcessIncludes(content, tempDir)
	if err == nil {
		t.Fatal("ProcessIncludes should fail when max depth exceeded")
	}

	if !strings.Contains(err.Error(), "maximum include depth") {
		t.Errorf("Error should mention maximum include depth: %v", err)
	}
}

func TestProcessIncludes_NoIncludes(t *testing.T) {
	tempDir := t.TempDir()

	content := `# Simple Prompt

No includes here, just plain content.`

	result, err := ProcessIncludes(content, tempDir)
	if err != nil {
		t.Fatalf("ProcessIncludes failed: %v", err)
	}

	if result != content {
		t.Errorf("Content without includes should be unchanged.\nGot: %s\nExpected: %s", result, content)
	}
}

func TestProcessIncludes_MultipleIncludes(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple included files
	err := os.WriteFile(filepath.Join(tempDir, "header.md"), []byte("# Header"), 0644)
	if err != nil {
		t.Fatalf("Failed to create header.md: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "rules.md"), []byte("- Rule 1\n- Rule 2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create rules.md: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "footer.md"), []byte("## End"), 0644)
	if err != nil {
		t.Fatalf("Failed to create footer.md: %v", err)
	}

	content := `{{include: header.md}}

{{include: rules.md}}

## Task
Do something.

{{include: footer.md}}`

	result, err := ProcessIncludes(content, tempDir)
	if err != nil {
		t.Fatalf("ProcessIncludes failed: %v", err)
	}

	if !strings.Contains(result, "# Header") {
		t.Error("Result should contain header")
	}
	if !strings.Contains(result, "- Rule 1") {
		t.Error("Result should contain rules")
	}
	if !strings.Contains(result, "## End") {
		t.Error("Result should contain footer")
	}
	if !strings.Contains(result, "## Task") {
		t.Error("Result should contain task section")
	}
}

func TestProcessIncludes_AutoExtension(t *testing.T) {
	tempDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tempDir, "rules.md"), []byte("Rules content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create rules.md: %v", err)
	}

	// Include without .md extension
	content := "{{include: rules}}"

	result, err := ProcessIncludes(content, tempDir)
	if err != nil {
		t.Fatalf("ProcessIncludes failed: %v", err)
	}

	if !strings.Contains(result, "Rules content") {
		t.Error("Result should contain included content (auto .md extension)")
	}
}

func TestProcessIncludes_WhitespaceInPath(t *testing.T) {
	tempDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tempDir, "file.md"), []byte("File content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file.md: %v", err)
	}

	// Include with extra whitespace
	content := "{{include:   file.md   }}"

	result, err := ProcessIncludes(content, tempDir)
	if err != nil {
		t.Fatalf("ProcessIncludes failed: %v", err)
	}

	if !strings.Contains(result, "File content") {
		t.Error("Result should handle whitespace in include path")
	}
}

func TestProcessIncludes_EmptyIncludedFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create empty file
	err := os.WriteFile(filepath.Join(tempDir, "empty.md"), []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty.md: %v", err)
	}

	content := "Before\n{{include: empty.md}}\nAfter"

	result, err := ProcessIncludes(content, tempDir)
	if err != nil {
		t.Fatalf("ProcessIncludes failed: %v", err)
	}

	if !strings.Contains(result, "Before") {
		t.Error("Result should contain content before include")
	}
	if !strings.Contains(result, "After") {
		t.Error("Result should contain content after include")
	}
}

func TestProcessIncludes_PreservesWhitespace(t *testing.T) {
	tempDir := t.TempDir()

	includedContent := "  Indented content\n\n  More indented"
	err := os.WriteFile(filepath.Join(tempDir, "indented.md"), []byte(includedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create indented.md: %v", err)
	}

	content := "{{include: indented.md}}"

	result, err := ProcessIncludes(content, tempDir)
	if err != nil {
		t.Fatalf("ProcessIncludes failed: %v", err)
	}

	if result != includedContent {
		t.Errorf("ProcessIncludes should preserve whitespace.\nGot: %q\nExpected: %q", result, includedContent)
	}
}

func TestProcessIncludes_SameFileTwiceInDifferentBranches(t *testing.T) {
	tempDir := t.TempDir()

	// Create a shared file
	err := os.WriteFile(filepath.Join(tempDir, "shared.md"), []byte("Shared content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create shared.md: %v", err)
	}

	// Create two files that both include shared.md
	err = os.WriteFile(filepath.Join(tempDir, "branch1.md"), []byte("Branch 1\n{{include: shared.md}}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create branch1.md: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "branch2.md"), []byte("Branch 2\n{{include: shared.md}}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create branch2.md: %v", err)
	}

	// Include both branches (same file included multiple times is OK)
	content := "{{include: branch1.md}}\n\n{{include: branch2.md}}"

	result, err := ProcessIncludes(content, tempDir)
	if err != nil {
		t.Fatalf("ProcessIncludes failed: %v", err)
	}

	// Should contain shared content twice
	if strings.Count(result, "Shared content") != 2 {
		t.Error("Result should contain 'Shared content' twice (from both branches)")
	}
}

func TestProcessIncludes_RelativePathFromIncludedFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create directory structure
	// tempDir/
	//   prompts/
	//     main.md (includes common/base.md)
	//     common/
	//       base.md (includes ../shared.md)
	//   shared.md

	promptsDir := filepath.Join(tempDir, "prompts")
	commonDir := filepath.Join(promptsDir, "common")
	err := os.MkdirAll(commonDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory structure: %v", err)
	}

	err = os.WriteFile(filepath.Join(tempDir, "shared.md"), []byte("Shared content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create shared.md: %v", err)
	}

	err = os.WriteFile(filepath.Join(commonDir, "base.md"), []byte("Base\n{{include: ../../shared.md}}"), 0644)
	if err != nil {
		t.Fatalf("Failed to create base.md: %v", err)
	}

	content := "Main\n{{include: common/base.md}}"

	result, err := ProcessIncludes(content, promptsDir)
	if err != nil {
		t.Fatalf("ProcessIncludes failed: %v", err)
	}

	if !strings.Contains(result, "Main") {
		t.Error("Result should contain main content")
	}
	if !strings.Contains(result, "Base") {
		t.Error("Result should contain base content")
	}
	if !strings.Contains(result, "Shared content") {
		t.Error("Result should contain shared content (resolved relative to included file)")
	}
}

func TestValidateIncludes_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Create included files
	commonDir := filepath.Join(tempDir, "common")
	err := os.MkdirAll(commonDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	err = os.WriteFile(filepath.Join(commonDir, "rules.md"), []byte("Rules"), 0644)
	if err != nil {
		t.Fatalf("Failed to create rules.md: %v", err)
	}

	err = os.WriteFile(filepath.Join(commonDir, "footer.md"), []byte("Footer"), 0644)
	if err != nil {
		t.Fatalf("Failed to create footer.md: %v", err)
	}

	content := "{{include: common/rules.md}}\n\nTask\n\n{{include: common/footer.md}}"

	includes, err := ValidateIncludes(content, tempDir)
	if err != nil {
		t.Fatalf("ValidateIncludes failed: %v", err)
	}

	if len(includes) != 2 {
		t.Errorf("Expected 2 includes, got %d", len(includes))
	}
}

func TestValidateIncludes_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()

	content := "{{include: missing.md}}"

	_, err := ValidateIncludes(content, tempDir)
	if err == nil {
		t.Fatal("ValidateIncludes should fail for missing file")
	}
}

func TestExtractIncludes(t *testing.T) {
	content := `# Header
{{include: common/rules.md}}

Task content

{{include: footer}}

{{include:   spaced.md   }}`

	includes := ExtractIncludes(content)

	if len(includes) != 3 {
		t.Errorf("Expected 3 includes, got %d: %v", len(includes), includes)
	}

	expected := []string{"common/rules.md", "footer", "spaced.md"}
	for i, exp := range expected {
		if i >= len(includes) {
			break
		}
		if includes[i] != exp {
			t.Errorf("Include %d: expected %q, got %q", i, exp, includes[i])
		}
	}
}

func TestExtractIncludes_NoIncludes(t *testing.T) {
	content := "No includes here"

	includes := ExtractIncludes(content)

	if len(includes) != 0 {
		t.Errorf("Expected 0 includes, got %d", len(includes))
	}
}

func TestResolveIncludePath_AbsolutePath(t *testing.T) {
	tempDir := t.TempDir()
	absPath := filepath.Join(tempDir, "absolute.md")

	err := os.WriteFile(absPath, []byte("Absolute content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	resolved, err := resolveIncludePath(absPath, "/some/other/dir")
	if err != nil {
		t.Fatalf("resolveIncludePath failed: %v", err)
	}

	if resolved != absPath {
		t.Errorf("Expected absolute path %q, got %q", absPath, resolved)
	}
}
