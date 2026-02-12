package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mj1618/swarm-cli/internal/agent"
	"github.com/spf13/cobra"
)

var (
	initTemplate string
	initPlan     string
	initPlanFile string
	initModel    string
)

type templateOption struct {
	Key         string
	Name        string
	Description string
}

var templateOptions = []templateOption{
	{
		Key:         "research",
		Name:        "Research Project",
		Description: "Research a topic, gather information, and produce a comprehensive report or knowledge base",
	},
	{
		Key:         "application",
		Name:        "Build an Application",
		Description: "Build a software application from scratch or extend an existing codebase iteratively",
	},
	{
		Key:         "book",
		Name:        "Write a Book / Long-form Content",
		Description: "Write a book, documentation, or other long-form content with structured chapters and editing",
	},
	{
		Key:         "data-analysis",
		Name:        "Data Analysis & Reporting",
		Description: "Analyze datasets, generate insights, create visualizations, and produce reports",
	},
	{
		Key:         "refactoring",
		Name:        "Code Refactoring / Migration",
		Description: "Systematically refactor, modernize, or migrate an existing codebase while maintaining correctness",
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new swarm project with a starter template",
	Long: `Initialize a new swarm project by selecting a template and providing a project plan.

This creates the swarm/ directory structure with:
  - swarm/PLAN.md - your project plan that all agents will reference
  - swarm/swarm.yaml - pipeline configuration with tasks and dependencies
  - swarm/prompts/*.md - prompt files for each pipeline task

An AI agent generates the pipeline configuration and prompts based on your
chosen template and plan. Each template follows a planner/doer pattern:
  - Planner reviews PLAN.md and current project state, writes bite-sized tasks
  - Doer agents pick up tasks and execute them to completion
  - Each iteration makes incremental progress on the overall plan`,
	Example: `  # Interactive mode (prompts for template and plan)
  swarm init

  # Specify template and plan via flags
  swarm init --template application --plan "Build a REST API for a todo app with user auth"

  # Read plan from a file
  swarm init --template book --plan-file ./my-book-outline.md

  # Use a specific model for generating the project
  swarm init --model sonnet`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVarP(&initTemplate, "template", "t", "", "Template to use (research, application, book, data-analysis, refactoring)")
	initCmd.Flags().StringVar(&initPlan, "plan", "", "Project plan description")
	initCmd.Flags().StringVar(&initPlanFile, "plan-file", "", "Read project plan from a file")
	initCmd.Flags().StringVarP(&initModel, "model", "m", "", "Model to use for generating the project (overrides config)")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Check for existing swarm/ directory
	swarmDir := "swarm"
	if info, err := os.Stat(swarmDir); err == nil && info.IsDir() {
		// Check if swarm.yaml already exists
		if _, err := os.Stat(filepath.Join(swarmDir, "swarm.yaml")); err == nil {
			fmt.Println("Warning: swarm/swarm.yaml already exists. Existing files may be overwritten.")
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Continue? [y/N] ")
			answer, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(answer)) != "y" {
				fmt.Println("Aborted.")
				return nil
			}
			fmt.Println()
		}
	}

	// Select template
	selectedTemplate, err := selectTemplate()
	if err != nil {
		return err
	}

	// Get plan
	plan, err := getProjectPlan()
	if err != nil {
		return err
	}

	// Create directories
	for _, dir := range []string{"swarm", "swarm/prompts"} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Write PLAN.md
	planContent := "# Project Plan\n\n" + plan + "\n"
	planPath := filepath.Join("swarm", "PLAN.md")
	if err := os.WriteFile(planPath, []byte(planContent), 0644); err != nil {
		return fmt.Errorf("failed to write PLAN.md: %w", err)
	}
	fmt.Printf("Created %s\n", planPath)

	// Build generation prompt
	genPrompt := buildGenerationPrompt(selectedTemplate, plan)

	// Determine model
	effectiveModel := appConfig.Model
	if initModel != "" {
		effectiveModel = initModel
	}

	fmt.Printf("\nGenerating project files using %s model...\n\n", effectiveModel)

	// Run agent to generate files
	cfg := agent.Config{
		Model:   effectiveModel,
		Prompt:  genPrompt,
		Command: appConfig.Command,
	}

	runner := agent.NewRunner(cfg)
	if err := runner.Run(os.Stdout); err != nil {
		fmt.Println()
		fmt.Println("Warning: agent failed to generate project files.")
		fmt.Println("You can manually create swarm/swarm.yaml and swarm/prompts/*.md")
		fmt.Printf("Error: %v\n", err)
		return nil
	}

	fmt.Println()
	fmt.Println("Project initialized! Next steps:")
	fmt.Println("  1. Review swarm/PLAN.md and edit if needed")
	fmt.Println("  2. Review swarm/swarm.yaml pipeline configuration")
	fmt.Println("  3. Review swarm/prompts/*.md prompt files")
	fmt.Println("  4. Run 'swarm up' to start the pipeline")
	fmt.Println("  5. Run 'swarm up -d' to run in the background")

	return nil
}

