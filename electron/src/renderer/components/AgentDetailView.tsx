import { useState, useEffect } from 'react'
import type { AgentState } from '../../preload/index'

interface AgentDetailViewProps {
  agent: AgentState
  onBack: () => void
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

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString()
}

function Section({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="mb-3">
      <div className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider mb-1.5">
        {label}
      </div>
      {children}
    </div>
  )
}

function DetailRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-baseline justify-between py-0.5">
      <span className="text-[11px] text-muted-foreground">{label}</span>
      <span className="text-[11px] text-foreground font-mono text-right max-w-[60%] truncate">
        {value}
      </span>
    </div>
  )
}

export default function AgentDetailView({ agent, onBack, onPause, onResume, onKill }: AgentDetailViewProps) {
  const [duration, setDuration] = useState(() => formatDuration(agent.started_at, agent.terminated_at))
  const isRunning = agent.status === 'running' && !agent.paused
  const isPaused = agent.paused
  const isActive = agent.status === 'running'

  // Tick the duration every second for running agents
  useEffect(() => {
    if (!isActive) return
    const interval = setInterval(() => {
      setDuration(formatDuration(agent.started_at, agent.terminated_at))
    }, 1000)
    return () => clearInterval(interval)
  }, [isActive, agent.started_at, agent.terminated_at])

  // Update duration when agent changes
  useEffect(() => {
    setDuration(formatDuration(agent.started_at, agent.terminated_at))
  }, [agent.started_at, agent.terminated_at])

  const statusLabel = isPaused ? 'Paused' : agent.status === 'running' ? 'Running' : 'Terminated'
  const statusColor = isPaused
    ? 'text-yellow-400'
    : agent.status === 'running'
      ? 'text-green-400'
      : 'text-zinc-400'

  const iterPct = agent.iterations > 0
    ? Math.round((agent.current_iteration / agent.iterations) * 100)
    : 0

  return (
    <div className="flex flex-col h-full">
      {/* Header with back button */}
      <div className="p-3 border-b border-border flex items-center gap-2">
        <button
          onClick={onBack}
          className="text-xs px-2 py-1 rounded bg-zinc-700 hover:bg-zinc-600 text-zinc-200 transition-colors"
        >
          ← Back
        </button>
        <h2 className="text-sm font-semibold text-foreground truncate flex-1">
          {agent.name || agent.id.slice(0, 8)}
        </h2>
        <span className={`text-[11px] font-medium ${statusColor}`}>
          {statusLabel}
        </span>
      </div>

      {/* Scrollable content */}
      <div className="flex-1 overflow-auto p-3">
        {/* Identity */}
        <Section label="Info">
          <div className="bg-zinc-800/50 rounded-lg p-2">
            <DetailRow label="ID" value={<span className="select-all">{agent.id}</span>} />
            <DetailRow label="PID" value={agent.pid} />
            <DetailRow label="Model" value={agent.model} />
            <DetailRow label="Started" value={formatTime(agent.started_at)} />
            {agent.terminated_at && (
              <DetailRow label="Terminated" value={formatTime(agent.terminated_at)} />
            )}
            <DetailRow label="Duration" value={duration} />
            <DetailRow label="Working Dir" value={agent.working_dir} />
          </div>
        </Section>

        {/* Progress */}
        {agent.iterations > 0 && (
          <Section label="Progress">
            <div className="bg-zinc-800/50 rounded-lg p-2">
              <div className="flex items-center justify-between text-[11px] text-foreground mb-1.5">
                <span>Iteration {agent.current_iteration} of {agent.iterations}</span>
                <span>{iterPct}%</span>
              </div>
              <div className="w-full h-1.5 bg-zinc-700 rounded-full overflow-hidden mb-1.5">
                <div
                  className="h-full bg-blue-500 rounded-full transition-all duration-300"
                  style={{ width: `${iterPct}%` }}
                />
              </div>
              <div className="flex gap-4 text-[11px]">
                <span className="text-green-400">
                  {agent.successful_iterations.toLocaleString()} succeeded
                </span>
                <span className="text-red-400">
                  {agent.failed_iterations.toLocaleString()} failed
                </span>
              </div>
            </div>
          </Section>
        )}

        {/* Usage */}
        <Section label="Usage">
          <div className="bg-zinc-800/50 rounded-lg p-2">
            <DetailRow label="Input Tokens" value={agent.input_tokens.toLocaleString()} />
            <DetailRow label="Output Tokens" value={agent.output_tokens.toLocaleString()} />
            <DetailRow label="Total Cost" value={formatCost(agent.total_cost_usd)} />
          </div>
        </Section>

        {/* Current Task */}
        {agent.current_task && isActive && (
          <Section label="Current Task">
            <div className="bg-zinc-800/50 rounded-lg p-2">
              <p className="text-[11px] text-foreground italic break-words">{agent.current_task}</p>
            </div>
          </Section>
        )}

        {/* Exit info for terminated agents */}
        {agent.status === 'terminated' && agent.exit_reason && (
          <Section label="Result">
            <div className="bg-zinc-800/50 rounded-lg p-2">
              <DetailRow
                label="Exit Reason"
                value={
                  agent.exit_reason === 'completed' ? '✓ Completed' :
                  agent.exit_reason === 'killed' ? '✕ Killed' :
                  agent.exit_reason === 'crashed' ? '✕ Crashed' :
                  agent.exit_reason
                }
              />
              {agent.last_error && (
                <div className="mt-1 text-[11px] text-red-400 break-words">
                  {agent.last_error}
                </div>
              )}
            </div>
          </Section>
        )}

        {/* Log File */}
        {agent.log_file && (
          <Section label="Log File">
            <p className="text-[10px] font-mono text-muted-foreground break-all select-all">
              {agent.log_file}
            </p>
          </Section>
        )}

        {/* Controls */}
        {isActive && (
          <div className="flex gap-2 mt-1">
            {isRunning && (
              <button
                onClick={() => onPause(agent.id)}
                className="text-xs px-3 py-1.5 rounded bg-zinc-700 hover:bg-zinc-600 text-zinc-200 transition-colors"
              >
                Pause
              </button>
            )}
            {isPaused && (
              <button
                onClick={() => onResume(agent.id)}
                className="text-xs px-3 py-1.5 rounded bg-zinc-700 hover:bg-zinc-600 text-zinc-200 transition-colors"
              >
                Resume
              </button>
            )}
            <button
              onClick={() => onKill(agent.id)}
              className="text-xs px-3 py-1.5 rounded bg-red-900/50 hover:bg-red-900/70 text-red-200 transition-colors"
            >
              Stop
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
