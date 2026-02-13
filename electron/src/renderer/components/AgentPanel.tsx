import { useState, useEffect, useCallback } from 'react'
import type { AgentState } from '../../preload/index'
import AgentCard from './AgentCard'

export default function AgentPanel() {
  const [agents, setAgents] = useState<AgentState[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [historyExpanded, setHistoryExpanded] = useState(true)
  const [expandedAgentId, setExpandedAgentId] = useState<string | null>(null)

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

    // Start watching the state file
    window.state.watch()

    // Listen for state changes
    const unsubscribe = window.state.onChanged((data) => {
      setAgents(data.agents)
      setError(null)
      setLoading(false)
    })

    return () => {
      unsubscribe()
      window.state.unwatch()
    }
  }, [loadAgents])

  const handlePause = async (agentId: string) => {
    await window.swarm.pause(agentId)
  }

  const handleResume = async (agentId: string) => {
    await window.swarm.resume(agentId)
  }

  const handleKill = async (agentId: string) => {
    await window.swarm.kill(agentId)
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
                    expanded={expandedAgentId === agent.id}
                    onToggleExpand={() => setExpandedAgentId(prev => prev === agent.id ? null : agent.id)}
                    onPause={handlePause}
                    onResume={handleResume}
                    onKill={handleKill}
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
                    expanded={expandedAgentId === agent.id}
                    onToggleExpand={() => setExpandedAgentId(prev => prev === agent.id ? null : agent.id)}
                    onPause={handlePause}
                    onResume={handleResume}
                    onKill={handleKill}
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
