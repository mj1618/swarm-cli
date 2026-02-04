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
	"time"

	"github.com/mj1618/swarm-cli/internal/scope"
)

// AgentState represents the state of a running agent.
type AgentState struct {
	ID            string            `json:"id"`
	Name          string            `json:"name,omitempty"`
	ParentID      string            `json:"parent_id,omitempty"` // Parent agent ID for sub-agents
	Labels        map[string]string `json:"labels,omitempty"`
	PID           int               `json:"pid"`
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
	EnvNames      []string   `json:"env_names,omitempty"` // Environment variable names (values not stored for security)
	TimeoutAt     *time.Time `json:"timeout_at,omitempty"`     // When total timeout will trigger
	TimeoutReason string     `json:"timeout_reason,omitempty"` // "total" or "iteration" when terminated by timeout

	// Termination tracking
	TerminatedAt *time.Time `json:"terminated_at,omitempty"` // When agent stopped
	ExitReason   string     `json:"exit_reason,omitempty"`   // completed, killed, signal, error

	// Iteration outcomes
	SuccessfulIters int    `json:"successful_iterations"` // Iterations that completed without error
	FailedIters     int    `json:"failed_iterations"`     // Iterations that errored
	LastError       string `json:"last_error,omitempty"`  // Last error message if any

	// Token and cost tracking
	InputTokens  int64   `json:"input_tokens"`        // Total input tokens used
	OutputTokens int64   `json:"output_tokens"`       // Total output tokens used
	TotalCost    float64 `json:"total_cost_usd"`      // Total cost in USD
	CurrentTask  string  `json:"current_task,omitempty"` // Last activity summary (e.g., "Read: auth.ts")

	// Hooks
	OnComplete string `json:"on_complete,omitempty"` // Command to run when agent completes
}

// State holds all agent states.
type State struct {
	Agents map[string]*AgentState `json:"agents"`
}

