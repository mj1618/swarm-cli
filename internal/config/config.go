package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Backend constants
const (
	BackendCursor     = "cursor"
	BackendClaudeCode = "claude-code"
	BackendCodex      = "codex"
)

// Config holds the application configuration.
type Config struct {
	// Backend specifies which agent CLI to use ("cursor", "claude-code", or "codex")
	Backend string `toml:"backend"`

	// Model is the default model to use (e.g., "opus-4.5-thinking" for cursor, "opus" for claude-code, "o4-mini" for codex)
	Model string `toml:"model"`

	// Iterations is the default number of iterations for run command
	Iterations int `toml:"iterations"`

	// Timeout is the default total timeout for run command (e.g., "30m", "2h")
	Timeout string `toml:"timeout"`

	// IterTimeout is the default per-iteration timeout (e.g., "10m")
	IterTimeout string `toml:"iter_timeout"`

	// Command holds the agent command configuration
	Command CommandConfig `toml:"command"`

	// Pricing holds model pricing configuration (model name -> pricing)
	Pricing map[string]*ModelPricing `toml:"pricing"`

	// SystemPrompt is the optional custom system prompt content passed to the
	// underlying agent (currently only honored by the claude-code backend via
	// the `--system-prompt` flag). When empty, no `--system-prompt` flag is
	// added to the agent invocation.
	SystemPrompt string `toml:"system_prompt"`
}

// CommandConfig holds the configuration for the agent command.
type CommandConfig struct {
	// Executable is the command to run (e.g., "agent", "claude", or "codex")
	Executable string `toml:"executable"`

	// Args is the list of arguments with {model} and {prompt} placeholders
	Args []string `toml:"args"`

	// RawOutput if true, streams output directly without parsing (for claude-code)
	// If false, output is parsed through the log parser (for cursor)
	RawOutput bool `toml:"raw_output"`
}

// ModelPricing holds the pricing for a model in USD per million tokens.
type ModelPricing struct {
	InputPerMillion  float64 `toml:"input_per_million"`
	OutputPerMillion float64 `toml:"output_per_million"`
}

// CalculateCost calculates the cost in USD for given token counts.
func (p *ModelPricing) CalculateCost(inputTokens, outputTokens int64) float64 {
	inputCost := float64(inputTokens) * p.InputPerMillion / 1_000_000
	outputCost := float64(outputTokens) * p.OutputPerMillion / 1_000_000
	return inputCost + outputCost
}

// DefaultPricing returns the default pricing map for common models.
func DefaultPricing() map[string]*ModelPricing {
	return map[string]*ModelPricing{
		// Claude Opus models
		"opus": {InputPerMillion: 15.0, OutputPerMillion: 75.0},
		"claude-opus": {InputPerMillion: 15.0, OutputPerMillion: 75.0},
		"opus-4.5-thinking": {InputPerMillion: 15.0, OutputPerMillion: 75.0},
		// Claude Sonnet models
		"sonnet": {InputPerMillion: 3.0, OutputPerMillion: 15.0},
		"claude-sonnet": {InputPerMillion: 3.0, OutputPerMillion: 15.0},
		"sonnet-4": {InputPerMillion: 3.0, OutputPerMillion: 15.0},
		// Claude Haiku models
		"haiku": {InputPerMillion: 0.25, OutputPerMillion: 1.25},
		"claude-haiku": {InputPerMillion: 0.25, OutputPerMillion: 1.25},
		// GPT-4 models
		"gpt-4": {InputPerMillion: 30.0, OutputPerMillion: 60.0},
		"gpt-4-turbo": {InputPerMillion: 10.0, OutputPerMillion: 30.0},
		"gpt-4o": {InputPerMillion: 2.5, OutputPerMillion: 10.0},
		// OpenAI Codex / o-series models
		"o4-mini":      {InputPerMillion: 1.1, OutputPerMillion: 4.4},
		"o3":           {InputPerMillion: 10.0, OutputPerMillion: 40.0},
		"gpt-5-codex":  {InputPerMillion: 2.0, OutputPerMillion: 8.0},
		"codex-mini":   {InputPerMillion: 1.1, OutputPerMillion: 4.4},
		// Default fallback
		"default": {InputPerMillion: 3.0, OutputPerMillion: 15.0},
	}
}

