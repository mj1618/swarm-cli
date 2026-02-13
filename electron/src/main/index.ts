import { app, BrowserWindow, dialog, ipcMain, Menu, Notification, shell } from 'electron'
import * as path from 'path'
import * as fs from 'fs/promises'
import { spawn } from 'child_process'
import { watch, FSWatcher } from 'chokidar'
import * as os from 'os'

let mainWindow: BrowserWindow | null = null

// Recent projects storage
const MAX_RECENT_PROJECTS = 5
const recentProjectsPath = path.join(app.getPath('userData'), 'recent-projects.json')

async function loadRecentProjects(): Promise<string[]> {
  try {
    const data = await fs.readFile(recentProjectsPath, 'utf-8')
    const parsed = JSON.parse(data)
    if (Array.isArray(parsed)) {
      // Filter out paths that no longer exist
      const valid: string[] = []
      for (const p of parsed) {
        try {
          await fs.access(p)
          valid.push(p)
        } catch {
          // Path no longer exists, skip it
        }
      }
      return valid.slice(0, MAX_RECENT_PROJECTS)
    }
  } catch {
    // File doesn't exist or is invalid
  }
  return []
}

async function saveRecentProjects(projects: string[]): Promise<void> {
  try {
    await fs.writeFile(recentProjectsPath, JSON.stringify(projects.slice(0, MAX_RECENT_PROJECTS), null, 2), 'utf-8')
  } catch (err) {
    console.error('Failed to save recent projects:', err)
  }
}

async function addRecentProject(projectPath: string): Promise<string[]> {
  const recents = await loadRecentProjects()
  // Remove if already exists (we'll add to top)
  const filtered = recents.filter(p => p !== projectPath)
  // Add to top
  const updated = [projectPath, ...filtered].slice(0, MAX_RECENT_PROJECTS)
  await saveRecentProjects(updated)
  // Rebuild menu to reflect changes
  await rebuildAppMenu()
  return updated
}

async function clearRecentProjects(): Promise<void> {
  await saveRecentProjects([])
  await rebuildAppMenu()
}

function shortenPath(fullPath: string): string {
  const home = app.getPath('home')
  if (fullPath.startsWith(home)) {
    return '~' + fullPath.slice(home.length)
  }
  return fullPath
}

const isDev = process.env.NODE_ENV === 'development' || !app.isPackaged

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1000,
    minHeight: 600,
    webPreferences: {
      preload: path.join(__dirname, '../preload/index.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
    titleBarStyle: 'hiddenInset',
    backgroundColor: '#0f172a',
  })

  if (isDev) {
    mainWindow.loadURL('http://localhost:5173')
    mainWindow.webContents.openDevTools()
  } else {
    mainWindow.loadFile(path.join(__dirname, '../renderer/index.html'))
  }

  mainWindow.on('closed', () => {
    mainWindow = null
  })
}

async function rebuildAppMenu() {
  await buildAppMenu()
}

