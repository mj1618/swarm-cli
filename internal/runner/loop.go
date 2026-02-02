package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/matt/swarm-cli/internal/agent"
	"github.com/matt/swarm-cli/internal/config"
	"github.com/matt/swarm-cli/internal/logparser"
	"github.com/matt/swarm-cli/internal/prompt"
	"github.com/matt/swarm-cli/internal/state"
)

// LoopConfig holds the configuration for running the multi-iteration agent loop.
type LoopConfig struct {
	// Manager is the state manager for reading/writing agent state
	Manager *state.Manager

	// AgentState is the agent state to run and update
	AgentState *state.AgentState

	// PromptContent is the full prompt content to pass to the agent
	PromptContent string

	// Command is the agent command configuration
	Command config.CommandConfig

	// Config is the full application config (for pricing lookups)
	Config *config.Config

	// Env is the list of environment variables in KEY=VALUE format
	Env []string

	// Output is where agent output is written
	Output io.Writer

	// StartingIteration is the first iteration to run (usually 1, higher for --continue)
	StartingIteration int

	// TotalTimeout is the total timeout for all iterations (0 = no timeout)
	TotalTimeout time.Duration

	// IterTimeout is the timeout per iteration (0 = no timeout)
	IterTimeout time.Duration
}

// LoopResult contains the result of running the loop.
type LoopResult struct {
	// TimedOut is true if the loop terminated due to total timeout
	TimedOut bool
}

