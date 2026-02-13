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
  getNodesBounds,
  getViewportForBounds,
} from '@xyflow/react'
import type { Node, Edge, NodeChange, EdgeChange, Connection } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { toPng, toSvg } from 'html-to-image'
import TaskNode from './TaskNode'
import ConnectionDialog from './ConnectionDialog'
import DagSearchBox from './DagSearchBox'
import { composeToFlow, parseComposeFile } from '../lib/yamlParser'
import type { ComposeFile, TaskDef, TaskDependency, TaskNodeData, AgentDisplayStatus } from '../lib/yamlParser'
import { validateDag } from '../lib/dagValidation'
import type { ValidationResult } from '../lib/dagValidation'
import type { AgentState } from '../../preload/index'
import type { EffectiveTheme } from '../lib/themeManager'
import type { ToastType } from './ToastContainer'

const nodeTypes = { taskNode: TaskNode }

interface PendingConnection {
  source: string
  target: string
  position: { x: number; y: number }
}

interface ViewportState {
  x: number
  y: number
  zoom: number
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
  onDuplicateTask?: (taskName: string, taskDef: TaskDef) => void
  onCreateTask?: () => void
  onDropCreateTask?: (promptName: string, position: { x: number; y: number }) => void
  savedPositions?: Record<string, { x: number; y: number }>
  onPositionsChange?: (positions: Record<string, { x: number; y: number }>) => void
  savedViewport?: ViewportState | null
  onViewportChange?: (viewport: ViewportState) => void
  onResetLayout?: () => void
  onFitViewReady?: (fitView: () => void) => void
  onToast?: (type: ToastType, message: string) => void
  theme?: EffectiveTheme
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
  onDuplicateTask,
  onCreateTask,
  onDropCreateTask,
  savedPositions,
  onPositionsChange,
  savedViewport,
  onViewportChange,
  onResetLayout,
  onFitViewReady,
  onToast,
  theme = 'dark',
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
          isInParallelPipeline: validation.parallelTasks.has(node.id),
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

  // Export dropdown state
  const [exportDropdownOpen, setExportDropdownOpen] = useState(false)
  const exportDropdownRef = useRef<HTMLDivElement>(null)
  const [isExporting, setIsExporting] = useState(false)

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
  const { flowToScreenPosition, screenToFlowPosition, fitView, getNodes, setCenter, getZoom, setViewport, getViewport } = useReactFlow()

  // Track if we've restored the viewport for this file (to avoid re-restoring after fitView)
  const viewportRestoredRef = useRef(false)

  // Restore saved viewport on mount (only once per file)
  useEffect(() => {
    if (savedViewport && !viewportRestoredRef.current && nodes.length > 0) {
      // Delay slightly to ensure React Flow is fully initialized
      const timer = setTimeout(() => {
        setViewport(savedViewport, { duration: 0 })
        viewportRestoredRef.current = true
      }, 50)
      return () => clearTimeout(timer)
    }
  }, [savedViewport, setViewport, nodes.length])

  // Reset the viewport restored flag when savedViewport changes (i.e., switching files)
  useEffect(() => {
    viewportRestoredRef.current = false
  }, [savedViewport])

  // Handle viewport changes (pan/zoom end)
  const handleMoveEnd = useCallback(
    (_event: MouseEvent | TouchEvent | null, viewport: { x: number; y: number; zoom: number }) => {
      if (onViewportChange) {
        onViewportChange({ x: viewport.x, y: viewport.y, zoom: viewport.zoom })
      }
    },
    [onViewportChange],
  )

  useEffect(() => {
    if (onFitViewReady) {
      onFitViewReady(() => fitView({ padding: 0.3 }))
    }
  }, [onFitViewReady, fitView])

  // Get task names for search (respecting pipeline filter)
  const searchableTaskNames = useMemo(() => {
    // If pipeline filter is active, only show tasks in that pipeline
    if (activePipeline && pipelineTasks) {
      const taskSet = new Set(pipelineTasks)
      return nodes.filter(n => taskSet.has(n.id)).map(n => n.id)
    }
    return nodes.map(n => n.id)
  }, [nodes, activePipeline, pipelineTasks])

  // Zoom to a specific node by ID
  const handleZoomToNode = useCallback((nodeId: string) => {
    const node = getNodes().find(n => n.id === nodeId)
    if (node) {
      const x = node.position.x + (node.measured?.width ?? 150) / 2
      const y = node.position.y + (node.measured?.height ?? 60) / 2
      setCenter(x, y, { zoom: Math.max(getZoom(), 1), duration: 300 })
    }
  }, [getNodes, setCenter, getZoom])

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

  // Keyboard handler for DAG canvas shortcuts
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      // Don't intercept when typing in inputs
      const tag = (e.target as HTMLElement)?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return

      // N — Create new task
      if (e.key === 'n' || e.key === 'N') {
        if (onCreateTask) {
          e.preventDefault()
          onCreateTask()
        }
        return
      }

      // F — Fit view
      if (e.key === 'f' || e.key === 'F') {
        e.preventDefault()
        fitView({ padding: 0.3 })
        return
      }

