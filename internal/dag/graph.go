package dag

import (
	"fmt"
	"sort"

	"github.com/matt/swarm-cli/internal/compose"
)

// Graph represents a directed acyclic graph of task dependencies.
type Graph struct {
	// nodes is the set of all task names in the graph
	nodes map[string]bool

	// edges maps each task to its dependencies (tasks it depends on)
	edges map[string][]compose.Dependency

	// reverseEdges maps each task to tasks that depend on it
	reverseEdges map[string][]string

	// tasks is a reference to the original task definitions
	tasks map[string]compose.Task
}

// NewGraph creates a new DAG from the given tasks.
// Only tasks in taskNames are included in the graph.
func NewGraph(tasks map[string]compose.Task, taskNames []string) *Graph {
	g := &Graph{
		nodes:        make(map[string]bool),
		edges:        make(map[string][]compose.Dependency),
		reverseEdges: make(map[string][]string),
		tasks:        tasks,
	}

	// Build node set from taskNames
	taskSet := make(map[string]bool)
	for _, name := range taskNames {
		taskSet[name] = true
		g.nodes[name] = true
	}

	// Build edges (only include dependencies that are also in the task set)
	for _, name := range taskNames {
		task := tasks[name]
		for _, dep := range task.DependsOn {
			if taskSet[dep.Task] {
				g.edges[name] = append(g.edges[name], dep)
				g.reverseEdges[dep.Task] = append(g.reverseEdges[dep.Task], name)
			}
		}
	}

	return g
}

// Validate checks the graph for cycles and returns an error if found.
func (g *Graph) Validate() error {
	// Check for cycles using DFS with coloring
	// white = 0 (unvisited), gray = 1 (visiting), black = 2 (visited)
	color := make(map[string]int)
	var path []string

	var visit func(node string) error
	visit = func(node string) error {
		color[node] = 1 // gray - visiting
		path = append(path, node)

		for _, dep := range g.edges[node] {
			if color[dep.Task] == 1 {
				// Found a cycle - build the cycle path
				cycleStart := -1
				for i, n := range path {
					if n == dep.Task {
						cycleStart = i
						break
					}
				}
				cyclePath := append(path[cycleStart:], dep.Task)
				return fmt.Errorf("cycle detected: %v", cyclePath)
			}
			if color[dep.Task] == 0 {
				if err := visit(dep.Task); err != nil {
					return err
				}
			}
		}

		path = path[:len(path)-1]
		color[node] = 2 // black - visited
		return nil
	}

	for node := range g.nodes {
		if color[node] == 0 {
			if err := visit(node); err != nil {
				return err
			}
		}
	}

	return nil
}

// TopologicalSort returns tasks in an order where dependencies come before dependents.
// Returns an error if the graph contains a cycle.
func (g *Graph) TopologicalSort() ([]string, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}

	var result []string
	visited := make(map[string]bool)

	var visit func(node string)
	visit = func(node string) {
		if visited[node] {
			return
		}
		visited[node] = true

		// Visit dependencies first
		for _, dep := range g.edges[node] {
			visit(dep.Task)
		}

		result = append(result, node)
	}

	// Sort nodes for deterministic output
	nodes := make([]string, 0, len(g.nodes))
	for node := range g.nodes {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)

	for _, node := range nodes {
		visit(node)
	}

	return result, nil
}

// GetRootTasks returns tasks with no dependencies (entry points).
func (g *Graph) GetRootTasks() []string {
	var roots []string
	for node := range g.nodes {
		if len(g.edges[node]) == 0 {
			roots = append(roots, node)
		}
	}
	sort.Strings(roots)
	return roots
}

// GetDependencies returns the dependencies for a task.
func (g *Graph) GetDependencies(task string) []compose.Dependency {
	return g.edges[task]
}

// GetDependents returns tasks that depend on the given task.
func (g *Graph) GetDependents(task string) []string {
	return g.reverseEdges[task]
}

// GetTask returns the task definition for a given task name.
func (g *Graph) GetTask(name string) (compose.Task, bool) {
	task, ok := g.tasks[name]
	return task, ok
}

// GetNodes returns all task names in the graph.
func (g *Graph) GetNodes() []string {
	nodes := make([]string, 0, len(g.nodes))
	for node := range g.nodes {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	return nodes
}

// FindReadyTasks returns tasks that are ready to run based on current task states.
// A task is ready if:
// 1. It hasn't been started yet (status is pending)
// 2. All its dependencies have completed (based on their conditions)
func (g *Graph) FindReadyTasks(states map[string]*TaskState) []string {
	var ready []string

	for node := range g.nodes {
		state := states[node]
		if state == nil || state.Status != TaskPending {
			continue // Only consider pending tasks
		}

		if g.canRun(node, states) {
			ready = append(ready, node)
		}
	}

	sort.Strings(ready)
	return ready
}

// canRun checks if a task can run based on its dependencies and their states.
func (g *Graph) canRun(task string, states map[string]*TaskState) bool {
	deps := g.edges[task]

	for _, dep := range deps {
		depState := states[dep.Task]
		if depState == nil {
			return false // Dependency state unknown
		}

		condition := dep.EffectiveCondition()

		switch condition {
		case compose.ConditionSuccess:
			// Run only if dependency succeeded
			if depState.Status != TaskSucceeded {
				return false
			}
		case compose.ConditionFailure:
			// Run only if dependency failed
			if depState.Status != TaskFailed {
				return false
			}
		case compose.ConditionAny:
			// Run if dependency completed (success or failure)
			if depState.Status != TaskSucceeded && depState.Status != TaskFailed {
				return false
			}
		case compose.ConditionAlways:
			// Run if dependency is done (including skipped)
			if depState.Status == TaskPending || depState.Status == TaskRunning {
				return false
			}
		}
	}

	return true
}

// ShouldSkip determines if a task should be skipped based on its dependencies.
// A task is skipped if its dependency conditions can never be satisfied.
func (g *Graph) ShouldSkip(task string, states map[string]*TaskState) bool {
	deps := g.edges[task]

	for _, dep := range deps {
		depState := states[dep.Task]
		if depState == nil {
			continue
		}

		condition := dep.EffectiveCondition()

		// Check if this dependency's condition can never be satisfied
		switch condition {
		case compose.ConditionSuccess:
			// If dependency failed or was skipped, this task should be skipped
			if depState.Status == TaskFailed || depState.Status == TaskSkipped {
				return true
			}
		case compose.ConditionFailure:
			// If dependency succeeded or was skipped, this task should be skipped
			if depState.Status == TaskSucceeded || depState.Status == TaskSkipped {
				return true
			}
		// ConditionAny and ConditionAlways don't cause skipping
		}
	}

	return false
}
