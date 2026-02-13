import { useState, useEffect, useCallback, useMemo } from 'react'
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
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | 'running' | 'terminated'>('all')

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
    const result = await window.swarm.pause(agentId)
    if (result.code !== 0) {
      onToast?.('error', `Failed to pause agent: ${result.stderr}`)
    } else {
      onToast?.('success', 'Agent paused')
    }
  }

  const handleResume = async (agentId: string) => {
    const result = await window.swarm.resume(agentId)
    if (result.code !== 0) {
      onToast?.('error', `Failed to resume agent: ${result.stderr}`)
    } else {
      onToast?.('success', 'Agent resumed')
    }
  }

  const handleKill = async (agentId: string) => {
    const result = await window.swarm.kill(agentId)
    if (result.code !== 0) {
      onToast?.('error', `Failed to stop agent: ${result.stderr}`)
    } else {
      onToast?.('success', 'Agent stopped')
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

  // Filter agents based on search query and status filter
  const filteredAgents = useMemo(() => {
    let result = agents

    // Apply status filter
    if (statusFilter === 'running') {
      result = result.filter(a => a.status === 'running')
    } else if (statusFilter === 'terminated') {
      result = result.filter(a => a.status !== 'running')
    }

    // Apply search query
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase()
      result = result.filter(a =>
        a.name?.toLowerCase().includes(query) ||
        a.id.toLowerCase().includes(query) ||
        a.model?.toLowerCase().includes(query) ||
        a.current_task?.toLowerCase().includes(query)
      )
    }

    return result
  }, [agents, searchQuery, statusFilter])

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

  // Split filtered agents into running/active vs history
  const runningAgents = filteredAgents.filter(a => a.status === 'running')
  const historyAgents = filteredAgents
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

      {/* Search and Filter */}
      <div className="p-2 border-b border-border flex items-center gap-2">
        <div className="relative flex-1">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search agents..."
            className="w-full h-7 rounded border border-border bg-background px-2 pr-7 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground text-xs"
              title="Clear search"
            >
              ✕
            </button>
          )}
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as 'all' | 'running' | 'terminated')}
          className="h-7 rounded border border-border bg-background px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
        >
          <option value="all">All</option>
          <option value="running">Running</option>
          <option value="terminated">History</option>
        </select>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-2">
        {loading ? (
          <div className="text-sm text-muted-foreground p-2">Loading...</div>
        ) : error ? (
          <div className="text-sm text-red-400 p-2">{error}</div>
        ) : agents.length === 0 ? (
          <div className="text-sm text-muted-foreground p-2">No agents</div>
        ) : filteredAgents.length === 0 ? (
          <div className="text-sm text-muted-foreground p-2">No agents match your search</div>
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
