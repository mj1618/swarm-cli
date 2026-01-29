package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/matt/swarm-cli/internal/compose"
	"github.com/matt/swarm-cli/internal/state"
)

func TestComposeLogsCommandFlags(t *testing.T) {
	cmd := composeLogsCmd

	// Test compose-file flag
	fileFlag := cmd.Flags().Lookup("compose-file")
	if fileFlag == nil {
		t.Error("expected 'compose-file' flag to exist")
	} else {
		if fileFlag.Shorthand != "c" {
			t.Errorf("compose-file flag shorthand = %q, want %q", fileFlag.Shorthand, "c")
		}
		if fileFlag.DefValue != "./swarm/swarm.yaml" {
			t.Errorf("compose-file flag default = %q, want %q", fileFlag.DefValue, "./swarm/swarm.yaml")
		}
	}

	// Test follow flag
	followFlag := cmd.Flags().Lookup("follow")
	if followFlag == nil {
		t.Error("expected 'follow' flag to exist")
	} else {
		if followFlag.Shorthand != "f" {
			t.Errorf("follow flag shorthand = %q, want %q", followFlag.Shorthand, "f")
		}
		if followFlag.DefValue != "false" {
			t.Errorf("follow flag default = %q, want %q", followFlag.DefValue, "false")
		}
	}

	// Test tail flag
	tailFlag := cmd.Flags().Lookup("tail")
	if tailFlag == nil {
		t.Error("expected 'tail' flag to exist")
	} else {
		if tailFlag.DefValue != "50" {
			t.Errorf("tail flag default = %q, want %q", tailFlag.DefValue, "50")
		}
	}

	// Test pretty flag
	prettyFlag := cmd.Flags().Lookup("pretty")
	if prettyFlag == nil {
		t.Error("expected 'pretty' flag to exist")
	} else {
		if prettyFlag.Shorthand != "P" {
			t.Errorf("pretty flag shorthand = %q, want %q", prettyFlag.Shorthand, "P")
		}
	}

	// Test since flag
	sinceFlag := cmd.Flags().Lookup("since")
	if sinceFlag == nil {
		t.Error("expected 'since' flag to exist")
	}

	// Test until flag
	untilFlag := cmd.Flags().Lookup("until")
	if untilFlag == nil {
		t.Error("expected 'until' flag to exist")
	}

	// Test grep flag
	grepFlag := cmd.Flags().Lookup("grep")
	if grepFlag == nil {
		t.Error("expected 'grep' flag to exist")
	}

	// Test invert flag
	invertFlag := cmd.Flags().Lookup("invert")
	if invertFlag == nil {
		t.Error("expected 'invert' flag to exist")
	}

	// Test case-sensitive flag
	caseFlag := cmd.Flags().Lookup("case-sensitive")
	if caseFlag == nil {
		t.Error("expected 'case-sensitive' flag to exist")
	}
}

