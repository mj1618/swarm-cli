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

interface PruneDialogState {
  open: boolean
  deleteLogs: boolean
  olderThan: string // '', '1d', '7d', '30d'
}

export default function AgentPanel({ onViewLog, onToast, selectedAgentId: externalSelectedAgentId, onClearSelectedAgent }: AgentPanelProps = {}) {
  const [agents, setAgents] = useState<AgentState[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [historyExpanded, setHistoryExpanded] = useState(true)
  const [internalSelectedAgentId, setInternalSelectedAgentId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | 'running' | 'terminated'>('all')
  const [pruneDialog, setPruneDialog] = useState<PruneDialogState>({ open: false, deleteLogs: false, olderThan: '' })
  const [pruning, setPruning] = useState(false)

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
    } catch {
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

  const handlePrune = async () => {
    setPruning(true)
    try {
      const args = ['prune', '--force']
      if (pruneDialog.deleteLogs) {
        args.push('--logs')
      }
      if (pruneDialog.olderThan) {
        args.push('--older-than', pruneDialog.olderThan)
      }
      const result = await window.swarm.run(args)
      if (result.code !== 0) {
        onToast?.('error', `Failed to clear history: ${result.stderr}`)
      } else {
        // Parse the output to get the count of removed agents
        const match = result.stdout.match(/Pruned (\d+) terminated agent/)
        const count = match ? match[1] : '0'
        onToast?.('success', `Removed ${count} terminated agent${count === '1' ? '' : 's'}`)
      }
    } finally {
      setPruning(false)
      setPruneDialog({ open: false, deleteLogs: false, olderThan: '' })
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
  // Count all terminated agents (not just filtered ones) for the prune button
  const allTerminatedAgents = agents.filter(a => a.status !== 'running')
  const hasTerminatedAgents = allTerminatedAgents.length > 0
  const historyAgents = filteredAgents
    .filter(a => a.status !== 'running')
    .sort((a, b) => {
      // Most recently terminated first
      const aTime = a.terminated_at || a.started_at
      const bTime = b.terminated_at || b.started_at
      return new Date(bTime).getTime() - new Date(aTime).getTime()
    })

  return (
    <div className="flex flex-col h-full" data-testid="agent-panel">
      {/* Header */}
      <div className="p-3 border-b border-border flex items-center justify-between">
        <h2 className="text-sm font-semibold text-foreground">Agents</h2>
        <div className="flex items-center gap-1">
          {hasTerminatedAgents && (
            <button
              onClick={() => setPruneDialog({ open: true, deleteLogs: false, olderThan: '' })}
              className="text-xs px-2 py-1 rounded bg-zinc-700 hover:bg-zinc-600 text-zinc-200 transition-colors"
              title="Clear terminated agents from history"
              data-testid="clear-history-button"
            >
              Clear History
            </button>
          )}
          <button
            onClick={loadAgents}
            className="text-xs px-2 py-1 rounded bg-zinc-700 hover:bg-zinc-600 text-zinc-200 transition-colors"
            title="Refresh"
            data-testid="refresh-agents-button"
          >
            ↻
          </button>
        </div>
      </div>

      {/* Search and Filter */}
      <div className="p-2 border-b border-border flex items-center gap-2" data-testid="agent-search-filter">
        <div className="relative flex-1">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search agents..."
            className="w-full h-7 rounded border border-border bg-background px-2 pr-7 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
            data-testid="agent-search-input"
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery('')}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground text-xs"
              title="Clear search"
              data-testid="agent-search-clear"
            >
              ✕
            </button>
          )}
        </div>
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as 'all' | 'running' | 'terminated')}
          className="h-7 rounded border border-border bg-background px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          data-testid="agent-status-filter"
        >
          <option value="all">All</option>
          <option value="running">Running</option>
          <option value="terminated">History</option>
        </select>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-2" data-testid="agent-list-content">
        {loading ? (
          <div className="text-sm text-muted-foreground p-2" data-testid="agent-loading">Loading...</div>
        ) : error ? (
          <div className="text-sm text-red-400 p-2" data-testid="agent-error">{error}</div>
        ) : agents.length === 0 ? (
          <div className="text-sm text-muted-foreground p-2" data-testid="no-agents">No agents</div>
        ) : filteredAgents.length === 0 ? (
          <div className="text-sm text-muted-foreground p-2" data-testid="no-matching-agents">No agents match your search</div>
        ) : (
          <>
            {/* Running agents section */}
            {runningAgents.length > 0 && (
              <div className="mb-4" data-testid="running-agents-section">
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
              <div data-testid="history-agents-section">
                <button
                  onClick={() => setHistoryExpanded(!historyExpanded)}
                  className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider px-1 mb-2 flex items-center gap-1 hover:text-foreground transition-colors w-full text-left"
                  data-testid="history-toggle-button"
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

      {/* Prune confirmation dialog */}
      {pruneDialog.open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" data-testid="prune-dialog-overlay">
          <div className="rounded-lg border border-border bg-card p-6 shadow-lg max-w-sm mx-4" data-testid="prune-dialog">
            <h3 className="text-sm font-semibold text-foreground mb-2">Clear History?</h3>
            <p className="text-sm text-muted-foreground mb-4">
              This will permanently remove {allTerminatedAgents.length} terminated agent{allTerminatedAgents.length === 1 ? '' : 's'} from the state file.
            </p>
            
            {/* Options */}
            <div className="space-y-3 mb-4">
              {/* Delete logs checkbox */}
              <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer">
                <input
                  type="checkbox"
                  checked={pruneDialog.deleteLogs}
                  onChange={(e) => setPruneDialog(prev => ({ ...prev, deleteLogs: e.target.checked }))}
                  className="rounded border-border bg-background"
                  data-testid="prune-delete-logs-checkbox"
                />
                Also delete log files
              </label>
              
              {/* Age filter dropdown */}
              <div className="flex items-center gap-2">
                <label className="text-sm text-muted-foreground">Only remove agents:</label>
                <select
                  value={pruneDialog.olderThan}
                  onChange={(e) => setPruneDialog(prev => ({ ...prev, olderThan: e.target.value }))}
                  className="h-7 rounded border border-border bg-background px-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                  data-testid="prune-older-than-select"
                >
                  <option value="">All terminated</option>
                  <option value="1d">Older than 1 day</option>
                  <option value="7d">Older than 7 days</option>
                  <option value="30d">Older than 30 days</option>
                </select>
              </div>
            </div>

            <p className="text-xs text-muted-foreground mb-4">
              This action cannot be undone.
            </p>
            
            <div className="flex justify-end gap-2">
              <button
                className="px-3 py-1.5 text-xs font-medium rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 border border-border transition-colors"
                onClick={() => setPruneDialog({ open: false, deleteLogs: false, olderThan: '' })}
                disabled={pruning}
                data-testid="prune-cancel-button"
              >
                Cancel
              </button>
              <button
                className="px-3 py-1.5 text-xs font-medium rounded-md bg-red-600 text-white hover:bg-red-700 transition-colors disabled:opacity-50"
                onClick={handlePrune}
                disabled={pruning}
                autoFocus
                data-testid="prune-confirm-button"
              >
                {pruning ? 'Clearing...' : 'Clear History'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