async function buildAppMenu() {
  const isMac = process.platform === 'darwin'
  const recentProjects = await loadRecentProjects()

  const sendToRenderer = (channel: string, data?: any) => {
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.webContents.send(channel, data)
    }
  }

  // Build Recent Projects submenu
  const recentProjectsSubmenu: Electron.MenuItemConstructorOptions[] = recentProjects.length > 0
    ? [
        ...recentProjects.map((p, index) => ({
          label: shortenPath(p),
          accelerator: index < 9 ? `CmdOrCtrl+${index + 1}` : undefined,
          click: () => sendToRenderer('menu:open-recent', p),
        })),
        { type: 'separator' as const },
        {
          label: 'Clear Recent Projects',
          click: () => clearRecentProjects(),
        },
      ]
    : [{ label: 'No Recent Projects', enabled: false }]

  const template: Electron.MenuItemConstructorOptions[] = [
    // macOS app menu
    ...(isMac
      ? [
          {
            label: app.name,
            submenu: [
              { role: 'about' as const },
              { type: 'separator' as const },
              {
                label: 'Settings...',
                accelerator: 'CmdOrCtrl+,',
                click: () => sendToRenderer('menu:settings'),
              },
              { type: 'separator' as const },
              { role: 'services' as const },
              { type: 'separator' as const },
              { role: 'hide' as const },
              { role: 'hideOthers' as const },
              { role: 'unhide' as const },
              { type: 'separator' as const },
              { role: 'quit' as const },
            ],
          } satisfies Electron.MenuItemConstructorOptions,
        ]
      : []),
    // File menu
    {
      label: 'File',
      submenu: [
        {
          label: 'Open Project',
          accelerator: 'CmdOrCtrl+O',
          click: () => sendToRenderer('menu:open-project'),
        },
        {
          label: 'Recent Projects',
          submenu: recentProjectsSubmenu,
        },
        { type: 'separator' },
        isMac ? { role: 'close' } : { role: 'quit' },
      ],
    },
    // Edit menu
    {
      label: 'Edit',
      submenu: [
        { role: 'undo' },
        { role: 'redo' },
        { type: 'separator' },
        { role: 'cut' },
        { role: 'copy' },
        { role: 'paste' },
        { role: 'selectAll' },
      ],
    },
    // View menu
    {
      label: 'View',
      submenu: [
        {
          label: 'Toggle Console',
          accelerator: 'CmdOrCtrl+J',
          click: () => sendToRenderer('menu:toggle-console'),
        },
        {
          label: 'Command Palette',
          accelerator: 'CmdOrCtrl+K',
          click: () => sendToRenderer('menu:command-palette'),
        },
        { type: 'separator' },
        { role: 'reload' },
        { role: 'toggleDevTools' },
        { type: 'separator' },
        { role: 'resetZoom' },
        { role: 'zoomIn' },
        { role: 'zoomOut' },
        { type: 'separator' },
        { role: 'togglefullscreen' },
      ],
    },
    // Window menu
    {
      label: 'Window',
      submenu: [
        { role: 'minimize' },
        { role: 'zoom' },
        ...(isMac
          ? [
              { type: 'separator' as const },
              { role: 'front' as const },
            ]
          : [{ role: 'close' as const }]),
      ],
    },
    // Help menu
    {
      label: 'Help',
      submenu: [
        {
          label: 'Keyboard Shortcuts',
          accelerator: 'CmdOrCtrl+/',
          click: () => sendToRenderer('menu:keyboard-shortcuts'),
        },
        { type: 'separator' },
        {
          label: 'Swarm CLI Documentation',
          click: () => shell.openExternal('https://github.com/your-org/swarm-cli#readme'),
        },
        {
          label: 'Report an Issue',
          click: () => shell.openExternal('https://github.com/your-org/swarm-cli/issues'),
        },
        { type: 'separator' },
        {
          label: 'About Swarm Desktop',
          click: () => sendToRenderer('menu:about'),
        },
      ],
    },
  ]

  const menu = Menu.buildFromTemplate(template)
  Menu.setApplicationMenu(menu)
}

app.whenReady().then(async () => {
  createWindow()
  await buildAppMenu()

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow()
    }
  })
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

// IPC handlers for recent projects
ipcMain.handle('recent:get', async (): Promise<string[]> => {
  return loadRecentProjects()
})

ipcMain.handle('recent:add', async (_event, projectPath: string): Promise<string[]> => {
  return addRecentProject(projectPath)
})

ipcMain.handle('recent:clear', async (): Promise<void> => {
  return clearRecentProjects()
})

// IPC handlers for swarm CLI interaction
ipcMain.handle('swarm:list', async () => {
  return runSwarmCommand(['list', '--json'])
})

ipcMain.handle('swarm:run', async (_event, args: string[]) => {
  return runSwarmCommand(args)
})

