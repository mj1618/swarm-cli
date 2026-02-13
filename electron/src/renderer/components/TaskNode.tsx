import { Handle, Position } from '@xyflow/react'
import type { NodeProps, Node } from '@xyflow/react'
import type { TaskNodeData } from '../lib/yamlParser'

type TaskNodeType = Node<TaskNodeData, 'taskNode'>

export default function TaskNode({ data }: NodeProps<TaskNodeType>) {
  return (
    <div className="bg-card border border-border rounded-lg shadow-lg min-w-[180px] overflow-hidden">
      <Handle type="target" position={Position.Top} className="!bg-primary !w-3 !h-1.5 !rounded-sm !border-0" />

      <div className="px-3 py-2 border-b border-border bg-primary/10">
        <span className="text-sm font-semibold text-card-foreground">{data.label}</span>
      </div>

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
      </div>

      <Handle type="source" position={Position.Bottom} className="!bg-primary !w-3 !h-1.5 !rounded-sm !border-0" />
    </div>
  )
}
