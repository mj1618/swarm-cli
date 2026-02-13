import { useState, useEffect } from 'react'

interface Agent {
  id: string
  status: string
  task?: string
  iterations?: number
  cost?: number
}

function App() {
  const [agents, setAgents] = useState<Agent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchAgents = async () => {
    try {
      const result = await window.swarm.list()
      if (result.code === 0 && result.stdout) {
        try {
          const parsed = JSON.parse(result.stdout)
          setAgents(Array.isArray(parsed) ? parsed : [])
        } catch {
          setAgents([])
        }
      } else {
        setAgents([])
      }
      setError(null)
    } catch (err) {
      setError('Failed to connect to swarm CLI')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchAgents()
    const interval = setInterval(fetchAgents, 5000)
    return () => clearInterval(interval)
  }, [])

  const handleKill = async (agentId: string) => {
    await window.swarm.kill(agentId)
    fetchAgents()
  }

  const handlePause = async (agentId: string) => {
    await window.swarm.pause(agentId)
    fetchAgents()
  }

  const handleResume = async (agentId: string) => {
    await window.swarm.resume(agentId)
    fetchAgents()
  }

  return (
    <div className="h-full flex flex-col">
      {/* Title bar drag region */}
      <div className="h-8 bg-background border-b border-border flex items-center px-4 drag-region">
        <span className="text-sm font-medium text-muted-foreground ml-16">Swarm Desktop</span>
      </div>

      {/* Main content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left sidebar - File tree */}
        <div className="w-64 border-r border-border bg-secondary/30 flex flex-col">
          <div className="p-3 border-b border-border">
            <h2 className="text-sm font-semibold text-foreground">Files</h2>
          </div>
          <div className="flex-1 p-2 text-sm text-muted-foreground">
            <div className="p-2 hover:bg-accent rounded cursor-pointer">
              📁 swarm/
            </div>
            <div className="pl-4">
              <div className="p-2 hover:bg-accent rounded cursor-pointer">
                📄 swarm.yaml
              </div>
              <div className="p-2 hover:bg-accent rounded cursor-pointer">
                📁 prompts/
              </div>
            </div>
          </div>
        </div>

        {/* Center - DAG Editor placeholder */}
        <div className="flex-1 flex flex-col">
          <div className="p-3 border-b border-border">
            <h2 className="text-sm font-semibold text-foreground">DAG Editor</h2>
          </div>
          <div className="flex-1 flex items-center justify-center text-muted-foreground">
            <div className="text-center">
              <div className="text-4xl mb-4">🔗</div>
              <p>DAG visualization coming soon</p>
              <p className="text-sm mt-2">Open a swarm.yaml to get started</p>
            </div>
          </div>
        </div>

        {/* Right sidebar - Agent panel */}
        <div className="w-72 border-l border-border bg-secondary/30 flex flex-col">
          <div className="p-3 border-b border-border flex items-center justify-between">
            <h2 className="text-sm font-semibold text-foreground">Agents</h2>
            <button
              onClick={fetchAgents}
              className="text-xs px-2 py-1 rounded bg-primary text-primary-foreground hover:bg-primary/90"
            >
              Refresh
            </button>
          </div>
          <div className="flex-1 overflow-auto p-2">
            {loading ? (
              <div className="text-sm text-muted-foreground p-2">Loading...</div>
            ) : error ? (
              <div className="text-sm text-red-400 p-2">{error}</div>
            ) : agents.length === 0 ? (
              <div className="text-sm text-muted-foreground p-2">No agents running</div>
            ) : (
              agents.map((agent) => (
                <div
                  key={agent.id}
                  className="p-3 mb-2 rounded bg-background border border-border"
                >
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium">
                      {agent.status === 'running' ? '🟢' : agent.status === 'paused' ? '🟡' : '⚪'}{' '}
                      {agent.id.slice(0, 8)}
                    </span>
                    <span className="text-xs text-muted-foreground">{agent.status}</span>
                  </div>
                  {agent.task && (
                    <p className="text-xs text-muted-foreground mb-2 truncate">{agent.task}</p>
                  )}
                  <div className="flex gap-1">
                    {agent.status === 'running' && (
                      <button
                        onClick={() => handlePause(agent.id)}
                        className="text-xs px-2 py-1 rounded bg-secondary hover:bg-secondary/80"
                      >
                        Pause
                      </button>
                    )}
                    {agent.status === 'paused' && (
                      <button
                        onClick={() => handleResume(agent.id)}
                        className="text-xs px-2 py-1 rounded bg-secondary hover:bg-secondary/80"
                      >
                        Resume
                      </button>
                    )}
                    <button
                      onClick={() => handleKill(agent.id)}
                      className="text-xs px-2 py-1 rounded bg-red-900/50 hover:bg-red-900/70 text-red-200"
                    >
                      Kill
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* Bottom - Console */}
      <div className="h-48 border-t border-border bg-background flex flex-col">
        <div className="p-2 border-b border-border">
          <h2 className="text-sm font-semibold text-foreground">Console</h2>
        </div>
        <div className="flex-1 p-2 font-mono text-xs text-muted-foreground overflow-auto">
          <div>Welcome to Swarm Desktop</div>
          <div className="text-green-400">✓ Ready</div>
        </div>
      </div>
    </div>
  )
}

export default App