// Manager handles state persistence for agents.
type Manager struct {
	statePath  string
	lockPath   string // Path to lock file for cross-process synchronization
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
		lockPath:   filepath.Join(swarmDir, "state.lock"),
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

// copyAgentState creates a deep copy of an AgentState to avoid race conditions
// when the returned pointer is used outside the lock.
func copyAgentState(agent *AgentState) *AgentState {
	if agent == nil {
		return nil
	}
	copy := *agent

	// Deep copy map
	if agent.Labels != nil {
		copy.Labels = make(map[string]string, len(agent.Labels))
		for k, v := range agent.Labels {
			copy.Labels[k] = v
		}
	}

	// Deep copy slices
	if agent.EnvNames != nil {
		copy.EnvNames = make([]string, len(agent.EnvNames))
		for i, v := range agent.EnvNames {
			copy.EnvNames[i] = v
		}
	}

	// Deep copy time pointers
	if agent.PausedAt != nil {
		t := *agent.PausedAt
		copy.PausedAt = &t
	}
	if agent.TerminatedAt != nil {
		t := *agent.TerminatedAt
		copy.TerminatedAt = &t
	}
	if agent.TimeoutAt != nil {
		t := *agent.TimeoutAt
		copy.TimeoutAt = &t
	}

	return &copy
}

// Register adds a new agent to the state.
// If the agent has a name that conflicts with a running agent, a number suffix is added.
func (m *Manager) Register(agent *AgentState) error {
	fl, err := m.lock()
	if err != nil {
		return err
	}
	defer m.unlock(fl)

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

// lock acquires both the in-process mutex and the cross-process file lock.
// Always call unlock() when done, typically via defer.
func (m *Manager) lock() (*fileLock, error) {
	m.mu.Lock()
	fl := newFileLock(m.lockPath)
	if err := fl.Lock(); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	return fl, nil
}

// unlock releases both locks.
func (m *Manager) unlock(fl *fileLock) {
	if fl != nil {
		fl.Unlock()
	}
	m.mu.Unlock()
}

// Update updates an existing agent's state.
// This replaces the entire agent state. For runner updates that should preserve
// external control field changes, use MergeUpdate() instead.
func (m *Manager) Update(agent *AgentState) error {
	fl, err := m.lock()
	if err != nil {
		return err
	}
	defer m.unlock(fl)

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

// MergeUpdate updates an existing agent's state while preserving "control signal"
// fields (Iterations, Model, TerminateMode, Paused) from the current disk state.
// This prevents the runner from overwriting changes made by `swarm top` or other commands.
// Use this from the runner loop instead of Update().
func (m *Manager) MergeUpdate(agent *AgentState) error {
	fl, err := m.lock()
	if err != nil {
		return err
	}
	defer m.unlock(fl)

	state, err := m.load()
	if err != nil {
		return err
	}

	existing, exists := state.Agents[agent.ID]
	if !exists {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	// Merge control signal fields from disk to preserve external changes
	mergeControlFields(existing, agent)

	state.Agents[agent.ID] = agent
	return m.save(state)
}

// mergeControlFields copies control signal fields from the existing (disk) state
// into the agent being saved, preserving external changes made by other processes.
// Control fields are those that can be modified by external commands like `swarm top`.
func mergeControlFields(existing, agent *AgentState) {
	// Iterations: preserve disk value if it differs (externally changed)
	// The runner reads this at iteration start and syncs it to its local copy
	agent.Iterations = existing.Iterations
	
	// Model: preserve disk value if it differs (externally changed)
	agent.Model = existing.Model
	
	// TerminateMode: preserve disk value - this is set by `swarm stop`
	agent.TerminateMode = existing.TerminateMode
	
	// Paused: preserve disk value - this is set by `swarm pause`
	agent.Paused = existing.Paused
}

// SetIterations atomically updates the Iterations field for an agent.
// Use this instead of Update() when explicitly changing the iteration count.
func (m *Manager) SetIterations(id string, iterations int) error {
	fl, err := m.lock()
	if err != nil {
		return err
	}
	defer m.unlock(fl)

	state, err := m.load()
	if err != nil {
		return err
	}

	agent, exists := state.Agents[id]
	if !exists {
		return fmt.Errorf("agent not found: %s", id)
	}

	agent.Iterations = iterations
	return m.save(state)
}

// SetModel atomically updates the Model field for an agent.
// Use this instead of Update() when explicitly changing the model.
func (m *Manager) SetModel(id string, model string) error {
	fl, err := m.lock()
	if err != nil {
		return err
	}
	defer m.unlock(fl)

	state, err := m.load()
	if err != nil {
		return err
	}

	agent, exists := state.Agents[id]
	if !exists {
		return fmt.Errorf("agent not found: %s", id)
	}

	agent.Model = model
	return m.save(state)
}

// SetTerminateMode atomically updates the TerminateMode field for an agent.
// Use this instead of Update() when explicitly setting termination mode.
func (m *Manager) SetTerminateMode(id string, mode string) error {
	fl, err := m.lock()
	if err != nil {
		return err
	}
	defer m.unlock(fl)

	state, err := m.load()
	if err != nil {
		return err
	}

	agent, exists := state.Agents[id]
	if !exists {
		return fmt.Errorf("agent not found: %s", id)
	}

	agent.TerminateMode = mode
	return m.save(state)
}

// SetPaused atomically updates the Paused field for an agent.
// Use this instead of Update() when explicitly pausing/resuming.
func (m *Manager) SetPaused(id string, paused bool) error {
	fl, err := m.lock()
	if err != nil {
		return err
	}
	defer m.unlock(fl)

	state, err := m.load()
	if err != nil {
		return err
	}

	agent, exists := state.Agents[id]
	if !exists {
		return fmt.Errorf("agent not found: %s", id)
	}

	agent.Paused = paused
	if paused {
		now := time.Now()
		agent.PausedAt = &now
	} else {
		agent.PausedAt = nil
	}
	return m.save(state)
}

// Get retrieves an agent's state by ID.
// Note: Get does not filter by scope - it retrieves the agent regardless of working directory.
// Returns a copy of the state to avoid race conditions.
func (m *Manager) Get(id string) (*AgentState, error) {
	fl, err := m.lock()
	if err != nil {
		return nil, err
	}
	defer m.unlock(fl)

	state, err := m.load()
	if err != nil {
		return nil, err
	}

	agent, exists := state.Agents[id]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", id)
	}

	return copyAgentState(agent), nil
}

// GetByNameOrID retrieves an agent's state by ID or name.
// It first tries to find by ID, then falls back to searching by name.
// Note: GetByNameOrID does not filter by scope - it retrieves the agent regardless of working directory.
// Returns a copy of the state to avoid race conditions.
func (m *Manager) GetByNameOrID(identifier string) (*AgentState, error) {
	fl, err := m.lock()
	if err != nil {
		return nil, err
	}
	defer m.unlock(fl)

	state, err := m.load()
	if err != nil {
		return nil, err
	}

	// First try direct ID lookup
	if agent, exists := state.Agents[identifier]; exists {
		return copyAgentState(agent), nil
	}

	// Fall back to name search
	for _, agent := range state.Agents {
		if agent.Name == identifier && identifier != "" {
			return copyAgentState(agent), nil
		}
	}

	return nil, fmt.Errorf("agent not found: %s", identifier)
}

// GetLast returns the most recently started agent.
// Respects the manager's scope setting.
// Returns an error if no agents are found.
// Returns a copy of the state to avoid race conditions.
func (m *Manager) GetLast() (*AgentState, error) {
	fl, err := m.lock()
	if err != nil {
		return nil, err
	}
	defer m.unlock(fl)

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

	return copyAgentState(latest), nil
}

// GetChildren returns all agents that have the given parentID as their parent.
// Note: GetChildren does not filter by scope - it returns all children regardless of working directory.
// Returns copies of the states to avoid race conditions.
func (m *Manager) GetChildren(parentID string) ([]*AgentState, error) {
	fl, err := m.lock()
	if err != nil {
		return nil, err
	}
	defer m.unlock(fl)

	state, err := m.load()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var children []*AgentState
	for _, agent := range state.Agents {
		if agent.ParentID == parentID {
			children = append(children, copyAgentState(agent))
		}
	}

	// Sort by StartedAt time (oldest first)
	sort.Slice(children, func(i, j int) bool {
		return children[i].StartedAt.Before(children[j].StartedAt)
	})

	return children, nil
}

// GetDescendants returns all agents in the subtree rooted at parentID (children, grandchildren, etc.).
// Note: GetDescendants does not filter by scope - it returns all descendants regardless of working directory.
// Returns copies of the states to avoid race conditions.
func (m *Manager) GetDescendants(parentID string) ([]*AgentState, error) {
	fl, err := m.lock()
	if err != nil {
		return nil, err
	}
	defer m.unlock(fl)

	state, err := m.load()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Build a map of parent -> children for efficient traversal
	childrenMap := make(map[string][]*AgentState)
	for _, agent := range state.Agents {
		if agent.ParentID != "" {
			childrenMap[agent.ParentID] = append(childrenMap[agent.ParentID], agent)
		}
	}

	// BFS to find all descendants
	var descendants []*AgentState
	queue := []string{parentID}

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		for _, child := range childrenMap[currentID] {
			descendants = append(descendants, copyAgentState(child))
			queue = append(queue, child.ID)
		}
	}

	// Sort by StartedAt time (oldest first)
	sort.Slice(descendants, func(i, j int) bool {
		return descendants[i].StartedAt.Before(descendants[j].StartedAt)
	})

	return descendants, nil
}

// List returns agents filtered by the manager's scope.
// For ScopeProject, only returns agents started in the manager's working directory.
// For ScopeGlobal, returns all agents.
// If onlyRunning is true, only returns agents with status "running".
// Results are always sorted by StartedAt time (oldest first).
// Returns copies of the states to avoid race conditions.
func (m *Manager) List(onlyRunning bool) ([]*AgentState, error) {
	fl, err := m.lock()
	if err != nil {
		return nil, err
	}
	defer m.unlock(fl)

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
		agents = append(agents, copyAgentState(agent))
	}

	// Sort by StartedAt time (oldest first)
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].StartedAt.Before(agents[j].StartedAt)
	})

	return agents, nil
}

