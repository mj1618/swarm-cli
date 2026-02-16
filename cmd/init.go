package cmd

import (
	"bufio"
	_ "embed"
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

//go:embed swarm.template.yaml
var swarmTemplateYAML string

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

You are setting up a swarm-cli project. Create two files based on the user's input and the provided template.

## User's Input

`)
	sb.WriteString("```\n")
	sb.WriteString(plan)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Project Type: " + tmpl.Name + "\n\n")
	sb.WriteString(tmpl.Description + "\n\n")

	sb.WriteString(`## Create These Files

### 1. ` + "`swarm/PLAN.md`" + ` — Detailed Project Plan

Expand the user's input into a comprehensive project plan. Include:
- **Project Overview**: What is being built and why
- **Goals & Success Criteria**: What does "done" look like?
- **Scope**: What's in scope and out of scope
- **Architecture / Approach**: High-level technical or structural decisions
- **Milestones**: Logical phases ordered by priority
- **Detailed Requirements**: Specific, actionable requirements

Be thorough — agents need enough detail to work autonomously.

### 2. ` + "`swarm/swarm.yaml`" + ` — Pipeline Configuration

**Start from this template and adapt it:**

` + "```yaml\n" + swarmTemplateYAML + "```" + `

**Adaptation rules:**
- Keep the same 3-step structure (planner → developer → reviewer)
- Keep the exact todo file lifecycle (the Step 1/2/3/4 flow in each task is critical — do not change it)
- Rename "developer" to something appropriate for the project type (e.g., "researcher", "writer", "analyst")
- Add project-specific context to each prompt's header (e.g., "# Planner — Research Project")
- Add a brief description of what the agent should focus on for THIS specific project
- Adjust ` + "`iterations`" + ` if needed (default 100 is good for most projects)
- Keep ` + "`parallelism: 4`" + ` and ` + "`concurrency: 1`" + ` on planner

**Do NOT change:**
- The step structure within each task (Step 1, Step 2, etc.)
- The todo file naming convention (` + "`{YYYY-MM-DD-HH-MM-SS}-{taskName}.todo.md`" + `)
- The file transitions (` + "`.todo.md`" + ` → ` + "`.processing.md`" + ` → ` + "`.done.md`" + `)
- The directories (` + "`swarm/todos/`" + `, ` + "`swarm/done/`" + `)

## Instructions

1. Create ` + "`swarm/PLAN.md`" + ` first
2. Create ` + "`swarm/swarm.yaml`" + ` by adapting the template above
3. Only create these two files — no other files or commands
`)

	return sb.String()
}
