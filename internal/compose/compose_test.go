package compose

import (
	"os"
	"path/filepath"
	"strings"
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
		name     string
		task     Task
		wantIter int
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

// Tests for dependencies

func TestLoadWithDependsOn_SimpleForm(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `version: "1"
tasks:
  coder:
    prompt: coder
  tester:
    prompt: tester
    depends_on:
      - coder
`
	path := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	task := cf.Tasks["tester"]
	if len(task.DependsOn) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(task.DependsOn))
	}
	if task.DependsOn[0].Task != "coder" {
		t.Errorf("dependency task = %q, want %q", task.DependsOn[0].Task, "coder")
	}
	if task.DependsOn[0].EffectiveCondition() != ConditionAny {
		t.Errorf("dependency condition = %q, want %q", task.DependsOn[0].Condition, ConditionAny)
	}
}

func TestLoadWithDependsOn_FullForm(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `version: "1"
tasks:
  coder:
    prompt: coder
  fixer:
    prompt: fixer
    depends_on:
      - task: coder
        condition: failure
`
	path := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	task := cf.Tasks["fixer"]
	if len(task.DependsOn) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(task.DependsOn))
	}
	if task.DependsOn[0].Task != "coder" {
		t.Errorf("dependency task = %q, want %q", task.DependsOn[0].Task, "coder")
	}
	if task.DependsOn[0].Condition != ConditionFailure {
		t.Errorf("dependency condition = %q, want %q", task.DependsOn[0].Condition, ConditionFailure)
	}
}

func TestLoadWithDependsOn_MixedForms(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `version: "1"
tasks:
  a:
    prompt: a
  b:
    prompt: b
  c:
    prompt: c
    depends_on:
      - a
      - task: b
        condition: any
`
	path := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	task := cf.Tasks["c"]
	if len(task.DependsOn) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(task.DependsOn))
	}

	// First dependency is simple form
	if task.DependsOn[0].Task != "a" {
		t.Errorf("first dependency task = %q, want %q", task.DependsOn[0].Task, "a")
	}
	if task.DependsOn[0].EffectiveCondition() != ConditionAny {
		t.Errorf("first dependency condition = %q, want %q", task.DependsOn[0].Condition, ConditionAny)
	}

	// Second dependency is full form
	if task.DependsOn[1].Task != "b" {
		t.Errorf("second dependency task = %q, want %q", task.DependsOn[1].Task, "b")
	}
	if task.DependsOn[1].Condition != ConditionAny {
		t.Errorf("second dependency condition = %q, want %q", task.DependsOn[1].Condition, ConditionAny)
	}
}

func TestValidate_DependsOnUnknownTask(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"a": {Prompt: "a"},
			"b": {Prompt: "b", DependsOn: []Dependency{{Task: "nonexistent"}}},
		},
	}

	err := cf.Validate()
	if err == nil {
		t.Error("expected error for unknown dependency task")
	}
}

func TestValidate_DependsOnSelf(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"a": {Prompt: "a", DependsOn: []Dependency{{Task: "a"}}},
		},
	}

	err := cf.Validate()
	if err == nil {
		t.Error("expected error for self-dependency")
	}
}

func TestValidate_InvalidCondition(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"a": {Prompt: "a"},
			"b": {Prompt: "b", DependsOn: []Dependency{{Task: "a", Condition: "invalid"}}},
		},
	}

	err := cf.Validate()
	if err == nil {
		t.Error("expected error for invalid condition")
	}
}

func TestValidate_EmptyDependencyTask(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"a": {Prompt: "a", DependsOn: []Dependency{{Task: ""}}},
		},
	}

	err := cf.Validate()
	if err == nil {
		t.Error("expected error for empty dependency task name")
	}
}

// Tests for pipelines

func TestLoadWithPipelines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `version: "1"
tasks:
  coder:
    prompt: coder
  tester:
    prompt: tester
    depends_on: [coder]

pipelines:
  development:
    iterations: 10
    tasks: [coder, tester]
`
	path := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cf.HasPipelines() {
		t.Error("expected HasPipelines() = true")
	}

	pipeline, err := cf.GetPipeline("development")
	if err != nil {
		t.Fatalf("GetPipeline() error = %v", err)
	}

	if pipeline.Iterations != 10 {
		t.Errorf("pipeline iterations = %d, want %d", pipeline.Iterations, 10)
	}
	if len(pipeline.Tasks) != 2 {
		t.Errorf("pipeline tasks count = %d, want %d", len(pipeline.Tasks), 2)
	}
}

func TestPipeline_EffectiveIterations(t *testing.T) {
	tests := []struct {
		iterations int
		want       int
	}{
		{0, 1},
		{-1, 1},
		{1, 1},
		{10, 10},
	}

	for _, tt := range tests {
		p := Pipeline{Iterations: tt.iterations}
		if got := p.EffectiveIterations(); got != tt.want {
			t.Errorf("EffectiveIterations() for %d = %d, want %d", tt.iterations, got, tt.want)
		}
	}
}

