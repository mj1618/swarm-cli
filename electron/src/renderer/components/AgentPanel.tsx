import { useState, useEffect, useCallback } from 'react'
import type { AgentState } from '../../preload/index'
import AgentCard from './AgentCard'
import AgentDetailView from './AgentDetailView'

interface AgentPanelProps {
  onViewLog?: (logPath: string) => void
  onToast?: (type: 'success' | 'error' | 'warning' | 'info', message: string) => void
  selectedAgentId?: string | null
  onClearSelectedAgent?: () => void
}

export default function AgentPanel({ onViewLog, onToast, selectedAgentId: externalSelectedAgentId, onClearSelectedAgent }: AgentPanelProps = {}) {
  const [agents, setAgents] = useState<AgentState[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [historyExpanded, setHistoryExpanded] = useState(true)
  const [internalSelectedAgentId, setInternalSelectedAgentId] = useState<string | null>(null)

  // Use external prop if provided, otherwise use internal state
  const selectedAgentId = externalSelectedAgentId ?? internalSelectedAgentId

  const setSelectedAgentId = useCallback((id: string | null) => {
    setInternalSelectedAgentId(id)
    if (id === null && onClearSelectedAgent) {
      onClearSelectedAgent()
    }
  }, [onClearSelectedAgent])

  // Sync internal state when external selection changes
  useEffect(() => {
    if (externalSelectedAgentId != null) {
      setInternalSelectedAgentId(externalSelectedAgentId)
    }
  }, [externalSelectedAgentId])

  const loadAgents = useCallback(async () => {
    try {
      const result = await window.state.read()
      if (result.error) {
        setError(result.error)
      } else {
        setAgents(result.agents)
        setError(null)
      }
    } catch (err) {
      setError('Failed to read state file')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    // Initial load
    loadAgents()

    // Listen for state changes (App.tsx manages watch/unwatch lifecycle)
    const unsubscribe = window.state.onChanged((data) => {
      setAgents(data.agents)
      setError(null)
      setLoading(false)
    })

    return () => {
      unsubscribe()
    }
  }, [loadAgents])

  const handlePause = async (agentId: string) => {
    try {
      await window.swarm.pause(agentId)
    } catch {
      onToast?.('error', 'Failed to pause agent')
    }
  }

  const handleResume = async (agentId: string) => {
    try {
      await window.swarm.resume(agentId)
    } catch {
      onToast?.('error', 'Failed to resume agent')
    }
  }

  const handleKill = async (agentId: string) => {
    try {
      await window.swarm.kill(agentId)
    } catch {
      onToast?.('error', 'Failed to kill agent')
    }
  }

  const handleSetIterations = async (agentId: string, iterations: number) => {
    const result = await window.swarm.run(['update', agentId, '--iterations', String(iterations)])
    if (result.code !== 0) {
      onToast?.('error', `Failed to set iterations: ${result.stderr}`)
    } else {
      onToast?.('success', `Updated iterations to ${iterations}`)
    }
  }

  const handleSetModel = async (agentId: string, model: string) => {
    const result = await window.swarm.run(['update', agentId, '--model', model])
    if (result.code !== 0) {
      onToast?.('error', `Failed to set model: ${result.stderr}`)
    } else {
      onToast?.('success', `Updated model to ${model}`)
    }
  }

  const handleClone = async (agentId: string) => {
    const result = await window.swarm.run(['clone', agentId, '-d'])
    if (result.code !== 0) {
      onToast?.('error', `Failed to clone agent: ${result.stderr}`)
    } else {
      onToast?.('success', 'Agent cloned')
    }
  }

  const handleReplay = async (agentId: string) => {
    const agent = agents.find(a => a.id === agentId)
    const name = agent?.name || agentId.slice(0, 8)
    const result = await window.swarm.run(['replay', agentId, '-d'])
    if (result.code !== 0) {
      onToast?.('error', `Failed to replay agent: ${result.stderr}`)
    } else {
      onToast?.('success', `Replaying agent ${name}`)
    }
  }

  // Find the selected agent from the live agents list (auto-updates via state:changed)
  const selectedAgent = selectedAgentId
    ? agents.find(a => a.id === selectedAgentId)
    : null

  // If viewing a detail but agent disappeared from state, go back to list
  useEffect(() => {
    if (selectedAgentId && !agents.find(a => a.id === selectedAgentId) && (agents.length > 0 || !loading)) {
      setSelectedAgentId(null)
    }
  }, [selectedAgentId, agents, loading])

  // Show detail view when an agent is selected
  if (selectedAgent) {
    return (
      <AgentDetailView
        agent={selectedAgent}
        onBack={() => setSelectedAgentId(null)}
        onPause={handlePause}
        onResume={handleResume}
        onKill={handleKill}
        onSetIterations={handleSetIterations}
        onSetModel={handleSetModel}
        onClone={handleClone}
        onReplay={handleReplay}
        onViewLog={onViewLog}
      />
    )
  }

  // Split agents into running/active vs history
  const runningAgents = agents.filter(a => a.status === 'running')
  const historyAgents = agents
    .filter(a => a.status !== 'running')
    .sort((a, b) => {
      // Most recently terminated first
      const aTime = a.terminated_at || a.started_at
      const bTime = b.terminated_at || b.started_at
      return new Date(bTime).getTime() - new Date(aTime).getTime()
    })

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="p-3 border-b border-border flex items-center justify-between">
        <h2 className="text-sm font-semibold text-foreground">Agents</h2>
        <button
          onClick={loadAgents}
          className="text-xs px-2 py-1 rounded bg-zinc-700 hover:bg-zinc-600 text-zinc-200 transition-colors"
          title="Refresh"
        >
          ↻
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-2">
        {loading ? (
          <div className="text-sm text-muted-foreground p-2">Loading...</div>
        ) : error ? (
          <div className="text-sm text-red-400 p-2">{error}</div>
        ) : agents.length === 0 ? (
          <div className="text-sm text-muted-foreground p-2">No agents</div>
        ) : (
          <>
            {/* Running agents section */}
            {runningAgents.length > 0 && (
              <div className="mb-4">
                <div className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider px-1 mb-2">
                  Running ({runningAgents.length})
                </div>
                {runningAgents.map(agent => (
                  <AgentCard
                    key={agent.id}
                    agent={agent}
                    onPause={handlePause}
                    onResume={handleResume}
                    onKill={handleKill}
                    onClick={(a) => setSelectedAgentId(a.id)}
                  />
                ))}
              </div>
            )}

            {/* History section */}
            {historyAgents.length > 0 && (
              <div>
                <button
                  onClick={() => setHistoryExpanded(!historyExpanded)}
                  className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider px-1 mb-2 flex items-center gap-1 hover:text-foreground transition-colors w-full text-left"
                >
                  <span className="text-[10px]">{historyExpanded ? '▼' : '▶'}</span>
                  History ({historyAgents.length})
                </button>
                {historyExpanded && historyAgents.map(agent => (
                  <AgentCard
                    key={agent.id}
                    agent={agent}
                    onPause={handlePause}
                    onResume={handleResume}
                    onKill={handleKill}
                    onClick={(a) => setSelectedAgentId(a.id)}
                  />
                ))}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
