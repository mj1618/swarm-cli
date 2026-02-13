import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { ReactFlowProvider } from '@xyflow/react'
import ErrorBoundary from './components/ErrorBoundary'
import FileTree from './components/FileTree'
import MonacoFileEditor from './components/MonacoFileEditor'
import DagCanvas from './components/DagCanvas'
import AgentPanel from './components/AgentPanel'
import ConsolePanel from './components/ConsolePanel'
import TaskDrawer from './components/TaskDrawer'
import PipelinePanel from './components/PipelinePanel'
import PipelineConfigBar from './components/PipelineConfigBar'
import CommandPalette from './components/CommandPalette'
import KeyboardShortcutsHelp from './components/KeyboardShortcutsHelp'
import AboutDialog from './components/AboutDialog'
import SettingsPanel from './components/SettingsPanel'
import OutputRunViewer, { isOutputRunFolder } from './components/OutputRunViewer'
import InitializeWorkspace from './components/InitializeWorkspace'
import type { Command } from './components/CommandPalette'
import ToastContainer, { useToasts } from './components/ToastContainer'
import type { ToastType } from './components/ToastContainer'
import { playSuccess, playFailure } from './lib/soundManager'
import { initThemeManager, getEffectiveTheme, onThemeChange } from './lib/themeManager'
import type { EffectiveTheme } from './lib/themeManager'
import { serializeCompose, parseComposeFile } from './lib/yamlParser'
import type { ComposeFile, TaskDef, TaskDependency, PipelineDef } from './lib/yamlParser'
import { addDependency, applyPipelineEdits, deletePipeline, deleteTask, deleteEdge } from './lib/yamlWriter'
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

function getViewportKey(filePath: string | null): string {
  return `swarm-dag-viewport:${filePath ?? 'swarm/swarm.yaml'}`
}

interface ViewportState {
  x: number
  y: number
  zoom: number
}

function loadViewport(filePath: string | null): ViewportState | null {
  try {
    const raw = localStorage.getItem(getViewportKey(filePath))
    if (raw) return JSON.parse(raw)
  } catch { /* ignore */ }
  return null
}

const DEFAULT_CONSOLE_HEIGHT = 192
const MIN_CONSOLE_HEIGHT = 100
const COLLAPSED_CONSOLE_HEIGHT = 28

const DEFAULT_LEFT_SIDEBAR_WIDTH = 256
const DEFAULT_RIGHT_SIDEBAR_WIDTH = 320
const MIN_SIDEBAR_WIDTH = 160
const MAX_SIDEBAR_WIDTH = 480
const COLLAPSED_SIDEBAR_WIDTH = 28

function shortenHomePath(fullPath: string): string {
  const homeMatch = fullPath.match(/^(\/Users\/[^/]+|\/home\/[^/]+)/)
  if (homeMatch) {
    return '~' + fullPath.slice(homeMatch[1].length)
  }
  return fullPath
}