func TestPipeline_GetPipelineTasks(t *testing.T) {
	allTasks := map[string]Task{
		"a": {Prompt: "a"},
		"b": {Prompt: "b"},
		"c": {Prompt: "c"},
	}

	// Pipeline with specific tasks
	p1 := Pipeline{Tasks: []string{"a", "b"}}
	tasks1 := p1.GetPipelineTasks(allTasks)
	if len(tasks1) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks1))
	}

	// Pipeline without tasks (should return all)
	p2 := Pipeline{}
	tasks2 := p2.GetPipelineTasks(allTasks)
	if len(tasks2) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks2))
	}
}

func TestValidate_PipelineUnknownTask(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"a": {Prompt: "a"},
		},
		Pipelines: map[string]Pipeline{
			"test": {Iterations: 1, Tasks: []string{"a", "nonexistent"}},
		},
	}

	err := cf.Validate()
	if err == nil {
		t.Error("expected error for unknown pipeline task")
	}
}

func TestValidate_PipelineNegativeIterations(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"a": {Prompt: "a"},
		},
		Pipelines: map[string]Pipeline{
			"test": {Iterations: -5, Tasks: []string{"a"}},
		},
	}

	err := cf.Validate()
	if err == nil {
		t.Error("expected error for negative pipeline iterations")
	}
}

func TestGetPipeline_NotFound(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"a": {Prompt: "a"},
		},
	}

	_, err := cf.GetPipeline("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent pipeline")
	}
}

func TestHasDependencies(t *testing.T) {
	// No dependencies
	cf1 := &ComposeFile{
		Tasks: map[string]Task{
			"a": {Prompt: "a"},
			"b": {Prompt: "b"},
		},
	}
	if cf1.HasDependencies() {
		t.Error("expected HasDependencies() = false for tasks without deps")
	}

	// With dependencies
	cf2 := &ComposeFile{
		Tasks: map[string]Task{
			"a": {Prompt: "a"},
			"b": {Prompt: "b", DependsOn: []Dependency{{Task: "a"}}},
		},
	}
	if !cf2.HasDependencies() {
		t.Error("expected HasDependencies() = true for tasks with deps")
	}
}

func TestDependency_EffectiveCondition(t *testing.T) {
	tests := []struct {
		condition string
		want      string
	}{
		{"", ConditionAny},
		{ConditionSuccess, ConditionSuccess},
		{ConditionFailure, ConditionFailure},
		{ConditionAny, ConditionAny},
		{ConditionAlways, ConditionAlways},
	}

	for _, tt := range tests {
		d := Dependency{Task: "test", Condition: tt.condition}
		if got := d.EffectiveCondition(); got != tt.want {
			t.Errorf("EffectiveCondition() for %q = %q, want %q", tt.condition, got, tt.want)
		}
	}
}

func TestWarnings_DependsOnWithoutPipeline(t *testing.T) {
	tests := []struct {
		name         string
		cf           *ComposeFile
		wantWarnings int
	}{
		{
			name: "depends_on without pipeline emits warning",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"a": {Prompt: "a"},
					"b": {Prompt: "b", DependsOn: []Dependency{{Task: "a"}}},
				},
			},
			wantWarnings: 1,
		},
		{
			name: "depends_on with pipeline emits no warning",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"a": {Prompt: "a"},
					"b": {Prompt: "b", DependsOn: []Dependency{{Task: "a"}}},
				},
				Pipelines: map[string]Pipeline{
					"main": {Tasks: []string{"a", "b"}},
				},
			},
			wantWarnings: 0,
		},
		{
			name: "no depends_on and no pipeline emits no warning",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"a": {Prompt: "a"},
					"b": {Prompt: "b"},
				},
			},
			wantWarnings: 0,
		},
		{
			name: "no depends_on with pipeline emits no warning",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"a": {Prompt: "a"},
				},
				Pipelines: map[string]Pipeline{
					"main": {Tasks: []string{"a"}},
				},
			},
			wantWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := tt.cf.Warnings()
			if len(warnings) != tt.wantWarnings {
				t.Errorf("Warnings() returned %d warnings, want %d: %v", len(warnings), tt.wantWarnings, warnings)
			}
			if tt.wantWarnings > 0 {
				// Verify warning mentions the task name and suggests pipelines
				w := warnings[0]
				if !strings.Contains(w, "depends_on") || !strings.Contains(w, "pipeline") {
					t.Errorf("warning should mention depends_on and pipeline, got: %s", w)
				}
			}
		})
	}
}

