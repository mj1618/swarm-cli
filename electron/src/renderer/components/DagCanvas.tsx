import { useMemo, useCallback, useState, useEffect, useRef } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  BackgroundVariant,
  Panel,
  applyNodeChanges,
  applyEdgeChanges,
  useReactFlow,
} from '@xyflow/react'
import type { Node, Edge, NodeChange, EdgeChange, Connection } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import TaskNode from './TaskNode'
import ConnectionDialog from './ConnectionDialog'
import { composeToFlow, parseComposeFile } from '../lib/yamlParser'
import type { ComposeFile, TaskDef, TaskDependency, TaskNodeData, AgentDisplayStatus } from '../lib/yamlParser'
import { validateDag } from '../lib/dagValidation'
import type { ValidationResult } from '../lib/dagValidation'
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
  activePipeline?: string | null
  pipelineTasks?: string[] | null
  onSelectTask?: (task: { name: string; def: TaskDef; compose: ComposeFile }) => void
  onNavigateToAgent?: (agentId: string) => void
  onAddDependency?: (dep: { source: string; target: string; condition: TaskDependency['condition'] }) => void
  onDeleteTask?: (taskName: string) => void
  onDeleteEdge?: (source: string, target: string) => void
  onRunTask?: (taskName: string, taskDef: TaskDef) => void
  onCreateTask?: () => void
  onDropCreateTask?: (promptName: string, position: { x: number; y: number }) => void
  savedPositions?: Record<string, { x: number; y: number }>
  onPositionsChange?: (positions: Record<string, { x: number; y: number }>) => void
  onResetLayout?: () => void
  onFitViewReady?: (fitView: () => void) => void
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
  activePipeline,
  pipelineTasks,
  onSelectTask,
  onNavigateToAgent,
  onAddDependency,
  onDeleteTask,
  onDeleteEdge,
  onRunTask,
  onCreateTask,
  onDropCreateTask,
  savedPositions,
  onPositionsChange,
  onResetLayout,
  onFitViewReady,
}: DagCanvasProps) {
  // Parse YAML and compute dagre layout (with saved positions applied)
  const { initialNodes, edges, parseError, compose, validation } = useMemo(() => {
    const empty = {
      initialNodes: [] as Node<TaskNodeData>[],
      edges: [] as Edge[],
      parseError: null as string | null,
      compose: null as ComposeFile | null,
      validation: null as ValidationResult | null,
    }
    if (!yamlContent) return empty
    try {
      const compose: ComposeFile = parseComposeFile(yamlContent)
      if (!compose.tasks || Object.keys(compose.tasks).length === 0) {
        return { ...empty, parseError: 'No tasks found in swarm.yaml' }
      }
      const result = composeToFlow(compose, savedPositions)
      const validation = validateDag(compose)

      // Inject validation state into node data
      const nodesWithValidation = result.nodes.map(node => ({
        ...node,
        data: {
          ...node.data,
          isInCycle: validation.cycleNodes.has(node.id),
          isOrphan: validation.orphanedTasks.has(node.id),
        },
      }))

      // Override edge styles for cycle edges and make all edges selectable
      const edgesWithValidation = result.edges.map(edge => {
        const base = { ...edge, selectable: true }
        if (validation.cycleEdges.has(edge.id)) {
          return {
            ...base,
            style: { stroke: '#ef4444', strokeWidth: 2.5 },
            labelStyle: { ...base.labelStyle, fill: '#ef4444' },
            animated: true,
          }
        }
        return base
      })

      return {
        initialNodes: nodesWithValidation,
        edges: edgesWithValidation,
        parseError: null,
        compose,
        validation,
      }
    } catch (e) {
      return { ...empty, parseError: (e as Error).message }
    }
  }, [yamlContent, savedPositions])

  // Enrich nodes with agent status data
  const enrichedNodes = useMemo(() => {
    // Build a set of pipeline tasks for quick lookup
    const pipelineTaskSet = pipelineTasks ? new Set(pipelineTasks) : null

    // Determine if the pipeline is actively running:
    // A pipeline is running when at least one agent with status 'running' matches a pipeline task
    const isPipelineRunning = !!(
      activePipeline &&
      pipelineTaskSet &&
      agents?.some(
        (a) =>
          a.status === 'running' &&
          (pipelineTaskSet.has(a.name) ||
            (a.labels?.task_id && pipelineTaskSet.has(a.labels.task_id)) ||
            (a.current_task && pipelineTaskSet.has(a.current_task)))
      )
    )

    return initialNodes.map((node) => {
      const agent = agents?.find(
        (a) => a.name === node.id || a.labels?.task_id === node.id || a.current_task === node.id,
      )

      // If the node has an agent, use the agent's status
      if (agent) {
        return {
          ...node,
          data: {
            ...node.data,
            agentStatus: resolveAgentStatus(agent),
            agentProgress: { current: agent.current_iteration, total: agent.iterations },
            agentCost: agent.total_cost_usd,
          },
        }
      }

      // If the pipeline is actively running and this task is part of it, show pending status
      if (isPipelineRunning && pipelineTaskSet?.has(node.id)) {
        return {
          ...node,
          data: {
            ...node.data,
            agentStatus: 'pending' as const,
          },
        }
      }

      return node
    })
  }, [initialNodes, agents, activePipeline, pipelineTasks])

  // Dim nodes not in the active pipeline
  const filteredNodes = useMemo(() => {
    if (!activePipeline || !pipelineTasks) return enrichedNodes
    const taskSet = new Set(pipelineTasks)
    return enrichedNodes.map(node => {
      const inPipeline = taskSet.has(node.id)
      return {
        ...node,
        style: inPipeline ? node.style : { ...node.style, opacity: 0.35 },
      }
    })
  }, [enrichedNodes, activePipeline, pipelineTasks])

  // Drop target visual indicator
  const [isDragOver, setIsDragOver] = useState(false)

  // Delete confirmation dialog state
  const [deleteConfirm, setDeleteConfirm] = useState<{ taskName: string } | null>(null)

  // Context menu state
  const [contextMenu, setContextMenu] = useState<{ taskName: string; x: number; y: number } | null>(null)
  const contextMenuRef = useRef<HTMLDivElement>(null)

  // Local node state for drag interactions
  const [nodes, setNodes] = useState<Node<TaskNodeData>[]>(filteredNodes)

  // Local edge state for selection interactions
  const [localEdges, setLocalEdges] = useState<Edge[]>(edges)

  // Sync local state when nodes change (YAML reload, positions reset, agent status, or pipeline filter)
  useEffect(() => {
    setNodes(filteredNodes)
  }, [filteredNodes])

  // Sync local edges when edges change
  useEffect(() => {
    setLocalEdges(edges)
  }, [edges])

  // Apply visual highlight to selected edges
  const styledEdges = useMemo(() => {
    return localEdges.map(edge => {
      if (!edge.selected) return edge
      return {
        ...edge,
        style: { ...edge.style, strokeWidth: 4, filter: 'brightness(1.5)' },
      }
    })
  }, [localEdges])

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
      // If this node has an active agent, navigate to its detail view instead of the task drawer
      if (onNavigateToAgent && agents && node.data.agentStatus) {
        const agent = agents.find(
          (a) => a.name === node.id || a.labels?.task_id === node.id || a.current_task === node.id,
        )
        if (agent) {
          onNavigateToAgent(agent.id)
          return
        }
      }
      if (onSelectTask && compose) {
        onSelectTask({ name: node.data.label, def: node.data.taskDef, compose })
      }
    },
    [onSelectTask, onNavigateToAgent, compose, agents],
  )

  // Connection dialog state
  const [pendingConnection, setPendingConnection] = useState<PendingConnection | null>(null)
  const { flowToScreenPosition, screenToFlowPosition, fitView } = useReactFlow()

  useEffect(() => {
    if (onFitViewReady) {
      onFitViewReady(() => fitView({ padding: 0.3 }))
    }
  }, [onFitViewReady, fitView])

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

  const onEdgesChange = useCallback(
    (changes: EdgeChange<Edge>[]) => {
      setLocalEdges((prev) => applyEdgeChanges(changes, prev))
    },
    [],
  )

  // Keyboard handler for deleting selected nodes/edges
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key !== 'Backspace' && e.key !== 'Delete') return
      // Don't intercept when typing in inputs
      const tag = (e.target as HTMLElement)?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return

      // Check for selected edges first
      const selectedEdge = localEdges.find(edge => edge.selected)
      if (selectedEdge && onDeleteEdge) {
        e.preventDefault()
        onDeleteEdge(selectedEdge.source, selectedEdge.target)
        return
      }

      // Check for selected nodes
      const selectedNode = nodes.find(node => node.selected)
      if (selectedNode && onDeleteTask) {
        e.preventDefault()
        setDeleteConfirm({ taskName: selectedNode.id })
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [nodes, localEdges, onDeleteTask, onDeleteEdge])

  // Close context menu on outside click
  useEffect(() => {
    if (!contextMenu) return
    function handleClick(e: MouseEvent) {
      if (contextMenuRef.current && !contextMenuRef.current.contains(e.target as globalThis.Node)) {
        setContextMenu(null)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [contextMenu])

  const handleNodeContextMenu = useCallback(
    (event: React.MouseEvent, node: Node<TaskNodeData>) => {
      event.preventDefault()
      if (!onDeleteTask && !onRunTask) return
      setContextMenu({ taskName: node.id, x: event.clientX, y: event.clientY })
    },
    [onDeleteTask, onRunTask],
  )

  const handleConfirmDelete = useCallback(() => {
    if (deleteConfirm && onDeleteTask) {
      onDeleteTask(deleteConfirm.taskName)
    }
    setDeleteConfirm(null)
  }, [deleteConfirm, onDeleteTask])

  const handleCancelDelete = useCallback(() => {
    setDeleteConfirm(null)
  }, [])

  const handleContextMenuRun = useCallback(() => {
    if (contextMenu && onRunTask && compose) {
      const taskDef = compose.tasks?.[contextMenu.taskName]
      if (taskDef) {
        onRunTask(contextMenu.taskName, taskDef)
      }
      setContextMenu(null)
    }
  }, [contextMenu, onRunTask, compose])

  const handleContextMenuDelete = useCallback(() => {
    if (contextMenu) {
      setDeleteConfirm({ taskName: contextMenu.taskName })
      setContextMenu(null)
    }
  }, [contextMenu])

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

  const handleDragOver = useCallback((e: React.DragEvent) => {
    if (e.dataTransfer.types.includes('application/swarm-prompt')) {
      e.preventDefault()
      e.dataTransfer.dropEffect = 'copy'
      setIsDragOver(true)
    }
  }, [])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    // Only clear when leaving the container (not entering a child)
    if (e.currentTarget === e.target || !e.currentTarget.contains(e.relatedTarget as globalThis.Node)) {
      setIsDragOver(false)
    }
  }, [])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setIsDragOver(false)
    const promptName = e.dataTransfer.getData('application/swarm-prompt')
    if (promptName && onDropCreateTask) {
      const position = screenToFlowPosition({ x: e.clientX, y: e.clientY })
      onDropCreateTask(promptName, position)
    }
  }, [onDropCreateTask, screenToFlowPosition])

  return (
    <div
      className={`flex-1 transition-colors ${isDragOver ? 'ring-2 ring-inset ring-blue-500/50 bg-blue-500/5' : ''}`}
      style={{ minHeight: 0 }}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <ReactFlow
        nodes={nodes}
        edges={styledEdges}
        nodeTypes={nodeTypes}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        fitView
        fitViewOptions={{ padding: 0.3 }}
        proOptions={{ hideAttribution: true }}
        nodesDraggable={true}
        nodesConnectable={true}
        elementsSelectable={true}
        edgesFocusable={true}
        onNodeClick={handleNodeClick}
        onNodeContextMenu={handleNodeContextMenu}
        onConnect={handleConnect}
        deleteKeyCode={null}
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
        {validation && (validation.cycleNodes.size > 0 || validation.orphanedTasks.size > 0) && (
          <Panel position="top-left">
            <div className="px-3 py-2 rounded-md bg-card/95 border border-border text-xs space-y-1 max-w-[300px]">
              {validation.cycleNodes.size > 0 && (
                <div className="flex items-start gap-1.5 text-red-400">
                  <span className="shrink-0 mt-px">&#9888;</span>
                  <span>Cycle detected: {[...validation.cycleNodes].join(', ')}</span>
                </div>
              )}
              {validation.orphanedTasks.size > 0 && (
                <div className="flex items-start gap-1.5 text-amber-400">
                  <span className="shrink-0 mt-px">&#9888;</span>
                  <span>Orphaned: {[...validation.orphanedTasks].join(', ')}</span>
                </div>
              )}
            </div>
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

      {/* Context menu */}
      {contextMenu && (
        <div
          ref={contextMenuRef}
          className="fixed z-50 min-w-[140px] rounded-md border border-border bg-popover py-1 shadow-md"
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          {onRunTask && (
            <button
              className="w-full px-3 py-1.5 text-left text-sm text-foreground hover:bg-secondary/80 transition-colors"
              onClick={handleContextMenuRun}
            >
              Run Task
            </button>
          )}
          {onDeleteTask && (
            <button
              className="w-full px-3 py-1.5 text-left text-sm text-red-400 hover:bg-secondary/80 transition-colors"
              onClick={handleContextMenuDelete}
            >
              Delete Task
            </button>
          )}
        </div>
      )}

      {/* Delete confirmation dialog */}
      {deleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="rounded-lg border border-border bg-card p-6 shadow-lg max-w-sm mx-4">
            <h3 className="text-sm font-semibold text-foreground mb-2">Delete task?</h3>
            <p className="text-sm text-muted-foreground mb-4">
              Delete task &ldquo;{deleteConfirm.taskName}&rdquo;? This will also remove all its dependencies.
            </p>
            <div className="flex justify-end gap-2">
              <button
                className="px-3 py-1.5 text-xs font-medium rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 border border-border transition-colors"
                onClick={handleCancelDelete}
              >
                Cancel
              </button>
              <button
                className="px-3 py-1.5 text-xs font-medium rounded-md bg-red-600 text-white hover:bg-red-700 transition-colors"
                onClick={handleConfirmDelete}
                autoFocus
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