// GetPricing returns the pricing for a model, falling back to default if not found.
func (c *Config) GetPricing(model string) *ModelPricing {
	// Normalize model name (lowercase, remove common prefixes/suffixes)
	normalizedModel := strings.ToLower(model)
	
	// Check user-configured pricing first
	if c.Pricing != nil {
		if pricing, ok := c.Pricing[model]; ok {
			return pricing
		}
		if pricing, ok := c.Pricing[normalizedModel]; ok {
			return pricing
		}
	}
	
	// Fall back to default pricing
	defaults := DefaultPricing()
	if pricing, ok := defaults[model]; ok {
		return pricing
	}
	if pricing, ok := defaults[normalizedModel]; ok {
		return pricing
	}
	
	// Check for partial matches (e.g., "opus" in "opus-4.5-thinking")
	for key, pricing := range defaults {
		if strings.Contains(normalizedModel, key) {
			return pricing
		}
	}
	
	// Return default fallback
	return defaults["default"]
}

// DefaultConfig returns the built-in default configuration (claude-code backend).
func DefaultConfig() *Config {
	return ClaudeCodeConfig()
}

// CursorConfig returns the configuration preset for Cursor's agent CLI.
func CursorConfig() *Config {
	return &Config{
		Backend:    BackendCursor,
		Model:      "opus-4.5-thinking",
		Iterations: 1,
		Command: CommandConfig{
			Executable: "agent",
			Args: []string{
				"--model", "{model}",
				"--output-format", "stream-json",
				"--stream-partial-output",
				"--sandbox", "disabled",
				"--print",
				"--force",
				"{prompt}",
			},
			RawOutput: false,
		},
	}
}

// ClaudeCodeConfig returns the configuration preset for Claude Code CLI.
func ClaudeCodeConfig() *Config {
	return &Config{
		Backend:    BackendClaudeCode,
		Model:      "opus",
		Iterations: 1,
		Command: CommandConfig{
			Executable: "claude",
			Args: []string{
				"-p",
				"--model", "{model}",
				"--output-format", "stream-json",
				"--verbose",
				"--dangerously-skip-permissions",
				"{prompt}",
			},
			RawOutput: false,
		},
	}
}

// CodexConfig returns the configuration preset for OpenAI's Codex CLI.
func CodexConfig() *Config {
	return &Config{
		Backend:    BackendCodex,
		Model:      "o4-mini",
		Iterations: 1,
		Command: CommandConfig{
			Executable: "codex",
			Args: []string{
				"exec",
				"--json",
				"--model", "{model}",
				"--sandbox", "danger-full-access",
				"{prompt}",
			},
			RawOutput: false,
		},
	}
}

// SetBackend updates the config to use the specified backend preset.
// It preserves the current Iterations value.
func (c *Config) SetBackend(backend string) error {
	var preset *Config
	switch backend {
	case BackendCursor:
		preset = CursorConfig()
	case BackendClaudeCode:
		preset = ClaudeCodeConfig()
	case BackendCodex:
		preset = CodexConfig()
	default:
		return fmt.Errorf("unknown backend: %s (valid options: %s)", backend, strings.Join(ValidBackends(), ", "))
	}

	// Preserve iterations
	iterations := c.Iterations
	if iterations == 0 {
		iterations = preset.Iterations
	}

	c.Backend = preset.Backend
	c.Model = preset.Model
	c.Iterations = iterations
	c.Command = preset.Command
	return nil
}

// ValidBackends returns the list of valid backend names.
func ValidBackends() []string {
	return []string{BackendCursor, BackendClaudeCode, BackendCodex}
}

// GlobalConfigPath returns the path to the global config file.
func GlobalConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "swarm", "config.toml"), nil
}

// ProjectConfigPath returns the path to the project config file.
func ProjectConfigPath() string {
	return "swarm/swarm.toml"
}

