package compose

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DefaultFileName is the default compose file path.
const DefaultFileName = "./swarm/swarm.yaml"

// Dependency condition constants
const (
	ConditionSuccess = "success" // Run only if dependency succeeded (default)
	ConditionFailure = "failure" // Run only if dependency failed
	ConditionAny     = "any"     // Run regardless of outcome (waits for completion)
	ConditionAlways  = "always"  // Always run after dependency, even if skipped
)

// Dependency represents a task dependency with an optional condition.
// Supports both simple string form ("depends_on: [task1]") and full form
// ("depends_on: [{task: task1, condition: success}]").
type Dependency struct {
	Task      string `yaml:"task"`      // Name of the task to depend on
	Condition string `yaml:"condition"` // success, failure, any, always (default: success)
}

// UnmarshalYAML implements custom unmarshaling to support both string and object forms.
func (d *Dependency) UnmarshalYAML(value *yaml.Node) error {
	// Try simple string form first
	if value.Kind == yaml.ScalarNode {
		d.Task = value.Value
		d.Condition = ConditionSuccess // default
		return nil
	}

	// Try full object form
	if value.Kind == yaml.MappingNode {
		type rawDependency struct {
			Task      string `yaml:"task"`
			Condition string `yaml:"condition"`
		}
		var raw rawDependency
		if err := value.Decode(&raw); err != nil {
			return err
		}
		d.Task = raw.Task
		d.Condition = raw.Condition
		if d.Condition == "" {
			d.Condition = ConditionSuccess
		}
		return nil
	}

	return fmt.Errorf("invalid dependency format: expected string or object")
}

// EffectiveCondition returns the condition to use, defaulting to "success".
func (d *Dependency) EffectiveCondition() string {
	if d.Condition == "" {
		return ConditionSuccess
	}
	return d.Condition
}

// Pipeline represents a named workflow that runs tasks in DAG order.
type Pipeline struct {
	// Iterations is the number of times to run the entire DAG
	Iterations int `yaml:"iterations"`

	// Tasks is an optional list of task names to include in this pipeline.
	// If empty, all tasks from the compose file are included.
	Tasks []string `yaml:"tasks"`
}

// EffectiveIterations returns the iterations to use, defaulting to 1.
func (p *Pipeline) EffectiveIterations() int {
	if p.Iterations <= 0 {
		return 1
	}
	return p.Iterations
}

// ComposeFile represents the structure of a swarm compose file.
type ComposeFile struct {
	// Version is the compose file format version
	Version string `yaml:"version"`

	// Tasks is a map of task name to task configuration
	Tasks map[string]Task `yaml:"tasks"`

	// Pipelines is a map of pipeline name to pipeline configuration
	Pipelines map[string]Pipeline `yaml:"pipelines"`
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

	// Prefix is content prepended to the prompt at runtime
	Prefix string `yaml:"prefix"`

	// Suffix is content appended to the prompt at runtime
	Suffix string `yaml:"suffix"`

	// DependsOn specifies task dependencies with optional conditions.
	// Tasks will only run after their dependencies complete (based on condition).
	DependsOn []Dependency `yaml:"depends_on"`
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

	// Validate task dependencies reference existing tasks
	for name, task := range cf.Tasks {
		for _, dep := range task.DependsOn {
			if _, exists := cf.Tasks[dep.Task]; !exists {
				return fmt.Errorf("task %q: depends on unknown task %q", name, dep.Task)
			}
			if dep.Task == name {
				return fmt.Errorf("task %q: cannot depend on itself", name)
			}
		}
	}

	// Validate pipelines
	for name, pipeline := range cf.Pipelines {
		if err := pipeline.Validate(name, cf.Tasks); err != nil {
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

	// Validate dependency conditions
	for i, dep := range t.DependsOn {
		if dep.Task == "" {
			return fmt.Errorf("task %q: dependency %d has no task name", name, i)
		}
		cond := dep.EffectiveCondition()
		if cond != ConditionSuccess && cond != ConditionFailure && cond != ConditionAny && cond != ConditionAlways {
			return fmt.Errorf("task %q: dependency on %q has invalid condition %q (must be success, failure, any, or always)", name, dep.Task, cond)
		}
	}

	return nil
}

// Validate checks a pipeline for errors.
func (p *Pipeline) Validate(name string, tasks map[string]Task) error {
	if p.Iterations < 0 {
		return fmt.Errorf("pipeline %q: iterations cannot be negative", name)
	}

	// Validate that all specified tasks exist
	for _, taskName := range p.Tasks {
		if _, exists := tasks[taskName]; !exists {
			return fmt.Errorf("pipeline %q: references unknown task %q", name, taskName)
		}
	}

	return nil
}

// GetPipelineTasks returns the tasks included in this pipeline.
// If p.Tasks is empty, returns all task names from the compose file.
func (p *Pipeline) GetPipelineTasks(allTasks map[string]Task) []string {
	if len(p.Tasks) > 0 {
		return p.Tasks
	}
	// Return all task names
	names := make([]string, 0, len(allTasks))
	for name := range allTasks {
		names = append(names, name)
	}
	return names
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

// HasDependencies returns true if any task has dependencies defined.
func (cf *ComposeFile) HasDependencies() bool {
	for _, task := range cf.Tasks {
		if len(task.DependsOn) > 0 {
			return true
		}
	}
	return false
}

// GetPipeline returns a pipeline by name.
func (cf *ComposeFile) GetPipeline(name string) (*Pipeline, error) {
	pipeline, exists := cf.Pipelines[name]
	if !exists {
		return nil, fmt.Errorf("pipeline %q not found in compose file", name)
	}
	return &pipeline, nil
}

// HasPipelines returns true if any pipelines are defined.
func (cf *ComposeFile) HasPipelines() bool {
	return len(cf.Pipelines) > 0
}

// GetStandaloneTasks returns tasks that are not part of any pipeline and have no dependencies.
// These are independent tasks that should run in parallel alongside pipelines.
func (cf *ComposeFile) GetStandaloneTasks() map[string]Task {
	// Build set of tasks that are in pipelines
	pipelineTasks := make(map[string]bool)
	for _, pipeline := range cf.Pipelines {
		for _, taskName := range pipeline.GetPipelineTasks(cf.Tasks) {
			pipelineTasks[taskName] = true
		}
	}

	// Find tasks that are standalone (not in pipeline AND no dependencies)
	standalone := make(map[string]Task)
	for name, task := range cf.Tasks {
		// Skip if task is in a pipeline
		if pipelineTasks[name] {
			continue
		}
		// Skip if task has dependencies
		if len(task.DependsOn) > 0 {
			continue
		}
		// Skip if another task depends on this task (it's part of a DAG)
		if cf.isDependent(name) {
			continue
		}
		standalone[name] = task
	}

	return standalone
}

// isDependent returns true if any task depends on the given task name.
func (cf *ComposeFile) isDependent(taskName string) bool {
	for _, task := range cf.Tasks {
		for _, dep := range task.DependsOn {
			if dep.Task == taskName {
				return true
			}
		}
	}
	return false
}
