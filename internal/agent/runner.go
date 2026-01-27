package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/matt/swarm-cli/internal/logparser"
)

// Runner manages the execution of an agent process.
type Runner struct {
	config Config
	cmd    *exec.Cmd
}

// NewRunner creates a new agent runner with the given configuration.
func NewRunner(cfg Config) *Runner {
	return &Runner{
		config: cfg,
	}
}

// Run executes the agent and streams output to the given writer.
// If RawOutput is false, output is passed through the log parser for pretty printing.
// If RawOutput is true, output is streamed directly (for Claude Code).
func (r *Runner) Run(out io.Writer) error {
	return r.RunWithContext(context.Background(), out)
}

// RunWithContext executes the agent with the given context for cancellation/timeout.
// If RawOutput is false, output is passed through the log parser for pretty printing.
// If RawOutput is true, output is streamed directly (for Claude Code).
func (r *Runner) RunWithContext(ctx context.Context, out io.Writer) error {
	// Set up context with timeout if configured
	if r.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.config.Timeout)
		defer cancel()
	}

	// Expand placeholders in command args
	args := r.config.Command.ExpandArgs(r.config.Model, r.config.Prompt)
	r.cmd = exec.CommandContext(ctx, r.config.Command.Executable, args...)

	// Apply custom environment variables if specified
	// Inherit parent environment and append custom vars (later values override earlier)
	if len(r.config.Env) > 0 {
		r.cmd.Env = append(os.Environ(), r.config.Env...)
	}

	// Set up pipes
	stdout, err := r.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := r.cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Start the command
	if err := r.cmd.Start(); err != nil {
		return err
	}

	// Process stdout based on RawOutput setting
	if r.config.Command.RawOutput {
		// Direct streaming for Claude Code - continuous stream without parsing
		go func() {
			io.Copy(out, stdout)
		}()
	} else {
		// Parsed output for Cursor agent
		parser := logparser.NewParser(out)
		go func() {
			scanner := bufio.NewScanner(stdout)
			// Increase buffer size for potentially long lines
			buf := make([]byte, 0, 64*1024)
			scanner.Buffer(buf, 1024*1024)

			for scanner.Scan() {
				line := scanner.Text()
				parser.ProcessLine(line)
			}
			parser.Flush()
		}()
	}

	// Forward stderr directly
	go func() {
		io.Copy(os.Stderr, stderr)
	}()

	// Wait for command to complete
	err = r.cmd.Wait()

	// Check if the error was due to context cancellation/timeout
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("iteration timed out after %v", r.config.Timeout)
	}
	if ctx.Err() == context.Canceled {
		return fmt.Errorf("iteration was cancelled")
	}

	return err
}

// PID returns the process ID of the running agent, or 0 if not running.
func (r *Runner) PID() int {
	if r.cmd != nil && r.cmd.Process != nil {
		return r.cmd.Process.Pid
	}
	return 0
}

// Kill sends a signal to terminate the agent process.
func (r *Runner) Kill() error {
	if r.cmd != nil && r.cmd.Process != nil {
		return r.cmd.Process.Kill()
	}
	return nil
}
