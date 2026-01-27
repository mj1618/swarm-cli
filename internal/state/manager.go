package state

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/matt/swarm-cli/internal/scope"
)

// AgentState represents the state of a running agent.
type AgentState struct {
	ID            string     `json:"id"`
	Name          string     `json:"name,omitempty"`
	PID           int        `json:"pid"`
	Prompt        string     `json:"prompt"`
	Model         string     `json:"model"`
	StartedAt     time.Time  `json:"started_at"`
	Iterations    int        `json:"iterations"`
	CurrentIter   int        `json:"current_iteration"`
	Status        string     `json:"status"`         // running, terminated
	TerminateMode string     `json:"terminate_mode"` // "", "immediate", "after_iteration"
	Paused        bool       `json:"paused"`         // Whether agent loop is paused
	PausedAt      *time.Time `json:"paused_at,omitempty"` // When agent entered pause loop
	LogFile       string     `json:"log_file"`
	WorkingDir    string     `json:"working_dir"` // Directory where agent was started
}

// State holds all agent states.
type State struct {
	Agents map[string]*AgentState `json:"agents"`
}

// Manager handles state persistence for agents.
type Manager struct {
	statePath  string
	scope      scope.Scope
	workingDir string // Used for filtering when scope is ScopeProject
	mu         sync.Mutex
}

// NewManager creates a new state manager.
// Deprecated: Use NewManagerWithScope instead.
func NewManager() (*Manager, error) {
	return NewManagerWithScope(scope.ScopeGlobal, "")
}

// NewManagerWithScope creates a new state manager with the specified scope.
// For ScopeProject, workingDir should be the current working directory to filter agents.
// For ScopeGlobal, workingDir is ignored.
func NewManagerWithScope(s scope.Scope, workingDir string) (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	swarmDir := filepath.Join(homeDir, ".swarm")
	if err := os.MkdirAll(swarmDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create swarm directory: %w", err)
	}

	// If project scope and no workingDir provided, get current directory
	if s == scope.ScopeProject && workingDir == "" {
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	mgr := &Manager{
		statePath:  filepath.Join(swarmDir, "state.json"),
		scope:      s,
		workingDir: workingDir,
	}

	// Clean up stale entries on startup
	if err := mgr.cleanup(); err != nil {
		// Non-fatal, just log
		fmt.Fprintf(os.Stderr, "Warning: failed to clean up stale state: %v\n", err)
	}

	return mgr, nil
}

// GenerateID generates a unique agent ID.
func GenerateID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Register adds a new agent to the state.
// If the agent has a name that conflicts with a running agent, a number suffix is added.
func (m *Manager) Register(agent *AgentState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.load()
	if err != nil {
		state = &State{Agents: make(map[string]*AgentState)}
	}

	// Ensure name uniqueness among running agents by appending number if needed
	if agent.Name != "" {
		agent.Name = m.uniqueName(state, agent.Name)
	}

	state.Agents[agent.ID] = agent
	return m.save(state)
}

// uniqueName returns a unique name by appending a number suffix if needed.
// Only considers running agents for conflicts.
func (m *Manager) uniqueName(state *State, baseName string) string {
	// Check if base name is available
	nameInUse := func(name string) bool {
		for _, existing := range state.Agents {
			if existing.Name == name && existing.Status == "running" {
				return true
			}
		}
		return false
	}

	if !nameInUse(baseName) {
		return baseName
	}

	// Find the next available number suffix
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", baseName, i)
		if !nameInUse(candidate) {
			return candidate
		}
	}
}

// Update updates an existing agent's state.
func (m *Manager) Update(agent *AgentState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.load()
	if err != nil {
		return err
	}

	if _, exists := state.Agents[agent.ID]; !exists {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	state.Agents[agent.ID] = agent
	return m.save(state)
}

// Get retrieves an agent's state by ID.
// Note: Get does not filter by scope - it retrieves the agent regardless of working directory.
func (m *Manager) Get(id string) (*AgentState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.load()
	if err != nil {
		return nil, err
	}

	agent, exists := state.Agents[id]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", id)
	}

	return agent, nil
}

// GetByNameOrID retrieves an agent's state by ID or name.
// It first tries to find by ID, then falls back to searching by name.
// Note: GetByNameOrID does not filter by scope - it retrieves the agent regardless of working directory.
func (m *Manager) GetByNameOrID(identifier string) (*AgentState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.load()
	if err != nil {
		return nil, err
	}

	// First try direct ID lookup
	if agent, exists := state.Agents[identifier]; exists {
		return agent, nil
	}

	// Fall back to name search
	for _, agent := range state.Agents {
		if agent.Name == identifier && identifier != "" {
			return agent, nil
		}
	}

	return nil, fmt.Errorf("agent not found: %s", identifier)
}

// GetLast returns the most recently started agent.
// Respects the manager's scope setting.
// Returns an error if no agents are found.
func (m *Manager) GetLast() (*AgentState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.load()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no agents found")
		}
		return nil, err
	}

	var latest *AgentState
	for _, agent := range state.Agents {
		// Filter by scope
		if m.scope == scope.ScopeProject && agent.WorkingDir != m.workingDir {
			continue
		}
		if latest == nil || agent.StartedAt.After(latest.StartedAt) {
			latest = agent
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no agents found")
	}

	return latest, nil
}

// List returns agents filtered by the manager's scope.
// For ScopeProject, only returns agents started in the manager's working directory.
// For ScopeGlobal, returns all agents.
// If onlyRunning is true, only returns agents with status "running".
// Results are always sorted by StartedAt time (oldest first).
func (m *Manager) List(onlyRunning bool) ([]*AgentState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.load()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var agents []*AgentState
	for _, agent := range state.Agents {
		// Filter by scope
		if m.scope == scope.ScopeProject && agent.WorkingDir != m.workingDir {
			continue
		}
		// Filter by status if onlyRunning is true
		if onlyRunning && agent.Status != "running" {
			continue
		}
		agents = append(agents, agent)
	}

	// Sort by StartedAt time (oldest first)
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].StartedAt.Before(agents[j].StartedAt)
	})

	return agents, nil
}

// Remove removes an agent from the state.
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.load()
	if err != nil {
		return err
	}

	delete(state.Agents, id)
	return m.save(state)
}

// WorkingDir returns the working directory used for filtering.
func (m *Manager) WorkingDir() string {
	return m.workingDir
}

func (m *Manager) load() (*State, error) {
	data, err := os.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Agents: make(map[string]*AgentState)}, nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	if state.Agents == nil {
		state.Agents = make(map[string]*AgentState)
	}

	return &state, nil
}

func (m *Manager) save(state *State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.statePath, data, 0644)
}

// cleanup removes stale entries (processes that are no longer running).
func (m *Manager) cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.load()
	if err != nil {
		return err
	}

	changed := false
	for id, agent := range state.Agents {
		// Check if process is still running
		if !isProcessRunning(agent.PID) {
			agent.Status = "terminated"
			state.Agents[id] = agent
			changed = true
		}
	}

	if changed {
		return m.save(state)
	}
	return nil
}

// isProcessRunning checks if a process with the given PID is still running.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, sending signal 0 checks if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
