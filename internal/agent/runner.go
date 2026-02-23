package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mj1618/swarm-cli/internal/logparser"
	"github.com/mj1618/swarm-cli/internal/process"
)

const defaultResultGracePeriod = 30 * time.Second

// UsageCallback is called when usage stats are updated during agent execution.
type UsageCallback func(stats logparser.UsageStats)

// Runner manages the execution of an agent process.
type Runner struct {
	config            Config
	cmd               *exec.Cmd
	cmdMu             sync.RWMutex // protects cmd
	usageCallback     UsageCallback
	usageStats        logparser.UsageStats
	statsMu           sync.Mutex
	resultCh          chan struct{}
	resultOnce        sync.Once
	killedAfterResult int32 // atomic: set to 1 if force-killed after result event
}

// NewRunner creates a new agent runner with the given configuration.
func NewRunner(cfg Config) *Runner {
	return &Runner{
		config:   cfg,
		resultCh: make(chan struct{}),
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

	// Set up process attributes for proper process group handling.
	// This allows ForceKill to terminate the entire process group including child processes.
	setProcAttr(r.cmd)
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

	// WaitGroup to ensure all output is consumed before cmd.Wait() closes pipes.
	// Per Go docs, cmd.Wait() closes StdoutPipe/StderrPipe, so all reads must
	// complete first to avoid losing data.
	var outputWg sync.WaitGroup

	// Grace period: if we see a result event but the process doesn't exit,
	// force-kill after a timeout to handle stuck child processes (e.g. dev servers).
	graceCtx, graceCancel := context.WithCancel(ctx)
	defer graceCancel()

	gracePeriod := r.config.ResultGracePeriod
	if gracePeriod == 0 {
		gracePeriod = defaultResultGracePeriod
	}
	if gracePeriod > 0 {
		go func() {
			select {
			case <-r.resultCh:
				select {
				case <-time.After(gracePeriod):
					atomic.StoreInt32(&r.killedAfterResult, 1)
					fmt.Fprintf(out, "\n[swarm] Agent completed but process still running after %v — killing stuck process\n", gracePeriod)
					r.cmdMu.RLock()
					if r.cmd != nil && r.cmd.Process != nil {
						process.ForceKill(r.cmd.Process.Pid)
					}
					r.cmdMu.RUnlock()
				case <-graceCtx.Done():
				}
			case <-graceCtx.Done():
			}
		}()
	}

	// Process stdout based on RawOutput setting
	if r.config.Command.RawOutput {
		// Direct streaming for Claude Code — tee stdout to parse for usage
		// stats and detect result events while streaming raw output.
		outputWg.Add(1)
		go func() {
			defer outputWg.Done()
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
		outputWg.Add(1)
		go func() {
			defer outputWg.Done()
			scanner := bufio.NewScanner(stdout)
			buf := make([]byte, 0, 64*1024)
			scanner.Buffer(buf, 1024*1024)

			for scanner.Scan() {
				line := scanner.Text()
				parser.ProcessLine(line)
				if event := logparser.ParseEvent(line); event != nil {
					if event.Type == "result" || event.Type == "turn.completed" {
						r.resultOnce.Do(func() { close(r.resultCh) })
					}
				}
			}
			parser.Flush()
		}()
	}

	// Forward stderr directly
	outputWg.Add(1)
	go func() {
		defer outputWg.Done()
		io.Copy(os.Stderr, stderr)
	}()

	// Wait for all output goroutines to finish reading before calling cmd.Wait(),
	// which closes the pipes and could cause data loss.
	outputWg.Wait()

	// Wait for command to complete and release resources
	err = r.cmd.Wait()

	// If we force-killed after a result event, the agent completed successfully
	// but had a stuck child process — treat as success.
	if atomic.LoadInt32(&r.killedAfterResult) == 1 {
		return nil
	}

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

	if event.Type == "result" || event.Type == "turn.completed" {
		r.resultOnce.Do(func() { close(r.resultCh) })
	}

	r.statsMu.Lock()

	updated := false

	// Find usage from the best available location:
	// 1. Top-level usage (result events, turn.completed)
	// 2. message.usage (assistant events)
	usage := event.Usage
	if usage == nil && event.Message != nil {
		usage = event.Message.Usage
	}

	if usage != nil {
		inputTokens := usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens + usage.CachedInputTokens
		if inputTokens == 0 {
			inputTokens = usage.PromptTokens
		}
		outputTokens := usage.OutputTokens
		if outputTokens == 0 {
			outputTokens = usage.CompletionTokens
		}
		if inputTokens > 0 || outputTokens > 0 {
			r.usageStats.InputTokens += inputTokens
			r.usageStats.OutputTokens += outputTokens
			updated = true
		}
	}

	// Capture total_cost_usd from result events (Claude CLI calculates this accurately)
	if event.TotalCostUSD != nil && *event.TotalCostUSD > 0 {
		r.usageStats.TotalCostUSD += *event.TotalCostUSD
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
