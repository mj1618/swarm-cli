package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/matt/swarm-cli/internal/logparser"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	logsFollow bool
	logsLines  int
	logsPretty bool
)

var logsCmd = &cobra.Command{
	Use:     "logs [agent-id-or-name]",
	Aliases: []string{"tail"},
	Short:   "View the output of a running or completed agent",
	Long: `View the log output of a detached agent.

The agent can be specified by its ID or name. Use -f to follow the output
in real-time, or --tail to specify the number of lines to show.`,
	Example: `  # Show last 50 lines of agent abc123
  swarm logs abc123

  # Follow output of agent named "myagent"
  swarm logs myagent -f

  # Show last 100 lines
  swarm logs abc123 --tail 100`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentIdentifier := args[0]

		// Create state manager with scope
		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		agent, err := mgr.GetByNameOrID(agentIdentifier)
		if err != nil {
			return fmt.Errorf("agent not found: %w", err)
		}

		if agent.LogFile == "" {
			return fmt.Errorf("agent %s was not started in detached mode (no log file)", agentIdentifier)
		}

		// Check if log file exists
		if _, err := os.Stat(agent.LogFile); os.IsNotExist(err) {
			return fmt.Errorf("log file not found: %s", agent.LogFile)
		}

		if logsFollow {
			return followFile(agent.LogFile)
		}

		return showLogLines(agent.LogFile, logsLines, nil)
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow the output in real-time")
	logsCmd.Flags().IntVar(&logsLines, "tail", 50, "Number of lines to show from the end of the logs")
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "Number of lines to show (alias for --tail)")
	logsCmd.Flags().MarkHidden("lines") // Keep -n working but prefer --tail in docs
	logsCmd.Flags().BoolVarP(&logsPretty, "pretty", "P", false, "Pretty-print log output with colors and formatting")
	rootCmd.AddCommand(logsCmd)
}

// showLogLines shows the last n lines of a file.
// If parser is provided, lines are processed through it for pretty-printing.
// If parser is nil and logsPretty is true, a new parser is created and flushed.
func showLogLines(filepath string, n int, parser *logparser.Parser) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	fileSize := stat.Size()
	if fileSize == 0 {
		fmt.Println("(log file is empty)")
		return nil
	}

	// Read the file and collect lines
	lines := make([]string, 0, n)
	scanner := bufio.NewScanner(file)

	// Use a larger buffer for potentially long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:] // Keep only last n lines
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log file: %w", err)
	}

	// Print the lines
	if logsPretty {
		ownParser := parser == nil
		if ownParser {
			parser = logparser.NewParser(os.Stdout)
		}
		for _, line := range lines {
			parser.ProcessLine(line)
		}
		if ownParser {
			parser.Flush()
		}
	} else {
		for _, line := range lines {
			fmt.Println(line)
		}
	}

	return nil
}

// followFile follows a file in real-time
func followFile(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Create parser if pretty mode is enabled - used for both initial lines and follow
	var parser *logparser.Parser
	if logsPretty {
		parser = logparser.NewParser(os.Stdout)
	}

	// First, show last few lines for context
	if err := showLogLines(filepath, logsLines, parser); err != nil {
		return err
	}

	// Seek to end of file
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}

	fmt.Println("\n--- Following log (Ctrl+C to stop) ---")

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// No new data, wait a bit
				time.Sleep(100 * time.Millisecond)
				continue
			}
			// Flush parser before returning error
			if parser != nil {
				parser.Flush()
			}
			return fmt.Errorf("error reading log file: %w", err)
		}

		if logsPretty && parser != nil {
			// Process through parser (strips the trailing newline itself)
			parser.ProcessLine(line)
		} else {
			// Print without extra newline since ReadString includes the \n
			fmt.Print(line)
		}
	}
}
