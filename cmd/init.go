package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
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
  - swarm/swarm.yaml - pipeline configuration with tasks, dependencies, and inline prompts

An AI agent generates the pipeline configuration with inline prompts based on your
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

// Reusable color styles for the init command
var (
	initBold      = color.New(color.Bold)
	initCyan      = color.New(color.FgCyan)
	initCyanBold  = color.New(color.FgCyan, color.Bold)
	initGreen     = color.New(color.FgGreen)
	initGreenBold = color.New(color.FgGreen, color.Bold)
	initYellow    = color.New(color.FgYellow)
	initRed       = color.New(color.FgRed)
	initFaint     = color.New(color.Faint)
	initWhiteBold = color.New(color.FgWhite, color.Bold)
)

func printInitHeader() {
	fmt.Println()
	initCyanBold.Println("  ╭──────────────────────────────────────╮")
	initCyanBold.Print("  │  ")
	initWhiteBold.Print("swarm init")
	initFaint.Print("  — project setup wizard")
	initCyanBold.Println("  │")
	initCyanBold.Println("  ╰──────────────────────────────────────╯")
	fmt.Println()
}

func printStepHeader(step, total int, title string) {
	initCyan.Printf("  (%d/%d) ", step, total)
	initBold.Println(title)
	fmt.Println()
}

func runInit(cmd *cobra.Command, args []string) error {
	totalSteps := 3

	printInitHeader()

	// Check for existing swarm/ directory
	swarmDir := "swarm"
	if info, err := os.Stat(swarmDir); err == nil && info.IsDir() {
		// Check if swarm.yaml already exists
		if _, err := os.Stat(filepath.Join(swarmDir, "swarm.yaml")); err == nil {
			initYellow.Print("  ⚠ ")
			fmt.Print("swarm/swarm.yaml already exists. Existing files may be overwritten.\n")
			reader := bufio.NewReader(os.Stdin)
			initFaint.Print("    Continue? ")
			initBold.Print("[y/N] ")
			answer, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(answer)) != "y" {
				initFaint.Println("    Aborted.")
				return nil
			}
			fmt.Println()
		}
	}

	// Step 1: Select template
	printStepHeader(1, totalSteps, "Choose a template")
	selectedTemplate, err := selectTemplate()
	if err != nil {
		return err
	}

	// Step 2: Get plan
	printStepHeader(2, totalSteps, "Describe your project")
	plan, err := getProjectPlan()
	if err != nil {
		return err
	}
	if plan == "" {
		return nil // User aborted
	}

	// Create swarm directory
	if err := os.MkdirAll("swarm", 0755); err != nil {
		return fmt.Errorf("failed to create directory swarm: %w", err)
	}

	// Build generation prompt (agent will create both PLAN.md and swarm.yaml)
	genPrompt := buildGenerationPrompt(selectedTemplate, plan)

	// Determine model
	effectiveModel := appConfig.Model
	if initModel != "" {
		effectiveModel = initModel
	}

	// Step 3: Generate
	printStepHeader(3, totalSteps, "Generate project")

	// Run agent to generate files with a spinner
	cfg := agent.Config{
		Model:   effectiveModel,
		Prompt:  genPrompt,
		Command: appConfig.Command,
	}

	runner := agent.NewRunner(cfg)

	spin := newSpinner(fmt.Sprintf("Generating project files using %s", initBold.Sprint(effectiveModel)))

	err = runner.Run(io.Discard)
	spin.Stop()

	if err != nil {
		fmt.Println()
		initRed.Print("  ✗ ")
		fmt.Println("Agent failed to generate project files.")
		initFaint.Println("    You can manually create swarm/swarm.yaml with inline prompt-string values")
		initFaint.Printf("    Error: %v\n", err)
		return nil
	}

	// Success
	fmt.Println()
	initGreenBold.Println("  ✓ Project initialized!")
	fmt.Println()
	initBold.Println("  Next steps")
	initFaint.Println("  ──────────")
	fmt.Println()
	initCyan.Print("    1. ")
	fmt.Print("Review ")
	initBold.Print("swarm/PLAN.md")
	fmt.Println(" and edit if needed")
	initCyan.Print("    2. ")
	fmt.Print("Review ")
	initBold.Print("swarm/swarm.yaml")
	fmt.Println(" pipeline configuration and inline prompts")
	initCyan.Print("    3. ")
	fmt.Print("Run ")
	initCyanBold.Print("swarm up -d")
	fmt.Println(" to start the pipeline")
	fmt.Println()

	return nil
}