func selectTemplate() (*templateOption, error) {
	if initTemplate != "" {
		for i := range templateOptions {
			if templateOptions[i].Key == initTemplate {
				return &templateOptions[i], nil
			}
		}
		validKeys := make([]string, len(templateOptions))
		for i, t := range templateOptions {
			validKeys[i] = t.Key
		}
		return nil, fmt.Errorf("unknown template: %s (valid: %s)", initTemplate, strings.Join(validKeys, ", "))
	}

	// Interactive selection
	fmt.Println("Select a project template:")
	fmt.Println()
	for i, t := range templateOptions {
		fmt.Printf("  %d. %s\n     %s\n\n", i+1, t.Name, t.Description)
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter your choice (1-5): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		var selection int
		if _, err := fmt.Sscanf(input, "%d", &selection); err == nil {
			if selection >= 1 && selection <= len(templateOptions) {
				selected := &templateOptions[selection-1]
				fmt.Printf("\nSelected: %s\n\n", selected.Name)
				return selected, nil
			}
		}

		// Try matching by key name
		for i := range templateOptions {
			if strings.EqualFold(templateOptions[i].Key, input) {
				fmt.Printf("\nSelected: %s\n\n", templateOptions[i].Name)
				return &templateOptions[i], nil
			}
		}

		fmt.Println("Invalid selection. Please try again.")
	}
}

func getProjectPlan() (string, error) {
	if initPlan != "" && initPlanFile != "" {
		return "", fmt.Errorf("only one of --plan or --plan-file can be specified")
	}

	if initPlanFile != "" {
		content, err := os.ReadFile(initPlanFile)
		if err != nil {
			return "", fmt.Errorf("failed to read plan file: %w", err)
		}
		plan := strings.TrimSpace(string(content))
		if plan == "" {
			return "", fmt.Errorf("plan file is empty")
		}
		return plan, nil
	}

	if initPlan != "" {
		return initPlan, nil
	}

	// Interactive plan input
	fmt.Println("Describe your project plan. This will be saved to swarm/PLAN.md and")
	fmt.Println("referenced by all agents throughout the project.")
	fmt.Println()
	fmt.Println("Enter your plan (press Enter on an empty line to finish):")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	var lines []string
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" && len(lines) > 0 {
			break
		}
		if trimmed != "" || len(lines) > 0 {
			lines = append(lines, trimmed)
		}
	}

	plan := strings.TrimSpace(strings.Join(lines, "\n"))
	if plan == "" {
		return "", fmt.Errorf("plan cannot be empty")
	}

	fmt.Println()
	return plan, nil
}

