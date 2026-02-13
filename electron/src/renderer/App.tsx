import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { ReactFlowProvider } from '@xyflow/react'
import FileTree from './components/FileTree'
import MonacoFileEditor from './components/MonacoFileEditor'
import DagCanvas from './components/DagCanvas'
import AgentPanel from './components/AgentPanel'
import ConsolePanel from './components/ConsolePanel'
import TaskDrawer from './components/TaskDrawer'
import PipelinePanel from './components/PipelinePanel'
import PipelineConfigBar from './components/PipelineConfigBar'
import CommandPalette from './components/CommandPalette'
import SettingsPanel from './components/SettingsPanel'
import type { Command } from './components/CommandPalette'
import ToastContainer, { useToasts } from './components/ToastContainer'
import type { ToastType } from './components/ToastContainer'
import { serializeCompose, parseComposeFile } from './lib/yamlParser'
import type { ComposeFile, TaskDef, TaskDependency, PipelineDef } from './lib/yamlParser'
import { addDependency, applyPipelineEdits, deletePipeline } from './lib/yamlWriter'
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
  const [activePipeline, setActivePipeline] = useState<string | null>(null)
  const [selectedPipeline, setSelectedPipeline] = useState<{ name: string; compose: ComposeFile } | null>(null)
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

  const handleCreateTask = useCallback(() => {
    const yamlContent = selectedIsYaml && selectedFile ? selectedYamlContent : defaultYamlContent
    if (!yamlContent) return
    const compose = parseComposeFile(yamlContent)
    setSelectedTask({ name: '', def: { prompt: '' }, compose })
  }, [selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent])

  const handleDropCreateTask = useCallback(async (promptName: string, position: { x: number; y: number }) => {
    const yamlContent = selectedIsYaml && selectedFile ? selectedYamlContent : defaultYamlContent
    if (!yamlContent) return

    const compose = parseComposeFile(yamlContent)

    // Determine unique task name; warn on duplicate
    let taskName = promptName
    if (compose.tasks?.[taskName]) {
      addToast('warning', `Task "${promptName}" already exists — creating "${promptName}" with a new name`)
      let counter = 2
      while (compose.tasks?.[`${promptName}-${counter}`]) counter++
      taskName = `${promptName}-${counter}`
    }

    // Add the new task
    if (!compose.tasks) compose.tasks = {}
    compose.tasks[taskName] = { prompt: promptName }

    const yamlStr = serializeCompose(compose)
    const filePath = selectedIsYaml && selectedFile ? selectedFile : 'swarm/swarm.yaml'
    const result = await window.fs.writefile(filePath, yamlStr)
    if (result.error) {
      console.error('Failed to save:', result.error)
      return
    }

    // Save the drop position
    const newPositions = { ...nodePositions, [taskName]: position }
    handlePositionsChange(newPositions)

    // Reload YAML to refresh the DAG
    if (selectedIsYaml && selectedFile) {
      const reloaded = await window.fs.readfile(selectedFile)
      if (!reloaded.error) setSelectedYamlContent(reloaded.content)
    } else {
      const reloaded = await window.fs.readfile('swarm/swarm.yaml')
      if (!reloaded.error) setDefaultYamlContent(reloaded.content)
    }

    // Open the task drawer for the new task
    const updatedCompose = { ...compose }
    setSelectedTask({ name: taskName, def: { prompt: promptName }, compose: updatedCompose })
  }, [selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent, nodePositions, handlePositionsChange, addToast])

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

  const handleUpdatePipeline = useCallback(
    async (pipelineName: string, updates: { iterations?: number; parallelism?: number }) => {
      const yamlContent = selectedIsYaml && selectedFile
        ? selectedYamlContent
        : defaultYamlContent
      if (!yamlContent) return

      const compose = parseComposeFile(yamlContent)
      const updated = applyPipelineEdits(compose, pipelineName, updates)
      const yamlStr = serializeCompose(updated)

      const filePath = selectedIsYaml && selectedFile ? selectedFile : 'swarm/swarm.yaml'
      const result = await window.fs.writefile(filePath, yamlStr)
      if (result.error) {
        console.error('Failed to save pipeline settings:', result.error)
        return
      }

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

  const handleSavePipeline = useCallback(async (pipelineName: string, pipelineDef: PipelineDef) => {
    const yamlContent = selectedIsYaml && selectedFile ? selectedYamlContent : defaultYamlContent
    if (!yamlContent) return

    const compose = parseComposeFile(yamlContent)
    const updated = applyPipelineEdits(compose, pipelineName, {
      iterations: pipelineDef.iterations,
      parallelism: pipelineDef.parallelism,
      tasks: pipelineDef.tasks,
    })
    const yamlStr = serializeCompose(updated)

    const filePath = selectedIsYaml && selectedFile ? selectedFile : 'swarm/swarm.yaml'
    const result = await window.fs.writefile(filePath, yamlStr)
    if (result.error) {
      console.error('Failed to save pipeline:', result.error)
      return
    }

    if (selectedIsYaml && selectedFile) {
      const reloaded = await window.fs.readfile(selectedFile)
      if (!reloaded.error) setSelectedYamlContent(reloaded.content)
    } else {
      const reloaded = await window.fs.readfile('swarm/swarm.yaml')
      if (!reloaded.error) setDefaultYamlContent(reloaded.content)
    }
    setSelectedPipeline(null)
  }, [selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent])

  const handleDeletePipeline = useCallback(async (pipelineName: string) => {
    const yamlContent = selectedIsYaml && selectedFile ? selectedYamlContent : defaultYamlContent
    if (!yamlContent) return

    const compose = parseComposeFile(yamlContent)
    const updated = deletePipeline(compose, pipelineName)
    const yamlStr = serializeCompose(updated)

    const filePath = selectedIsYaml && selectedFile ? selectedFile : 'swarm/swarm.yaml'
    const result = await window.fs.writefile(filePath, yamlStr)
    if (result.error) {
      console.error('Failed to delete pipeline:', result.error)
      return
    }

    if (activePipeline === pipelineName) setActivePipeline(null)

    if (selectedIsYaml && selectedFile) {
      const reloaded = await window.fs.readfile(selectedFile)
      if (!reloaded.error) setSelectedYamlContent(reloaded.content)
    } else {
      const reloaded = await window.fs.readfile('swarm/swarm.yaml')
      if (!reloaded.error) setDefaultYamlContent(reloaded.content)
    }
    setSelectedPipeline(null)
  }, [selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent, activePipeline])

  const handleRunPipeline = useCallback(async (pipelineName: string) => {
    const result = await window.swarm.run(['pipeline', '--name', pipelineName])
    if (result.code !== 0) {
      addToast('error', `Pipeline failed: ${result.stderr || 'unknown error'}`)
    } else {
      addToast('success', `Pipeline "${pipelineName}" started`)
    }
  }, [addToast])

  const handleEditPipeline = useCallback((pipelineName: string) => {
    const yamlContent = selectedIsYaml && selectedFile ? selectedYamlContent : defaultYamlContent
    if (!yamlContent) return
    const compose = parseComposeFile(yamlContent)
    setSelectedPipeline({ name: pipelineName, compose })
    setSelectedTask(null)
  }, [selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent])

  const handleCreatePipeline = useCallback(() => {
    const yamlContent = selectedIsYaml && selectedFile ? selectedYamlContent : defaultYamlContent
    if (!yamlContent) return
    const compose = parseComposeFile(yamlContent)
    setSelectedPipeline({ name: '', compose })
    setSelectedTask(null)
  }, [selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent])

  const handleClosePipelinePanel = useCallback(() => {
    setSelectedPipeline(null)
  }, [])

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

        // Fire system notification if enabled
        const notificationsEnabled = localStorage.getItem('swarm-system-notifications') !== 'false'
        if (notificationsEnabled && agent.exit_reason !== 'killed') {
          const isFailed = agent.exit_reason === 'crashed'
          const costStr = agent.total_cost_usd != null ? ` — $${agent.total_cost_usd.toFixed(2)}` : ''
          window.notify.send({
            title: isFailed ? 'Agent failed' : 'Agent completed',
            body: `${label} — ${agent.current_iteration}/${agent.iterations} iterations${costStr}`,
          })
        }
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

  const currentCompose = useMemo(() => {
    const yamlContent = selectedIsYaml && selectedFile ? selectedYamlContent : defaultYamlContent
    if (!yamlContent) return null
    try {
      return parseComposeFile(yamlContent)
    } catch {
      return null
    }
  }, [selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent])

  const paletteCommands = useMemo<Command[]>(() => {
    const cmds: Command[] = []

    // Dynamic pipeline commands
    const pipelineNames = currentCompose?.pipelines ? Object.keys(currentCompose.pipelines) : []
    if (pipelineNames.length > 0) {
      for (const name of pipelineNames) {
        cmds.push({
          id: `run-pipeline-${name}`,
          name: `Run pipeline: ${name}`,
          description: `Start the ${name} pipeline`,
          action: () => { window.swarm.run(['pipeline', '--name', name]) },
        })
      }
    } else {
      cmds.push({
        id: 'run-pipeline',
        name: 'Run pipeline: main',
        description: 'Start the main pipeline',
        action: () => { window.swarm.run(['pipeline']) },
      })
    }
    cmds.push({
      id: 'create-pipeline',
      name: 'Create new pipeline',
      description: 'Open pipeline panel to create a new pipeline',
      action: handleCreatePipeline,
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
  }, [agents, selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent, handleResetLayout, currentCompose, handleCreatePipeline])

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
          <FileTree selectedPath={selectedFile} onSelectFile={handleSelectFile} onToast={addToast} />
        </div>

        {/* Center - Settings panel, File viewer, or DAG canvas */}
        <div className="flex-1 flex flex-col min-w-0">
          {settingsOpen ? (
            <SettingsPanel
              onClose={() => setSettingsOpen(false)}
              onToast={addToast}
            />
          ) : selectedFile && !selectedIsYaml ? (
            <MonacoFileEditor filePath={selectedFile} />
          ) : (
            <>
              <div className="p-3 border-b border-border">
                <h2 className="text-sm font-semibold text-foreground">{dagLabel}</h2>
              </div>
              {currentCompose && (
                <PipelineConfigBar
                  compose={currentCompose}
                  activePipeline={activePipeline}
                  onSelectPipeline={setActivePipeline}
                  onUpdatePipeline={handleUpdatePipeline}
                  onEditPipeline={handleEditPipeline}
                  onCreatePipeline={handleCreatePipeline}
                  onRunPipeline={handleRunPipeline}
                />
              )}
              <ReactFlowProvider>
                <DagCanvas
                  yamlContent={selectedIsYaml ? selectedYamlContent : defaultYamlContent}
                  loading={selectedIsYaml ? selectedYamlLoading : defaultYamlLoading}
                  error={selectedIsYaml ? selectedYamlError : defaultYamlError}
                  agents={agents}
                  activePipeline={activePipeline}
                  pipelineTasks={activePipeline && currentCompose?.pipelines?.[activePipeline]?.tasks || null}
                  onSelectTask={handleSelectTask}
                  onAddDependency={handleAddDependency}
                  onCreateTask={handleCreateTask}
                  onDropCreateTask={handleDropCreateTask}
                  savedPositions={nodePositions}
                  onPositionsChange={handlePositionsChange}
                  onResetLayout={handleResetLayout}
                />
              </ReactFlowProvider>
            </>
          )}
        </div>

        {/* Right sidebar - Task drawer, Pipeline panel, or Agent panel */}
        {selectedTask ? (
          <TaskDrawer
            taskName={selectedTask.name}
            compose={selectedTask.compose}
            onSave={handleSaveTask}
            onClose={handleCloseDrawer}
          />
        ) : selectedPipeline ? (
          <PipelinePanel
            pipelineName={selectedPipeline.name}
            compose={selectedPipeline.compose}
            onSave={handleSavePipeline}
            onDelete={handleDeletePipeline}
            onClose={handleClosePipelinePanel}
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