// Template icons for visual differentiation
var templateIcons = map[string]string{
	"research":      "🔍",
	"application":   "⚡",
	"book":          "📖",
	"data-analysis": "📊",
	"refactoring":   "🔧",
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
	for i, t := range templateOptions {
		icon := templateIcons[t.Key]
		initCyanBold.Printf("    %d ", i+1)
		fmt.Printf("%s  ", icon)
		initBold.Println(t.Name)
		initFaint.Printf("         %s\n", t.Description)
		fmt.Println()
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		initCyan.Print("  → ")
		fmt.Print("Enter your choice ")
		initFaint.Print("(1-5)")
		fmt.Print(": ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		var selection int
		if _, err := fmt.Sscanf(input, "%d", &selection); err == nil {
			if selection >= 1 && selection <= len(templateOptions) {
				selected := &templateOptions[selection-1]
				icon := templateIcons[selected.Key]
				fmt.Println()
				initGreen.Print("  ✓ ")
				fmt.Printf("%s  %s\n", icon, selected.Name)
				fmt.Println()
				return selected, nil
			}
		}

		// Try matching by key name
		for i := range templateOptions {
			if strings.EqualFold(templateOptions[i].Key, input) {
				icon := templateIcons[templateOptions[i].Key]
				fmt.Println()
				initGreen.Print("  ✓ ")
				fmt.Printf("%s  %s\n", icon, templateOptions[i].Name)
				fmt.Println()
				return &templateOptions[i], nil
			}
		}

		initRed.Println("    Invalid selection. Please try again.")
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

	// Interactive plan input using vim
	fmt.Print("  Describe your project and an AI agent will expand it into a detailed plan\n")
	fmt.Print("  in ")
	initBold.Print("swarm/PLAN.md")
	fmt.Println(" that all agents will reference throughout the project.")
	fmt.Println()

	// Create a temporary file for vim editing
	tmpFile, err := os.CreateTemp("", "swarm-plan-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write initial content/instructions
	initialContent := `# Project Plan

<!-- Describe your project below. An AI agent will expand this into a detailed plan. -->
<!-- Save and quit (:wq) when done. Leave empty or quit without saving (:q!) to abort. -->

`
	if _, err := tmpFile.WriteString(initialContent); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	// Open vim
	initFaint.Println("  Opening vim...")
	fmt.Println()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor failed: %w", err)
	}

	// Read the result
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to read plan: %w", err)
	}

	// Remove comment lines and trim
	lines := strings.Split(string(content), "\n")
	var planLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->") {
			continue
		}
		// Also skip the header if unchanged
		if trimmed == "# Project Plan" {
			continue
		}
		planLines = append(planLines, line)
	}
	plan := strings.TrimSpace(strings.Join(planLines, "\n"))

	if plan == "" {
		fmt.Println()
		initFaint.Println("    Aborted (empty plan).")
		return "", nil
	}

	fmt.Println()
	initGreen.Print("  ✓ ")
	lineCount := len(strings.Split(plan, "\n"))
	if lineCount == 1 {
		initFaint.Println("Plan captured (1 line)")
	} else {
		initFaint.Printf("Plan captured (%d lines)\n", lineCount)
	}
	fmt.Println()
	return plan, nil
}

// spinner shows an animated indicator with elapsed time while work is in progress.
type spinner struct {
	message string
	start   time.Time
	done    chan struct{}
	wg      sync.WaitGroup
}

