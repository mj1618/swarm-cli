package telegram

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mj1618/swarm-cli/internal/agent"
	"github.com/mj1618/swarm-cli/internal/config"
	"github.com/mj1618/swarm-cli/internal/detach"
	"github.com/mj1618/swarm-cli/internal/logparser"
	"github.com/mj1618/swarm-cli/internal/prompt"
	"github.com/mj1618/swarm-cli/internal/scope"
	"github.com/mj1618/swarm-cli/internal/state"
)

// ServerConfig holds configuration for the Telegram server.
type ServerConfig struct {
	Client         *Client
	AllowedUsers   []string // Telegram usernames allowed to interact
	Model          string
	Command        config.CommandConfig
	AppConfig      *config.Config
	WorkingDir     string
	Scope          scope.Scope
	Labels         map[string]string
	AgentTimeout   time.Duration
	MaxConcurrent  int
}

// Server manages the Telegram bot lifecycle.
type Server struct {
	config   ServerConfig
	stateMgr *state.Manager
	active   sync.WaitGroup
	sem      chan struct{} // concurrency semaphore
}

// NewServer creates a new Telegram bot server.
func NewServer(cfg ServerConfig) (*Server, error) {
	mgr, err := state.NewManagerWithScope(cfg.Scope, cfg.WorkingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	maxConcurrent := cfg.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}

	return &Server{
		config:   cfg,
		stateMgr: mgr,
		sem:      make(chan struct{}, maxConcurrent),
	}, nil
}

// Run starts the long-polling loop. Blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	offset := 0
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		if ctx.Err() != nil {
			return nil
		}

		updates, err := s.config.Client.GetUpdates(ctx, offset, 30)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			fmt.Fprintf(os.Stderr, "[telegram] Poll error: %v (retrying in %v)\n", err, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}
		backoff = time.Second // reset on success

		for _, update := range updates {
			offset = update.UpdateID + 1

			if update.Message == nil || update.Message.Text == "" {
				continue
			}

			if !s.isAuthorized(update.Message) {
				continue
			}

			// Handle bot commands
			if update.Message.Text == "/start" || update.Message.Text == "/help" {
				s.config.Client.SendMessage(ctx, update.Message.Chat.ID,
					"Send me a message and I'll spawn an agent to handle it.")
				continue
			}

			msg := update.Message
			s.active.Add(1)
			go func() {
				defer s.active.Done()
				// Acquire semaphore slot
				select {
				case s.sem <- struct{}{}:
					defer func() { <-s.sem }()
				case <-ctx.Done():
					return
				}
				s.handleMessage(ctx, msg)
			}()
		}
	}
}

// Shutdown waits for all running agents to complete with a timeout.
func (s *Server) Shutdown(timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		s.active.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Fprintln(os.Stderr, "[telegram] All agents finished")
	case <-time.After(timeout):
		fmt.Fprintln(os.Stderr, "[telegram] Shutdown timeout reached, some agents may still be running")
	}
}