function App() {
  const [effectiveTheme, setEffectiveTheme] = useState<EffectiveTheme>(getEffectiveTheme)
  const [projectPath, setProjectPath] = useState<string | null>(() => {
    return localStorage.getItem('swarm-project-path')
  })
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
  const [shortcutsOpen, setShortcutsOpen] = useState(false)
  const [aboutOpen, setAboutOpen] = useState(false)
  const [activePipeline, setActivePipeline] = useState<string | null>(null)
  const [selectedPipeline, setSelectedPipeline] = useState<{ name: string; compose: ComposeFile } | null>(null)
  const [agents, setAgents] = useState<AgentState[]>([])
  const [selectedAgentId, setSelectedAgentId] = useState<string | null>(null)
  const { toasts, addToast, removeToast } = useToasts()
  const prevAgentsRef = useRef<Map<string, AgentState>>(new Map())

  // Track dirty files for unsaved changes warning
  const [dirtyFiles, setDirtyFiles] = useState<Set<string>>(new Set())
  const [triggerSave, setTriggerSave] = useState(0)
  const pendingSaveCloseRef = useRef(false)

  // Console panel tab + collapse/resize state
  const [consoleActiveTab, setConsoleActiveTab] = useState<string>('console')
  const [consoleCollapsed, setConsoleCollapsed] = useState<boolean>(() => {
    return localStorage.getItem('swarm-console-collapsed') === 'true'
  })
  const [consoleHeight, setConsoleHeight] = useState<number>(() => {
    const saved = localStorage.getItem('swarm-console-height')
    return saved ? parseInt(saved, 10) || DEFAULT_CONSOLE_HEIGHT : DEFAULT_CONSOLE_HEIGHT
  })
  const fitViewRef = useRef<(() => void) | null>(null)
  const isDraggingConsole = useRef(false)
  const dragStartY = useRef(0)
  const dragStartHeight = useRef(0)

  // Sidebar resize state
  const [leftSidebarWidth, setLeftSidebarWidth] = useState<number>(() => {
    const saved = localStorage.getItem('swarm-left-sidebar-width')
    return saved ? parseInt(saved, 10) || DEFAULT_LEFT_SIDEBAR_WIDTH : DEFAULT_LEFT_SIDEBAR_WIDTH
  })
  const [rightSidebarWidth, setRightSidebarWidth] = useState<number>(() => {
    const saved = localStorage.getItem('swarm-right-sidebar-width')
    return saved ? parseInt(saved, 10) || DEFAULT_RIGHT_SIDEBAR_WIDTH : DEFAULT_RIGHT_SIDEBAR_WIDTH
  })
  const [leftSidebarCollapsed, setLeftSidebarCollapsed] = useState<boolean>(() => {
    return localStorage.getItem('swarm-left-sidebar-collapsed') === 'true'
  })
  const [rightSidebarCollapsed, setRightSidebarCollapsed] = useState<boolean>(() => {
    return localStorage.getItem('swarm-right-sidebar-collapsed') === 'true'
  })
  const isDraggingLeftSidebar = useRef(false)
  const isDraggingRightSidebar = useRef(false)
  const sidebarDragStartX = useRef(0)
  const sidebarDragStartWidth = useRef(0)

  const activeYamlPath = selectedFile && isYamlFile(selectedFile) ? selectedFile : null
  const [nodePositions, setNodePositions] = useState<Record<string, { x: number; y: number }>>(() =>
    loadPositions(activeYamlPath),
  )
  const [savedViewport, setSavedViewport] = useState<ViewportState | null>(() =>
    loadViewport(activeYamlPath),
  )

  // Reload saved positions and viewport when the active YAML file changes
  useEffect(() => {
    setNodePositions(loadPositions(activeYamlPath))
    setSavedViewport(loadViewport(activeYamlPath))
  }, [activeYamlPath])

  const handlePositionsChange = useCallback(
    (positions: Record<string, { x: number; y: number }>) => {
      setNodePositions(positions)
      localStorage.setItem(getPositionsKey(activeYamlPath), JSON.stringify(positions))
    },
    [activeYamlPath],
  )

  const handleViewportChange = useCallback(
    (viewport: ViewportState) => {
      setSavedViewport(viewport)
      localStorage.setItem(getViewportKey(activeYamlPath), JSON.stringify(viewport))
    },
    [activeYamlPath],
  )

  const handleResetLayout = useCallback(() => {
    setNodePositions({})
    setSavedViewport(null)
    localStorage.removeItem(getPositionsKey(activeYamlPath))
    localStorage.removeItem(getViewportKey(activeYamlPath))
  }, [activeYamlPath])

  const selectedIsYaml = selectedFile ? isYamlFile(selectedFile) : false
  const selectedIsOutputRun = selectedFile ? isOutputRunFolder(selectedFile) : false

  const handleSelectTask = useCallback((task: { name: string; def: TaskDef; compose: ComposeFile }) => {
    setSelectedTask(task)
    setSelectedAgentId(null)
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

  const handleNavigateToAgent = useCallback((agentId: string) => {
    setSelectedAgentId(agentId)
    setSelectedTask(null)
    setSelectedPipeline(null)
  }, [])

  const handleClearSelectedAgent = useCallback(() => {
    setSelectedAgentId(null)
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

  const handleDeleteTask = useCallback(
    async (taskName: string) => {
      const yamlContent = selectedIsYaml && selectedFile
        ? selectedYamlContent
        : defaultYamlContent
      if (!yamlContent) return

      const compose = parseComposeFile(yamlContent)
      const updated = deleteTask(compose, taskName)
      const yamlStr = serializeCompose(updated)

      const filePath = selectedIsYaml && selectedFile ? selectedFile : 'swarm/swarm.yaml'
      const result = await window.fs.writefile(filePath, yamlStr)
      if (result.error) {
        console.error('Failed to delete task:', result.error)
        return
      }

      // Close drawer if the deleted task was selected
      if (selectedTask?.name === taskName) setSelectedTask(null)

      addToast('success', `Deleted task "${taskName}"`)

      if (selectedIsYaml && selectedFile) {
        const reloaded = await window.fs.readfile(selectedFile)
        if (!reloaded.error) setSelectedYamlContent(reloaded.content)
      } else {
        const reloaded = await window.fs.readfile('swarm/swarm.yaml')
        if (!reloaded.error) setDefaultYamlContent(reloaded.content)
      }
    },
    [selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent, selectedTask, addToast],
  )

  const handleDeleteEdge = useCallback(
    async (source: string, target: string) => {
      const yamlContent = selectedIsYaml && selectedFile
        ? selectedYamlContent
        : defaultYamlContent
      if (!yamlContent) return

      const compose = parseComposeFile(yamlContent)
      const updated = deleteEdge(compose, source, target)
      const yamlStr = serializeCompose(updated)

      const filePath = selectedIsYaml && selectedFile ? selectedFile : 'swarm/swarm.yaml'
      const result = await window.fs.writefile(filePath, yamlStr)
      if (result.error) {
        console.error('Failed to delete edge:', result.error)
        return
      }

      addToast('success', `Removed dependency: ${source} → ${target}`)

      if (selectedIsYaml && selectedFile) {
        const reloaded = await window.fs.readfile(selectedFile)
        if (!reloaded.error) setSelectedYamlContent(reloaded.content)
      } else {
        const reloaded = await window.fs.readfile('swarm/swarm.yaml')
        if (!reloaded.error) setDefaultYamlContent(reloaded.content)
      }
    },
    [selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent, addToast],
  )

  const handleDuplicateTask = useCallback(
    async (taskName: string, taskDef: TaskDef) => {
      const yamlContent = selectedIsYaml && selectedFile
        ? selectedYamlContent
        : defaultYamlContent
      if (!yamlContent) return

      const compose = parseComposeFile(yamlContent)

      // Generate unique name: taskName-copy or taskName-copy-N
      let newName = `${taskName}-copy`
      let counter = 2
      while (compose.tasks?.[newName]) {
        newName = `${taskName}-copy-${counter}`
        counter++
      }

      // Clone the task definition without dependencies (clean slate)
      const newTaskDef: TaskDef = {
        ...taskDef,
        depends_on: undefined,
      }

      // Add the new task
      if (!compose.tasks) compose.tasks = {}
      compose.tasks[newName] = newTaskDef

      const yamlStr = serializeCompose(compose)
      const filePath = selectedIsYaml && selectedFile ? selectedFile : 'swarm/swarm.yaml'
      const result = await window.fs.writefile(filePath, yamlStr)
      if (result.error) {
        console.error('Failed to duplicate task:', result.error)
        addToast('error', `Failed to duplicate task: ${result.error}`)
        return
      }

      // Position the new task near the original (offset by 50px)
      const originalPos = nodePositions[taskName]
      if (originalPos) {
        const newPositions = { ...nodePositions, [newName]: { x: originalPos.x + 50, y: originalPos.y + 50 } }
        handlePositionsChange(newPositions)
      }

      addToast('success', `Duplicated task "${taskName}" as "${newName}"`)

      // Reload YAML to refresh the DAG
      if (selectedIsYaml && selectedFile) {
        const reloaded = await window.fs.readfile(selectedFile)
        if (!reloaded.error) setSelectedYamlContent(reloaded.content)
      } else {
        const reloaded = await window.fs.readfile('swarm/swarm.yaml')
        if (!reloaded.error) setDefaultYamlContent(reloaded.content)
      }

      // Open the task drawer for the new task
      const updatedCompose = parseComposeFile(yamlStr)
      setSelectedTask({ name: newName, def: newTaskDef, compose: updatedCompose })
    },
    [selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent, nodePositions, handlePositionsChange, addToast],
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
    const result = await window.swarm.run(['up', pipelineName])
    if (result.code !== 0) {
      addToast('error', `Pipeline failed: ${result.stderr || 'unknown error'}`)
    } else {
      addToast('success', `Pipeline "${pipelineName}" started`)
    }
  }, [addToast])

  const handleRunTask = useCallback(async (taskName: string, taskDef: TaskDef) => {
    const args: string[] = ['run']
    if (taskDef['prompt-file']) {
      args.push('-f', taskDef['prompt-file'])
    } else if (taskDef['prompt-string']) {
      args.push('-s', taskDef['prompt-string'])
    } else if (taskDef.prompt) {
      args.push('-p', taskDef.prompt)
    } else {
      addToast('error', `Task "${taskName}" has no prompt configured`)
      return
    }
    if (taskDef.model) {
      args.push('-m', taskDef.model)
    }
    args.push('-n', '1', '-d')
    const result = await window.swarm.run(args)
    if (result.code !== 0) {
      addToast('error', `Failed to start task "${taskName}": ${result.stderr || 'unknown error'}`)
    } else {
      addToast('success', `Started agent for task "${taskName}"`)
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

  // Handle dirty state changes from MonacoFileEditor
  const handleDirtyChange = useCallback((filePath: string, isDirty: boolean) => {
    setDirtyFiles(prev => {
      const next = new Set(prev)
      if (isDirty) {
        next.add(filePath)
      } else {
        next.delete(filePath)
      }
      return next
    })
  }, [])

  // Handle save completion from MonacoFileEditor (for save-and-close flow)
  const handleSaveComplete = useCallback(() => {
    if (pendingSaveCloseRef.current) {
      pendingSaveCloseRef.current = false
      window.editor.notifySaveComplete()
    }
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

  // Auto-reload YAML when externally modified (e.g. by an agent or text editor)
  useEffect(() => {
    const unsubscribe = window.fs.onChanged((data) => {
      const activePath = selectedIsYaml && selectedFile ? selectedFile : 'swarm/swarm.yaml'
      if (data.path === activePath || data.path.endsWith('/' + activePath) || activePath.endsWith('/' + data.path)) {
        window.fs.readfile(activePath).then((result) => {
          if (result.error) return
          if (selectedIsYaml && selectedFile) {
            setSelectedYamlContent(result.content)
          } else {
            setDefaultYamlContent(result.content)
          }
        })
      }
    })
    return () => { unsubscribe() }
  }, [selectedFile, selectedIsYaml])

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

  // Load initial project path from main process CWD
  useEffect(() => {
    window.workspace.getCwd().then(cwd => {
      setProjectPath(cwd)
      localStorage.setItem('swarm-project-path', cwd)
    })
  }, [])

  // Initialize theme manager and subscribe to changes
  useEffect(() => {
    const cleanup = initThemeManager()
    const unsubscribe = onThemeChange(setEffectiveTheme)
    return () => {
      cleanup()
      unsubscribe()
    }
  }, [])

  // Report dirty state to main process
  useEffect(() => {
    window.editor.setDirtyState(dirtyFiles.size > 0)
  }, [dirtyFiles])

  // Listen for save-and-close requests from main process
  useEffect(() => {
    const cleanup = window.editor.onSaveAndClose(() => {
      if (dirtyFiles.size > 0) {
        pendingSaveCloseRef.current = true
        setTriggerSave(prev => prev + 1)
      } else {
        // No dirty files, notify completion immediately
        window.editor.notifySaveComplete()
      }
    })
    return cleanup
  }, [dirtyFiles])

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

        // Play sound alert
        if (agent.exit_reason === 'crashed') {
          playFailure()
        } else if (agent.exit_reason !== 'killed') {
          playSuccess()
        }

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

  // Handle "View Log" from agent detail — switch console tab and expand
  const handleViewLog = useCallback((logPath: string) => {
    setConsoleActiveTab(logPath)
    setConsoleCollapsed(prev => {
      if (prev) {
        localStorage.setItem('swarm-console-collapsed', 'false')
        return false
      }
      return prev
    })
  }, [])

  const handleFitViewReady = useCallback((fn: () => void) => {
    fitViewRef.current = fn
  }, [])

  // Initialize workspace (create swarm/ directory structure)
  const handleInitializeWorkspace = useCallback(async () => {
    const result = await window.workspace.init()
    if (result.error) {
      addToast('error', `Failed to initialize: ${result.error}`)
      return
    }
    addToast('success', 'Swarm project initialized!')
    // Reload the default yaml
    const reloaded = await window.fs.readfile('swarm/swarm.yaml')
    if (!reloaded.error) {
      setDefaultYamlContent(reloaded.content)
      setDefaultYamlError(null)
    }
  }, [addToast])

  // Open project directory picker
  const handleOpenProject = useCallback(async () => {
    const result = await window.workspace.open()
    if (!result.path) return
    if (result.error === 'no-swarm-dir') {
      addToast('warning', `No swarm/ directory found in ${result.path}`)
      setProjectPath(result.path)
      localStorage.setItem('swarm-project-path', result.path)
      // Still add to recents even without swarm/ dir
      await window.recent.add(result.path)
      return
    }
    setProjectPath(result.path)
    localStorage.setItem('swarm-project-path', result.path)
    // Add to recent projects
    await window.recent.add(result.path)
    // Reset state for new workspace
    setSelectedFile(null)
    setSelectedTask(null)
    setSelectedPipeline(null)
    // Reload swarm.yaml from new workspace
    const reloaded = await window.fs.readfile('swarm/swarm.yaml')
    if (reloaded.error) {
      setDefaultYamlError(reloaded.error)
      setDefaultYamlContent(null)
    } else {
      setDefaultYamlContent(reloaded.content)
      setDefaultYamlError(null)
    }
    addToast('success', `Switched to ${result.path}`)
  }, [addToast])

  // Open a recent project directly (called from menu)
  const handleOpenRecentProject = useCallback(async (recentPath: string) => {
    // Switch workspace in main process
    const result = await window.workspace.switch(recentPath)
    
    if (result.error === 'Directory not found') {
      addToast('error', `Directory not found: ${shortenHomePath(recentPath)}`)
      return
    }
    
    setProjectPath(recentPath)
    localStorage.setItem('swarm-project-path', recentPath)
    
    // Add to recents (moves to top)
    await window.recent.add(recentPath)
    
    // Reset state for new workspace
    setSelectedFile(null)
    setSelectedTask(null)
    setSelectedPipeline(null)
    
    // Try to reload swarm.yaml
    const reloaded = await window.fs.readfile('swarm/swarm.yaml')
    if (reloaded.error || result.error === 'no-swarm-dir') {
      setDefaultYamlError(reloaded.error || 'No swarm directory')
      setDefaultYamlContent(null)
      addToast('warning', `No swarm/swarm.yaml found in ${shortenHomePath(recentPath)}`)
    } else {
      setDefaultYamlContent(reloaded.content)
      setDefaultYamlError(null)
      addToast('success', `Switched to ${shortenHomePath(recentPath)}`)
    }
  }, [addToast])

  // Console panel toggle
  const toggleConsole = useCallback(() => {
    setConsoleCollapsed(prev => {
      const next = !prev
      localStorage.setItem('swarm-console-collapsed', String(next))
      return next
    })
  }, [])

  // Sidebar collapse toggles
  const toggleLeftSidebar = useCallback(() => {
    setLeftSidebarCollapsed(prev => {
      const next = !prev
      localStorage.setItem('swarm-left-sidebar-collapsed', String(next))
      return next
    })
  }, [])

  const toggleRightSidebar = useCallback(() => {
    setRightSidebarCollapsed(prev => {
      const next = !prev
      localStorage.setItem('swarm-right-sidebar-collapsed', String(next))
      return next
    })
  }, [])

  // Persist console height to localStorage
  const updateConsoleHeight = useCallback((h: number) => {
    const maxH = Math.floor(window.innerHeight * 0.6)
    const clamped = Math.max(MIN_CONSOLE_HEIGHT, Math.min(h, maxH))
    setConsoleHeight(clamped)
    localStorage.setItem('swarm-console-height', String(clamped))
  }, [])

  // Console resize drag handlers
  const handleConsoleResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    isDraggingConsole.current = true
    dragStartY.current = e.clientY
    dragStartHeight.current = consoleHeight
    document.body.style.cursor = 'row-resize'
    document.body.style.userSelect = 'none'
  }, [consoleHeight])

  useEffect(() => {
    function onMouseMove(e: MouseEvent) {
      if (!isDraggingConsole.current) return
      const delta = dragStartY.current - e.clientY
      updateConsoleHeight(dragStartHeight.current + delta)
    }
    function onMouseUp() {
      if (!isDraggingConsole.current) return
      isDraggingConsole.current = false
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
    return () => {
      document.removeEventListener('mousemove', onMouseMove)
      document.removeEventListener('mouseup', onMouseUp)
    }
  }, [updateConsoleHeight])

  // Sidebar resize helpers
  const clampSidebarWidth = useCallback((w: number) => {
    return Math.max(MIN_SIDEBAR_WIDTH, Math.min(w, MAX_SIDEBAR_WIDTH))
  }, [])

  const handleLeftSidebarResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    isDraggingLeftSidebar.current = true
    sidebarDragStartX.current = e.clientX
    sidebarDragStartWidth.current = leftSidebarWidth
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }, [leftSidebarWidth])

  const handleRightSidebarResizeStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    isDraggingRightSidebar.current = true
    sidebarDragStartX.current = e.clientX
    sidebarDragStartWidth.current = rightSidebarWidth
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }, [rightSidebarWidth])

  useEffect(() => {
    function onMouseMove(e: MouseEvent) {
      if (isDraggingLeftSidebar.current) {
        const delta = e.clientX - sidebarDragStartX.current
        const newWidth = clampSidebarWidth(sidebarDragStartWidth.current + delta)
        setLeftSidebarWidth(newWidth)
        localStorage.setItem('swarm-left-sidebar-width', String(newWidth))
      } else if (isDraggingRightSidebar.current) {
        const delta = sidebarDragStartX.current - e.clientX
        const newWidth = clampSidebarWidth(sidebarDragStartWidth.current + delta)
        setRightSidebarWidth(newWidth)
        localStorage.setItem('swarm-right-sidebar-width', String(newWidth))
      }
    }
    function onMouseUp() {
      if (isDraggingLeftSidebar.current || isDraggingRightSidebar.current) {
        isDraggingLeftSidebar.current = false
        isDraggingRightSidebar.current = false
        document.body.style.cursor = ''
        document.body.style.userSelect = ''
      }
    }
    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
    return () => {
      document.removeEventListener('mousemove', onMouseMove)
      document.removeEventListener('mouseup', onMouseUp)
    }
  }, [clampSidebarWidth])

  // Keyboard shortcuts: Cmd+K, Cmd+J, Cmd+B, Cmd+Shift+B, ?
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setPaletteOpen(prev => !prev)
      }
      if ((e.metaKey || e.ctrlKey) && e.key === 'j') {
        e.preventDefault()
        setConsoleCollapsed(prev => {
          const next = !prev
          localStorage.setItem('swarm-console-collapsed', String(next))
          return next
        })
      }
      // Cmd+B to toggle left sidebar
      if ((e.metaKey || e.ctrlKey) && e.key === 'b' && !e.shiftKey) {
        e.preventDefault()
        toggleLeftSidebar()
      }
      // Cmd+Shift+B to toggle right sidebar
      if ((e.metaKey || e.ctrlKey) && e.key === 'b' && e.shiftKey) {
        e.preventDefault()
        toggleRightSidebar()
      }
      if (e.key === '?' && !e.metaKey && !e.ctrlKey && !e.altKey) {
        const tag = (e.target as HTMLElement)?.tagName
        const editable = (e.target as HTMLElement)?.isContentEditable
        if (tag === 'INPUT' || tag === 'TEXTAREA' || editable) return
        e.preventDefault()
        setShortcutsOpen(prev => !prev)
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [toggleLeftSidebar, toggleRightSidebar])

  // Native menu IPC listeners
  useEffect(() => {
    const cleanups = [
      window.electronMenu.on('menu:settings', () => setSettingsOpen(true)),
      window.electronMenu.on('menu:toggle-console', toggleConsole),
      window.electronMenu.on('menu:toggle-left-sidebar', toggleLeftSidebar),
      window.electronMenu.on('menu:toggle-right-sidebar', toggleRightSidebar),
      window.electronMenu.on('menu:command-palette', () => setPaletteOpen(prev => !prev)),
      window.electronMenu.on('menu:open-project', handleOpenProject),
      window.electronMenu.on('menu:open-recent', (path: string) => handleOpenRecentProject(path)),
      window.electronMenu.on('menu:keyboard-shortcuts', () => setShortcutsOpen(true)),
      window.electronMenu.on('menu:about', () => setAboutOpen(true)),
    ]
    return () => { cleanups.forEach(fn => fn()) }
  }, [toggleConsole, toggleLeftSidebar, toggleRightSidebar, handleOpenProject, handleOpenRecentProject])

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
          action: () => { window.swarm.run(['up', name]) },
        })
      }
    } else {
      cmds.push({
        id: 'run-pipeline',
        name: 'Run pipeline: main',
        description: 'Start the main pipeline',
        action: () => { window.swarm.run(['up']) },
      })
    }
    // Dynamic per-task run commands
    const taskNames = currentCompose?.tasks ? Object.keys(currentCompose.tasks) : []
    for (const name of taskNames) {
      const taskDef = currentCompose!.tasks[name]
      cmds.push({
        id: `run-task-${name}`,
        name: `Run task: ${name}`,
        description: `Start a detached agent for task "${name}"`,
        action: () => { handleRunTask(name, taskDef) },
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
      action: async () => {
        const toPause = agents.filter(a => a.status === 'running' && !a.paused)
        if (toPause.length === 0) return
        const results = await Promise.all(toPause.map(a => window.swarm.pause(a.id)))
        const failed = results.filter(r => r.code !== 0).length
        if (failed > 0) {
          addToast('error', `Failed to pause ${failed} agent(s)`)
        } else {
          addToast('success', `Paused ${toPause.length} agent(s)`)
        }
      },
    })
    cmds.push({
      id: 'resume-all',
      name: 'Resume all agents',
      description: 'Resume all paused agents',
      action: async () => {
        const toResume = agents.filter(a => a.paused)
        if (toResume.length === 0) return
        const results = await Promise.all(toResume.map(a => window.swarm.resume(a.id)))
        const failed = results.filter(r => r.code !== 0).length
        if (failed > 0) {
          addToast('error', `Failed to resume ${failed} agent(s)`)
        } else {
          addToast('success', `Resumed ${toResume.length} agent(s)`)
        }
      },
    })
    cmds.push({
      id: 'kill-all',
      name: 'Kill all agents',
      description: 'Stop all running agents',
      action: async () => {
        const toKill = agents.filter(a => a.status === 'running')
        if (toKill.length === 0) return
        const results = await Promise.all(toKill.map(a => window.swarm.kill(a.id)))
        const failed = results.filter(r => r.code !== 0).length
        if (failed > 0) {
          addToast('error', `Failed to stop ${failed} agent(s)`)
        } else {
          addToast('success', `Stopped ${toKill.length} agent(s)`)
        }
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
    cmds.push({
      id: 'keyboard-shortcuts',
      name: 'Show keyboard shortcuts',
      description: 'Display all keyboard shortcuts',
      shortcut: '?',
      action: () => setShortcutsOpen(true),
    })
    cmds.push({
      id: 'open-project',
      name: 'Open project',
      description: 'Switch to a different project directory',
      action: handleOpenProject,
    })
    cmds.push({
      id: 'toggle-console',
      name: 'Toggle console',
      description: 'Show or hide the console panel (Cmd+J)',
      action: toggleConsole,
    })
    cmds.push({
      id: 'toggle-left-sidebar',
      name: 'Toggle left sidebar',
      description: 'Show or hide the file tree sidebar (Cmd+B)',
      action: toggleLeftSidebar,
    })
    cmds.push({
      id: 'toggle-right-sidebar',
      name: 'Toggle right sidebar',
      description: 'Show or hide the agent panel sidebar (Cmd+Shift+B)',
      action: toggleRightSidebar,
    })
    cmds.push({
      id: 'fit-dag',
      name: 'Fit DAG to view',
      description: 'Center and fit the DAG in the viewport',
      action: () => { fitViewRef.current?.() },
    })

    // Dynamic: per-agent commands
    agents.filter(a => a.status === 'running').forEach(a => {
      cmds.push({
        id: `kill-${a.id}`,
        name: `Kill agent: ${a.name || a.id.slice(0, 8)}`,
        action: async () => {
          const result = await window.swarm.kill(a.id)
          if (result.code !== 0) {
            addToast('error', `Failed to stop agent: ${result.stderr}`)
          } else {
            addToast('success', 'Agent stopped')
          }
        },
      })
      if (!a.paused) {
        cmds.push({
          id: `pause-${a.id}`,
          name: `Pause agent: ${a.name || a.id.slice(0, 8)}`,
          action: async () => {
            const result = await window.swarm.pause(a.id)
            if (result.code !== 0) {
              addToast('error', `Failed to pause agent: ${result.stderr}`)
            } else {
              addToast('success', 'Agent paused')
            }
          },
        })
      }
    })
    agents.filter(a => a.paused).forEach(a => {
      cmds.push({
        id: `resume-${a.id}`,
        name: `Resume agent: ${a.name || a.id.slice(0, 8)}`,
        action: async () => {
          const result = await window.swarm.resume(a.id)
          if (result.code !== 0) {
            addToast('error', `Failed to resume agent: ${result.stderr}`)
          } else {
            addToast('success', 'Agent resumed')
          }
        },
      })
    })

    return cmds
  }, [agents, selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent, handleResetLayout, currentCompose, handleCreatePipeline, handleRunTask, toggleConsole, toggleLeftSidebar, toggleRightSidebar, handleOpenProject])

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
      <KeyboardShortcutsHelp
        open={shortcutsOpen}
        onClose={() => setShortcutsOpen(false)}
      />
      <AboutDialog
        open={aboutOpen}
        onClose={() => setAboutOpen(false)}
      />

      {/* Title bar drag region */}
      <div className="h-8 bg-background border-b border-border flex items-center justify-between px-4 drag-region">
        <span className="text-sm font-medium text-muted-foreground ml-16">Swarm Desktop</span>
        <button
          onClick={handleOpenProject}
          className="no-drag text-xs text-muted-foreground hover:text-foreground bg-secondary/50 hover:bg-secondary px-2 py-0.5 rounded transition-colors truncate max-w-[300px]"
          title="Click to switch project"
        >
          {projectPath ? shortenHomePath(projectPath) : 'No project'}
        </button>
      </div>

      {/* Main content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left sidebar - File tree */}
        <div
          style={{ width: leftSidebarCollapsed ? COLLAPSED_SIDEBAR_WIDTH : leftSidebarWidth }}
          className="border-r border-border bg-secondary/30 flex flex-col shrink-0 relative transition-[width] duration-200 ease-in-out"
        >
          {leftSidebarCollapsed ? (
            <div className="h-full flex flex-col items-center pt-2">
              <button
                onClick={toggleLeftSidebar}
                className="text-xs text-muted-foreground hover:text-foreground p-1 leading-none"
                title="Expand sidebar (Cmd+B)"
              >
                ▶
              </button>
            </div>
          ) : (
            <>
              <div className="flex items-center justify-between px-2 py-1 border-b border-border shrink-0">
                <span className="text-xs font-medium text-muted-foreground">Files</span>
                <button
                  onClick={toggleLeftSidebar}
                  className="text-xs text-muted-foreground hover:text-foreground p-1 leading-none"
                  title="Collapse sidebar (Cmd+B)"
                >
                  ◀
                </button>
              </div>
              <ErrorBoundary name="File Tree">
                <FileTree selectedPath={selectedFile} onSelectFile={handleSelectFile} onToast={addToast} />
              </ErrorBoundary>
              {/* Left sidebar drag handle */}
              <div
                className="absolute top-0 right-0 w-1 h-full cursor-col-resize hover:bg-primary/30 active:bg-primary/50 transition-colors z-10"
                onMouseDown={handleLeftSidebarResizeStart}
                onDoubleClick={() => {
                  setLeftSidebarWidth(DEFAULT_LEFT_SIDEBAR_WIDTH)
                  localStorage.setItem('swarm-left-sidebar-width', String(DEFAULT_LEFT_SIDEBAR_WIDTH))
                }}
              />
            </>
          )}
        </div>

        {/* Center - Settings panel, File viewer, or DAG canvas */}
        <div className="flex-1 flex flex-col min-w-0">
          <ErrorBoundary name="Center Panel">
            {settingsOpen ? (
              <SettingsPanel
                onClose={() => setSettingsOpen(false)}
                onToast={addToast}
              />
            ) : selectedIsOutputRun && selectedFile ? (
              <OutputRunViewer folderPath={selectedFile} onOpenFile={handleSelectFile} />
            ) : selectedFile && !selectedIsYaml ? (
              <MonacoFileEditor
                filePath={selectedFile}
                theme={effectiveTheme}
                onDirtyChange={handleDirtyChange}
                triggerSave={triggerSave}
                onSaveComplete={handleSaveComplete}
              />
            ) : defaultYamlError && !selectedFile ? (
              <InitializeWorkspace
                projectPath={projectPath}
                onInitialize={handleInitializeWorkspace}
              />
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
                    pipelineTasks={activePipeline ? currentCompose?.pipelines?.[activePipeline]?.tasks ?? null : null}
                    onSelectTask={handleSelectTask}
                    onNavigateToAgent={handleNavigateToAgent}
                    onAddDependency={handleAddDependency}
                    onDeleteTask={handleDeleteTask}
                    onDeleteEdge={handleDeleteEdge}
                    onRunTask={handleRunTask}
                    onDuplicateTask={handleDuplicateTask}
                    onCreateTask={handleCreateTask}
                    onDropCreateTask={handleDropCreateTask}
                    savedPositions={nodePositions}
                    onPositionsChange={handlePositionsChange}
                    savedViewport={savedViewport}
                    onViewportChange={handleViewportChange}
                    onResetLayout={handleResetLayout}
                    onFitViewReady={handleFitViewReady}
                    theme={effectiveTheme}
                    onToast={addToast}
                  />
                </ReactFlowProvider>
              </>
            )}
          </ErrorBoundary>
        </div>

        {/* Right sidebar - Task drawer, Pipeline panel, or Agent panel */}
        <div
          style={{ width: rightSidebarCollapsed ? COLLAPSED_SIDEBAR_WIDTH : rightSidebarWidth }}
          className="border-l border-border bg-secondary/30 flex flex-col shrink-0 relative transition-[width] duration-200 ease-in-out"
        >
          {rightSidebarCollapsed ? (
            <div className="h-full flex flex-col items-center pt-2">
              <button
                onClick={toggleRightSidebar}
                className="text-xs text-muted-foreground hover:text-foreground p-1 leading-none"
                title="Expand sidebar (Cmd+Shift+B)"
              >
                ◀
              </button>
            </div>
          ) : (
            <>
              {/* Right sidebar drag handle */}
              <div
                className="absolute top-0 left-0 w-1 h-full cursor-col-resize hover:bg-primary/30 active:bg-primary/50 transition-colors z-10"
                onMouseDown={handleRightSidebarResizeStart}
                onDoubleClick={() => {
                  setRightSidebarWidth(DEFAULT_RIGHT_SIDEBAR_WIDTH)
                  localStorage.setItem('swarm-right-sidebar-width', String(DEFAULT_RIGHT_SIDEBAR_WIDTH))
                }}
              />
              <ErrorBoundary name="Right Panel">
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
                  <AgentPanel
                    onViewLog={handleViewLog}
                    onToast={addToast}
                    selectedAgentId={selectedAgentId}
                    onClearSelectedAgent={handleClearSelectedAgent}
                  />
                )}
              </ErrorBoundary>
            </>
          )}
        </div>
      </div>

      {/* Bottom - Console (collapsible & resizable) */}
      <div
        style={{ height: consoleCollapsed ? COLLAPSED_CONSOLE_HEIGHT : consoleHeight }}
        className="border-t border-border bg-background flex flex-col shrink-0"
      >
        {/* Drag handle / header bar */}
        <div
          className="flex items-center h-7 px-2 shrink-0 select-none border-b border-border"
          style={{ cursor: consoleCollapsed ? 'default' : 'row-resize' }}
          onMouseDown={consoleCollapsed ? undefined : handleConsoleResizeStart}
        >
          <button
            onClick={toggleConsole}
            className="text-xs text-muted-foreground hover:text-foreground mr-1.5 leading-none"
            title={consoleCollapsed ? 'Expand console' : 'Collapse console'}
          >
            {consoleCollapsed ? '\u25B6' : '\u25BC'}
          </button>
          <span className="text-xs text-muted-foreground font-medium">Console</span>
        </div>
        {!consoleCollapsed && (
          <div className="flex-1 min-h-0">
            <ErrorBoundary name="Console">
              <ConsolePanel activeTab={consoleActiveTab} onActiveTabChange={setConsoleActiveTab} />
            </ErrorBoundary>
          </div>
        )}
      </div>

      <ToastContainer toasts={toasts} onDismiss={removeToast} />
    </div>
  )
}

export default App