func newSpinner(message string) *spinner {
	s := &spinner{
		message: message,
		start:   time.Now(),
		done:    make(chan struct{}),
	}
	s.wg.Add(1)
	go s.run()
	return s
}

func (s *spinner) run() {
	defer s.wg.Done()
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	cyan := color.New(color.FgCyan)
	faint := color.New(color.Faint)
	i := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			// Clear the spinner line
			fmt.Print("\r\033[K")
			return
		case <-ticker.C:
			elapsed := time.Since(s.start).Truncate(time.Second)
			fmt.Print("\r\033[K")
			fmt.Print("  ")
			cyan.Print(frames[i%len(frames)])
			fmt.Printf(" %s ", s.message)
			faint.Printf("(%s)", elapsed)
			i++
		}
	}
}

func (s *spinner) Stop() {
	close(s.done)
	s.wg.Wait()
}

func buildGenerationPrompt(tmpl *templateOption, plan string) string {
	var sb strings.Builder

	sb.WriteString(`# Initialize Swarm Project

You are setting up a swarm-cli project. Your job is to create a detailed project plan and the pipeline configuration that will orchestrate AI agents to accomplish the user's goals iteratively.

## User's Input

The user described their project as follows:

`)
	sb.WriteString("```\n")
	sb.WriteString(plan)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Template Type: " + tmpl.Name + "\n\n")
	sb.WriteString(tmpl.Description + "\n\n")

	sb.WriteString(`## What You Need to Create

Create two files using the appropriate file-writing tool:
1. ` + "`swarm/PLAN.md`" + ` — A detailed project plan
2. ` + "`swarm/swarm.yaml`" + ` — Pipeline configuration

### ` + "`swarm/PLAN.md`" + ` — Detailed Project Plan

Take the user's input above and expand it into a comprehensive, well-structured project plan. This file is the single source of truth that ALL agents will reference throughout the project lifecycle.

The plan should include:
- **Project Overview**: A clear summary of what is being built/done and why
- **Goals & Success Criteria**: What does "done" look like? How will we know the project succeeded?
- **Scope**: What is in scope and what is explicitly out of scope
- **Architecture / Approach**: High-level technical or structural decisions (e.g., tech stack, content structure, research methodology — whatever fits the project type)
- **Milestones / Phases**: Break the work into logical phases or milestones, ordered by priority
- **Detailed Requirements**: Specific, actionable requirements grouped by area (features, chapters, research questions, etc.)
- **Constraints & Assumptions**: Any known limitations, dependencies, or assumptions

Write this in Markdown. Be thorough and specific — the agents executing this plan need enough detail to make good decisions autonomously. Expand on the user's input with sensible defaults and best practices where they were vague, but stay true to their intent. If the user provided very little detail, use your best judgment to flesh out a reasonable plan for the chosen template type.

### ` + "`swarm/swarm.yaml`" + ` — Pipeline Configuration

This defines the tasks and pipeline. Prompts are inlined directly as ` + "`prompt-string`" + ` values. Here is the format:

` + "```yaml" + `
version: "1"
tasks:
  planner:
    prompt-string: |
      # Planner
      Your detailed prompt instructions go here...
      Multi-line prompts are supported using YAML block scalars.
    concurrency: 1                     # only one planner runs at a time (critical for task coordination)
  doer-name:
    prompt-string: |
      # Doer
      Your detailed prompt instructions go here...
    depends_on: [planner]              # runs after planner completes
  reviewer-name:
    prompt-string: |
      # Reviewer
      Your detailed prompt instructions go here...
    depends_on: [doer-name]            # runs after doer completes

pipelines:
  main:
    iterations: 10                     # how many full cycles to run
    parallelism: 1                     # number of concurrent pipeline instances (set explicitly)
    tasks: [planner, doer-name, reviewer-name]
` + "```" + `

Rules:
- Use ` + "`prompt-string`" + ` with YAML block scalar (` + "`|`" + `) to inline multi-line prompts directly in the YAML file
- Do NOT create separate prompt files — everything goes in swarm.yaml
- Use ` + "`depends_on`" + ` to define execution order — downstream tasks wait for upstream tasks
- Set ` + "`parallelism: 1`" + ` explicitly so users can see where to change it
- Set ` + "`concurrency: 1`" + ` on all planner tasks — only one planner should run at a time to avoid creating duplicate tasks
- Choose an appropriate number of iterations (10-20 is typical)
- Design 2-5 tasks that make sense for this template type
- Name tasks descriptively for the template (e.g. "researcher" not just "doer")

### Prompt Content Guidelines

Each task's ` + "`prompt-string`" + ` tells the agent exactly what to do.

#### How the runtime works:

When the pipeline runs, each agent automatically receives:
- **SWARM_STATE_DIR**: A shared directory unique to the current pipeline iteration. Agents read previous task outputs and write their own results here. This is injected automatically — prompts don't need to mention the path.
- **SWARM_AGENT_ID**: A unique ID for each agent instance.
- **Iteration number**: Which iteration of the pipeline this is.

Between tasks, you can reference another task's output using:
- ` + "`{{output:task_name}}`" + ` — This directive is replaced at runtime with the full contents of what the named task wrote to SWARM_STATE_DIR.

For example, if the planner writes its output, the doer prompt can include ` + "`{{output:planner}}`" + ` to read it.

#### Task File Lifecycle (CRITICAL):

All pipelines MUST use this task file lifecycle for coordination between planner and doer agents:

1. **Planner creates** a task file in ` + "`swarm/todos/`" + ` named: ` + "`{YYYY-MM-DD-HH-MM-SS}-{taskName}.todo.md`" + `
   - Create the ` + "`swarm/todos/`" + ` directory if it doesn't exist
   - The timestamp is the current time when the planner creates the task
   - ` + "`{taskName}`" + ` is a short kebab-case name describing the task (e.g. ` + "`implement-auth`" + `, ` + "`write-chapter-3`" + `, ` + "`analyze-dataset`" + `)
   - The file contents are the full task description and instructions
   - Example: ` + "`swarm/todos/2026-02-12-14-30-00-implement-user-auth.todo.md`" + `

2. **Doer picks up** the task by renaming ` + "`.todo.md`" + ` → ` + "`.processing.md`" + `
   - Search ` + "`swarm/todos/`" + ` for any ` + "`.todo.md`" + ` file
   - Rename it in place (e.g. ` + "`swarm/todos/2026-02-12-14-30-00-implement-user-auth.processing.md`" + `)
   - This signals that work is in progress

3. **Doer completes** the task by moving the file to ` + "`swarm/done/`" + ` and renaming ` + "`.processing.md`" + ` → ` + "`.done.md`" + `
   - Create the ` + "`swarm/done/`" + ` directory if it doesn't exist
   - Move the file (e.g. ` + "`swarm/done/2026-02-12-14-30-00-implement-user-auth.done.md`" + `)
   - Append a summary of what was accomplished to the end of the file before moving it

#### The Planner/Doer Pattern (CRITICAL):

The entire pipeline revolves around this pattern:

**Planner** (first task in each iteration):
1. Read ` + "`swarm/PLAN.md`" + ` for the overall project plan
2. Review the current state of the project directory — what files exist, what's been done already
3. Review ` + "`swarm/done/`" + ` to see all previously completed tasks (` + "`.done.md`" + ` files) and avoid repeating work
4. Check ` + "`swarm/todos/`" + ` for any existing ` + "`.todo.md`" + ` or ` + "`.processing.md`" + ` files — if one exists, DO NOT create a new task (a previous task is still pending)
5. If a pending task exists, exit early without creating a new task
6. Determine what single piece of work should be done next
7. Write a specific, bite-sized task as a ` + "`{YYYY-MM-DD-HH-MM-SS}-{taskName}.todo.md`" + ` file in ` + "`swarm/todos/`" + ` (following the Task File Lifecycle above)
8. The task must be actionable and concrete, not vague
9. Each iteration the planner MUST identify DIFFERENT work (check ` + "`swarm/done/`" + ` to see what's already been completed)

**Doer(s)** (middle tasks):
1. Read ` + "`swarm/PLAN.md`" + ` for context on the overall project
2. Search ` + "`swarm/todos/`" + ` for a ` + "`.todo.md`" + ` file and read it for the task assignment
3. Rename the ` + "`.todo.md`" + ` file to ` + "`.processing.md`" + ` in ` + "`swarm/todos/`" + ` to signal work has started
4. Execute the task to completion — actually create/modify files, write content, etc.
5. Be thorough but focused — complete the single assigned task, don't try to do everything at once
6. **Test your work** — verify that what you built actually works before considering the task complete
7. When done, move the ` + "`.processing.md`" + ` file to ` + "`swarm/done/`" + ` and rename to ` + "`.done.md`" + ` (append a completion summary first)

**Reviewer** (optional final task):
1. Read ` + "`swarm/PLAN.md`" + ` for context
2. Read the most recent ` + "`.done.md`" + ` file in ` + "`swarm/done/`" + ` to review what the doer accomplished
3. Review the work for quality, correctness, and completeness
4. **Fix any issues found** — don't just report problems, actually fix them in the code/content

#### Prompt writing guidelines:

- **Keep prompts concise** — be direct and to the point, avoid verbose instructions or excessive explanation
- Start every prompt by instructing the agent to read ` + "`swarm/PLAN.md`" + `
- Make prompts specific to the template type and the user's plan
- Include clear exit conditions (when should the agent stop/consider the task done?)
- Use ` + "`{{output:task_name}}`" + ` to pass data between stages — don't hardcode paths
- Instruct planners to always check current project state before planning (to avoid repeating work)
- Keep planner output bite-sized: each task should be completable in a single agent session
- Tell doer agents to actually create/modify project files, not just describe what to do
- **Doers must test their work** before marking tasks complete
- **Reviewers must fix issues** they find, not just report them
- **CRITICAL**: All prompts MUST follow the Task File Lifecycle:
  - Planners MUST check ` + "`swarm/todos/`" + ` for existing ` + "`.todo.md`" + ` or ` + "`.processing.md`" + ` files FIRST — if found, exit without creating a new task
  - Planners MUST write tasks as ` + "`{YYYY-MM-DD-HH-MM-SS}-{taskName}.todo.md`" + ` in ` + "`swarm/todos/`" + `
  - Doers MUST search ` + "`swarm/todos/`" + ` for a ` + "`.todo.md`" + ` file and rename it to ` + "`.processing.md`" + ` when starting work
  - Doers MUST move completed tasks to ` + "`swarm/done/`" + ` as ` + "`.done.md`" + ` when finished
  - Planners MUST check ` + "`swarm/done/`" + ` to avoid repeating completed work
- Instruct planner prompts to create ` + "`swarm/todos/`" + ` directory if it doesn't exist before creating task files
- Instruct doer prompts to create ` + "`swarm/done/`" + ` directory if it doesn't exist before moving files

## Generate the files now

Based on the "` + tmpl.Name + `" template and the user's input above, create these two files:

1. ` + "`swarm/PLAN.md`" + ` — a detailed, expanded project plan (see guidelines above)
2. ` + "`swarm/swarm.yaml`" + ` with an appropriate pipeline (2-5 tasks, sensible dependencies, explicit parallelism: 1, concurrency: 1 on all planner tasks, and all prompts inlined as prompt-string values)

Create PLAN.md FIRST, then create swarm.yaml. The prompts in swarm.yaml should reference PLAN.md and be specific and tailored to the user's plan. The pipeline should be practical and effective for this kind of work. Don't be generic — reference the specific goals from the user's plan in your prompts.

IMPORTANT: Only create swarm/PLAN.md and swarm/swarm.yaml. Do not create any other files (no prompt files, no additional directories), and do not run any commands. Everything pipeline-related goes into the single swarm/swarm.yaml file using prompt-string for each task.
`)

	return sb.String()
}
