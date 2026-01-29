package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/fatih/color"
	"github.com/matt/swarm-cli/internal/process"
	"github.com/matt/swarm-cli/internal/state"
	"github.com/spf13/cobra"
)

var (
	attachNoInteractive bool
	attachTail          int
)

var attachCmd = &cobra.Command{
	Use:   "attach [process-id-or-name]",
	Short: "Attach to a running agent for interactive monitoring",
	Long: `Attach to a running detached agent for interactive monitoring.

Shows a status header with agent info that updates in real-time,
follows log output, and provides keyboard shortcuts for control.

The agent can be specified by its ID, name, or special identifier:
  - @last or _ : the most recently started agent

Press 'q' or Ctrl+C to detach without killing the agent.`,
	Example: `  # Attach to agent by ID
  swarm attach abc123

  # Attach to agent by name
  swarm attach my-agent

  # Attach to the most recent agent
  swarm attach @last
  swarm attach _

  # Attach without keyboard controls
  swarm attach my-agent --no-interactive

  # Show last 100 lines when attaching
  swarm attach my-agent --tail 100`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentIdentifier := args[0]

		mgr, err := state.NewManagerWithScope(GetScope(), "")
		if err != nil {
			return fmt.Errorf("failed to initialize state manager: %w", err)
		}

		agent, err := ResolveAgentIdentifier(mgr, agentIdentifier)
		if err != nil {
			return fmt.Errorf("agent not found: %w", err)
		}

		if agent.Status != "running" {
			return fmt.Errorf("agent is not running (status: %s)", agent.Status)
		}

		if agent.LogFile == "" {
			return fmt.Errorf("agent was not started in detached mode (no log file)")
		}

		// Check if log file exists
		if _, err := os.Stat(agent.LogFile); os.IsNotExist(err) {
			return fmt.Errorf("log file not found: %s", agent.LogFile)
		}

		if attachNoInteractive {
			return attachNonInteractive(mgr, agent)
		}

		return attachInteractive(mgr, agent)
	},
}

