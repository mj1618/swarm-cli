import { useState, useEffect, useCallback, useMemo } from 'react'
import { ReactFlowProvider } from '@xyflow/react'
import FileTree from './components/FileTree'
import FileViewer from './components/FileViewer'
import DagCanvas from './components/DagCanvas'
import AgentPanel from './components/AgentPanel'
import ConsolePanel from './components/ConsolePanel'
import TaskDrawer from './components/TaskDrawer'
import { serializeCompose, parseComposeFile } from './lib/yamlParser'
import type { ComposeFile, TaskDef } from './lib/yamlParser'
import { addDependency } from './lib/yamlWriter'

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
    async (dep: { source: string; target: string; condition: string }) => {
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

  const dagLabel = useMemo(() => {
    if (!selectedFile) return 'DAG Editor'
    return selectedFile.split('/').pop() || 'DAG Editor'
  }, [selectedFile])

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
          <FileTree selectedPath={selectedFile} onSelectFile={handleSelectFile} />
        </div>

        {/* Center - File viewer or DAG canvas */}
        <div className="flex-1 flex flex-col min-w-0">
          {selectedFile && !selectedIsYaml ? (
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
    </div>
  )
}

export default App