ipcMain.handle('swarm:kill', async (_event, agentId: string) => {
  return runSwarmCommand(['kill', agentId])
})

ipcMain.handle('swarm:pause', async (_event, agentId: string) => {
  return runSwarmCommand(['stop', agentId])
})

ipcMain.handle('swarm:resume', async (_event, agentId: string) => {
  return runSwarmCommand(['start', agentId])
})

ipcMain.handle('swarm:logs', async (_event, agentId: string) => {
  return runSwarmCommand(['logs', agentId])
})

ipcMain.handle('swarm:inspect', async (_event, agentId: string) => {
  return runSwarmCommand(['inspect', agentId])
})

// Filesystem IPC handlers scoped to the swarm/ directory
let workingDir = process.cwd()
let swarmRoot = path.join(workingDir, 'swarm')

function isWithinSwarmDir(targetPath: string): boolean {
  const resolved = path.resolve(targetPath)
  const root = path.resolve(swarmRoot)
  return resolved === root || resolved.startsWith(root + path.sep)
}

export interface DirEntry {
  name: string
  path: string
  isDirectory: boolean
}

ipcMain.handle('fs:readdir', async (_event, dirPath: string): Promise<{ entries: DirEntry[]; error?: string }> => {
  try {
    const fullPath = path.resolve(dirPath)
    if (!isWithinSwarmDir(fullPath)) {
      return { entries: [], error: 'Access denied: path outside swarm/ directory' }
    }
    const items = await fs.readdir(fullPath, { withFileTypes: true })
    const entries: DirEntry[] = items
      .filter(item => !item.name.startsWith('.') && item.name !== 'node_modules')
      .map(item => ({
        name: item.name,
        path: path.join(fullPath, item.name),
        isDirectory: item.isDirectory(),
      }))
      .sort((a, b) => {
        if (a.isDirectory !== b.isDirectory) return a.isDirectory ? -1 : 1
        return a.name.localeCompare(b.name)
      })
    return { entries }
  } catch (err: any) {
    return { entries: [], error: err.message }
  }
})

ipcMain.handle('fs:readfile', async (_event, filePath: string): Promise<{ content: string; error?: string }> => {
  try {
    const fullPath = path.resolve(filePath)
    if (!isWithinSwarmDir(fullPath)) {
      return { content: '', error: 'Access denied: path outside swarm/ directory' }
    }
    const content = await fs.readFile(fullPath, 'utf-8')
    return { content }
  } catch (err: any) {
    return { content: '', error: err.message }
  }
})

