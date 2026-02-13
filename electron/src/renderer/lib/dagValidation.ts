import type { ComposeFile } from './yamlParser'

export interface ValidationResult {
  cycleNodes: Set<string>
  cycleEdges: Set<string> // format: "source->target"
  orphanedTasks: Set<string>
}

/**
 * Detect cycles using Kahn's algorithm (topological sort via in-degree counting).
 * Any nodes remaining after the sort are part of cycles.
 */
function detectCycles(tasks: ComposeFile['tasks']): { cycleNodes: Set<string>; cycleEdges: Set<string> } {
  const taskNames = Object.keys(tasks)
  const adj = new Map<string, string[]>() // source -> targets (dependents)
  const inDegree = new Map<string, number>()

  for (const name of taskNames) {
    adj.set(name, [])
    inDegree.set(name, 0)
  }

  // Build adjacency: if B depends_on A, then edge A -> B
  for (const [name, task] of Object.entries(tasks)) {
    if (!task.depends_on) continue
    for (const rawDep of task.depends_on) {
      const depTask = typeof rawDep === 'string' ? rawDep : rawDep.task
      if (!tasks[depTask]) continue
      adj.get(depTask)!.push(name)
      inDegree.set(name, (inDegree.get(name) ?? 0) + 1)
    }
  }

  // Kahn's algorithm
  const queue: string[] = []
  for (const [node, deg] of inDegree) {
    if (deg === 0) queue.push(node)
  }

  const sorted: string[] = []
  while (queue.length > 0) {
    const node = queue.shift()!
    sorted.push(node)
    for (const neighbor of adj.get(node) ?? []) {
      const newDeg = (inDegree.get(neighbor) ?? 1) - 1
      inDegree.set(neighbor, newDeg)
      if (newDeg === 0) queue.push(neighbor)
    }
  }

  const cycleNodes = new Set<string>()
  for (const name of taskNames) {
    if (!sorted.includes(name)) {
      cycleNodes.add(name)
    }
  }

  // Find edges that are part of cycles (both endpoints are cycle nodes)
  const cycleEdges = new Set<string>()
  for (const [name, task] of Object.entries(tasks)) {
    if (!task.depends_on || !cycleNodes.has(name)) continue
    for (const rawDep of task.depends_on) {
      const depTask = typeof rawDep === 'string' ? rawDep : rawDep.task
      if (cycleNodes.has(depTask)) {
        cycleEdges.add(`${depTask}->${name}`)
      }
    }
  }

  return { cycleNodes, cycleEdges }
}

/**
 * Detect orphaned tasks: tasks that have dependencies but are not listed
 * in any pipeline's tasks array.
 */
function detectOrphans(compose: ComposeFile): Set<string> {
  const orphaned = new Set<string>()
  const tasks = compose.tasks ?? {}
  const pipelines = compose.pipelines ?? {}

  // Collect all tasks that appear in at least one pipeline
  const pipelinedTasks = new Set<string>()
  for (const pipeline of Object.values(pipelines)) {
    if (pipeline.tasks) {
      for (const t of pipeline.tasks) {
        pipelinedTasks.add(t)
      }
    }
  }

  // If there are no pipelines defined, no tasks can be orphaned
  if (Object.keys(pipelines).length === 0) return orphaned

  // Tasks with dependencies that are NOT in any pipeline are orphaned
  for (const [name, task] of Object.entries(tasks)) {
    if (task.depends_on && task.depends_on.length > 0 && !pipelinedTasks.has(name)) {
      orphaned.add(name)
    }
  }

  return orphaned
}

export function validateDag(compose: ComposeFile): ValidationResult {
  const tasks = compose.tasks ?? {}
  if (Object.keys(tasks).length === 0) {
    return { cycleNodes: new Set(), cycleEdges: new Set(), orphanedTasks: new Set() }
  }

  const { cycleNodes, cycleEdges } = detectCycles(tasks)
  const orphanedTasks = detectOrphans(compose)

  return { cycleNodes, cycleEdges, orphanedTasks }
}