func TestComposeLogsCommandUsage(t *testing.T) {
	cmd := composeLogsCmd

	if cmd.Use != "compose-logs [task...]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "compose-logs [task...]")
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

func TestComposeLogsComposeFileIntegration(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "compose-logs-integration-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a valid compose file
	composeContent := `version: "1"
tasks:
  frontend:
    prompt-string: "Frontend task"
    iterations: 2
  backend:
    prompt-string: "Backend task"
    model: sonnet-4.5
    name: api-server
`
	composePath := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	// Load and validate - this tests the compose file loading logic used by compose-logs
	cf, err := compose.Load(composePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if err := cf.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// Test effective names mapping (used for agent matching)
	tasks, _ := cf.GetTasks(nil)
	effectiveNames := make(map[string]string)
	for taskName, task := range tasks {
		effectiveNames[task.EffectiveName(taskName)] = taskName
	}

	// Verify expected effective names
	if effectiveNames["frontend"] != "frontend" {
		t.Errorf("expected effective name 'frontend' to map to 'frontend'")
	}
	if effectiveNames["api-server"] != "backend" {
		t.Errorf("expected effective name 'api-server' to map to 'backend'")
	}
}

func TestComposeLogsEffectiveNamesMatching(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "compose-logs-matching-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a compose file with various name configurations
	composeContent := `version: "1"
tasks:
  frontend:
    prompt-string: "Frontend task"
  backend:
    prompt-string: "Backend task"
    name: api-server
  worker:
    prompt-string: "Worker task"
    name: background-worker
`
	composePath := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	cf, err := compose.Load(composePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	tasks, err := cf.GetTasks(nil)
	if err != nil {
		t.Fatalf("GetTasks() error = %v", err)
	}

	// Build effective names map (same logic as compose-logs command)
	effectiveNames := make(map[string]string)
	for taskName, task := range tasks {
		effectiveNames[task.EffectiveName(taskName)] = taskName
	}

	// Verify expected effective names
	expectedMappings := map[string]string{
		"frontend":          "frontend",
		"api-server":        "backend",
		"background-worker": "worker",
	}
	for effName, taskKey := range expectedMappings {
		if effectiveNames[effName] != taskKey {
			t.Errorf("expected effective name %q to map to %q, got %q", effName, taskKey, effectiveNames[effName])
		}
	}

	// Verify task names without custom names use the key
	unexpectedNames := []string{"backend", "worker"}
	for _, name := range unexpectedNames {
		if effectiveNames[name] != "" {
			t.Errorf("did not expect effective name %q (should use custom name)", name)
		}
	}
}

func TestComposeLogsTaskFiltering(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "compose-logs-filter-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a compose file with multiple tasks
	composeContent := `version: "1"
tasks:
  web:
    prompt-string: "Web server"
  api:
    prompt-string: "API server"
  db:
    prompt-string: "Database worker"
`
	composePath := filepath.Join(tmpDir, "swarm.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("failed to write compose file: %v", err)
	}

	cf, err := compose.Load(composePath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Test filtering to specific tasks
	filtered, err := cf.GetTasks([]string{"web", "api"})
	if err != nil {
		t.Fatalf("GetTasks() error = %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("GetTasks() returned %d tasks, want 2", len(filtered))
	}
	if _, ok := filtered["web"]; !ok {
		t.Error("GetTasks() should contain web")
	}
	if _, ok := filtered["api"]; !ok {
		t.Error("GetTasks() should contain api")
	}
	if _, ok := filtered["db"]; ok {
		t.Error("GetTasks() should not contain db")
	}

	// Test filtering with nonexistent task
	_, err = cf.GetTasks([]string{"nonexistent"})
	if err == nil {
		t.Error("GetTasks() should return error for nonexistent task")
	}
}

func TestReadLastLines(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "compose-logs-readlines-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a log file with multiple lines
	logContent := `2024-01-28 10:00:00 | Line 1
2024-01-28 10:00:01 | Line 2
2024-01-28 10:00:02 | Line 3
2024-01-28 10:00:03 | Line 4
2024-01-28 10:00:04 | Line 5
`
	logPath := filepath.Join(tmpDir, "test.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0644); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	// Test reading last 3 lines
	lines, err := readLastLines(logPath, 3, time.Time{}, time.Time{}, nil, false, false, false)
	if err != nil {
		t.Fatalf("readLastLines() error = %v", err)
	}
	if len(lines) != 3 {
		t.Errorf("readLastLines() returned %d lines, want 3", len(lines))
	}
	if lines[0] != "2024-01-28 10:00:02 | Line 3" {
		t.Errorf("first line = %q, want %q", lines[0], "2024-01-28 10:00:02 | Line 3")
	}

	// Test reading more lines than exist
	lines, err = readLastLines(logPath, 100, time.Time{}, time.Time{}, nil, false, false, false)
	if err != nil {
		t.Fatalf("readLastLines() error = %v", err)
	}
	if len(lines) != 5 {
		t.Errorf("readLastLines() returned %d lines, want 5", len(lines))
	}

	// Test with time filter
	since, _ := ParseTimeFlag("2024-01-28 10:00:02")
	lines, err = readLastLines(logPath, 100, since, time.Time{}, nil, false, true, false)
	if err != nil {
		t.Fatalf("readLastLines() error = %v", err)
	}
	if len(lines) != 3 {
		t.Errorf("readLastLines() with time filter returned %d lines, want 3", len(lines))
	}
}

func TestTimestampedLineSorting(t *testing.T) {
	// Test that lines are sorted correctly by timestamp
	lines := []timestampedLine{
		{line: "Line B", timestamp: mustParseTime("2024-01-28 10:00:02"), agentName: "agent1"},
		{line: "Line A", timestamp: mustParseTime("2024-01-28 10:00:01"), agentName: "agent2"},
		{line: "Line C", timestamp: mustParseTime("2024-01-28 10:00:03"), agentName: "agent1"},
		{line: "Line no timestamp", timestamp: time.Time{}, agentName: "agent2"},
	}

	// Sort using the same logic as showComposeLogs
	for i := 0; i < len(lines)-1; i++ {
		for j := i + 1; j < len(lines); j++ {
			ti, tj := lines[i].timestamp, lines[j].timestamp
			shouldSwap := false
			if ti.IsZero() && !tj.IsZero() {
				shouldSwap = true
			} else if !ti.IsZero() && !tj.IsZero() && ti.After(tj) {
				shouldSwap = true
			}
			if shouldSwap {
				lines[i], lines[j] = lines[j], lines[i]
			}
		}
	}

	// Verify order
	expectedOrder := []string{"Line A", "Line B", "Line C", "Line no timestamp"}
	for i, expected := range expectedOrder {
		if lines[i].line != expected {
			t.Errorf("lines[%d] = %q, want %q", i, lines[i].line, expected)
		}
	}
}

func TestAgentMatching(t *testing.T) {
	// Test the agent matching logic used by compose-logs
	workingDir := "/test/project"

	// Create mock agents
	agents := []*state.AgentState{
		{ID: "1", Name: "frontend", WorkingDir: workingDir, LogFile: "/logs/1.log"},
		{ID: "2", Name: "backend", WorkingDir: workingDir, LogFile: "/logs/2.log"},
		{ID: "3", Name: "frontend", WorkingDir: "/other/project", LogFile: "/logs/3.log"},
		{ID: "4", Name: "unrelated", WorkingDir: workingDir, LogFile: "/logs/4.log"},
		{ID: "5", Name: "no-logs", WorkingDir: workingDir, LogFile: ""},
	}

	// Effective names from compose file
	effectiveNames := map[string]string{
		"frontend": "frontend",
		"backend":  "backend",
	}

	// Filter for matching agents (same logic as compose-logs)
	var matchingAgents []*state.AgentState
	for _, agent := range agents {
		if agent.WorkingDir == workingDir && effectiveNames[agent.Name] != "" && agent.LogFile != "" {
			matchingAgents = append(matchingAgents, agent)
		}
	}

	if len(matchingAgents) != 2 {
		t.Errorf("expected 2 matching agents, got %d", len(matchingAgents))
	}

	// Verify correct agents matched
	matchedIDs := make(map[string]bool)
	for _, a := range matchingAgents {
		matchedIDs[a.ID] = true
	}

	if !matchedIDs["1"] {
		t.Error("expected agent 1 (frontend in correct dir) to match")
	}
	if !matchedIDs["2"] {
		t.Error("expected agent 2 (backend in correct dir) to match")
	}
	if matchedIDs["3"] {
		t.Error("agent 3 (different working dir) should not match")
	}
	if matchedIDs["4"] {
		t.Error("agent 4 (unrelated name) should not match")
	}
	if matchedIDs["5"] {
		t.Error("agent 5 (no log file) should not match")
	}
}

// Helper function for parsing test times
func mustParseTime(s string) time.Time {
	t, err := ParseTimeFlag(s)
	if err != nil {
		panic(err)
	}
	return t
}
