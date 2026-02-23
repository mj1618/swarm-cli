package agent

import (
	"time"

	"github.com/mj1618/swarm-cli/internal/config"
)

// Config holds the configuration for running an agent.
type Config struct {
	// Model is the model to use (e.g., "opus-4.5-thinking")
	Model string

	// Prompt is the full prompt content (already wrapped with system/user tags)
	Prompt string

	// Command holds the command configuration (executable and args template)
	Command config.CommandConfig

	// Env holds environment variables in KEY=VALUE format to pass to the agent process
	Env []string

	// Timeout is the per-iteration timeout (0 means no timeout)
	Timeout time.Duration

	// ResultGracePeriod is how long to wait after seeing a result event
	// before force-killing a hung process. 0 uses the default (30s).
	// Negative values disable this feature.
	ResultGracePeriod time.Duration
}
