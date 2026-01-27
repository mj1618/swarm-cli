package compose

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DefaultFileName is the default compose file path.
const DefaultFileName = "./swarm/swarm.yaml"

// ComposeFile represents the structure of a swarm compose file.
type ComposeFile struct {
	// Version is the compose file format version
	Version string `yaml:"version"`

	// Tasks is a map of task name to task configuration
	Tasks map[string]Task `yaml:"tasks"`
}

// Task represents a single task definition in the compose file.
type Task struct {
	// Prompt is the name of a prompt from the prompts directory
	Prompt string `yaml:"prompt"`

	// PromptFile is the path to an arbitrary prompt file
	PromptFile string `yaml:"prompt-file"`

	// PromptString is a direct prompt string
	PromptString string `yaml:"prompt-string"`

	// Model is the model to use (optional, overrides config)
	Model string `yaml:"model"`

	// Iterations is the number of iterations to run (optional, default 1)
	Iterations int `yaml:"iterations"`

	// Name is a custom name for the agent (optional, defaults to task name)
	Name string `yaml:"name"`
}

// DefaultPath returns the default compose file path.
func DefaultPath() string {
	return DefaultFileName
}

// Load reads and parses a compose file from the given path.
func Load(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	var cf ComposeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	return &cf, nil
}

// Validate checks the compose file for errors.
func (cf *ComposeFile) Validate() error {
	if len(cf.Tasks) == 0 {
		return fmt.Errorf("no tasks defined in compose file")
	}

	for name, task := range cf.Tasks {
		if err := task.Validate(name); err != nil {
			return err
		}
	}

	return nil
}

// Validate checks a single task for errors.
func (t *Task) Validate(name string) error {
	// Count how many prompt sources are specified
	promptCount := 0
	if t.Prompt != "" {
		promptCount++
	}
	if t.PromptFile != "" {
		promptCount++
	}
	if t.PromptString != "" {
		promptCount++
	}

	if promptCount == 0 {
		return fmt.Errorf("task %q: no prompt source specified (use prompt, prompt-file, or prompt-string)", name)
	}
	if promptCount > 1 {
		return fmt.Errorf("task %q: only one prompt source allowed (prompt, prompt-file, or prompt-string)", name)
	}

	if t.Iterations < 0 {
		return fmt.Errorf("task %q: iterations cannot be negative", name)
	}

	return nil
}

// GetTasks returns the tasks to run, filtered by the given names.
// If names is empty, all tasks are returned.
func (cf *ComposeFile) GetTasks(names []string) (map[string]Task, error) {
	if len(names) == 0 {
		return cf.Tasks, nil
	}

	result := make(map[string]Task)
	for _, name := range names {
		task, ok := cf.Tasks[name]
		if !ok {
			return nil, fmt.Errorf("task %q not found in compose file", name)
		}
		result[name] = task
	}

	return result, nil
}

// EffectiveName returns the agent name to use for this task.
// If Name is set, it returns Name; otherwise it returns the task key.
func (t *Task) EffectiveName(taskKey string) string {
	if t.Name != "" {
		return t.Name
	}
	return taskKey
}

// EffectiveIterations returns the iterations to use for this task.
// If Iterations is 0, it returns 1 (the default).
func (t *Task) EffectiveIterations() int {
	if t.Iterations == 0 {
		return 1
	}
	return t.Iterations
}
