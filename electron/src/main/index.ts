import { app, BrowserWindow, ipcMain } from 'electron'
import * as path from 'path'
import * as fs from 'fs/promises'
import { spawn } from 'child_process'
import { watch, FSWatcher } from 'chokidar'
import * as os from 'os'

let mainWindow: BrowserWindow | null = null

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

app.whenReady().then(() => {
  createWindow()

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

// IPC handlers for swarm CLI interaction
ipcMain.handle('swarm:list', async () => {
  return runSwarmCommand(['list', '--json'])
})

ipcMain.handle('swarm:run', async (_event, args: string[]) => {
  return runSwarmCommand(['run', ...args])
})

ipcMain.handle('swarm:kill', async (_event, agentId: string) => {
  return runSwarmCommand(['kill', agentId])
})

ipcMain.handle('swarm:pause', async (_event, agentId: string) => {
  return runSwarmCommand(['pause', agentId])
})

ipcMain.handle('swarm:resume', async (_event, agentId: string) => {
  return runSwarmCommand(['resume', agentId])
})

ipcMain.handle('swarm:logs', async (_event, agentId: string) => {
  return runSwarmCommand(['logs', agentId])
})

ipcMain.handle('swarm:inspect', async (_event, agentId: string) => {
  return runSwarmCommand(['inspect', agentId])
})

// Filesystem IPC handlers scoped to the swarm/ directory
const workingDir = process.cwd()
const swarmRoot = path.join(workingDir, 'swarm')

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

// Logs directory IPC handlers — read log files from ~/swarm/logs/
const logsDir = path.join(os.homedir(), 'swarm', 'logs')

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
const configFilePath = path.join(swarmRoot, '.swarm.toml')

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
    const content = await fs.readFile(configFilePath, 'utf-8')
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
    let content = await fs.readFile(configFilePath, 'utf-8')
    if (updates.backend !== undefined) {
      content = content.replace(/^(backend\s*=\s*)"[^"]*"/m, `$1"${updates.backend}"`)
    }
    if (updates.model !== undefined) {
      content = content.replace(/^(model\s*=\s*)"[^"]*"/m, `$1"${updates.model}"`)
    }
    await fs.writeFile(configFilePath, content, 'utf-8')
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})

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
    const proc = spawn('swarm', args)
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
