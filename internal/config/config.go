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
	BackendCursor    = "cursor"
	BackendClaudeCode = "claude-code"
)

// Config holds the application configuration.
type Config struct {
	// Backend specifies which agent CLI to use ("cursor" or "claude-code")
	Backend string `toml:"backend"`

	// Model is the default model to use (e.g., "opus-4.5-thinking" for cursor, "opus" for claude-code)
	Model string `toml:"model"`

	// Iterations is the default number of iterations for loop command
	Iterations int `toml:"iterations"`

	// Command holds the agent command configuration
	Command CommandConfig `toml:"command"`
}

// CommandConfig holds the configuration for the agent command.
type CommandConfig struct {
	// Executable is the command to run (e.g., "agent" or "claude")
	Executable string `toml:"executable"`

	// Args is the list of arguments with {model} and {prompt} placeholders
	Args []string `toml:"args"`

	// RawOutput if true, streams output directly without parsing (for claude-code)
	// If false, output is parsed through the log parser (for cursor)
	RawOutput bool `toml:"raw_output"`
}

// DefaultConfig returns the built-in default configuration (cursor backend).
func DefaultConfig() *Config {
	return CursorConfig()
}

// CursorConfig returns the configuration preset for Cursor's agent CLI.
func CursorConfig() *Config {
	return &Config{
		Backend:    BackendCursor,
		Model:      "opus-4.5-thinking",
		Iterations: 20,
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
		Iterations: 20,
		Command: CommandConfig{
			Executable: "claude",
			Args: []string{
				"-p",
				"--model", "{model}",
				"--dangerously-skip-permissions",
				"{prompt}",
			},
			RawOutput: true,
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
	default:
		return fmt.Errorf("unknown backend: %s (valid options: %s, %s)", backend, BackendCursor, BackendClaudeCode)
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
	return []string{BackendCursor, BackendClaudeCode}
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
	return ".swarm.toml"
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
		Backend    string           `toml:"backend"`
		Model      string           `toml:"model"`
		Iterations int              `toml:"iterations"`
		Command    rawCommandConfig `toml:"command"`
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
	if fileCfg.Command.Executable != "" {
		cfg.Command.Executable = fileCfg.Command.Executable
	}
	if len(fileCfg.Command.Args) > 0 {
		cfg.Command.Args = fileCfg.Command.Args
	}
	if fileCfg.Command.RawOutput != nil {
		cfg.Command.RawOutput = *fileCfg.Command.RawOutput
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
	sb.WriteString("# Options: \"cursor\" (Cursor's agent CLI) or \"claude-code\" (Anthropic's Claude Code CLI)\n")
	sb.WriteString("backend = \"")
	sb.WriteString(c.Backend)
	sb.WriteString("\"\n\n")

	sb.WriteString("# Default model for agent runs\n")
	sb.WriteString("# For cursor: e.g., \"opus-4.5-thinking\"\n")
	sb.WriteString("# For claude-code: e.g., \"opus\", \"sonnet\"\n")
	sb.WriteString("model = \"")
	sb.WriteString(c.Model)
	sb.WriteString("\"\n\n")

	sb.WriteString("# Default iterations for loop command\n")
	sb.WriteString("iterations = ")
	sb.WriteString(itoa(c.Iterations))
	sb.WriteString("\n\n")

	sb.WriteString("# Agent command configuration\n")
	sb.WriteString("[command]\n")
	sb.WriteString("# The base command to run (e.g., \"agent\" for cursor, \"claude\" for claude-code)\n")
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