      // R — Reset layout
      if (e.key === 'r' || e.key === 'R') {
        if (onResetLayout) {
          e.preventDefault()
          onResetLayout()
        }
        return
      }

      // Escape — Deselect all nodes and edges
      if (e.key === 'Escape') {
        setNodes(prev => prev.map(n => ({ ...n, selected: false })))
        setLocalEdges(prev => prev.map(edge => ({ ...edge, selected: false })))
        return
      }

      // Delete/Backspace — Delete selected node or edge
      if (e.key !== 'Backspace' && e.key !== 'Delete') return

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
  }, [nodes, localEdges, onDeleteTask, onDeleteEdge, onCreateTask, onResetLayout, fitView])

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

  // Close export dropdown on outside click
  useEffect(() => {
    if (!exportDropdownOpen) return
    function handleClick(e: MouseEvent) {
      if (exportDropdownRef.current && !exportDropdownRef.current.contains(e.target as globalThis.Node)) {
        setExportDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [exportDropdownOpen])

  // Export DAG as image
  const handleExport = useCallback(async (format: 'png' | 'svg') => {
    setExportDropdownOpen(false)
    setIsExporting(true)

    try {
      // Find the React Flow viewport element
      const viewport = document.querySelector('.react-flow__viewport') as HTMLElement
      if (!viewport) {
        onToast?.('error', 'Could not find DAG viewport')
        setIsExporting(false)
        return
      }

      // Calculate bounds for all nodes with padding
      const currentNodes = getNodes()
      if (currentNodes.length === 0) {
        onToast?.('error', 'No nodes to export')
        setIsExporting(false)
        return
      }

      const bounds = getNodesBounds(currentNodes)
      const padding = 50
      const imageWidth = bounds.width + padding * 2
      const imageHeight = bounds.height + padding * 2

      // Get viewport transformation for centering
      const transform = getViewportForBounds(bounds, imageWidth, imageHeight, 0.5, 2, padding)

      // Generate the image
      const bgColor = theme === 'light' ? '#ffffff' : '#0f172a'
      let dataUrl: string
      if (format === 'svg') {
        dataUrl = await toSvg(viewport, {
          backgroundColor: bgColor,
          width: imageWidth,
          height: imageHeight,
          style: {
            transform: `translate(${transform.x}px, ${transform.y}px) scale(${transform.zoom})`,
          },
        })
      } else {
        dataUrl = await toPng(viewport, {
          backgroundColor: bgColor,
          width: imageWidth,
          height: imageHeight,
          pixelRatio: 2, // High DPI for better quality
          style: {
            transform: `translate(${transform.x}px, ${transform.y}px) scale(${transform.zoom})`,
          },
        })
      }

      // Prompt save dialog
      const timestamp = new Date().toISOString().slice(0, 10)
      const defaultName = `dag-export-${timestamp}.${format}`
      const result = await window.dialog.saveImage({ defaultName, dataUrl, format })

      if (result.error) {
        onToast?.('error', `Export failed: ${result.error}`)
      } else if (!result.canceled) {
        onToast?.('success', `DAG exported as ${format.toUpperCase()}`)
      }
    } catch (err) {
      console.error('Export error:', err)
      onToast?.('error', `Export failed: ${(err as Error).message}`)
    } finally {
      setIsExporting(false)
    }
  }, [getNodes, onToast, theme])

  const handleNodeContextMenu = useCallback(
    (event: React.MouseEvent, node: Node<TaskNodeData>) => {
      event.preventDefault()
      if (!onDeleteTask && !onRunTask && !onDuplicateTask) return
      setContextMenu({ taskName: node.id, x: event.clientX, y: event.clientY })
    },
    [onDeleteTask, onRunTask, onDuplicateTask],
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

  const handleContextMenuDuplicate = useCallback(() => {
    if (contextMenu && onDuplicateTask && compose) {
      const taskDef = compose.tasks?.[contextMenu.taskName]
      if (taskDef) {
        onDuplicateTask(contextMenu.taskName, taskDef)
      }
      setContextMenu(null)
    }
  }, [contextMenu, onDuplicateTask, compose])

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
    // Reference export state to satisfy TypeScript (export only available when nodes exist)
    void isExporting
    void handleExport
    return (
      <div className="flex-1 flex items-center justify-center text-muted-foreground">
        <div className="text-center max-w-md px-6">
          {/* Visual icon */}
          <div className="mb-6 flex justify-center">
            <div className="w-20 h-20 rounded-2xl bg-secondary/50 border border-border flex items-center justify-center">
              <svg
                className="w-10 h-10 text-muted-foreground/60"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={1.5}
                  d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7m0 10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2h-2a2 2 0 00-2 2"
                />
              </svg>
            </div>
          </div>

          {/* Heading */}
          <h2 className="text-lg font-semibold text-foreground mb-2">No tasks yet</h2>

          {/* Explanation */}
          <p className="text-sm text-muted-foreground mb-6">
            Tasks are the building blocks of your pipeline. Each task runs an AI agent with a specific prompt.
          </p>

          {/* Create Task button */}
          {onCreateTask && (
            <button
              onClick={onCreateTask}
              className="px-4 py-2 text-sm font-medium rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors mb-6"
            >
              + Create Task
            </button>
          )}

          {/* Tips */}
          <div className="space-y-3 text-xs text-muted-foreground/80">
            <div className="flex items-start gap-2">
              <span className="text-blue-400 mt-0.5">💡</span>
              <span className="text-left">
                <strong className="text-muted-foreground">Drag &amp; drop:</strong> Drag a prompt from the File Tree on the left to create a task
              </span>
            </div>
            <div className="flex items-start gap-2">
              <span className="text-blue-400 mt-0.5">📝</span>
              <span className="text-left">
                <strong className="text-muted-foreground">Edit directly:</strong> Add tasks to <code className="px-1 py-0.5 rounded bg-secondary text-foreground">swarm/swarm.yaml</code>
              </span>
            </div>
          </div>
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
        fitView={!savedViewport}
        fitViewOptions={{ padding: 0.3 }}
        defaultViewport={savedViewport ?? undefined}
        proOptions={{ hideAttribution: true }}
        nodesDraggable={true}
        nodesConnectable={true}
        elementsSelectable={true}
        edgesFocusable={true}
        onNodeClick={handleNodeClick}
        onNodeContextMenu={handleNodeContextMenu}
        onConnect={handleConnect}
        onMoveEnd={handleMoveEnd}
        deleteKeyCode={null}
        colorMode={theme}
      >
        <Background variant={BackgroundVariant.Dots} gap={20} size={1} color="hsl(240 5% 20%)" />
        <Controls showInteractive={false} />
        <MiniMap
          nodeColor={(node) => {
            // Return color based on node status
            const status = (node.data as TaskNodeData)?.agentStatus
            if (status === 'running') return '#3b82f6' // blue
            if (status === 'paused') return '#f59e0b' // amber
            if (status === 'succeeded') return '#22c55e' // green
            if (status === 'failed') return '#ef4444' // red
            if (status === 'pending') return '#6b7280' // gray
            // Default color for idle/no status
            return theme === 'light' ? '#64748b' : '#94a3b8'
          }}
          maskColor={theme === 'light' ? 'rgba(255, 255, 255, 0.8)' : 'rgba(0, 0, 0, 0.7)'}
          bgColor={theme === 'light' ? 'rgba(248, 250, 252, 0.9)' : 'rgba(15, 23, 42, 0.9)'}
          style={{
            borderRadius: 8,
            border: theme === 'light' ? '1px solid hsl(214 32% 91%)' : '1px solid hsl(217 33% 17%)',
          }}
          pannable
          zoomable
        />
        <Panel position="top-right">
          <div className="flex items-center gap-2">
            {/* Export dropdown */}
            <div className="relative" ref={exportDropdownRef}>
              <button
                onClick={() => setExportDropdownOpen(prev => !prev)}
                disabled={isExporting || nodes.length === 0}
                className="px-3 py-1.5 text-xs font-medium rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 border border-border transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5"
                title="Export DAG as image"
              >
                {isExporting ? (
                  <>
                    <svg className="animate-spin h-3 w-3" viewBox="0 0 24 24" fill="none">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                    </svg>
                    Exporting...
                  </>
                ) : (
                  <>
                    <svg className="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" />
                      <polyline points="7 10 12 15 17 10" />
                      <line x1="12" y1="15" x2="12" y2="3" />
                    </svg>
                    Export
                  </>
                )}
              </button>
              {exportDropdownOpen && (
                <div className="absolute right-0 mt-1 w-36 rounded-md border border-border bg-popover shadow-lg py-1 z-50">
                  <button
                    onClick={() => handleExport('png')}
                    className="w-full px-3 py-1.5 text-left text-xs text-foreground hover:bg-secondary/80 transition-colors"
                  >
                    Export as PNG
                  </button>
                  <button
                    onClick={() => handleExport('svg')}
                    className="w-full px-3 py-1.5 text-left text-xs text-foreground hover:bg-secondary/80 transition-colors"
                  >
                    Export as SVG
                  </button>
                </div>
              )}
            </div>
            {onResetLayout && (
              <button
                onClick={onResetLayout}
                className="px-3 py-1.5 text-xs font-medium rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 border border-border transition-colors"
              >
                Reset Layout
              </button>
            )}
          </div>
        </Panel>
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
        {/* Top-left panel: Search + Validation warnings */}
        <Panel position="top-left">
          <div className="space-y-2">
            {/* Search box */}
            <DagSearchBox
              taskNames={searchableTaskNames}
              onSelectTask={handleZoomToNode}
              disabled={nodes.length === 0}
            />
            {/* Validation warnings */}
            {validation && (validation.cycleNodes.size > 0 || validation.orphanedTasks.size > 0) && (
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
            )}
          </div>
        </Panel>
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
          {onDuplicateTask && (
            <button
              className="w-full px-3 py-1.5 text-left text-sm text-foreground hover:bg-secondary/80 transition-colors"
              onClick={handleContextMenuDuplicate}
            >
              Duplicate Task
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