ipcMain.handle('fs:writefile', async (_event, filePath: string, content: string): Promise<{ error?: string }> => {
  try {
    const fullPath = path.resolve(filePath)
    if (!isWithinSwarmDir(fullPath)) {
      return { error: 'Access denied: path outside swarm/ directory' }
    }
    await fs.writeFile(fullPath, content, 'utf-8')
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})

ipcMain.handle('fs:rename', async (_event, oldPath: string, newPath: string): Promise<{ error?: string }> => {
  try {
    const resolvedOld = path.resolve(oldPath)
    const resolvedNew = path.resolve(newPath)
    if (!isWithinSwarmDir(resolvedOld) || !isWithinSwarmDir(resolvedNew)) {
      return { error: 'Access denied: path outside swarm/ directory' }
    }
    await fs.rename(resolvedOld, resolvedNew)
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})

ipcMain.handle('fs:delete', async (_event, targetPath: string): Promise<{ error?: string }> => {
  try {
    const resolved = path.resolve(targetPath)
    if (!isWithinSwarmDir(resolved)) {
      return { error: 'Access denied: path outside swarm/ directory' }
    }
    await fs.rm(resolved, { recursive: true })
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})

ipcMain.handle('fs:duplicate', async (_event, filePath: string): Promise<{ error?: string }> => {
  try {
    const resolved = path.resolve(filePath)
    if (!isWithinSwarmDir(resolved)) {
      return { error: 'Access denied: path outside swarm/ directory' }
    }
    const dir = path.dirname(resolved)
    const ext = path.extname(resolved)
    const base = path.basename(resolved, ext)
    const dest = path.join(dir, `${base}-copy${ext}`)
    await fs.copyFile(resolved, dest)
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})

ipcMain.handle('fs:createfile', async (_event, filePath: string): Promise<{ error?: string }> => {
  try {
    const resolved = path.resolve(filePath)
    if (!isWithinSwarmDir(resolved)) {
      return { error: 'Access denied: path outside swarm/ directory' }
    }
    // Ensure parent directory exists
    await fs.mkdir(path.dirname(resolved), { recursive: true })
    await fs.writeFile(resolved, '', 'utf-8')
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})

ipcMain.handle('fs:createdir', async (_event, dirPath: string): Promise<{ error?: string }> => {
  try {
    const resolved = path.resolve(dirPath)
    if (!isWithinSwarmDir(resolved)) {
      return { error: 'Access denied: path outside swarm/ directory' }
    }
    await fs.mkdir(resolved, { recursive: true })
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})

ipcMain.handle('fs:listprompts', async (): Promise<{ prompts: string[]; error?: string }> => {
  try {
    const promptsDir = path.join(swarmRoot, 'prompts')
    const items = await fs.readdir(promptsDir, { withFileTypes: true })
    const prompts = items
      .filter(item => item.isFile() && !item.name.startsWith('.'))
      .map(item => item.name)
      .sort()
    return { prompts }
  } catch (err: any) {
    if (err.code === 'ENOENT') {
      return { prompts: [] }
    }
    return { prompts: [], error: err.message }
  }
})

ipcMain.handle('fs:swarmroot', async (): Promise<string> => {
  return swarmRoot
})

// Workspace IPC handlers — get/set the current project directory
ipcMain.handle('workspace:getCwd', async (): Promise<string> => {
  return workingDir
})

ipcMain.handle('workspace:open', async (): Promise<{ path: string | null; error?: string }> => {
  const result = await dialog.showOpenDialog({
    properties: ['openDirectory'],
    title: 'Open Project Directory',
  })
  if (result.canceled || result.filePaths.length === 0) {
    return { path: null }
  }

  const newDir = result.filePaths[0]
  const newSwarmRoot = path.join(newDir, 'swarm')

  // Check if swarm/ subdirectory exists
  try {
    await fs.access(newSwarmRoot)
  } catch {
    return { path: newDir, error: 'no-swarm-dir' }
  }

  // Update working directory and swarm root
  workingDir = newDir
  swarmRoot = newSwarmRoot

  // Restart file watchers for new paths
  if (swarmWatcher) {
    await swarmWatcher.close()
    swarmWatcher = null
  }
  if (stateWatcher) {
    await stateWatcher.close()
    stateWatcher = null
  }
  if (logsWatcher) {
    await logsWatcher.close()
    logsWatcher = null
  }

  // Re-initialize watchers
  swarmWatcher = watch(swarmRoot, {
    ignoreInitial: true,
    depth: 10,
    ignored: /(^|[\/\\])\../,
  })
  swarmWatcher.on('all', (event, filePath) => {
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.webContents.send('fs:changed', { event, path: filePath })
    }
  })

  stateWatcher = watch(stateFilePath, {
    ignoreInitial: true,
    awaitWriteFinish: { stabilityThreshold: 100, pollInterval: 50 },
  })
  stateWatcher.on('change', async () => {
    if (!mainWindow || mainWindow.isDestroyed()) return
    try {
      const data = await fs.readFile(stateFilePath, 'utf-8')
      const parsed = JSON.parse(data)
      const agentsMap = parsed.agents || {}
      const agents = Object.values(agentsMap)
      mainWindow.webContents.send('state:changed', { agents })
    } catch {
      // File may be mid-write; ignore transient errors
    }
  })

  try {
    await fs.mkdir(logsDir, { recursive: true })
  } catch { /* ignore */ }
  logsWatcher = watch(logsDir, {
    ignoreInitial: true,
    depth: 0,
    ignored: /(^|[\/\\])\../,
    awaitWriteFinish: { stabilityThreshold: 200, pollInterval: 50 },
  })
  logsWatcher.on('all', (event, filePath) => {
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.webContents.send('logs:changed', { event, path: filePath })
    }
  })

  // Update window title
  if (mainWindow && !mainWindow.isDestroyed()) {
    const dirName = path.basename(newDir)
    mainWindow.setTitle(`Swarm Desktop — ${dirName}`)
  }

  return { path: newDir }
})

// File watcher using chokidar
let swarmWatcher: FSWatcher | null = null

ipcMain.handle('fs:watch', async () => {
  if (swarmWatcher) return

  swarmWatcher = watch(swarmRoot, {
    ignoreInitial: true,
    depth: 10,
    ignored: /(^|[\/\\])\../,
  })

  swarmWatcher.on('all', (event, filePath) => {
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.webContents.send('fs:changed', { event, path: filePath })
    }
  })
})

ipcMain.handle('fs:unwatch', async () => {
  if (swarmWatcher) {
    await swarmWatcher.close()
    swarmWatcher = null
  }
})

// State file IPC handlers — read agent state directly from ~/.swarm/state.json
const stateFilePath = path.join(os.homedir(), '.swarm', 'state.json')

ipcMain.handle('state:read', async (): Promise<{ agents: any[]; error?: string }> => {
  try {
    const data = await fs.readFile(stateFilePath, 'utf-8')
    const parsed = JSON.parse(data)
    // state.json has { agents: { id: {...} } } — convert map to array
    const agentsMap = parsed.agents || {}
    const agents = Object.values(agentsMap)
    return { agents }
  } catch (err: any) {
    if (err.code === 'ENOENT') {
      return { agents: [] }
    }
    return { agents: [], error: err.message }
  }
})

let stateWatcher: FSWatcher | null = null

ipcMain.handle('state:watch', async () => {
  if (stateWatcher) return

  stateWatcher = watch(stateFilePath, {
    ignoreInitial: true,
    awaitWriteFinish: { stabilityThreshold: 100, pollInterval: 50 },
  })

  stateWatcher.on('change', async () => {
    if (!mainWindow || mainWindow.isDestroyed()) return
    try {
      const data = await fs.readFile(stateFilePath, 'utf-8')
      const parsed = JSON.parse(data)
      const agentsMap = parsed.agents || {}
      const agents = Object.values(agentsMap)
      mainWindow.webContents.send('state:changed', { agents })
    } catch {
      // File may be mid-write; ignore transient errors
    }
  })
})

ipcMain.handle('state:unwatch', async () => {
  if (stateWatcher) {
    await stateWatcher.close()
    stateWatcher = null
  }
})

// Logs directory IPC handlers — read log files from ~/.swarm/logs/
const logsDir = path.join(os.homedir(), '.swarm', 'logs')

export interface LogEntry {
  name: string
  path: string
  modifiedAt: number
}

ipcMain.handle('logs:list', async (): Promise<{ entries: LogEntry[]; error?: string }> => {
  try {
    const items = await fs.readdir(logsDir, { withFileTypes: true })
    const entries: LogEntry[] = []
    for (const item of items) {
      if (item.isFile() && !item.name.startsWith('.')) {
        const fullPath = path.join(logsDir, item.name)
        const stat = await fs.stat(fullPath)
        entries.push({
          name: item.name,
          path: fullPath,
          modifiedAt: stat.mtimeMs,
        })
      }
    }
    entries.sort((a, b) => b.modifiedAt - a.modifiedAt)
    return { entries }
  } catch (err: any) {
    if (err.code === 'ENOENT') {
      return { entries: [] }
    }
    return { entries: [], error: err.message }
  }
})

ipcMain.handle('logs:read', async (_event, filePath: string): Promise<{ content: string; error?: string }> => {
  try {
    const resolved = path.resolve(filePath)
    const logsRoot = path.resolve(logsDir)
    if (resolved !== logsRoot && !resolved.startsWith(logsRoot + path.sep)) {
      return { content: '', error: 'Access denied: path outside logs directory' }
    }
    const content = await fs.readFile(resolved, 'utf-8')
    return { content }
  } catch (err: any) {
    if (err.code === 'ENOENT') {
      return { content: '', error: 'Log file not found' }
    }
    return { content: '', error: err.message }
  }
})

let logsWatcher: FSWatcher | null = null

ipcMain.handle('logs:watch', async () => {
  if (logsWatcher) return

  try {
    await fs.mkdir(logsDir, { recursive: true })
  } catch {
    // ignore
  }

  logsWatcher = watch(logsDir, {
    ignoreInitial: true,
    depth: 0,
    ignored: /(^|[\/\\])\../,
    awaitWriteFinish: { stabilityThreshold: 200, pollInterval: 50 },
  })

  logsWatcher.on('all', (event, filePath) => {
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.webContents.send('logs:changed', { event, path: filePath })
    }
  })
})

ipcMain.handle('logs:unwatch', async () => {
  if (logsWatcher) {
    await logsWatcher.close()
    logsWatcher = null
  }
})

// Settings IPC handlers — read/write swarm config from swarm/.swarm.toml
// Use a getter so it tracks the current swarmRoot after workspace switches
function getConfigFilePath() {
  return path.join(swarmRoot, '.swarm.toml')
}

export interface SwarmConfig {
  backend: string
  model: string
  statePath: string
  logsDir: string
}

ipcMain.handle('settings:read', async (): Promise<{ config: SwarmConfig; error?: string }> => {
  const defaults: SwarmConfig = {
    backend: 'claude-code',
    model: 'opus',
    statePath: path.join(os.homedir(), '.swarm', 'state.json'),
    logsDir: path.join(os.homedir(), 'swarm', 'logs'),
  }
  try {
    const content = await fs.readFile(getConfigFilePath(), 'utf-8')
    const backendMatch = content.match(/^backend\s*=\s*"([^"]*)"$/m)
    const modelMatch = content.match(/^model\s*=\s*"([^"]*)"$/m)
    if (backendMatch) defaults.backend = backendMatch[1]
    if (modelMatch) defaults.model = modelMatch[1]
    return { config: defaults }
  } catch (err: any) {
    if (err.code === 'ENOENT') {
      return { config: defaults }
    }
    return { config: defaults, error: err.message }
  }
})

