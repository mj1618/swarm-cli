package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPath(t *testing.T) {
	if got := DefaultPath(); got != "./swarm/swarm.yaml" {
		t.Errorf("DefaultPath() = %q, want %q", got, "./swarm/swarm.yaml")
	}
}

func TestLoad(t *testing.T) {
	// Create a temp directory for test files
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		content   string
		wantErr   bool
		wantTasks int
	}{
		{
			name: "valid compose file",
			content: `version: "1"
tasks:
  frontend:
    prompt: my-prompt
    model: sonnet-4.5
    iterations: 10
  backend:
    prompt: backend-task
`,
			wantErr:   false,
			wantTasks: 2,
		},
		{
			name: "empty tasks",
			content: `version: "1"
tasks: {}
`,
			wantErr:   false,
			wantTasks: 0,
		},
		{
			name: "with prompt-file",
			content: `version: "1"
tasks:
  custom:
    prompt-file: ./path/to/file.md
    iterations: 5
`,
			wantErr:   false,
			wantTasks: 1,
		},
		{
			name: "with prompt-string",
			content: `version: "1"
tasks:
  inline:
    prompt-string: "Do something cool"
`,
			wantErr:   false,
			wantTasks: 1,
		},
		{
			name:      "invalid yaml",
			content:   `this is not valid yaml: [`,
			wantErr:   true,
			wantTasks: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test file
			path := filepath.Join(tmpDir, "swarm.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			cf, err := Load(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(cf.Tasks) != tt.wantTasks {
				t.Errorf("Load() got %d tasks, want %d", len(cf.Tasks), tt.wantTasks)
			}
		})
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/swarm.yaml")
	if err == nil {
		t.Error("Load() expected error for nonexistent file, got nil")
	}
}