func TestGetStandaloneTasks(t *testing.T) {
	tests := []struct {
		name     string
		cf       *ComposeFile
		expected []string
	}{
		{
			name: "no pipelines, no dependencies - all standalone",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"a": {Prompt: "a"},
					"b": {Prompt: "b"},
				},
			},
			expected: []string{"a", "b"},
		},
		{
			name: "task in pipeline - not standalone",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"a": {Prompt: "a"},
					"b": {Prompt: "b"},
				},
				Pipelines: map[string]Pipeline{
					"main": {Tasks: []string{"a"}},
				},
			},
			expected: []string{"b"},
		},
		{
			name: "task with dependency - not standalone",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"a": {Prompt: "a"},
					"b": {Prompt: "b", DependsOn: []Dependency{{Task: "a"}}},
				},
			},
			expected: []string{}, // 'a' is depended upon, 'b' has dependencies
		},
		{
			name: "mix of pipeline, dependency, and standalone",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"pipeline-task": {Prompt: "pt"},
					"dep-parent":    {Prompt: "dp"},
					"dep-child":     {Prompt: "dc", DependsOn: []Dependency{{Task: "dep-parent"}}},
					"standalone":    {Prompt: "s"},
				},
				Pipelines: map[string]Pipeline{
					"main": {Tasks: []string{"pipeline-task"}},
				},
			},
			expected: []string{"standalone"},
		},
		{
			name: "all tasks in pipeline - none standalone",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"a": {Prompt: "a"},
					"b": {Prompt: "b"},
				},
				Pipelines: map[string]Pipeline{
					"main": {Tasks: []string{"a", "b"}},
				},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cf.GetStandaloneTasks()

			// Check count
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d standalone tasks, got %d", len(tt.expected), len(result))
				return
			}

			// Check each expected task is present
			for _, name := range tt.expected {
				if _, ok := result[name]; !ok {
					t.Errorf("expected task %q to be standalone", name)
				}
			}
		})
	}
}

// Tests for parallelism

func TestTaskEffectiveParallelism(t *testing.T) {
	tests := []struct {
		name        string
		parallelism int
		want        int
	}{
		{"zero returns 1", 0, 1},
		{"negative returns 1", -1, 1},
		{"one returns 1", 1, 1},
		{"positive value returned", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{Prompt: "test", Parallelism: tt.parallelism}
			if got := task.EffectiveParallelism(); got != tt.want {
				t.Errorf("EffectiveParallelism() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPipelineEffectiveParallelism(t *testing.T) {
	tests := []struct {
		name        string
		parallelism int
		want        int
	}{
		{"zero returns 1", 0, 1},
		{"negative returns 1", -1, 1},
		{"one returns 1", 1, 1},
		{"positive value returned", 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Pipeline{Parallelism: tt.parallelism}
			if got := p.EffectiveParallelism(); got != tt.want {
				t.Errorf("EffectiveParallelism() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestValidate_TaskNegativeParallelism(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"a": {Prompt: "a", Parallelism: -1},
		},
	}

	err := cf.Validate()
	if err == nil {
		t.Error("expected error for negative task parallelism")
	}
	if !strings.Contains(err.Error(), "parallelism cannot be negative") {
		t.Errorf("error should mention parallelism, got: %v", err)
	}
}

func TestValidate_PipelineNegativeParallelism(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"a": {Prompt: "a"},
		},
		Pipelines: map[string]Pipeline{
			"test": {Parallelism: -1, Tasks: []string{"a"}},
		},
	}

	err := cf.Validate()
	if err == nil {
		t.Error("expected error for negative pipeline parallelism")
	}
	if !strings.Contains(err.Error(), "parallelism cannot be negative") {
		t.Errorf("error should mention parallelism, got: %v", err)
	}
}

func TestLoadWithParallelism(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := `version: "1"
tasks:
  coder:
    prompt: coder
    iterations: 5
    parallelism: 3

pipelines:
  main:
    iterations: 10
    parallelism: 2
    tasks: [coder]
`
	path := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	task := cf.Tasks["coder"]
	if task.Parallelism != 3 {
		t.Errorf("task parallelism = %d, want 3", task.Parallelism)
	}
	if task.EffectiveParallelism() != 3 {
		t.Errorf("task EffectiveParallelism() = %d, want 3", task.EffectiveParallelism())
	}

	pipeline, err := cf.GetPipeline("main")
	if err != nil {
		t.Fatalf("GetPipeline() error = %v", err)
	}
	if pipeline.Parallelism != 2 {
		t.Errorf("pipeline parallelism = %d, want 2", pipeline.Parallelism)
	}
	if pipeline.EffectiveParallelism() != 2 {
		t.Errorf("pipeline EffectiveParallelism() = %d, want 2", pipeline.EffectiveParallelism())
	}
}

func TestValidate_TaskParallelismNameCollision(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"worker":   {Prompt: "a", Parallelism: 2},
			"worker.1": {Prompt: "b"},
		},
	}

	err := cf.Validate()
	if err == nil {
		t.Error("expected error for task parallelism name collision")
	}
	if !strings.Contains(err.Error(), "would collide") {
		t.Errorf("error should mention collision, got: %v", err)
	}
}

func TestValidate_PipelineParallelismNameCollision(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"a": {Prompt: "a"},
		},
		Pipelines: map[string]Pipeline{
			"main":   {Parallelism: 2, Tasks: []string{"a"}},
			"main.1": {Tasks: []string{"a"}},
		},
	}

	err := cf.Validate()
	if err == nil {
		t.Error("expected error for pipeline parallelism name collision")
	}
	if !strings.Contains(err.Error(), "would collide") {
		t.Errorf("error should mention collision, got: %v", err)
	}
}

