package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/mj1618/swarm-cli/internal/compose"
	"github.com/mj1618/swarm-cli/internal/logparser"
	"github.com/mj1618/swarm-cli/internal/output"
	"github.com/mj1618/swarm-cli/internal/scope"
	"github.com/mj1618/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	composeLogsFile         string
	composeLogsFollow       bool
	composeLogsTail         int
	composeLogsPretty       bool
	composeLogsSince        string
	composeLogsUntil        string
	composeLogsGrep         []string
	composeLogsGrepInvert   bool
	composeLogsGrepCase     bool
)

var composeLogsCmd = &cobra.Command{
	Use:   "compose-logs [task...]",
	Short: "View logs from compose tasks",
	Long: `View aggregated logs from agents started via 'swarm up -d'.

This command displays logs from all tasks defined in a compose file,
using the same colored, prefixed output format as 'swarm up' foreground mode.

By default, shows the last 50 lines from each agent. Use --tail to adjust
the number of lines per agent, or -f to follow logs in real-time.

Agents are matched by name and working directory to ensure only agents
started from the specified compose file are shown.`,
	Example: `  # View logs from all compose tasks
  swarm compose-logs

  # View specific tasks only  
  swarm compose-logs frontend backend

  # Follow logs in real-time (Ctrl-C to stop, agents keep running)
  swarm compose-logs -f

  # Show more lines per agent
  swarm compose-logs --tail 100

  # Filter logs by pattern
  swarm compose-logs --grep error

  # Show logs from the last 30 minutes
  swarm compose-logs --since 30m

  # Use a custom compose file
  swarm compose-logs -c custom.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load compose file
		cf, err := compose.Load(composeLogsFile)
		if err != nil {
			return fmt.Errorf("failed to load compose file %s: %w", composeLogsFile, err)
		}

		// Validate compose file
		if err := cf.Validate(); err != nil {
			return fmt.Errorf("invalid compose file: %w", err)
		}

		// Get tasks (filtered by args if provided)
		tasks, err := cf.GetTasks(args)
		if err != nil {
			return err
		}

		// Get current working directory
		workingDir, err := scope.CurrentWorkingDir()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Build set of effective task names
		effectiveNames := make(map[string]string) // effective name -> task key
		for taskName, task := range tasks {
			effectiveNames[task.EffectiveName(taskName)] = taskName
		}

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		// List all agents (including terminated ones, they may have logs)
		agents, err := mgr.List(false) // false = include terminated
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}

		// Filter for agents that match our compose file tasks and have log files
		var matchingAgents []*state.AgentState
		for _, agent := range agents {
			if agent.WorkingDir == workingDir && effectiveNames[agent.Name] != "" && agent.LogFile != "" {
				matchingAgents = append(matchingAgents, agent)
			}
		}

		if len(matchingAgents) == 0 {
			fmt.Println("No matching agents with logs found")
			fmt.Println("Hint: Use 'swarm up -d' to start agents in detached mode")
			return nil
		}

		// Parse time flags
		var sinceTime, untilTime time.Time
		if composeLogsSince != "" {
			sinceTime, err = ParseTimeFlag(composeLogsSince)
			if err != nil {
				return fmt.Errorf("invalid --since format: %w", err)
			}
		}
		if composeLogsUntil != "" {
			untilTime, err = ParseTimeFlag(composeLogsUntil)
			if err != nil {
				return fmt.Errorf("invalid --until format: %w", err)
			}
		}

		// Validate since is before until
		if !sinceTime.IsZero() && !untilTime.IsZero() && sinceTime.After(untilTime) {
			return fmt.Errorf("--since time must be before --until time")
		}

		// Compile grep patterns
		var grepPatterns []*regexp.Regexp
		for _, pattern := range composeLogsGrep {
			flags := ""
			if !composeLogsGrepCase {
				flags = "(?i)"
			}
			re, err := regexp.Compile(flags + pattern)
			if err != nil {
				return fmt.Errorf("invalid grep pattern %q: %w", pattern, err)
			}
			grepPatterns = append(grepPatterns, re)
		}

		if composeLogsFollow {
			// Warn if --until is used with --follow
			if composeLogsUntil != "" {
				fmt.Println("Warning: --until is ignored when using --follow")
				untilTime = time.Time{}
			}
			return followComposeLogs(matchingAgents, sinceTime, grepPatterns, composeLogsGrepInvert)
		}

		return showComposeLogs(matchingAgents, composeLogsTail, sinceTime, untilTime, grepPatterns, composeLogsGrepInvert)
	},
}

func init() {
	composeLogsCmd.Flags().StringVarP(&composeLogsFile, "compose-file", "c", compose.DefaultPath(), "Path to compose file")
	composeLogsCmd.Flags().BoolVarP(&composeLogsFollow, "follow", "f", false, "Follow logs in real-time")
	composeLogsCmd.Flags().IntVar(&composeLogsTail, "tail", 50, "Number of lines to show per agent")
	composeLogsCmd.Flags().BoolVarP(&composeLogsPretty, "pretty", "P", false, "Pretty-print log output with colors and formatting")
	composeLogsCmd.Flags().StringVar(&composeLogsSince, "since", "", "Show logs since timestamp (e.g., 30m, 2h, 2024-01-28 10:00)")
	composeLogsCmd.Flags().StringVar(&composeLogsUntil, "until", "", "Show logs until timestamp (e.g., 1h, 2024-01-28 12:00)")
	composeLogsCmd.Flags().StringArrayVar(&composeLogsGrep, "grep", nil, "Filter lines matching pattern (regex, case-insensitive by default)")
	composeLogsCmd.Flags().BoolVar(&composeLogsGrepInvert, "invert", false, "Invert match (show non-matching lines)")
	composeLogsCmd.Flags().BoolVar(&composeLogsGrepCase, "case-sensitive", false, "Make grep pattern case-sensitive")
	rootCmd.AddCommand(composeLogsCmd)
}

// timestampedLine holds a log line with its parsed timestamp and source agent.
type timestampedLine struct {
	line      string
	timestamp time.Time
	agentName string
}

// showComposeLogs displays merged historical logs from all matching agents.
func showComposeLogs(agents []*state.AgentState, tailLines int, since, until time.Time, grepPatterns []*regexp.Regexp, invert bool) error {
	hasTimeFilter := !since.IsZero() || !until.IsZero()
	hasGrepFilter := len(grepPatterns) > 0

	// Collect lines from all agents
	var allLines []timestampedLine

	for _, agent := range agents {
		lines, err := readLastLines(agent.LogFile, tailLines, since, until, grepPatterns, invert, hasTimeFilter, hasGrepFilter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read logs for %s: %v\n", agent.Name, err)
			continue
		}

		for _, line := range lines {
			ts := ExtractTimestamp(line)
			allLines = append(allLines, timestampedLine{
				line:      line,
				timestamp: ts,
				agentName: agent.Name,
			})
		}
	}

	if len(allLines) == 0 {
		if hasTimeFilter || hasGrepFilter {
			fmt.Println("(no matching log lines)")
		} else {
			fmt.Println("(no log lines)")
		}
		return nil
	}

	// Sort by timestamp (lines without timestamps go to the end)
	sort.SliceStable(allLines, func(i, j int) bool {
		ti, tj := allLines[i].timestamp, allLines[j].timestamp
		if ti.IsZero() && tj.IsZero() {
			return false // Keep original order for lines without timestamps
		}
		if ti.IsZero() {
			return false // Lines without timestamps go after
		}
		if tj.IsZero() {
			return true // Lines with timestamps go before
		}
		return ti.Before(tj)
	})

	// Create writer group for colored output
	agentNames := make([]string, 0, len(agents))
	for _, agent := range agents {
		agentNames = append(agentNames, agent.Name)
	}
	sort.Strings(agentNames)
	writers := output.NewWriterGroup(os.Stdout, agentNames)

	// Create parsers for pretty printing if enabled
	var parsers map[string]*logparser.Parser
	if composeLogsPretty {
		parsers = make(map[string]*logparser.Parser)
		for _, name := range agentNames {
			parsers[name] = logparser.NewParser(writers.Get(name))
		}
	}

	// Output lines with prefixes
	for _, tl := range allLines {
		writer := writers.Get(tl.agentName)
		if writer == nil {
			continue
		}

		if composeLogsPretty && parsers[tl.agentName] != nil {
			parsers[tl.agentName].ProcessLine(tl.line)
		} else {
			fmt.Fprintln(writer, tl.line)
		}
	}

	// Flush all writers and parsers
	if composeLogsPretty {
		for _, parser := range parsers {
			parser.Flush()
		}
	}
	writers.FlushAll()

	return nil
}

// readLastLines reads the last n lines from a file, applying filters.
func readLastLines(filepath string, n int, since, until time.Time, grepPatterns []*regexp.Regexp, invert, hasTimeFilter, hasGrepFilter bool) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var lines []string
	for scanner.Scan() {
		line := scanner.Text()

		// Apply time filter
		if hasTimeFilter && !IsLineInTimeRange(line, since, until) {
			continue
		}

		// Apply grep filter
		if hasGrepFilter && !MatchesGrep(line, grepPatterns, invert) {
			continue
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Keep last n lines
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	return lines, nil
}

// followComposeLogs follows logs from all matching agents in real-time.
func followComposeLogs(agents []*state.AgentState, since time.Time, grepPatterns []*regexp.Regexp, invert bool) error {
	// Create writer group for colored output
	agentNames := make([]string, 0, len(agents))
	for _, agent := range agents {
		agentNames = append(agentNames, agent.Name)
	}
	sort.Strings(agentNames)
	writers := output.NewWriterGroup(os.Stdout, agentNames)

	// Create parsers for pretty printing if enabled
	var parsers map[string]*logparser.Parser
	if composeLogsPretty {
		parsers = make(map[string]*logparser.Parser)
		for _, name := range agentNames {
			parsers[name] = logparser.NewParser(writers.Get(name))
		}
	}

	// Show initial lines for context
	fmt.Printf("Showing logs from %d agent(s)...\n\n", len(agents))
	
	// Show last few lines from each agent for context
	for _, agent := range agents {
		lines, err := readLastLines(agent.LogFile, composeLogsTail, since, time.Time{}, grepPatterns, invert, !since.IsZero(), len(grepPatterns) > 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read logs for %s: %v\n", agent.Name, err)
			continue
		}

		writer := writers.Get(agent.Name)
		for _, line := range lines {
			if composeLogsPretty && parsers[agent.Name] != nil {
				parsers[agent.Name].ProcessLine(line)
			} else {
				fmt.Fprintln(writer, line)
			}
		}
	}

	fmt.Println("\n--- Following logs (Ctrl+C to stop, agents keep running) ---")

	// Start a goroutine for each agent to tail its log file
	var wg sync.WaitGroup
	
	for _, agent := range agents {
		wg.Add(1)
		go func(a *state.AgentState) {
			defer wg.Done()
			tailAgentLog(a, writers.Get(a.Name), parsers[a.Name], since, grepPatterns, invert)
		}(agent)
	}

	// Wait forever (until Ctrl+C)
	wg.Wait()

	return nil
}

// tailAgentLog tails a single agent's log file, writing to the prefixed writer.
func tailAgentLog(agent *state.AgentState, writer *output.PrefixedWriter, parser *logparser.Parser, since time.Time, grepPatterns []*regexp.Regexp, invert bool) {
	file, err := os.Open(agent.LogFile)
	if err != nil {
		fmt.Fprintf(writer, "Error opening log file: %v\n", err)
		return
	}
	defer file.Close()

	// Seek to end of file
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		fmt.Fprintf(writer, "Error seeking log file: %v\n", err)
		return
	}

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// No new data, wait a bit
				time.Sleep(100 * time.Millisecond)
				continue
			}
			// Other error, stop tailing this file
			return
		}

		// Remove trailing newline for processing
		line = line[:len(line)-1]

		// Apply time filter
		if !since.IsZero() && !IsLineInTimeRange(line, since, time.Time{}) {
			continue
		}

		// Apply grep filter
		if len(grepPatterns) > 0 && !MatchesGrep(line, grepPatterns, invert) {
			continue
		}

		if composeLogsPretty && parser != nil {
			parser.ProcessLine(line)
		} else {
			fmt.Fprintln(writer, line)
		}
	}
}