// RunLoop executes the multi-iteration agent loop with state management,
// signal handling, pause/resume support, and graceful termination.
// Returns when all iterations complete, termination is requested, or a signal is received.
func RunLoop(cfg LoopConfig) (*LoopResult, error) {
	mgr := cfg.Manager
	agentState := cfg.AgentState
	result := &LoopResult{}

	// Mutex to protect concurrent access to agentState fields
	var stateMu sync.Mutex

	// Set up total timeout context
	var timeoutCtx context.Context
	var timeoutCancel context.CancelFunc
	if cfg.TotalTimeout > 0 {
		timeoutCtx, timeoutCancel = context.WithTimeout(context.Background(), cfg.TotalTimeout)
		defer timeoutCancel()
	} else {
		timeoutCtx = context.Background()
	}

	// Ensure cleanup on exit
	defer func() {
		stateMu.Lock()
		if result.TimedOut {
			agentState.TimeoutReason = "total"
		}
		agentState.Status = "terminated"
		now := time.Now()
		agentState.TerminatedAt = &now
		if agentState.ExitReason == "" {
			agentState.ExitReason = "completed"
		}
		_ = mgr.MergeUpdate(agentState)

		// Execute on-complete hook (copy hook value while holding lock)
		onComplete := agentState.OnComplete
		stateMu.Unlock()

		if onComplete != "" {
			if err := agent.ExecuteOnCompleteHook(agentState); err != nil {
				fmt.Fprintf(cfg.Output, "[swarm] Warning: on-complete hook failed: %v\n", err)
			}
		}
	}()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Determine starting iteration
	startingIteration := cfg.StartingIteration
	if startingIteration <= 0 {
		startingIteration = 1
	}

	// Run iterations (0 means unlimited), starting from startingIteration
	for i := startingIteration; ; i++ {
		// Check loop condition under lock
		stateMu.Lock()
		iterations := agentState.Iterations
		stateMu.Unlock()
		if iterations != 0 && i > iterations {
			break
		}
		// Check for total timeout before starting iteration
		select {
		case <-timeoutCtx.Done():
			fmt.Fprintln(cfg.Output, "\n[swarm] Total timeout reached, stopping")
			result.TimedOut = true
			return result, nil
		default:
			// Continue
		}

		// Check for control signals from state
		stateMu.Lock()
		agentID := agentState.ID
		stateMu.Unlock()
		currentState, err := mgr.Get(agentID)
		if err == nil && currentState != nil {
			stateMu.Lock()
			// Update iterations if changed
			if currentState.Iterations != agentState.Iterations {
				agentState.Iterations = currentState.Iterations
				if agentState.Iterations == 0 {
					fmt.Fprintln(cfg.Output, "\n[swarm] Now running indefinitely")
				} else {
					fmt.Fprintf(cfg.Output, "\n[swarm] Iterations updated to %d\n", agentState.Iterations)
				}
			}

			// Update model if changed
			if currentState.Model != agentState.Model {
				agentState.Model = currentState.Model
				fmt.Fprintf(cfg.Output, "\n[swarm] Model updated to %s\n", agentState.Model)
			}

			// Check for termination
			if currentState.TerminateMode == "immediate" {
				fmt.Fprintln(cfg.Output, "\n[swarm] Received immediate termination signal")
				agentState.ExitReason = "killed"
				stateMu.Unlock()
				return result, nil
			}
			if currentState.TerminateMode == "after_iteration" && i > 1 {
				fmt.Fprintln(cfg.Output, "\n[swarm] Terminating after iteration as requested")
				agentState.ExitReason = "killed"
				stateMu.Unlock()
				return result, nil
			}

			// Check for pause state and wait while paused
			if currentState.Paused {
				fmt.Fprintln(cfg.Output, "\n[swarm] Agent paused, waiting for resume...")
				agentState.Paused = true
				now := time.Now()
				agentState.PausedAt = &now
				_ = mgr.MergeUpdate(agentState)
				stateMu.Unlock()

				for currentState.Paused && currentState.Status == "running" {
					time.Sleep(1 * time.Second)
					currentState, err = mgr.Get(agentID)
					if err != nil {
						break
					}
					// Allow termination while paused
					if currentState.TerminateMode != "" {
						if currentState.TerminateMode == "immediate" {
							fmt.Fprintln(cfg.Output, "\n[swarm] Received immediate termination signal")
							stateMu.Lock()
							agentState.ExitReason = "killed"
							stateMu.Unlock()
							return result, nil
						}
						break
					}
				}

				if !currentState.Paused {
					fmt.Fprintln(cfg.Output, "\n[swarm] Agent resumed")
					stateMu.Lock()
					agentState.Paused = false
					agentState.PausedAt = nil
					_ = mgr.MergeUpdate(agentState)
					stateMu.Unlock()
				}
			} else {
				stateMu.Unlock()
			}
		}

		// Update current iteration and get values needed for this iteration
		stateMu.Lock()
		agentState.CurrentIter = i
		_ = mgr.MergeUpdate(agentState)
		iterationsForDisplay := agentState.Iterations
		modelForConfig := agentState.Model
		stateMu.Unlock()

		if iterationsForDisplay == 0 {
			fmt.Fprintf(cfg.Output, "\n[swarm] === Iteration %d ===\n", i)
		} else {
			fmt.Fprintf(cfg.Output, "\n[swarm] === Iteration %d/%d ===\n", i, iterationsForDisplay)
		}

		// Generate a per-iteration agent ID and inject it into the prompt.
		iterationAgentID := state.GenerateID()
		iterationPrompt := prompt.InjectAgentID(cfg.PromptContent, iterationAgentID)

		// Create agent config with per-iteration timeout
		agentCfg := agent.Config{
			Model:   modelForConfig,
			Prompt:  iterationPrompt,
			Command: cfg.Command,
			Env:     cfg.Env,
			Timeout: cfg.IterTimeout,
		}

		// Run agent with usage tracking
		runner := agent.NewRunner(agentCfg)
		
		// Set up usage callback to update state
		runner.SetUsageCallback(func(stats logparser.UsageStats) {
			stateMu.Lock()
			// Update token counts (stats are cumulative within iteration)
			agentState.InputTokens = stats.InputTokens
			agentState.OutputTokens = stats.OutputTokens
			agentState.CurrentTask = stats.CurrentTask
			
			// Calculate cost if config is available
			if cfg.Config != nil {
				pricing := cfg.Config.GetPricing(agentState.Model)
				agentState.TotalCost = pricing.CalculateCost(agentState.InputTokens, agentState.OutputTokens)
			}
			
			// Update state (will be throttled by the parser's update frequency)
			_ = mgr.MergeUpdate(agentState)
			stateMu.Unlock()
		})

		// Run agent - errors should NOT stop the run (including iteration timeouts)
		if err := runner.RunWithContext(timeoutCtx, cfg.Output); err != nil {
			stateMu.Lock()
			agentState.FailedIters++
			agentState.LastError = err.Error()
			if strings.Contains(err.Error(), "timed out") {
				fmt.Fprintf(cfg.Output, "\n[swarm] Iteration %d timed out after %v (continuing)\n", i, cfg.IterTimeout)
				// Record that this iteration timed out
				agentState.TimeoutReason = "iteration"
				_ = mgr.MergeUpdate(agentState)
				// Reset timeout reason after recording (will be set to "total" if total timeout hit)
				agentState.TimeoutReason = ""
			} else {
				fmt.Fprintf(cfg.Output, "\n[swarm] Agent error (continuing): %v\n", err)
			}
			stateMu.Unlock()
		} else {
			stateMu.Lock()
			agentState.SuccessfulIters++
			stateMu.Unlock()
		}
		
		// Capture final usage stats from this iteration
		finalStats := runner.UsageStats()
		stateMu.Lock()
		agentState.InputTokens = finalStats.InputTokens
		agentState.OutputTokens = finalStats.OutputTokens
		if finalStats.CurrentTask != "" {
			agentState.CurrentTask = finalStats.CurrentTask
		}
		if cfg.Config != nil {
			pricing := cfg.Config.GetPricing(agentState.Model)
			agentState.TotalCost = pricing.CalculateCost(agentState.InputTokens, agentState.OutputTokens)
		}
		_ = mgr.MergeUpdate(agentState)
		stateMu.Unlock()

		// Check for signals and total timeout
		select {
		case sig := <-sigChan:
			fmt.Fprintf(cfg.Output, "\n[swarm] Received signal %v, stopping\n", sig)
			stateMu.Lock()
			agentState.ExitReason = "signal"
			stateMu.Unlock()
			return result, nil
		case <-timeoutCtx.Done():
			fmt.Fprintln(cfg.Output, "\n[swarm] Total timeout reached, stopping")
			result.TimedOut = true
			return result, nil
		default:
			// Continue
		}
	}

	stateMu.Lock()
	currentIter := agentState.CurrentIter
	stateMu.Unlock()
	fmt.Fprintf(cfg.Output, "\n[swarm] Run completed (%d iterations)\n", currentIter)
	return result, nil
}
