package scope

import (
	"os"
	"path/filepath"
)

// Scope represents whether operations are scoped to the current project or globally.
type Scope int

const (
	// ScopeProject scopes operations to the current working directory.
	// - Lists only agents started in this directory
	// - Uses ./swarm/prompts/ for prompts
	ScopeProject Scope = iota

	// ScopeGlobal scopes operations globally.
	// - Lists all agents regardless of where they were started
	// - Uses ~/.swarm/prompts/ for prompts
	ScopeGlobal
)

// String returns the string representation of the scope.
func (s Scope) String() string {
	switch s {
	case ScopeProject:
		return "project"
	case ScopeGlobal:
		return "global"
	default:
		return "unknown"
	}
}

// PromptsDir returns the prompts directory for the given scope.
func (s Scope) PromptsDir() (string, error) {
	switch s {
	case ScopeGlobal:
		return GlobalPromptsDir()
	default:
		return ProjectPromptsDir(), nil
	}
}

// GlobalPromptsDir returns the global prompts directory (~/.swarm/prompts/).
func GlobalPromptsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".swarm", "prompts"), nil
}

// ProjectPromptsDir returns the project prompts directory (./swarm/prompts/).
func ProjectPromptsDir() string {
	return filepath.Join(".", "swarm", "prompts")
}

// CurrentWorkingDir returns the current working directory.
func CurrentWorkingDir() (string, error) {
	return os.Getwd()
}
