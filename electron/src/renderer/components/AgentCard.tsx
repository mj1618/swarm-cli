import type { AgentState } from '../../preload/index'

interface AgentCardProps {
  agent: AgentState
  onPause: (id: string) => void
  onResume: (id: string) => void
  onKill: (id: string) => void
}

function formatDuration(startedAt: string, terminatedAt?: string): string {
  const start = new Date(startedAt).getTime()
  const end = terminatedAt ? new Date(terminatedAt).getTime() : Date.now()
  const seconds = Math.floor((end - start) / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const secs = seconds % 60
  if (minutes < 60) return `${minutes}m ${secs}s`
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  return `${hours}h ${mins}m`
}

function formatCost(cost: number): string {
  if (cost === 0) return '$0.00'
  if (cost < 0.01) return `$${cost.toFixed(4)}`
  return `$${cost.toFixed(2)}`
}

function StatusDot({ status, paused }: { status: string; paused: boolean }) {
  if (paused) {
    return <span className="inline-block w-2 h-2 rounded-full bg-yellow-400" />
  }
  if (status === 'running') {
    return <span className="inline-block w-2 h-2 rounded-full bg-green-400 animate-pulse" />
  }
  if (status === 'terminated') {
    return <span className="inline-block w-2 h-2 rounded-full bg-zinc-500" />
  }
  return <span className="inline-block w-2 h-2 rounded-full bg-zinc-600" />
}

function ProgressBar({ current, total }: { current: number; total: number }) {
  const pct = total > 0 ? Math.min((current / total) * 100, 100) : 0
  return (
    <div className="w-full h-1.5 bg-zinc-700 rounded-full overflow-hidden">
      <div
        className="h-full bg-blue-500 rounded-full transition-all duration-300"
        style={{ width: `${pct}%` }}
      />
    </div>
  )
}

export default function AgentCard({ agent, onPause, onResume, onKill }: AgentCardProps) {
  const displayName = agent.name || agent.id.slice(0, 8)
  const isRunning = agent.status === 'running' && !agent.paused
  const isPaused = agent.paused
  const isActive = agent.status === 'running'

  return (
    <div className="p-3 mb-2 rounded-lg bg-background border border-border hover:border-zinc-600 transition-colors">
      {/* Header: status + name + model */}
      <div className="flex items-center gap-2 mb-1.5">
        <StatusDot status={agent.status} paused={agent.paused} />
        <span className="text-sm font-medium text-foreground truncate flex-1">
          {displayName}
        </span>
        <span className="text-[10px] text-muted-foreground bg-zinc-800 px-1.5 py-0.5 rounded">
          {agent.model}
        </span>
      </div>

      {/* Iteration progress */}
      {agent.iterations > 0 && (
        <div className="mb-2">
          <div className="flex items-center justify-between text-[11px] text-muted-foreground mb-1">
            <span>{agent.current_iteration} of {agent.iterations}</span>
            <span>{Math.round((agent.current_iteration / agent.iterations) * 100)}%</span>
          </div>
          <ProgressBar current={agent.current_iteration} total={agent.iterations} />
        </div>
      )}

      {/* Stats row */}
      <div className="flex items-center gap-3 text-[11px] text-muted-foreground mb-2">
        <span>{formatCost(agent.total_cost_usd)}</span>
        <span>{formatDuration(agent.started_at, agent.terminated_at)}</span>
      </div>

      {/* Current task */}
      {agent.current_task && isActive && (
        <p className="text-[11px] text-muted-foreground mb-2 truncate italic">
          {agent.current_task}
        </p>
      )}

      {/* Status label for paused */}
      {isPaused && (
        <p className="text-[11px] text-yellow-400 mb-2">Paused</p>
      )}

      {/* Exit reason for terminated */}
      {agent.status === 'terminated' && agent.exit_reason && (
        <p className="text-[11px] text-muted-foreground mb-2">
          {agent.exit_reason === 'completed' ? '✓ Completed' :
           agent.exit_reason === 'killed' ? '✕ Killed' :
           agent.exit_reason === 'crashed' ? '✕ Crashed' :
           agent.exit_reason}
          {agent.last_error && ` — ${agent.last_error}`}
        </p>
      )}

      {/* Controls */}
      {isActive && (
        <div className="flex gap-1.5">
          {isRunning && (
            <button
              onClick={() => onPause(agent.id)}
              className="text-[11px] px-2 py-1 rounded bg-zinc-700 hover:bg-zinc-600 text-zinc-200 transition-colors"
            >
              Pause
            </button>
          )}
          {isPaused && (
            <button
              onClick={() => onResume(agent.id)}
              className="text-[11px] px-2 py-1 rounded bg-zinc-700 hover:bg-zinc-600 text-zinc-200 transition-colors"
            >
              Resume
            </button>
          )}
          <button
            onClick={() => onKill(agent.id)}
            className="text-[11px] px-2 py-1 rounded bg-red-900/50 hover:bg-red-900/70 text-red-200 transition-colors"
          >
            Stop
          </button>
        </div>
      )}
    </div>
  )
}
