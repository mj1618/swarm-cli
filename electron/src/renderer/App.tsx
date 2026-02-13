import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { ReactFlowProvider } from '@xyflow/react'
import FileTree from './components/FileTree'
import FileViewer from './components/FileViewer'
import DagCanvas from './components/DagCanvas'
import AgentPanel from './components/AgentPanel'
import ConsolePanel from './components/ConsolePanel'
import TaskDrawer from './components/TaskDrawer'
import CommandPalette from './components/CommandPalette'
import SettingsPanel from './components/SettingsPanel'
import type { Command } from './components/CommandPalette'
import ToastContainer, { useToasts } from './components/ToastContainer'
import type { ToastType } from './components/ToastContainer'
import { serializeCompose, parseComposeFile } from './lib/yamlParser'
import type { ComposeFile, TaskDef, TaskDependency } from './lib/yamlParser'
import { addDependency } from './lib/yamlWriter'
import type { AgentState } from '../preload/index'

function isYamlFile(filePath: string): boolean {
  const ext = filePath.split('.').pop()?.toLowerCase()
  return ext === 'yaml' || ext === 'yml'
}

function getPositionsKey(filePath: string | null): string {
  return `swarm-dag-positions:${filePath ?? 'swarm/swarm.yaml'}`
}

function loadPositions(filePath: string | null): Record<string, { x: number; y: number }> {
  try {
    const raw = localStorage.getItem(getPositionsKey(filePath))
    if (raw) return JSON.parse(raw)
  } catch { /* ignore */ }
  return {}
}

