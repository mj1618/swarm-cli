import { useMemo, useCallback, useState, useEffect } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  BackgroundVariant,
  Panel,
  applyNodeChanges,
  useReactFlow,
} from '@xyflow/react'
import type { Node, Edge, NodeChange, Connection } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import TaskNode from './TaskNode'
import ConnectionDialog from './ConnectionDialog'
import { composeToFlow, parseComposeFile } from '../lib/yamlParser'
import type { ComposeFile, TaskDef, TaskDependency, TaskNodeData, AgentDisplayStatus } from '../lib/yamlParser'
import type { AgentState } from '../../preload/index'

const nodeTypes = { taskNode: TaskNode }

interface PendingConnection {
  source: string
  target: string
  position: { x: number; y: number }
}

interface DagCanvasProps {
  yamlContent: string | null
  loading: boolean
  error: string | null
  agents?: AgentState[]
  onSelectTask?: (task: { name: string; def: TaskDef; compose: ComposeFile }) => void
  onAddDependency?: (dep: { source: string; target: string; condition: TaskDependency['condition'] }) => void
  onCreateTask?: () => void
  savedPositions?: Record<string, { x: number; y: number }>
  onPositionsChange?: (positions: Record<string, { x: number; y: number }>) => void
  onResetLayout?: () => void
}

function resolveAgentStatus(agent: AgentState): AgentDisplayStatus {
  if (agent.status === 'running' && agent.paused) return 'paused'
  if (agent.status === 'running') return 'running'
  if (agent.status === 'terminated') {
    if (agent.exit_reason === 'crashed' || agent.exit_reason === 'killed') return 'failed'
    return 'succeeded'
  }
  return 'running'
}