// handleMessage spawns an agent for the incoming message and sends the response back.
func (s *Server) handleMessage(ctx context.Context, msg *Message) {
	chatID := msg.Chat.ID
	username := ""
	if msg.From != nil {
		username = msg.From.Username
	}

	taskID := state.GenerateID()

	// Prepare prompt
	promptContent := prompt.WrapPromptString(msg.Text)
	promptContent = prompt.InjectTaskID(promptContent, taskID)
	agentID := state.GenerateID()
	promptContent = prompt.InjectAgentID(promptContent, agentID)

	// Create log file
	logFile, err := detach.LogFilePath(taskID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[telegram] Failed to create log file for %s: %v\n", taskID, err)
		s.config.Client.SendMessage(ctx, chatID, "Internal error: failed to create log file")
		return
	}

	// Build labels
	labels := make(map[string]string)
	for k, v := range s.config.Labels {
		labels[k] = v
	}
	labels["source"] = "telegram"
	if username != "" {
		labels["telegram_user"] = username
	}
	labels["telegram_chat"] = strconv.FormatInt(chatID, 10)

	// Register agent state
	now := time.Now()
	agentState := &state.AgentState{
		ID:            taskID,
		Name:          fmt.Sprintf("tg-%s", taskID),
		Prompt:        "<telegram>",
		PromptContent: msg.Text,
		Model:         s.config.Model,
		StartedAt:     now,
		Iterations:    1,
		CurrentIter:   1,
		Status:        "running",
		WorkingDir:    s.config.WorkingDir,
		LogFile:       logFile,
		Labels:        labels,
	}

	if err := s.stateMgr.Register(agentState); err != nil {
		fmt.Fprintf(os.Stderr, "[telegram] Failed to register agent %s: %v\n", taskID, err)
		s.config.Client.SendMessage(ctx, chatID, "Internal error: failed to register agent")
		return
	}

	fmt.Fprintf(os.Stderr, "[telegram] Agent %s started for @%s: %s\n", taskID, username, truncate(msg.Text, 80))

	// Open log file for writing
	lf, err := os.Create(logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[telegram] Failed to open log file %s: %v\n", logFile, err)
		s.config.Client.SendMessage(ctx, chatID, "Internal error: failed to open log file")
		s.terminateAgent(agentState, "error", err.Error())
		return
	}
	defer lf.Close()

	// Run agent, capture output
	var buf bytes.Buffer
	out := io.MultiWriter(lf, &buf)

	cfg := agent.Config{
		Model:   s.config.Model,
		Prompt:  promptContent,
		Command: s.config.Command,
		Timeout: s.config.AgentTimeout,
	}

	runner := agent.NewRunner(cfg)

	// Update PID and usage stats
	runner.SetUsageCallback(func(stats logparser.UsageStats) {
		if pid := runner.PID(); pid > 0 && agentState.PID == 0 {
			agentState.PID = pid
		}
		agentState.InputTokens = int64(stats.InputTokens)
		agentState.OutputTokens = int64(stats.OutputTokens)
		s.stateMgr.MergeUpdate(agentState)
	})

	runErr := runner.RunWithContext(ctx, out)

	// Extract summary from output (last ~4000 chars)
	output := buf.String()
	summary := extractSummary(output)

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "[telegram] Agent %s failed: %v\n", taskID, runErr)
		if summary == "" {
			summary = fmt.Sprintf("Agent error: %v", runErr)
		}
		s.terminateAgent(agentState, "error", runErr.Error())
	} else {
		fmt.Fprintf(os.Stderr, "[telegram] Agent %s completed\n", taskID)
		if summary == "" {
			summary = "(no output)"
		}
		s.terminateAgent(agentState, "completed", "")
	}

	// Send response
	if err := s.config.Client.SendMessageChunked(ctx, chatID, summary); err != nil {
		fmt.Fprintf(os.Stderr, "[telegram] Failed to send response for %s: %v\n", taskID, err)
	}
}

// terminateAgent marks an agent as terminated in the state.
func (s *Server) terminateAgent(agentState *state.AgentState, reason, lastError string) {
	now := time.Now()
	agentState.Status = "terminated"
	agentState.TerminatedAt = &now
	agentState.ExitReason = reason
	if reason == "completed" {
		agentState.SuccessfulIters = 1
	} else {
		agentState.FailedIters = 1
		agentState.LastError = lastError
	}
	if err := s.stateMgr.Update(agentState); err != nil {
		fmt.Fprintf(os.Stderr, "[telegram] Failed to update agent state for %s: %v\n", agentState.ID, err)
	}
}

// isAuthorized checks if the message sender is in the allowed users list.
func (s *Server) isAuthorized(msg *Message) bool {
	if len(s.config.AllowedUsers) == 0 {
		return false // no users configured = deny all
	}
	if msg.From == nil || msg.From.Username == "" {
		return false
	}
	for _, u := range s.config.AllowedUsers {
		if strings.EqualFold(u, msg.From.Username) {
			return true
		}
	}
	return false
}

// extractSummary returns the last meaningful portion of the output (up to ~4000 chars).
func extractSummary(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}

	const maxLen = 4000
	if len(output) <= maxLen {
		return output
	}

	// Take the tail, break at a newline
	tail := output[len(output)-maxLen:]
	idx := strings.Index(tail, "\n")
	if idx >= 0 && idx < len(tail)/2 {
		tail = tail[idx+1:]
	}

	return "...\n" + tail
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
