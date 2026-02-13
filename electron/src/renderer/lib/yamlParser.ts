import yaml from 'js-yaml'
import dagre from 'dagre'
import type { Node, Edge } from '@xyflow/react'

export interface TaskDependency {
  task: string
  condition: 'success' | 'failure' | 'any' | 'always'
}

export interface TaskDef {
  prompt?: string
  'prompt-file'?: string
  'prompt-string'?: string
  model?: string
  prefix?: string
  suffix?: string
  depends_on?: (string | TaskDependency)[]
}

export interface PipelineDef {
  iterations?: number
  parallelism?: number
  tasks?: string[]
}

export interface ComposeFile {
  version: string
  tasks: Record<string, TaskDef>
  pipelines?: Record<string, PipelineDef>
}

export interface TaskNodeData {
  label: string
  promptSource: string
  model?: string
  taskDef: TaskDef
  [key: string]: unknown
}

const NODE_WIDTH = 200
const NODE_HEIGHT = 100

function getPromptSource(task: TaskDef): string {
  if (task.prompt) return task.prompt
  if (task['prompt-file']) return task['prompt-file']
  if (task['prompt-string']) return 'inline'
  return 'none'
}

function normalizeDependency(dep: string | TaskDependency): TaskDependency {
  if (typeof dep === 'string') {
    return { task: dep, condition: 'success' }
  }
  return dep
}

function getEdgeColor(condition: string): string {
  switch (condition) {
    case 'success': return '#22c55e'
    case 'failure': return '#ef4444'
    case 'any': return '#eab308'
    case 'always': return '#3b82f6'
    default: return '#6b7280'
  }
}

export function parseComposeFile(content: string): ComposeFile {
  return yaml.load(content) as ComposeFile
}

export function composeToFlow(compose: ComposeFile): { nodes: Node<TaskNodeData>[]; edges: Edge[] } {
  const g = new dagre.graphlib.Graph()
  g.setGraph({ rankdir: 'TB', nodesep: 60, ranksep: 80 })
  g.setDefaultEdgeLabel(() => ({}))

  const tasks = compose.tasks ?? {}
  const taskNames = Object.keys(tasks)

  // Add nodes to dagre
  for (const name of taskNames) {
    g.setNode(name, { width: NODE_WIDTH, height: NODE_HEIGHT })
  }

  // Build edges from depends_on
  const edges: Edge[] = []
  for (const [name, task] of Object.entries(tasks)) {
    if (!task.depends_on) continue
    for (const rawDep of task.depends_on) {
      const dep = normalizeDependency(rawDep)
      if (!tasks[dep.task]) continue

      g.setEdge(dep.task, name)
      edges.push({
        id: `${dep.task}->${name}`,
        source: dep.task,
        target: name,
        label: dep.condition,
        style: { stroke: getEdgeColor(dep.condition), strokeWidth: 2 },
        labelStyle: { fill: getEdgeColor(dep.condition), fontWeight: 600, fontSize: 11 },
        labelBgStyle: { fill: 'hsl(240 10% 10%)', fillOpacity: 0.9 },
        labelBgPadding: [6, 3] as [number, number],
        animated: dep.condition === 'any' || dep.condition === 'always',
      })
    }
  }

  // Run dagre layout
  dagre.layout(g)

  // Create positioned nodes
  const nodes: Node<TaskNodeData>[] = taskNames.map((name) => {
    const pos = g.node(name)
    const task = tasks[name]
    return {
      id: name,
      type: 'taskNode',
      position: { x: pos.x - NODE_WIDTH / 2, y: pos.y - NODE_HEIGHT / 2 },
      data: {
        label: name,
        promptSource: getPromptSource(task),
        model: task.model,
        taskDef: task,
      },
    }
  })

  return { nodes, edges }
}