export default function DagCanvas({
  yamlContent,
  loading,
  error,
  agents,
  onSelectTask,
  onAddDependency,
  onCreateTask,
  savedPositions,
  onPositionsChange,
  onResetLayout,
}: DagCanvasProps) {
  // Parse YAML and compute dagre layout (with saved positions applied)
  const { initialNodes, edges, parseError, compose } = useMemo(() => {
    const empty = {
      initialNodes: [] as Node<TaskNodeData>[],
      edges: [] as Edge[],
      parseError: null as string | null,
      compose: null as ComposeFile | null,
    }
    if (!yamlContent) return empty
    try {
      const compose: ComposeFile = parseComposeFile(yamlContent)
      if (!compose.tasks || Object.keys(compose.tasks).length === 0) {
        return { ...empty, parseError: 'No tasks found in swarm.yaml' }
      }
      const result = composeToFlow(compose, savedPositions)
      return { initialNodes: result.nodes, edges: result.edges, parseError: null, compose }
    } catch (e) {
      return { ...empty, parseError: (e as Error).message }
    }
  }, [yamlContent, savedPositions])

  // Enrich nodes with agent status data
  const enrichedNodes = useMemo(() => {
    if (!agents || agents.length === 0) return initialNodes
    return initialNodes.map((node) => {
      const agent = agents.find(
        (a) => a.name === node.id || a.labels?.task_id === node.id || a.current_task === node.id,
      )
      if (!agent) return node
      return {
        ...node,
        data: {
          ...node.data,
          agentStatus: resolveAgentStatus(agent),
          agentProgress: { current: agent.current_iteration, total: agent.iterations },
          agentCost: agent.total_cost_usd,
        },
      }
    })
  }, [initialNodes, agents])

  // Local node state for drag interactions
  const [nodes, setNodes] = useState<Node<TaskNodeData>[]>(enrichedNodes)

  // Sync local state when nodes change (YAML reload, positions reset, or agent status update)
  useEffect(() => {
    setNodes(enrichedNodes)
  }, [enrichedNodes])

  const onNodesChange = useCallback(
    (changes: NodeChange<Node<TaskNodeData>>[]) => {
      setNodes((prev) => {
        const updated = applyNodeChanges(changes, prev)

        // Persist position changes when drag ends
        const hasDragEnd = changes.some(
          (c) => c.type === 'position' && !c.dragging,
        )
        if (hasDragEnd && onPositionsChange) {
          const positions: Record<string, { x: number; y: number }> = {}
          for (const node of updated) {
            positions[node.id] = { x: node.position.x, y: node.position.y }
          }
          onPositionsChange(positions)
        }

        return updated
      })
    },
    [onPositionsChange],
  )

  const handleNodeClick = useCallback(
    (_event: React.MouseEvent, node: Node<TaskNodeData>) => {
      if (onSelectTask && compose) {
        onSelectTask({ name: node.data.label, def: node.data.taskDef, compose })
      }
    },
    [onSelectTask, compose],
  )

  // Connection dialog state
  const [pendingConnection, setPendingConnection] = useState<PendingConnection | null>(null)
  const { flowToScreenPosition } = useReactFlow()

  const handleConnect = useCallback(
    (connection: Connection) => {
      if (!connection.source || !connection.target) return
      // Prevent self-references
      if (connection.source === connection.target) return
      // Prevent duplicate: check if edge already exists
      if (edges.some(e => e.source === connection.source && e.target === connection.target)) return

      // Find midpoint between source and target nodes for dialog placement
      const sourceNode = nodes.find(n => n.id === connection.source)
      const targetNode = nodes.find(n => n.id === connection.target)
      if (!sourceNode || !targetNode) return

      const midX = (sourceNode.position.x + targetNode.position.x) / 2 + 100
      const midY = (sourceNode.position.y + targetNode.position.y) / 2 + 50
      const screenPos = flowToScreenPosition({ x: midX, y: midY })

      setPendingConnection({
        source: connection.source,
        target: connection.target,
        position: { x: screenPos.x, y: screenPos.y },
      })
    },
    [nodes, edges, flowToScreenPosition],
  )

  const handleConditionSelect = useCallback(
    (condition: 'success' | 'failure' | 'any' | 'always') => {
      if (!pendingConnection || !onAddDependency) return
      onAddDependency({
        source: pendingConnection.source,
        target: pendingConnection.target,
        condition,
      })
      setPendingConnection(null)
    },
    [pendingConnection, onAddDependency],
  )

  const handleConnectionCancel = useCallback(() => {
    setPendingConnection(null)
  }, [])

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center text-muted-foreground">
        <p className="text-sm">Loading swarm.yaml...</p>
      </div>
    )
  }

  if (error || parseError) {
    return (
      <div className="flex-1 flex items-center justify-center text-muted-foreground">
        <div className="text-center">
          <p className="text-red-400 text-sm">{error || parseError}</p>
          <p className="text-xs mt-2">Check that swarm/swarm.yaml exists and is valid</p>
        </div>
      </div>
    )
  }

  if (nodes.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center text-muted-foreground">
        <div className="text-center">
          <p>No tasks to display</p>
          <p className="text-xs mt-2">Add tasks to swarm.yaml to see the DAG</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1" style={{ minHeight: 0 }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodesChange={onNodesChange}
        fitView
        fitViewOptions={{ padding: 0.3 }}
        proOptions={{ hideAttribution: true }}
        nodesDraggable={true}
        nodesConnectable={true}
        elementsSelectable={true}
        onNodeClick={handleNodeClick}
        onConnect={handleConnect}
        colorMode="dark"
      >
        <Background variant={BackgroundVariant.Dots} gap={20} size={1} color="hsl(240 5% 20%)" />
        <Controls showInteractive={false} />
        <MiniMap
          nodeColor="hsl(217 91% 60%)"
          maskColor="hsl(222 84% 5% / 0.7)"
          bgColor="hsl(222 84% 5%)"
          style={{ borderRadius: 8, border: '1px solid hsl(217 33% 17%)' }}
        />
        {onResetLayout && (
          <Panel position="top-right">
            <button
              onClick={onResetLayout}
              className="px-3 py-1.5 text-xs font-medium rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 border border-border transition-colors"
            >
              Reset Layout
            </button>
          </Panel>
        )}
        {onCreateTask && (
          <Panel position="bottom-left">
            <button
              onClick={onCreateTask}
              className="px-3 py-1.5 text-xs font-medium rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              + Add Task
            </button>
          </Panel>
        )}
      </ReactFlow>
      {pendingConnection && (
        <ConnectionDialog
          sourceTask={pendingConnection.source}
          targetTask={pendingConnection.target}
          position={pendingConnection.position}
          onSelect={handleConditionSelect}
          onCancel={handleConnectionCancel}
        />
      )}
    </div>
  )
}
