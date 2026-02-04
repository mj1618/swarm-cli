package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/mj1618/swarm-cli/internal/logparser"
)

// UsageCallback is called when usage stats are updated during agent execution.
type UsageCallback func(stats logparser.UsageStats)

// Runner manages the execution of an agent process.
type Runner struct {
	config        Config
	cmd           *exec.Cmd
	cmdMu         sync.RWMutex // protects cmd
	usageCallback UsageCallback
	usageStats    logparser.UsageStats
	statsMu       sync.Mutex
}

// NewRunner creates a new agent runner with the given configuration.
func NewRunner(cfg Config) *Runner {
	return &Runner{
		config: cfg,
	}
}

// SetUsageCallback sets a callback function that is called when usage stats are updated.
func (r *Runner) SetUsageCallback(cb UsageCallback) {
	r.usageCallback = cb
}

// UsageStats returns the current usage statistics.
func (r *Runner) UsageStats() logparser.UsageStats {
	r.statsMu.Lock()
	defer r.statsMu.Unlock()
	return r.usageStats
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
	r.cmdMu.Lock()
	r.cmd = exec.CommandContext(ctx, r.config.Command.Executable, args...)
	r.cmdMu.Unlock()

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
		// Still try to extract usage stats if callback is set
		go func() {
			if r.usageCallback != nil {
				// Wrap with a tee reader to parse while streaming
				pr, pw := io.Pipe()
				go func() {
					defer pw.Close()
					io.Copy(io.MultiWriter(out, pw), stdout)
				}()
				scanner := bufio.NewScanner(pr)
				buf := make([]byte, 0, 64*1024)
				scanner.Buffer(buf, 1024*1024)
				for scanner.Scan() {
					line := scanner.Text()
					r.extractUsageFromLine(line)
				}
			} else {
				io.Copy(out, stdout)
			}
		}()
	} else {
		// Parsed output for Cursor agent with usage tracking
		parser := logparser.NewStreamingParser(out, func(stats logparser.UsageStats) {
			r.statsMu.Lock()
			r.usageStats = stats
			r.statsMu.Unlock()
			if r.usageCallback != nil {
				r.usageCallback(stats)
			}
		})
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

// extractUsageFromLine tries to extract usage info from a raw output line.
// This is used for Claude Code which outputs raw text.
func (r *Runner) extractUsageFromLine(line string) {
	event := logparser.ParseEvent(line)
	if event == nil {
		return
	}

	r.statsMu.Lock()

	updated := false

	// Extract usage from various possible locations
	if event.Usage != nil {
		inputTokens := event.Usage.InputTokens
		if inputTokens == 0 {
			inputTokens = event.Usage.PromptTokens
		}
		outputTokens := event.Usage.OutputTokens
		if outputTokens == 0 {
			outputTokens = event.Usage.CompletionTokens
		}
		if inputTokens > 0 || outputTokens > 0 {
			r.usageStats.InputTokens += inputTokens
			r.usageStats.OutputTokens += outputTokens
			updated = true
		}
	}

	// Check for direct token fields
	if event.InputTokens != nil && *event.InputTokens > 0 {
		r.usageStats.InputTokens += *event.InputTokens
		updated = true
	}
	if event.OutputTokens != nil && *event.OutputTokens > 0 {
		r.usageStats.OutputTokens += *event.OutputTokens
		updated = true
	}

	// Update current task based on event type
	var newTask string
	switch event.Type {
	case "tool_call":
		if event.ToolCall != nil {
			for toolName := range event.ToolCall {
				newTask = toolName
				break
			}
		}
	}

	if newTask != "" && newTask != r.usageStats.CurrentTask {
		r.usageStats.CurrentTask = newTask
		updated = true
	}

	// Copy stats and callback reference before releasing lock
	var statsCopy logparser.UsageStats
	var callback UsageCallback
	if updated && r.usageCallback != nil {
		statsCopy = r.usageStats
		callback = r.usageCallback
	}
	r.statsMu.Unlock()

	// Invoke callback outside of lock to prevent potential deadlock/contention
	if callback != nil {
		callback(statsCopy)
	}
}

// PID returns the process ID of the running agent, or 0 if not running.
func (r *Runner) PID() int {
	r.cmdMu.RLock()
	defer r.cmdMu.RUnlock()
	if r.cmd != nil && r.cmd.Process != nil {
		return r.cmd.Process.Pid
	}
	return 0
}

// Kill sends a signal to terminate the agent process.
func (r *Runner) Kill() error {
	r.cmdMu.RLock()
	defer r.cmdMu.RUnlock()
	if r.cmd != nil && r.cmd.Process != nil {
		return r.cmd.Process.Kill()
	}
	return nil
}