func TestComposeFileValidate(t *testing.T) {
	tests := []struct {
		name    string
		cf      *ComposeFile
		wantErr bool
	}{
		{
			name: "valid with prompt",
			cf: &ComposeFile{
				Version: "1",
				Tasks: map[string]Task{
					"test": {Prompt: "my-prompt"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid with prompt-file",
			cf: &ComposeFile{
				Version: "1",
				Tasks: map[string]Task{
					"test": {PromptFile: "./file.md"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid with prompt-string",
			cf: &ComposeFile{
				Version: "1",
				Tasks: map[string]Task{
					"test": {PromptString: "do something"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid with all optional fields",
			cf: &ComposeFile{
				Version: "1",
				Tasks: map[string]Task{
					"test": {
						Prompt:     "my-prompt",
						Model:      "opus-4.5",
						Iterations: 10,
						Name:       "custom-name",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no tasks",
			cf: &ComposeFile{
				Version: "1",
				Tasks:   map[string]Task{},
			},
			wantErr: true,
		},
		{
			name: "nil tasks",
			cf: &ComposeFile{
				Version: "1",
				Tasks:   nil,
			},
			wantErr: true,
		},
		{
			name: "no prompt source",
			cf: &ComposeFile{
				Version: "1",
				Tasks: map[string]Task{
					"test": {Model: "opus"},
				},
			},
			wantErr: true,
		},
		{
			name: "multiple prompt sources",
			cf: &ComposeFile{
				Version: "1",
				Tasks: map[string]Task{
					"test": {
						Prompt:     "my-prompt",
						PromptFile: "./file.md",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "all three prompt sources",
			cf: &ComposeFile{
				Version: "1",
				Tasks: map[string]Task{
					"test": {
						Prompt:       "my-prompt",
						PromptFile:   "./file.md",
						PromptString: "inline",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "negative iterations",
			cf: &ComposeFile{
				Version: "1",
				Tasks: map[string]Task{
					"test": {
						Prompt:     "my-prompt",
						Iterations: -1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "multiple tasks with one invalid",
			cf: &ComposeFile{
				Version: "1",
				Tasks: map[string]Task{
					"valid":   {Prompt: "my-prompt"},
					"invalid": {Model: "opus"}, // no prompt source
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cf.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskValidate(t *testing.T) {
	tests := []struct {
		name    string
		task    Task
		wantErr bool
	}{
		{
			name:    "valid prompt",
			task:    Task{Prompt: "test"},
			wantErr: false,
		},
		{
			name:    "valid prompt-file",
			task:    Task{PromptFile: "./test.md"},
			wantErr: false,
		},
		{
			name:    "valid prompt-string",
			task:    Task{PromptString: "do something"},
			wantErr: false,
		},
		{
			name:    "no prompt",
			task:    Task{},
			wantErr: true,
		},
		{
			name:    "two prompts",
			task:    Task{Prompt: "test", PromptFile: "./test.md"},
			wantErr: true,
		},
		{
			name:    "negative iterations",
			task:    Task{Prompt: "test", Iterations: -5},
			wantErr: true,
		},
		{
			name:    "zero iterations is valid",
			task:    Task{Prompt: "test", Iterations: 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate("test-task")
			if (err != nil) != tt.wantErr {
				t.Errorf("Task.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTasks(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"frontend": {Prompt: "frontend-prompt"},
			"backend":  {Prompt: "backend-prompt"},
			"tests":    {Prompt: "test-prompt"},
		},
	}

	tests := []struct {
		name      string
		names     []string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "empty names returns all",
			names:     []string{},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "nil names returns all",
			names:     nil,
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "single task",
			names:     []string{"frontend"},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name:      "multiple tasks",
			names:     []string{"frontend", "backend"},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "all tasks by name",
			names:     []string{"frontend", "backend", "tests"},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:      "nonexistent task",
			names:     []string{"nonexistent"},
			wantCount: 0,
			wantErr:   true,
		},
		{
			name:      "mix of valid and invalid",
			names:     []string{"frontend", "nonexistent"},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cf.GetTasks(tt.names)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTasks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantCount {
				t.Errorf("GetTasks() got %d tasks, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestTaskEffectiveName(t *testing.T) {
	tests := []struct {
		name     string
		task     Task
		taskKey  string
		wantName string
	}{
		{
			name:     "uses Name when set",
			task:     Task{Prompt: "test", Name: "custom-name"},
			taskKey:  "task-key",
			wantName: "custom-name",
		},
		{
			name:     "uses taskKey when Name not set",
			task:     Task{Prompt: "test"},
			taskKey:  "task-key",
			wantName: "task-key",
		},
		{
			name:     "empty Name uses taskKey",
			task:     Task{Prompt: "test", Name: ""},
			taskKey:  "my-task",
			wantName: "my-task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.EffectiveName(tt.taskKey); got != tt.wantName {
				t.Errorf("EffectiveName() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

func TestTaskEffectiveIterations(t *testing.T) {
	tests := []struct {
		name       string
		task       Task
		wantIter   int
	}{
		{
			name:     "zero returns 1",
			task:     Task{Prompt: "test", Iterations: 0},
			wantIter: 1,
		},
		{
			name:     "positive value returned",
			task:     Task{Prompt: "test", Iterations: 10},
			wantIter: 10,
		},
		{
			name:     "one returns 1",
			task:     Task{Prompt: "test", Iterations: 1},
			wantIter: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.EffectiveIterations(); got != tt.wantIter {
				t.Errorf("EffectiveIterations() = %d, want %d", got, tt.wantIter)
			}
		})
	}
}

func TestLoadParsesAllFields(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `version: "1"
tasks:
  my-task:
    prompt: my-prompt
    model: opus-4.5-thinking
    iterations: 25
    name: custom-agent-name
`
	path := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cf.Version != "1" {
		t.Errorf("Version = %q, want %q", cf.Version, "1")
	}

	task, ok := cf.Tasks["my-task"]
	if !ok {
		t.Fatal("task 'my-task' not found")
	}

	if task.Prompt != "my-prompt" {
		t.Errorf("Prompt = %q, want %q", task.Prompt, "my-prompt")
	}
	if task.Model != "opus-4.5-thinking" {
		t.Errorf("Model = %q, want %q", task.Model, "opus-4.5-thinking")
	}
	if task.Iterations != 25 {
		t.Errorf("Iterations = %d, want %d", task.Iterations, 25)
	}
	if task.Name != "custom-agent-name" {
		t.Errorf("Name = %q, want %q", task.Name, "custom-agent-name")
	}
}

func TestLoadWithPromptFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `version: "1"
tasks:
  file-task:
    prompt-file: ./custom/path.md
    iterations: 3
`
	path := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	task := cf.Tasks["file-task"]
	if task.PromptFile != "./custom/path.md" {
		t.Errorf("PromptFile = %q, want %q", task.PromptFile, "./custom/path.md")
	}
}

func TestLoadWithPromptString(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `version: "1"
tasks:
  string-task:
    prompt-string: "This is an inline prompt with special chars: !@#$%"
`
	path := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	task := cf.Tasks["string-task"]
	expected := "This is an inline prompt with special chars: !@#$%"
	if task.PromptString != expected {
		t.Errorf("PromptString = %q, want %q", task.PromptString, expected)
	}
}

func TestLoadWithPrefixAndSuffix(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `version: "1"
tasks:
  prefixed-task:
    prompt: my-prompt
    prefix: "Focus on security best practices."
    suffix: "Output only the code, no explanations."
    iterations: 5
`
	path := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	task := cf.Tasks["prefixed-task"]
	if task.Prompt != "my-prompt" {
		t.Errorf("Prompt = %q, want %q", task.Prompt, "my-prompt")
	}
	if task.Prefix != "Focus on security best practices." {
		t.Errorf("Prefix = %q, want %q", task.Prefix, "Focus on security best practices.")
	}
	if task.Suffix != "Output only the code, no explanations." {
		t.Errorf("Suffix = %q, want %q", task.Suffix, "Output only the code, no explanations.")
	}
	if task.Iterations != 5 {
		t.Errorf("Iterations = %d, want %d", task.Iterations, 5)
	}
}

func TestLoadWithPrefixOnly(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `version: "1"
tasks:
  prefix-only:
    prompt: coder
    prefix: "You are analyzing a Go codebase."
`
	path := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	task := cf.Tasks["prefix-only"]
	if task.Prefix != "You are analyzing a Go codebase." {
		t.Errorf("Prefix = %q, want %q", task.Prefix, "You are analyzing a Go codebase.")
	}
	if task.Suffix != "" {
		t.Errorf("Suffix should be empty, got %q", task.Suffix)
	}
}

func TestLoadWithSuffixOnly(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `version: "1"
tasks:
  suffix-only:
    prompt: reviewer
    suffix: "Provide results in JSON format."
`
	path := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	task := cf.Tasks["suffix-only"]
	if task.Prefix != "" {
		t.Errorf("Prefix should be empty, got %q", task.Prefix)
	}
	if task.Suffix != "Provide results in JSON format." {
		t.Errorf("Suffix = %q, want %q", task.Suffix, "Provide results in JSON format.")
	}
}
