package agent

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/mj1618/swarm-cli/internal/state"
)

// ExecuteOnCompleteHook runs the on-complete command for an agent.
// The command is executed in a shell with agent context as environment variables.
// Returns nil if no hook is configured, or an error if the hook fails.
func ExecuteOnCompleteHook(agent *state.AgentState) error {
	if agent.OnComplete == "" {
		return nil
	}

	// Calculate duration
	var duration int64
	if agent.TerminatedAt != nil {
		duration = int64(agent.TerminatedAt.Sub(agent.StartedAt).Seconds())
	} else {
		duration = int64(time.Since(agent.StartedAt).Seconds())
	}

	// Set up environment with agent context
	env := os.Environ()
	env = append(env,
		"SWARM_AGENT_ID="+agent.ID,
		"SWARM_AGENT_NAME="+agent.Name,
		"SWARM_AGENT_STATUS="+agent.Status,
		fmt.Sprintf("SWARM_AGENT_ITERATIONS=%d", agent.Iterations),
		fmt.Sprintf("SWARM_AGENT_COMPLETED=%d", agent.CurrentIter),
		"SWARM_AGENT_PROMPT="+agent.Prompt,
		"SWARM_AGENT_MODEL="+agent.Model,
		"SWARM_AGENT_LOG_FILE="+agent.LogFile,
		fmt.Sprintf("SWARM_AGENT_DURATION=%d", duration),
		"SWARM_AGENT_EXIT_REASON="+agent.ExitReason,
		fmt.Sprintf("SWARM_AGENT_SUCCESSFUL_ITERS=%d", agent.SuccessfulIters),
		fmt.Sprintf("SWARM_AGENT_FAILED_ITERS=%d", agent.FailedIters),
	)

	// Execute command in shell
	cmd := exec.Command("sh", "-c", agent.OnComplete)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run in the agent's working directory if available
	if agent.WorkingDir != "" {
		cmd.Dir = agent.WorkingDir
	}

	return cmd.Run()
}