function App() {
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [defaultYamlContent, setDefaultYamlContent] = useState<string | null>(null)
  const [defaultYamlLoading, setDefaultYamlLoading] = useState(true)
  const [defaultYamlError, setDefaultYamlError] = useState<string | null>(null)
  const [selectedYamlContent, setSelectedYamlContent] = useState<string | null>(null)
  const [selectedYamlLoading, setSelectedYamlLoading] = useState(false)
  const [selectedYamlError, setSelectedYamlError] = useState<string | null>(null)
  const [selectedTask, setSelectedTask] = useState<{ name: string; def: TaskDef; compose: ComposeFile } | null>(null)
  const [paletteOpen, setPaletteOpen] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [agents, setAgents] = useState<AgentState[]>([])
  const { toasts, addToast, removeToast } = useToasts()
  const prevAgentsRef = useRef<Map<string, AgentState>>(new Map())

  const activeYamlPath = selectedFile && isYamlFile(selectedFile) ? selectedFile : null
  const [nodePositions, setNodePositions] = useState<Record<string, { x: number; y: number }>>(() =>
    loadPositions(activeYamlPath),
  )

  // Reload saved positions when the active YAML file changes
  useEffect(() => {
    setNodePositions(loadPositions(activeYamlPath))
  }, [activeYamlPath])

  const handlePositionsChange = useCallback(
    (positions: Record<string, { x: number; y: number }>) => {
      setNodePositions(positions)
      localStorage.setItem(getPositionsKey(activeYamlPath), JSON.stringify(positions))
    },
    [activeYamlPath],
  )

  const handleResetLayout = useCallback(() => {
    setNodePositions({})
    localStorage.removeItem(getPositionsKey(activeYamlPath))
  }, [activeYamlPath])

  const selectedIsYaml = selectedFile ? isYamlFile(selectedFile) : false

  const handleSelectTask = useCallback((task: { name: string; def: TaskDef; compose: ComposeFile }) => {
    setSelectedTask(task)
  }, [])

  const handleSaveTask = useCallback(async (taskName: string, updatedDef: TaskDef) => {
    if (!selectedTask) return
    // Normalize dependencies: use string shorthand for 'success', object form otherwise
    if (updatedDef.depends_on) {
      updatedDef = {
        ...updatedDef,
        depends_on: updatedDef.depends_on.map(dep => {
          if (typeof dep === 'string') return dep
          return dep.condition === 'success' ? dep.task : dep
        }),
      }
    }
    const compose = { ...selectedTask.compose }
    compose.tasks = { ...compose.tasks, [taskName]: updatedDef }
    const yamlStr = serializeCompose(compose)
    const filePath = selectedIsYaml && selectedFile ? selectedFile : 'swarm/swarm.yaml'
    const result = await window.fs.writefile(filePath, yamlStr)
    if (result.error) {
      console.error('Failed to save:', result.error)
      return
    }
    // Reload YAML content to refresh the DAG
    if (selectedIsYaml && selectedFile) {
      const reloaded = await window.fs.readfile(selectedFile)
      if (!reloaded.error) setSelectedYamlContent(reloaded.content)
    } else {
      const reloaded = await window.fs.readfile('swarm/swarm.yaml')
      if (!reloaded.error) setDefaultYamlContent(reloaded.content)
    }
    setSelectedTask(null)
  }, [selectedTask, selectedIsYaml, selectedFile])

  const handleCloseDrawer = useCallback(() => {
    setSelectedTask(null)
  }, [])

  const handleAddDependency = useCallback(
    async (dep: { source: string; target: string; condition: TaskDependency['condition'] }) => {
      // Get current YAML content to parse fresh compose data
      const yamlContent = selectedIsYaml && selectedFile
        ? selectedYamlContent
        : defaultYamlContent
      if (!yamlContent) return

      const compose = parseComposeFile(yamlContent)
      const updated = addDependency(compose, dep.target, dep.source, dep.condition)
      const yamlStr = serializeCompose(updated)

      const filePath = selectedIsYaml && selectedFile ? selectedFile : 'swarm/swarm.yaml'
      const result = await window.fs.writefile(filePath, yamlStr)
      if (result.error) {
        console.error('Failed to save dependency:', result.error)
        return
      }

      // Reload YAML to refresh the DAG
      if (selectedIsYaml && selectedFile) {
        const reloaded = await window.fs.readfile(selectedFile)
        if (!reloaded.error) setSelectedYamlContent(reloaded.content)
      } else {
        const reloaded = await window.fs.readfile('swarm/swarm.yaml')
        if (!reloaded.error) setDefaultYamlContent(reloaded.content)
      }
    },
    [selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent],
  )

  const handleSelectFile = useCallback((filePath: string) => {
    setSelectedFile(filePath)
  }, [])

  // Load default swarm.yaml for when no file is selected
  useEffect(() => {
    let cancelled = false
    window.fs.readfile('swarm/swarm.yaml').then((result) => {
      if (cancelled) return
      if (result.error) {
        setDefaultYamlError(result.error)
      } else {
        setDefaultYamlContent(result.content)
      }
      setDefaultYamlLoading(false)
    }).catch(() => {
      if (cancelled) return
      setDefaultYamlError('Failed to read swarm.yaml')
      setDefaultYamlLoading(false)
    })
    return () => { cancelled = true }
  }, [])

  // Load selected YAML file content when a YAML file is selected
  useEffect(() => {
    if (!selectedFile || !isYamlFile(selectedFile)) {
      setSelectedYamlContent(null)
      setSelectedYamlError(null)
      setSelectedYamlLoading(false)
      return
    }

    let cancelled = false
    setSelectedYamlLoading(true)
    setSelectedYamlError(null)
    setSelectedYamlContent(null)

    window.fs.readfile(selectedFile).then((result) => {
      if (cancelled) return
      if (result.error) {
        setSelectedYamlError(result.error)
      } else {
        setSelectedYamlContent(result.content)
      }
      setSelectedYamlLoading(false)
    }).catch(() => {
      if (cancelled) return
      setSelectedYamlError('Failed to read file')
      setSelectedYamlLoading(false)
    })

    return () => { cancelled = true }
  }, [selectedFile])

  // Watch agent state for command palette dynamic commands
  useEffect(() => {
    window.state.read().then(result => {
      if (!result.error) setAgents(result.agents)
    })
    window.state.watch()
    const unsubscribe = window.state.onChanged(data => {
      setAgents(data.agents)
    })
    return () => {
      unsubscribe()
      window.state.unwatch()
    }
  }, [])

  // Detect agent state transitions and fire toasts
  useEffect(() => {
    const prevMap = prevAgentsRef.current
    const newMap = new Map(agents.map(a => [a.id, a]))

    for (const agent of agents) {
      const prev = prevMap.get(agent.id)
      const label = agent.name || agent.id.slice(0, 8)

      if (!prev) {
        if (agent.status === 'running') {
          addToast('info', `${label} started`)
        }
      } else if (prev.status === 'running' && agent.status === 'terminated') {
        let type: ToastType = 'success'
        let msg = `${label} completed successfully`

        if (agent.exit_reason === 'crashed') {
          type = 'error'
          msg = `${label} failed: crashed`
        } else if (agent.exit_reason === 'killed') {
          type = 'warning'
          msg = `${label} was stopped`
        }

        addToast(type, msg)
      }
    }

    prevAgentsRef.current = newMap
  }, [agents, addToast])

  // Cmd+K / Ctrl+K keyboard shortcut
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setPaletteOpen(prev => !prev)
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [])

  const paletteCommands = useMemo<Command[]>(() => {
    const cmds: Command[] = []

    // Static commands
    cmds.push({
      id: 'run-pipeline',
      name: 'Run pipeline: main',
      description: 'Start the main pipeline',
      action: () => { window.swarm.run(['pipeline']) },
    })
    cmds.push({
      id: 'pause-all',
      name: 'Pause all agents',
      description: 'Pause every running agent',
      action: () => {
        agents.filter(a => a.status === 'running' && !a.paused).forEach(a => window.swarm.pause(a.id))
      },
    })
    cmds.push({
      id: 'resume-all',
      name: 'Resume all agents',
      description: 'Resume all paused agents',
      action: () => {
        agents.filter(a => a.paused).forEach(a => window.swarm.resume(a.id))
      },
    })
    cmds.push({
      id: 'kill-all',
      name: 'Kill all agents',
      description: 'Stop all running agents',
      action: () => {
        agents.filter(a => a.status === 'running').forEach(a => window.swarm.kill(a.id))
      },
    })
    cmds.push({
      id: 'open-swarm-yaml',
      name: 'Open swarm.yaml',
      description: 'View the compose file',
      action: () => setSelectedFile('swarm/swarm.yaml'),
    })
    cmds.push({
      id: 'create-task',
      name: 'Create new task',
      description: 'Open task drawer with empty task',
      action: () => {
        const yamlContent = selectedIsYaml && selectedFile ? selectedYamlContent : defaultYamlContent
        if (!yamlContent) return
        const compose = parseComposeFile(yamlContent)
        setSelectedTask({ name: '', def: { prompt: '' }, compose })
      },
    })
    cmds.push({
      id: 'reset-layout',
      name: 'Reset DAG layout',
      description: 'Clear saved positions and re-layout',
      action: handleResetLayout,
    })
    cmds.push({
      id: 'refresh-agents',
      name: 'Refresh agents',
      description: 'Reload agent state',
      action: () => { window.state.read().then(r => { if (!r.error) setAgents(r.agents) }) },
    })
    cmds.push({
      id: 'open-settings',
      name: 'Open settings',
      description: 'Open the settings panel',
      action: () => setSettingsOpen(true),
    })

    // Dynamic: per-agent commands
    agents.filter(a => a.status === 'running').forEach(a => {
      cmds.push({
        id: `kill-${a.id}`,
        name: `Kill agent: ${a.name || a.id.slice(0, 8)}`,
        action: () => { window.swarm.kill(a.id) },
      })
      if (!a.paused) {
        cmds.push({
          id: `pause-${a.id}`,
          name: `Pause agent: ${a.name || a.id.slice(0, 8)}`,
          action: () => { window.swarm.pause(a.id) },
        })
      }
    })
    agents.filter(a => a.paused).forEach(a => {
      cmds.push({
        id: `resume-${a.id}`,
        name: `Resume agent: ${a.name || a.id.slice(0, 8)}`,
        action: () => { window.swarm.resume(a.id) },
      })
    })

    return cmds
  }, [agents, selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent, handleResetLayout])

  const dagLabel = useMemo(() => {
    if (!selectedFile) return 'DAG Editor'
    return selectedFile.split('/').pop() || 'DAG Editor'
  }, [selectedFile])

  return (
    <div className="h-full flex flex-col">
      <CommandPalette
        open={paletteOpen}
        onClose={() => setPaletteOpen(false)}
        commands={paletteCommands}
      />

      {/* Title bar drag region */}
      <div className="h-8 bg-background border-b border-border flex items-center px-4 drag-region">
        <span className="text-sm font-medium text-muted-foreground ml-16">Swarm Desktop</span>
      </div>

      {/* Main content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left sidebar - File tree */}
        <div className="w-64 border-r border-border bg-secondary/30 flex flex-col">
          <FileTree selectedPath={selectedFile} onSelectFile={handleSelectFile} />
        </div>

        {/* Center - Settings panel, File viewer, or DAG canvas */}
        <div className="flex-1 flex flex-col min-w-0">
          {settingsOpen ? (
            <SettingsPanel
              onClose={() => setSettingsOpen(false)}
              onToast={addToast}
            />
          ) : selectedFile && !selectedIsYaml ? (
            <FileViewer filePath={selectedFile} />
          ) : (
            <>
              <div className="p-3 border-b border-border">
                <h2 className="text-sm font-semibold text-foreground">{dagLabel}</h2>
              </div>
              <ReactFlowProvider>
                <DagCanvas
                  yamlContent={selectedIsYaml ? selectedYamlContent : defaultYamlContent}
                  loading={selectedIsYaml ? selectedYamlLoading : defaultYamlLoading}
                  error={selectedIsYaml ? selectedYamlError : defaultYamlError}
                  onSelectTask={handleSelectTask}
                  onAddDependency={handleAddDependency}
                  savedPositions={nodePositions}
                  onPositionsChange={handlePositionsChange}
                  onResetLayout={handleResetLayout}
                />
              </ReactFlowProvider>
            </>
          )}
        </div>

        {/* Right sidebar - Task drawer or Agent panel */}
        {selectedTask ? (
          <TaskDrawer
            taskName={selectedTask.name}
            compose={selectedTask.compose}
            onSave={handleSaveTask}
            onClose={handleCloseDrawer}
          />
        ) : (
          <div className="w-72 border-l border-border bg-secondary/30 flex flex-col">
            <AgentPanel />
          </div>
        )}
      </div>

      {/* Bottom - Console */}
      <div className="h-48 border-t border-border bg-background flex flex-col">
        <ConsolePanel />
      </div>

      <ToastContainer toasts={toasts} onDismiss={removeToast} />
    </div>
  )
}

export default App
