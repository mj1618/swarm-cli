package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	tailFollow bool
	tailLines  int
)

var tailCmd = &cobra.Command{
	Use:   "tail [agent-id-or-name]",
	Short: "View the output of a running or completed agent",
	Long: `View the log output of a detached agent.

The agent can be specified by its ID or name. Use -f to follow the output
in real-time (like tail -f), or -n to specify the number of lines to show.`,
	Example: `  # Show last 50 lines of agent abc123
  swarm tail abc123

  # Follow output of agent named "myagent"
  swarm tail myagent -f

  # Show last 100 lines
  swarm tail abc123 -n 100`,
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

		if tailFollow {
			return followFile(agent.LogFile)
		}

		return tailFile(agent.LogFile, tailLines)
	},
}

func init() {
	tailCmd.Flags().BoolVarP(&tailFollow, "follow", "f", false, "Follow the output (like tail -f)")
	tailCmd.Flags().IntVarP(&tailLines, "lines", "n", 50, "Number of lines to show")
	rootCmd.AddCommand(tailCmd)
}

// tailFile shows the last n lines of a file
func tailFile(filepath string, n int) error {
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
	for _, line := range lines {
		fmt.Println(line)
	}

	return nil
}

// followFile follows a file like tail -f
func followFile(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// First, show last few lines for context
	if err := tailFile(filepath, tailLines); err != nil {
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
			return fmt.Errorf("error reading log file: %w", err)
		}

		// Print without extra newline since ReadString includes the \n
		fmt.Print(line)
	}
}