func TestValidate_TaskParallelismNoCollision(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"worker":  {Prompt: "a", Parallelism: 2},
			"builder": {Prompt: "b"},
		},
	}

	if err := cf.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWarnings_TaskParallelismInPipeline(t *testing.T) {
	tests := []struct {
		name         string
		cf           *ComposeFile
		wantWarnings int
		wantContains string
	}{
		{
			name: "task with parallelism in pipeline emits warning",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"worker": {Prompt: "a", Parallelism: 3},
				},
				Pipelines: map[string]Pipeline{
					"main": {Tasks: []string{"worker"}},
				},
			},
			wantWarnings: 1,
			wantContains: "parallelism",
		},
		{
			name: "task without parallelism in pipeline emits no warning",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"worker": {Prompt: "a"},
				},
				Pipelines: map[string]Pipeline{
					"main": {Tasks: []string{"worker"}},
				},
			},
			wantWarnings: 0,
		},
		{
			name: "task with parallelism=1 in pipeline emits no warning",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"worker": {Prompt: "a", Parallelism: 1},
				},
				Pipelines: map[string]Pipeline{
					"main": {Tasks: []string{"worker"}},
				},
			},
			wantWarnings: 0,
		},
		{
			name: "task with parallelism NOT in pipeline emits no warning",
			cf: &ComposeFile{
				Tasks: map[string]Task{
					"worker":     {Prompt: "a", Parallelism: 3},
					"standalone": {Prompt: "b"},
				},
				Pipelines: map[string]Pipeline{
					"main": {Tasks: []string{"standalone"}},
				},
			},
			wantWarnings: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := tt.cf.Warnings()
			if len(warnings) != tt.wantWarnings {
				t.Errorf("Warnings() returned %d warnings, want %d: %v", len(warnings), tt.wantWarnings, warnings)
			}
			if tt.wantWarnings > 0 && tt.wantContains != "" {
				found := false
				for _, w := range warnings {
					if strings.Contains(w, tt.wantContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected a warning containing %q, got: %v", tt.wantContains, warnings)
				}
			}
		})
	}
}

// Tests for concurrency

func TestTaskEffectiveConcurrency(t *testing.T) {
	tests := []struct {
		name        string
		concurrency int
		want        int
	}{
		{"zero returns 0 (unlimited)", 0, 0},
		{"negative returns 0 (unlimited)", -1, 0},
		{"one returns 1", 1, 1},
		{"positive value returned", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := Task{Prompt: "test", Concurrency: tt.concurrency}
			if got := task.EffectiveConcurrency(); got != tt.want {
				t.Errorf("EffectiveConcurrency() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestValidate_TaskNegativeConcurrency(t *testing.T) {
	cf := &ComposeFile{
		Version: "1",
		Tasks: map[string]Task{
			"a": {Prompt: "a", Concurrency: -1},
		},
	}

	err := cf.Validate()
	if err == nil {
		t.Error("expected error for negative task concurrency")
	}
	if !strings.Contains(err.Error(), "concurrency cannot be negative") {
		t.Errorf("error should mention concurrency, got: %v", err)
	}
}

func TestLoadWithConcurrency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compose-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `version: "1"
tasks:
  planner:
    prompt: test
    concurrency: 1
  implementer:
    prompt: test
    concurrency: 3
`
	path := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	planner := cf.Tasks["planner"]
	if planner.Concurrency != 1 {
		t.Errorf("planner concurrency = %d, want 1", planner.Concurrency)
	}
	if planner.EffectiveConcurrency() != 1 {
		t.Errorf("planner EffectiveConcurrency() = %d, want 1", planner.EffectiveConcurrency())
	}

	implementer := cf.Tasks["implementer"]
	if implementer.Concurrency != 3 {
		t.Errorf("implementer concurrency = %d, want 3", implementer.Concurrency)
	}
	if implementer.EffectiveConcurrency() != 3 {
		t.Errorf("implementer EffectiveConcurrency() = %d, want 3", implementer.EffectiveConcurrency())
	}
}
