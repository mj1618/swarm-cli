import { useMemo } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  BackgroundVariant,
} from '@xyflow/react'
import type { Node, Edge } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import TaskNode from './TaskNode'
import { composeToFlow, parseComposeFile } from '../lib/yamlParser'
import type { ComposeFile, TaskNodeData } from '../lib/yamlParser'

const nodeTypes = { taskNode: TaskNode }

interface DagCanvasProps {
  yamlContent: string | null
  loading: boolean
  error: string | null
}

export default function DagCanvas({ yamlContent, loading, error }: DagCanvasProps) {
  const { nodes, edges, parseError } = useMemo(() => {
    const empty = { nodes: [] as Node<TaskNodeData>[], edges: [] as Edge[], parseError: null as string | null }
    if (!yamlContent) return empty
    try {
      const compose: ComposeFile = parseComposeFile(yamlContent)
      if (!compose.tasks || Object.keys(compose.tasks).length === 0) {
        return { ...empty, parseError: 'No tasks found in swarm.yaml' }
      }
      const result = composeToFlow(compose)
      return { ...result, parseError: null }
    } catch (e) {
      return { ...empty, parseError: (e as Error).message }
    }
  }, [yamlContent])

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
        fitView
        fitViewOptions={{ padding: 0.3 }}
        proOptions={{ hideAttribution: true }}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
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
      </ReactFlow>
    </div>
  )
}
