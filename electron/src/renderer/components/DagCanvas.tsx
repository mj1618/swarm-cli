import { useMemo, useCallback, useState, useEffect } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  BackgroundVariant,
  Panel,
  applyNodeChanges,
} from '@xyflow/react'
import type { Node, Edge, NodeChange } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import TaskNode from './TaskNode'
import { composeToFlow, parseComposeFile } from '../lib/yamlParser'
import type { ComposeFile, TaskDef, TaskNodeData } from '../lib/yamlParser'

const nodeTypes = { taskNode: TaskNode }

interface DagCanvasProps {
  yamlContent: string | null
  loading: boolean
  error: string | null
  onSelectTask?: (task: { name: string; def: TaskDef; compose: ComposeFile }) => void
  savedPositions?: Record<string, { x: number; y: number }>
  onPositionsChange?: (positions: Record<string, { x: number; y: number }>) => void
  onResetLayout?: () => void
}

export default function DagCanvas({
  yamlContent,
  loading,
  error,
  onSelectTask,
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

  // Local node state for drag interactions
  const [nodes, setNodes] = useState<Node<TaskNodeData>[]>(initialNodes)

  // Sync local state when initial nodes change (YAML reload or positions reset)
  useEffect(() => {
    setNodes(initialNodes)
  }, [initialNodes])

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
        nodesConnectable={false}
        elementsSelectable={true}
        onNodeClick={handleNodeClick}
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
      </ReactFlow>
    </div>
  )
}