ipcMain.handle('settings:write', async (_event, updates: { backend?: string; model?: string }): Promise<{ error?: string }> => {
  try {
    let content: string
    try {
      content = await fs.readFile(getConfigFilePath(), 'utf-8')
    } catch (readErr: any) {
      if (readErr.code === 'ENOENT') {
        // Config file doesn't exist yet — create with defaults
        content = `backend = "claude-code"\nmodel = "sonnet"\n`
      } else {
        throw readErr
      }
    }
    if (updates.backend !== undefined) {
      if (/^backend\s*=/m.test(content)) {
        content = content.replace(/^(backend\s*=\s*)"[^"]*"/m, `$1"${updates.backend}"`)
      } else {
        content = `backend = "${updates.backend}"\n` + content
      }
    }
    if (updates.model !== undefined) {
      if (/^model\s*=/m.test(content)) {
        content = content.replace(/^(model\s*=\s*)"[^"]*"/m, `$1"${updates.model}"`)
      } else {
        content = `model = "${updates.model}"\n` + content
      }
    }
    await fs.writeFile(getConfigFilePath(), content, 'utf-8')
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})

// System notification IPC handler
ipcMain.handle('notify:send', async (_event, payload: { title: string; body: string }) => {
  if (!mainWindow || mainWindow.isDestroyed() || mainWindow.isFocused()) return
  const notification = new Notification({
    title: payload.title,
    body: payload.body,
  })
  notification.on('click', () => {
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.show()
      mainWindow.focus()
    }
  })
  notification.show()
})