func buildGenerationPrompt(tmpl *templateOption, plan string) string {
	var sb strings.Builder

	sb.WriteString(`# Initialize Swarm Project

You are setting up a swarm-cli project. Your job is to create the pipeline configuration and prompt files that will orchestrate AI agents to accomplish the user's plan iteratively.

## User's Project Plan

The user has already saved this plan to ` + "`swarm/PLAN.md`" + `:

`)
	sb.WriteString("```\n")
	sb.WriteString(plan)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Template Type: " + tmpl.Name + "\n\n")
	sb.WriteString(tmpl.Description + "\n\n")

	sb.WriteString(`## What You Need to Create

Create the following files. Use the appropriate file-writing tools to create each file.

### 1. ` + "`swarm/swarm.yaml`" + ` — Pipeline Configuration

This defines the tasks and pipeline. Here is the format:

` + "```yaml" + `
version: "1"
tasks:
  planner:
    prompt: planner                    # references swarm/prompts/planner.md
  doer-name:
    prompt: doer-name                  # references swarm/prompts/doer-name.md
    depends_on: [planner]              # runs after planner completes
  reviewer-name:
    prompt: reviewer-name              # references swarm/prompts/reviewer-name.md
    depends_on: [doer-name]            # runs after doer completes

pipelines:
  main:
    iterations: 10                     # how many full cycles to run
    parallelism: 1                     # number of concurrent pipeline instances (set explicitly)
    tasks: [planner, doer-name, reviewer-name]
` + "```" + `

Rules:
- The ` + "`prompt`" + ` field references a markdown file in ` + "`swarm/prompts/`" + ` by name (without .md extension)
- Use ` + "`depends_on`" + ` to define execution order — downstream tasks wait for upstream tasks
- Set ` + "`parallelism: 1`" + ` explicitly so users can see where to change it
- Choose an appropriate number of iterations (10-20 is typical)
- Design 2-5 tasks that make sense for this template type
- Name tasks descriptively for the template (e.g. "researcher" not just "doer")

### 2. ` + "`swarm/prompts/<task-name>.md`" + ` — One Per Task

Each task needs a corresponding prompt file in ` + "`swarm/prompts/`" + `. These markdown files tell the agent exactly what to do.

#### How the runtime works:

When the pipeline runs, each agent automatically receives:
- **SWARM_STATE_DIR**: A shared directory unique to the current pipeline iteration. Agents read previous task outputs and write their own results here. This is injected automatically — prompts don't need to mention the path.
- **SWARM_AGENT_ID**: A unique ID for each agent instance.
- **Iteration number**: Which iteration of the pipeline this is.

Between tasks, you can reference another task's output using:
- ` + "`{{output:task_name}}`" + ` — This directive is replaced at runtime with the full contents of what the named task wrote to SWARM_STATE_DIR.

For example, if the planner writes its output, the doer prompt can include ` + "`{{output:planner}}`" + ` to read it.

#### The Planner/Doer Pattern (CRITICAL):

The entire pipeline revolves around this pattern:

**Planner** (first task in each iteration):
1. Read ` + "`swarm/PLAN.md`" + ` for the overall project plan
2. Review the current state of the project directory — what files exist, what's been done already
3. Determine what single piece of work should be done next
4. Write a specific, bite-sized task description — something an agent can complete in one session
5. The task must be actionable and concrete, not vague
6. Each iteration the planner MUST identify DIFFERENT work (check what's already been done to avoid repetition)

**Doer(s)** (middle tasks):
1. Read ` + "`swarm/PLAN.md`" + ` for context on the overall project
2. Read the planner's task via ` + "`{{output:planner}}`" + `
3. Execute the task to completion — actually create/modify files, write content, etc.
4. Be thorough but focused — complete the single assigned task, don't try to do everything at once

**Reviewer** (optional final task):
1. Read ` + "`swarm/PLAN.md`" + ` for context
2. Read the doer's output via ` + "`{{output:doer-name}}`" + `
3. Review the work for quality, correctness, and completeness
4. Report any issues that should be addressed in the next iteration

#### Prompt writing guidelines:

- Start every prompt by instructing the agent to read ` + "`swarm/PLAN.md`" + `
- Make prompts detailed and specific to the template type and the user's plan
- Include clear exit conditions (when should the agent stop/consider the task done?)
- Use ` + "`{{output:task_name}}`" + ` to pass data between stages — don't hardcode paths
- Instruct planners to always check current project state before planning (to avoid repeating work)
- Keep planner output bite-sized: each task should be completable in a single agent session
- Tell doer agents to actually create/modify project files, not just describe what to do

## Generate the files now

Based on the "` + tmpl.Name + `" template and the user's plan above, create:

1. ` + "`swarm/swarm.yaml`" + ` with an appropriate pipeline (2-5 tasks, sensible dependencies, explicit parallelism: 1)
2. ` + "`swarm/prompts/<task>.md`" + ` for each task in the pipeline

Make the prompts specific and tailored to the user's plan. The pipeline should be practical and effective for this kind of work. Don't be generic — reference the specific goals from the user's plan in your prompts.

IMPORTANT: Only create the files listed above. Do not create any other files, do not modify PLAN.md (it already exists), and do not run any commands. Just create swarm/swarm.yaml and the prompt files.
`)

	return sb.String()
}
