package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mj1618/swarm-cli/internal/compose"
)

func TestLoadTaskPrompt(t *testing.T) {
	// Create temp directory with prompts
	tmpDir, err := os.MkdirTemp("", "up-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a prompts directory
	promptsDir := filepath.Join(tmpDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	// Create a test prompt file
	promptContent := "This is my test prompt content"
	if err := os.WriteFile(filepath.Join(promptsDir, "test-prompt.md"), []byte(promptContent), 0644); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	// Create a custom prompt file outside prompts dir
	customPromptContent := "Custom prompt from file"
	customPath := filepath.Join(tmpDir, "custom-prompt.md")
	if err := os.WriteFile(customPath, []byte(customPromptContent), 0644); err != nil {
		t.Fatalf("failed to write custom prompt file: %v", err)
	}

	tests := []struct {
		name        string
		task        compose.Task
		wantContent string
		wantLabel   string
		wantErr     bool
	}{
		{
			name:        "prompt from directory",
			task:        compose.Task{Prompt: "test-prompt"},
			wantContent: promptContent,
			wantLabel:   "test-prompt",
			wantErr:     false,
		},
		{
			name:        "prompt from file",
			task:        compose.Task{PromptFile: customPath},
			wantContent: customPromptContent,
			wantLabel:   customPath,
			wantErr:     false,
		},
		{
			name:        "prompt string",
			task:        compose.Task{PromptString: "inline prompt"},
			wantContent: "inline prompt",
			wantLabel:   "<string>",
			wantErr:     false,
		},
		{
			name:    "nonexistent prompt",
			task:    compose.Task{Prompt: "nonexistent"},
			wantErr: true,
		},
		{
			name:    "nonexistent file",
			task:    compose.Task{PromptFile: "/nonexistent/file.md"},
			wantErr: true,
		},
		{
			name:    "no prompt source",
			task:    compose.Task{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, label, err := loadTaskPrompt(tt.task, promptsDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadTaskPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if content != tt.wantContent {
					t.Errorf("loadTaskPrompt() content = %q, want %q", content, tt.wantContent)
				}
				if label != tt.wantLabel {
					t.Errorf("loadTaskPrompt() label = %q, want %q", label, tt.wantLabel)
				}
			}
		})
	}
}

func TestUpCommandFlags(t *testing.T) {
	// Test that the command has the expected flags
	cmd := upCmd

	fileFlag := cmd.Flags().Lookup("file")
	if fileFlag == nil {
		t.Error("expected 'file' flag to exist")
	} else {
		if fileFlag.Shorthand != "f" {
			t.Errorf("file flag shorthand = %q, want %q", fileFlag.Shorthand, "f")
		}
		if fileFlag.DefValue != "./swarm/swarm.yaml" {
			t.Errorf("file flag default = %q, want %q", fileFlag.DefValue, "./swarm/swarm.yaml")
		}
	}

	detachFlag := cmd.Flags().Lookup("detach")
	if detachFlag == nil {
		t.Error("expected 'detach' flag to exist")
	} else {
		if detachFlag.Shorthand != "d" {
			t.Errorf("detach flag shorthand = %q, want %q", detachFlag.Shorthand, "d")
		}
		if detachFlag.DefValue != "false" {
			t.Errorf("detach flag default = %q, want %q", detachFlag.DefValue, "false")
		}
	}
}

func TestUpCommandUsage(t *testing.T) {
	cmd := upCmd

	if cmd.Use != "up [task...]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "up [task...]")
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if cmd.Example == "" {
		t.Error("Example should not be empty")
	}
}

func TestComposeFileIntegration(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "up-integration-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid compose file
	composeContent := `version: "1"
tasks:
  task1:
    prompt-string: "Do task 1"
    iterations: 2
  task2:
    prompt-string: "Do task 2"
    model: sonnet-4.5
    name: custom-name
`
	composePath := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	// Load and validate
	cf, err := compose.Load(composePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cf.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Check task1
	task1 := cf.Tasks["task1"]
	if task1.PromptString != "Do task 1" {
		t.Errorf("task1.PromptString = %q, want %q", task1.PromptString, "Do task 1")
	}
	if task1.EffectiveIterations() != 2 {
		t.Errorf("task1.EffectiveIterations() = %d, want %d", task1.EffectiveIterations(), 2)
	}
	if task1.EffectiveName("task1") != "task1" {
		t.Errorf("task1.EffectiveName() = %q, want %q", task1.EffectiveName("task1"), "task1")
	}

	// Check task2
	task2 := cf.Tasks["task2"]
	if task2.PromptString != "Do task 2" {
		t.Errorf("task2.PromptString = %q, want %q", task2.PromptString, "Do task 2")
	}
	if task2.Model != "sonnet-4.5" {
		t.Errorf("task2.Model = %q, want %q", task2.Model, "sonnet-4.5")
	}
	if task2.EffectiveName("task2") != "custom-name" {
		t.Errorf("task2.EffectiveName() = %q, want %q", task2.EffectiveName("task2"), "custom-name")
	}
	if task2.EffectiveIterations() != 1 {
		t.Errorf("task2.EffectiveIterations() = %d, want %d", task2.EffectiveIterations(), 1)
	}

	// Test filtering
	filtered, err := cf.GetTasks([]string{"task1"})
	if err != nil {
		t.Fatalf("GetTasks() error = %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("GetTasks() returned %d tasks, want 1", len(filtered))
	}
	if _, ok := filtered["task1"]; !ok {
		t.Error("GetTasks() should contain task1")
	}
}

func TestSkipAlreadyRunningTasks(t *testing.T) {
	// This tests the logic for determining which tasks to skip based on running agents.
	// We use the same logic as runTasksDetached/runTasksForeground to build the running names map.

	tests := []struct {
		name              string
		runningNames      map[string]bool // Names of running agents
		tasks             map[string]compose.Task
		expectedSkipped   []string
		expectedToStart   []string
	}{
		{
			name:         "no running agents - all tasks start",
			runningNames: map[string]bool{},
			tasks: map[string]compose.Task{
				"frontend": {PromptString: "test"},
				"backend":  {PromptString: "test"},
			},
			expectedSkipped: []string{},
			expectedToStart: []string{"frontend", "backend"},
		},
		{
			name: "one running agent - skip matching task",
			runningNames: map[string]bool{
				"frontend": true,
			},
			tasks: map[string]compose.Task{
				"frontend": {PromptString: "test"},
				"backend":  {PromptString: "test"},
			},
			expectedSkipped: []string{"frontend"},
			expectedToStart: []string{"backend"},
		},
		{
			name: "all agents running - skip all tasks",
			runningNames: map[string]bool{
				"frontend": true,
				"backend":  true,
			},
			tasks: map[string]compose.Task{
				"frontend": {PromptString: "test"},
				"backend":  {PromptString: "test"},
			},
			expectedSkipped: []string{"frontend", "backend"},
			expectedToStart: []string{},
		},
		{
			name: "task with custom name - match by effective name",
			runningNames: map[string]bool{
				"custom-frontend": true,
			},
			tasks: map[string]compose.Task{
				"frontend": {PromptString: "test", Name: "custom-frontend"},
				"backend":  {PromptString: "test"},
			},
			expectedSkipped: []string{"frontend"},
			expectedToStart: []string{"backend"},
		},
		{
			name: "running agent name doesn't match task key but matches custom name",
			runningNames: map[string]bool{
				"my-agent": true,
			},
			tasks: map[string]compose.Task{
				"task1": {PromptString: "test", Name: "my-agent"},
				"task2": {PromptString: "test"},
			},
			expectedSkipped: []string{"task1"},
			expectedToStart: []string{"task2"},
		},
		{
			name: "no overlap between running and tasks",
			runningNames: map[string]bool{
				"other-agent": true,
			},
			tasks: map[string]compose.Task{
				"frontend": {PromptString: "test"},
				"backend":  {PromptString: "test"},
			},
			expectedSkipped: []string{},
			expectedToStart: []string{"frontend", "backend"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var skipped []string
			var toStart []string

			for taskName, task := range tt.tasks {
				effectiveName := task.EffectiveName(taskName)
				if tt.runningNames[effectiveName] {
					skipped = append(skipped, taskName)
				} else {
					toStart = append(toStart, taskName)
				}
			}

			// Check skipped count matches
			if len(skipped) != len(tt.expectedSkipped) {
				t.Errorf("skipped count = %d, want %d", len(skipped), len(tt.expectedSkipped))
			}

			// Check toStart count matches
			if len(toStart) != len(tt.expectedToStart) {
				t.Errorf("toStart count = %d, want %d", len(toStart), len(tt.expectedToStart))
			}

			// Verify skipped tasks are correct
			for _, expected := range tt.expectedSkipped {
				found := false
				for _, s := range skipped {
					if s == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected task %q to be skipped, but it wasn't", expected)
				}
			}

			// Verify toStart tasks are correct
			for _, expected := range tt.expectedToStart {
				found := false
				for _, s := range toStart {
					if s == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected task %q to start, but it wasn't", expected)
				}
			}
		})
	}
}

func TestBuildRunningNamesMap(t *testing.T) {
	// This tests building the running names lookup map from agent states
	// Similar to the logic in runTasksDetached/runTasksForeground

	tests := []struct {
		name           string
		agents         []struct {
			name   string
			status string
		}
		onlyRunning    bool
		expectedNames  map[string]bool
	}{
		{
			name: "all running agents",
			agents: []struct {
				name   string
				status string
			}{
				{"frontend", "running"},
				{"backend", "running"},
			},
			onlyRunning: true,
			expectedNames: map[string]bool{
				"frontend": true,
				"backend":  true,
			},
		},
		{
			name: "mix of running and terminated",
			agents: []struct {
				name   string
				status string
			}{
				{"frontend", "running"},
				{"backend", "terminated"},
			},
			onlyRunning: true,
			expectedNames: map[string]bool{
				"frontend": true,
			},
		},
		{
			name: "no running agents",
			agents: []struct {
				name   string
				status string
			}{
				{"frontend", "terminated"},
				{"backend", "terminated"},
			},
			onlyRunning:   true,
			expectedNames: map[string]bool{},
		},
		{
			name:          "empty agents list",
			agents:        []struct {
				name   string
				status string
			}{},
			onlyRunning:   true,
			expectedNames: map[string]bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the map as the code does
			runningNames := make(map[string]bool)
			for _, a := range tt.agents {
				if !tt.onlyRunning || a.status == "running" {
					runningNames[a.name] = true
				}
			}

			// Verify the map matches expected
			if len(runningNames) != len(tt.expectedNames) {
				t.Errorf("runningNames length = %d, want %d", len(runningNames), len(tt.expectedNames))
			}

			for name := range tt.expectedNames {
				if !runningNames[name] {
					t.Errorf("expected name %q in runningNames, but not found", name)
				}
			}

			for name := range runningNames {
				if !tt.expectedNames[name] {
					t.Errorf("unexpected name %q in runningNames", name)
				}
			}
		})
	}
}

func TestComposeFileValidationErrors(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "up-validation-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "no prompt source",
			content: `version: "1"
tasks:
  bad-task:
    model: opus
`,
			wantErr: "no prompt source",
		},
		{
			name: "multiple prompt sources",
			content: `version: "1"
tasks:
  bad-task:
    prompt: test
    prompt-string: "inline"
`,
			wantErr: "only one prompt source",
		},
		{
			name: "empty tasks",
			content: `version: "1"
tasks: {}
`,
			wantErr: "no tasks defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, "test.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			cf, err := compose.Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			err = cf.Validate()
			if err == nil {
				t.Errorf("Validate() expected error containing %q, got nil", tt.wantErr)
			}
		})
	}
}