func attachInteractive(mgr *state.Manager, agent *state.AgentState) error {
	// Initialize keyboard
	if err := keyboard.Open(); err != nil {
		// Fall back to non-interactive if keyboard fails
		fmt.Println("Warning: keyboard input unavailable, running in non-interactive mode")
		return attachNonInteractive(mgr, agent)
	}
	defer keyboard.Close()

	// Clear screen and print initial header
	fmt.Print("\033[2J\033[H")
	printAttachStatusHeader(agent)
	printAttachHelpLine()

	// Open log file
	file, err := os.Open(agent.LogFile)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Show last N lines
	if err := showLastLinesAttach(file, attachTail); err != nil {
		return err
	}

	// Seek to end for following
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}

	// Channels for coordination
	done := make(chan struct{})
	logLines := make(chan string, 100)

	// Goroutine: read log file
	go func() {
		reader := bufio.NewReader(file)
		for {
			select {
			case <-done:
				return
			default:
				line, err := reader.ReadString('\n')
				if err == io.EOF {
					time.Sleep(100 * time.Millisecond)
					continue
				}
				if err != nil {
					return
				}
				select {
				case logLines <- line:
				case <-done:
					return
				}
			}
		}
	}()

	// Goroutine: update status periodically
	statusTicker := time.NewTicker(2 * time.Second)
	defer statusTicker.Stop()

	// Non-blocking keyboard check channel
	keyChan := make(chan keyboard.Key, 10)
	charChan := make(chan rune, 10)

	go func() {
		for {
			char, key, err := keyboard.GetKey()
			if err != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			select {
			case charChan <- char:
			case <-done:
				return
			}
			select {
			case keyChan <- key:
			case <-done:
				return
			}
		}
	}()

	// Main loop
	for {
		select {
		case line := <-logLines:
			fmt.Print(line)

		case <-statusTicker.C:
			// Refresh agent state
			updated, err := mgr.Get(agent.ID)
			if err == nil {
				agent = updated
				// Refresh status header in place
				refreshAttachStatusHeader(agent)
			}
			// Check if terminated
			if agent.Status == "terminated" {
				close(done)
				fmt.Println("\n[swarm] Agent terminated")
				return nil
			}

		case char := <-charChan:
			<-keyChan // consume corresponding key

			if char == 'q' {
				close(done)
				fmt.Println("\n[swarm] Detached from agent (agent still running)")
				return nil
			}

			switch char {
			case 'p':
				if !agent.Paused {
					agent.Paused = true
					now := time.Now()
					agent.PausedAt = &now
					if err := mgr.Update(agent); err != nil {
						fmt.Printf("\n[swarm] Error pausing: %v\n", err)
					} else {
						fmt.Println("\n[swarm] Agent paused (will pause after current iteration)")
					}
				}
			case 'r':
				if agent.Paused {
					agent.Paused = false
					agent.PausedAt = nil
					if err := mgr.Update(agent); err != nil {
						fmt.Printf("\n[swarm] Error resuming: %v\n", err)
					} else {
						fmt.Println("\n[swarm] Agent resumed")
					}
				}
			case '+':
				agent.Iterations++
				if err := mgr.Update(agent); err != nil {
					fmt.Printf("\n[swarm] Error updating iterations: %v\n", err)
				} else {
					fmt.Printf("\n[swarm] Iterations increased to %d\n", agent.Iterations)
				}
			case '-':
				if agent.Iterations > agent.CurrentIter && agent.Iterations > 0 {
					agent.Iterations--
					if err := mgr.Update(agent); err != nil {
						fmt.Printf("\n[swarm] Error updating iterations: %v\n", err)
					} else {
						fmt.Printf("\n[swarm] Iterations decreased to %d\n", agent.Iterations)
					}
				}
			case 'k':
				fmt.Print("\n[swarm] Kill agent? (y/n): ")
				// Wait for confirmation
				confirmChar := <-charChan
				<-keyChan
				if confirmChar == 'y' || confirmChar == 'Y' {
					agent.TerminateMode = "immediate"
					if err := mgr.Update(agent); err != nil {
						fmt.Printf("\n[swarm] Error setting terminate mode: %v\n", err)
					}
					if err := process.Kill(agent.PID); err != nil {
						fmt.Printf("\n[swarm] Warning: could not send signal: %v\n", err)
					}
					close(done)
					fmt.Println("\n[swarm] Agent killed")
					return nil
				}
				fmt.Println("Cancelled")
			case 0: // Check for Ctrl+C
				// handled by keyboard library key
			}

		default:
			// Small sleep to prevent busy loop
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func attachNonInteractive(mgr *state.Manager, agent *state.AgentState) error {
	// Print initial header
	fmt.Print("\033[2J\033[H")
	printAttachStatusHeader(agent)
	fmt.Println("\nPress Ctrl+C to detach")

	// Open log file
	file, err := os.Open(agent.LogFile)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Show last N lines
	if err := showLastLinesAttach(file, attachTail); err != nil {
		return err
	}

	// Seek to end for following
	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}

	fmt.Println("\n--- Following log (Ctrl+C to stop) ---")

	reader := bufio.NewReader(file)
	statusTicker := time.NewTicker(5 * time.Second)
	defer statusTicker.Stop()

	for {
		select {
		case <-statusTicker.C:
			// Refresh agent state
			updated, err := mgr.Get(agent.ID)
			if err == nil {
				agent = updated
			}
			// Check if terminated
			if agent.Status == "terminated" {
				fmt.Println("\n[swarm] Agent terminated")
				return nil
			}
		default:
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if err != nil {
				return fmt.Errorf("error reading log file: %w", err)
			}
			fmt.Print(line)
		}
	}
}

func printAttachStatusHeader(agent *state.AgentState) {
	bold := color.New(color.Bold)

	name := agent.Name
	if name == "" {
		name = "-"
	}

	statusStr := agent.Status
	if agent.Paused {
		if agent.PausedAt != nil {
			statusStr = "paused"
		} else {
			statusStr = "pausing"
		}
	}

	iterStr := fmt.Sprintf("%d/%d", agent.CurrentIter, agent.Iterations)
	if agent.Iterations == 0 {
		iterStr = fmt.Sprintf("%d/∞", agent.CurrentIter)
	}

	bold.Printf("╭─ Agent: %s (%s) ", name, agent.ID)
	fmt.Println(strings.Repeat("─", 40) + "╮")

	fmt.Printf("│ Status: %-10s │ Iteration: %-8s │ Model: %-20s │\n",
		statusStr, iterStr, truncateString(agent.Model, 20))
	fmt.Println("╰" + strings.Repeat("─", 73) + "╯")
}

func printAttachHelpLine() {
	dim := color.New(color.Faint)
	dim.Println("Press: [p]ause  [r]esume  [+]iter  [-]iter  [k]ill  [q]uit")
	fmt.Println()
}

func refreshAttachStatusHeader(agent *state.AgentState) {
	statusStr := agent.Status
	if agent.Paused {
		if agent.PausedAt != nil {
			statusStr = "paused"
		} else {
			statusStr = "pausing"
		}
	}

	iterStr := fmt.Sprintf("%d/%d", agent.CurrentIter, agent.Iterations)
	if agent.Iterations == 0 {
		iterStr = fmt.Sprintf("%d/∞", agent.CurrentIter)
	}

	// Save cursor, move to line 2, update, restore cursor
	fmt.Print("\033[s")    // Save cursor
	fmt.Print("\033[2;1H") // Move to line 2, column 1

	fmt.Printf("│ Status: %-10s │ Iteration: %-8s │ Model: %-20s │",
		statusStr, iterStr, truncateString(agent.Model, 20))
	fmt.Print("\033[K") // Clear to end of line
	fmt.Print("\033[u") // Restore cursor
}

func showLastLinesAttach(file *os.File, n int) error {
	// Get file size
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	if stat.Size() == 0 {
		fmt.Println("(log file is empty)")
		return nil
	}

	// Read all lines
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading log file: %w", err)
	}

	// Keep last n lines
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	// Print the lines
	for _, line := range lines {
		fmt.Println(line)
	}

	return nil
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func init() {
	attachCmd.Flags().BoolVar(&attachNoInteractive, "no-interactive", false, "Disable keyboard controls")
	attachCmd.Flags().IntVar(&attachTail, "tail", 50, "Number of lines to show from the end")
	rootCmd.AddCommand(attachCmd)

	// Add dynamic completion for agent identifier
	attachCmd.ValidArgsFunction = completeAgentIdentifier
}