// Load reads and merges configuration from global and project config files.
// Priority (highest to lowest): project config > global config > defaults
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Load global config
	globalPath, err := GlobalConfigPath()
	if err == nil {
		if _, err := os.Stat(globalPath); err == nil {
			if err := loadConfigFile(globalPath, cfg); err != nil {
				return nil, err
			}
		}
	}

	// Load project config (overrides global)
	projectPath := ProjectConfigPath()
	if _, err := os.Stat(projectPath); err == nil {
		if err := loadConfigFile(projectPath, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// loadConfigFile reads a TOML config file and merges it into the given config.
func loadConfigFile(path string, cfg *Config) error {
	// We need a separate struct to detect which fields were actually set in the file
	type rawCommandConfig struct {
		Executable string   `toml:"executable"`
		Args       []string `toml:"args"`
		RawOutput  *bool    `toml:"raw_output"` // pointer to detect if set
	}
	type rawConfig struct {
		Backend      string                    `toml:"backend"`
		Model        string                    `toml:"model"`
		Iterations   int                       `toml:"iterations"`
		Timeout      string                    `toml:"timeout"`
		IterTimeout  string                    `toml:"iter_timeout"`
		Command      rawCommandConfig          `toml:"command"`
		Pricing      map[string]*ModelPricing  `toml:"pricing"`
		SystemPrompt *string                   `toml:"system_prompt"` // pointer to detect explicit removal
	}

	var fileCfg rawConfig
	if _, err := toml.DecodeFile(path, &fileCfg); err != nil {
		return err
	}

	// If backend is specified, apply that preset first
	if fileCfg.Backend != "" {
		if err := cfg.SetBackend(fileCfg.Backend); err != nil {
			return err
		}
	}

	// Then merge non-zero values (these override the preset)
	if fileCfg.Model != "" {
		cfg.Model = fileCfg.Model
	}
	if fileCfg.Iterations != 0 {
		cfg.Iterations = fileCfg.Iterations
	}
	if fileCfg.Timeout != "" {
		cfg.Timeout = fileCfg.Timeout
	}
	if fileCfg.IterTimeout != "" {
		cfg.IterTimeout = fileCfg.IterTimeout
	}
	if fileCfg.Command.Executable != "" {
		cfg.Command.Executable = fileCfg.Command.Executable
	}
	if len(fileCfg.Command.Args) > 0 {
		cfg.Command.Args = fileCfg.Command.Args
	}
	if fileCfg.Command.RawOutput != nil {
		cfg.Command.RawOutput = *fileCfg.Command.RawOutput
	}

	// Merge system prompt (project file overrides global; empty string explicitly clears it)
	if fileCfg.SystemPrompt != nil {
		cfg.SystemPrompt = *fileCfg.SystemPrompt
	}

	// Merge pricing (add/override individual models)
	if len(fileCfg.Pricing) > 0 {
		if cfg.Pricing == nil {
			cfg.Pricing = make(map[string]*ModelPricing)
		}
		for model, pricing := range fileCfg.Pricing {
			cfg.Pricing[model] = pricing
		}
	}

	return nil
}

// ExpandArgs expands {model} and {prompt} placeholders in the command args.
func (c *CommandConfig) ExpandArgs(model, prompt string) []string {
	result := make([]string, len(c.Args))
	for i, arg := range c.Args {
		expanded := arg
		expanded = strings.ReplaceAll(expanded, "{model}", model)
		expanded = strings.ReplaceAll(expanded, "{prompt}", prompt)
		result[i] = expanded
	}
	return result
}

// ToTOML returns the config as a TOML string.
func (c *Config) ToTOML() string {
	var sb strings.Builder
	sb.WriteString("# swarm-cli configuration\n\n")

	sb.WriteString("# Backend specifies which agent CLI to use\n")
	sb.WriteString("# Options: \"cursor\", \"claude-code\", or \"codex\"\n")
	sb.WriteString("backend = \"")
	sb.WriteString(c.Backend)
	sb.WriteString("\"\n\n")

	sb.WriteString("# Default model for agent runs\n")
	sb.WriteString("# For cursor: e.g., \"opus-4.5-thinking\"\n")
	sb.WriteString("# For claude-code: e.g., \"opus\", \"sonnet\"\n")
	sb.WriteString("# For codex: e.g., \"o4-mini\", \"o3\"\n")
	sb.WriteString("model = \"")
	sb.WriteString(c.Model)
	sb.WriteString("\"\n\n")

	sb.WriteString("# Default iterations for run command\n")
	sb.WriteString("iterations = ")
	sb.WriteString(itoa(c.Iterations))
	sb.WriteString("\n\n")

	sb.WriteString("# Default total timeout for run command (e.g., \"30m\", \"2h\")\n")
	sb.WriteString("# Set to \"\" or omit for no timeout\n")
	sb.WriteString("# timeout = \"")
	sb.WriteString(c.Timeout)
	sb.WriteString("\"\n\n")

	sb.WriteString("# Default per-iteration timeout (e.g., \"10m\")\n")
	sb.WriteString("# Set to \"\" or omit for no timeout\n")
	sb.WriteString("# iter_timeout = \"")
	sb.WriteString(c.IterTimeout)
	sb.WriteString("\"\n\n")

	// System prompt MUST be written before any [section] header — once we
	// enter `[command]`, subsequent top-level keys would be parsed as
	// `command.<key>` per TOML semantics.
	sb.WriteString("# Custom system prompt for the agent.\n")
	sb.WriteString("# When set, claude-code runs are invoked with `--system-prompt <content>`.\n")
	sb.WriteString("# Manage via `swarm config set-system-prompt` / `swarm config remove-system-prompt`,\n")
	sb.WriteString("# or `swarm run --system-prompt[-file] ...` to set + persist before running.\n")
	if c.SystemPrompt == "" {
		sb.WriteString("system_prompt = \"\"\n\n")
	} else {
		sb.WriteString("system_prompt = ")
		sb.WriteString(tomlQuoteMultiline(c.SystemPrompt))
		sb.WriteString("\n\n")
	}

	sb.WriteString("# Agent command configuration\n")
	sb.WriteString("[command]\n")
	sb.WriteString("# The base command to run (e.g., \"agent\" for cursor, \"claude\" for claude-code, \"codex\" for codex)\n")
	sb.WriteString("executable = \"")
	sb.WriteString(c.Command.Executable)
	sb.WriteString("\"\n\n")

	sb.WriteString("# Arguments template - {model} and {prompt} are replaced at runtime\n")
	sb.WriteString("args = [\n")
	for i, arg := range c.Command.Args {
		sb.WriteString("  \"")
		sb.WriteString(arg)
		sb.WriteString("\"")
		if i < len(c.Command.Args)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("]\n\n")

	sb.WriteString("# If true, output streams directly without parsing (for claude-code)\n")
	sb.WriteString("# If false, output is parsed through the log parser (for cursor)\n")
	sb.WriteString("raw_output = ")
	if c.Command.RawOutput {
		sb.WriteString("true")
	} else {
		sb.WriteString("false")
	}
	sb.WriteString("\n")

	return sb.String()
}

// tomlQuoteMultiline returns a TOML-safe representation of s, preferring a
// triple-quoted multiline string for content containing newlines and falling
// back to a basic quoted string otherwise.
func tomlQuoteMultiline(s string) string {
	if strings.ContainsAny(s, "\n\r") {
		// Triple-quoted multiline basic string. Escape any embedded `"""` to be safe.
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"""`, `\"\"\"`)
		return "\"\"\"\n" + escaped + "\"\"\""
	}
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return "\"" + escaped + "\""
}

// AgentCommand returns the effective CommandConfig used to launch the agent.
// It is a copy of c.Command with c.SystemPrompt injected as `--system-prompt`
// before the `{prompt}` placeholder when both are set and the active backend
// is claude-code (the only backend that currently supports the flag).
//
// Callers should use this in place of c.Command when constructing agent.Config
// so that a configured system prompt is honored uniformly across `swarm run`,
// `swarm up`, `swarm restart`, `swarm clone`, the DAG executor, and the
// multi-iteration loop runner.
func (c *Config) AgentCommand() CommandConfig {
	cmd := c.Command
	if c.SystemPrompt == "" || c.Backend != BackendClaudeCode {
		return cmd
	}
	cmd.Args = CommandArgsWithSystemPrompt(c.Command.Args, c.SystemPrompt)
	return cmd
}

// CommandArgsWithSystemPrompt returns a copy of the command args with
// `--system-prompt <content>` injected before the `{prompt}` placeholder when
// systemPrompt is non-empty. If `{prompt}` is not present, the flag is
// appended to the end. When systemPrompt is empty, args are returned unchanged.
func CommandArgsWithSystemPrompt(args []string, systemPrompt string) []string {
	if systemPrompt == "" {
		return args
	}
	out := make([]string, 0, len(args)+2)
	inserted := false
	for _, a := range args {
		if !inserted && a == "{prompt}" {
			out = append(out, "--system-prompt", systemPrompt)
			inserted = true
		}
		out = append(out, a)
	}
	if !inserted {
		out = append(out, "--system-prompt", systemPrompt)
	}
	return out
}

// itoa converts an int to string (simple implementation to avoid strconv import)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
