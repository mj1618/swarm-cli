import { Handle, Position } from '@xyflow/react'
import type { NodeProps, Node } from '@xyflow/react'
import type { TaskNodeData, AgentDisplayStatus } from '../lib/yamlParser'

type TaskNodeType = Node<TaskNodeData, 'taskNode'>

function StatusIndicator({ status }: { status: AgentDisplayStatus }) {
  switch (status) {
    case 'running':
      return <span className="inline-block w-2 h-2 rounded-full bg-blue-500 animate-pulse" />
    case 'paused':
      return <span className="inline-block w-2 h-2 rounded-full bg-yellow-500" />
    case 'succeeded':
      return (
        <svg className="w-3 h-3 text-green-500" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2.5">
          <path d="M3 8.5l3.5 3.5L13 4" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      )
    case 'failed':
      return (
        <svg className="w-3 h-3 text-red-500" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2.5">
          <path d="M4 4l8 8M12 4l-8 8" strokeLinecap="round" />
        </svg>
      )
  }
}

export default function TaskNode({ data, selected }: NodeProps<TaskNodeType>) {
  const { agentStatus, agentProgress, agentCost } = data
  const progressPct =
    agentProgress && agentProgress.total > 0
      ? Math.round((agentProgress.current / agentProgress.total) * 100)
      : 0

  return (
    <div className={`bg-card border rounded-lg shadow-lg min-w-[180px] overflow-hidden cursor-pointer transition-colors ${
      selected ? 'border-primary ring-2 ring-primary/30' : 'border-border hover:border-primary/50'
    } ${agentStatus === 'running' ? 'border-blue-500/50' : ''} ${agentStatus === 'failed' ? 'border-red-500/50' : ''} ${agentStatus === 'succeeded' ? 'border-green-500/50' : ''}`}>
      <Handle
        type="target"
        position={Position.Top}
        className="!bg-primary !w-3 !h-1.5 !rounded-sm !border-0 hover:!bg-green-400 hover:!w-4 hover:!h-2 !transition-all"
      />

      <div className="px-3 py-2 border-b border-border bg-primary/10">
        <div className="flex items-center gap-1.5">
          {agentStatus && <StatusIndicator status={agentStatus} />}
          <span className="text-sm font-semibold text-card-foreground">{data.label}</span>
        </div>
      </div>

      {/* Progress bar for running agents */}
      {agentStatus === 'running' && agentProgress && agentProgress.total > 0 && (
        <div className="h-0.5 bg-muted">
          <div
            className="h-full bg-blue-500 transition-all duration-500"
            style={{ width: `${progressPct}%` }}
          />
        </div>
      )}

      <div className="px-3 py-2 space-y-1">
        <div className="flex items-center gap-1.5">
          <span className="text-[10px] text-muted-foreground">prompt:</span>
          <span className="text-xs text-foreground truncate">{data.promptSource}</span>
        </div>
        {data.model && (
          <div className="flex items-center gap-1.5">
            <span className="text-[10px] text-muted-foreground">model:</span>
            <span className="text-xs px-1.5 py-0.5 rounded bg-primary/20 text-primary font-medium">
              {data.model}
            </span>
          </div>
        )}
        {/* Running agent stats */}
        {agentStatus === 'running' && (
          <div className="flex items-center justify-between">
            {agentProgress && agentProgress.total > 0 && (
              <span className="text-[10px] text-muted-foreground">
                iter {agentProgress.current}/{agentProgress.total}
              </span>
            )}
            {agentCost != null && agentCost > 0 && (
              <span className="text-[10px] text-muted-foreground">${agentCost.toFixed(2)}</span>
            )}
          </div>
        )}
        {/* Paused label */}
        {agentStatus === 'paused' && (
          <span className="text-[10px] text-yellow-500 font-medium">paused</span>
        )}
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className="!bg-primary !w-3 !h-1.5 !rounded-sm !border-0 hover:!bg-green-400 hover:!w-4 hover:!h-2 !transition-all"
      />
    </div>
  )
}