// Dialog IPC handler — save file dialog for log export
ipcMain.handle('dialog:saveFile', async (_event, options: { defaultName: string; content: string }) => {
  const result = await dialog.showSaveDialog({
    defaultPath: options.defaultName,
    filters: [
      { name: 'Log files', extensions: ['log', 'txt'] },
      { name: 'All files', extensions: ['*'] },
    ],
  })
  if (result.canceled || !result.filePath) {
    return { canceled: true }
  }
  try {
    await fs.writeFile(result.filePath, options.content, 'utf-8')
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})

// Prompt resolution IPC handler — recursively expands {{include:path}} directives
ipcMain.handle('prompt:resolve', async (_event, filePath: string): Promise<{ content: string; error?: string }> => {
  try {
    const fullPath = path.resolve(filePath)
    if (!isWithinSwarmDir(fullPath)) {
      return { content: '', error: 'Access denied: path outside swarm/ directory' }
    }
    const content = await fs.readFile(fullPath, 'utf-8')
    const resolved = await resolveIncludes(content, path.dirname(fullPath), new Set())
    return { content: resolved }
  } catch (err: any) {
    if (err.code === 'ENOENT') {
      return { content: '', error: 'Prompt file not found' }
    }
    return { content: '', error: err.message }
  }
})

async function resolveIncludes(content: string, baseDir: string, seen: Set<string>): Promise<string> {
  const includePattern = /\{\{include:([^}]+)\}\}/g
  const parts: string[] = []
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = includePattern.exec(content)) !== null) {
    parts.push(content.slice(lastIndex, match.index))
    const includePath = match[1].trim()

    // Resolve relative to baseDir first, then relative to swarm/ root
    let resolvedPath = path.resolve(baseDir, includePath)
    if (!isWithinSwarmDir(resolvedPath)) {
      resolvedPath = path.resolve(swarmRoot, includePath)
    }
    if (!isWithinSwarmDir(resolvedPath)) {
      parts.push(`[ERROR: include path outside swarm directory: ${includePath}]`)
      lastIndex = match.index + match[0].length
      continue
    }

    if (seen.has(resolvedPath)) {
      parts.push(`[ERROR: circular include: ${includePath}]`)
    } else {
      try {
        const includeContent = await fs.readFile(resolvedPath, 'utf-8')
        seen.add(resolvedPath)
        const resolved = await resolveIncludes(includeContent, path.dirname(resolvedPath), seen)
        parts.push(resolved)
      } catch {
        parts.push(`[ERROR: file not found: ${includePath}]`)
      }
    }
    lastIndex = match.index + match[0].length
  }

  parts.push(content.slice(lastIndex))
  return parts.join('')
}

app.on('before-quit', () => {
  if (swarmWatcher) {
    swarmWatcher.close()
    swarmWatcher = null
  }
  if (stateWatcher) {
    stateWatcher.close()
    stateWatcher = null
  }
  if (logsWatcher) {
    logsWatcher.close()
    logsWatcher = null
  }
})

async function runSwarmCommand(args: string[]): Promise<{ stdout: string; stderr: string; code: number }> {
  return new Promise((resolve) => {
    const proc = spawn('swarm', args, { cwd: workingDir })
    let stdout = ''
    let stderr = ''

    proc.stdout.on('data', (data) => {
      stdout += data.toString()
    })

    proc.stderr.on('data', (data) => {
      stderr += data.toString()
    })

    proc.on('close', (code) => {
      resolve({ stdout, stderr, code: code ?? 1 })
    })

    proc.on('error', (err) => {
      resolve({ stdout: '', stderr: err.message, code: 1 })
    })
  })
}