// Remove removes an agent from the state.
func (m *Manager) Remove(id string) error {
	fl, err := m.lock()
	if err != nil {
		return err
	}
	defer m.unlock(fl)

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
	fl, err := m.lock()
	if err != nil {
		return err
	}
	defer m.unlock(fl)

	state, err := m.load()
	if err != nil {
		return err
	}

	changed := false
	now := time.Now()
	for id, agent := range state.Agents {
		// Handle agents with PID=0 (registered but child never started or updated PID)
		if agent.Status == "running" && agent.PID == 0 {
			// Give some time for the parent to update the PID after starting child
			if time.Since(agent.StartedAt) > 30*time.Second {
				agent.Status = "terminated"
				agent.ExitReason = "crashed"
				agent.TerminatedAt = &now
				state.Agents[id] = agent
				changed = true
			}
			continue
		}

		// Check if process is still running
		if agent.Status == "running" && !isProcessRunning(agent.PID) {
			agent.Status = "terminated"
			// If the process died without setting exit reason, it crashed
			if agent.ExitReason == "" {
				agent.ExitReason = "crashed"
			}
			if agent.TerminatedAt == nil {
				agent.TerminatedAt = &now
			}
			state.Agents[id] = agent
			changed = true
		}
	}

	if changed {
		return m.save(state)
	}
	return nil
}
