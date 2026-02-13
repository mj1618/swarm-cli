import { app, BrowserWindow, ipcMain } from 'electron'
import * as path from 'path'
import * as fs from 'fs/promises'
import { spawn } from 'child_process'
import { watch, FSWatcher } from 'chokidar'

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
  return resolved.startsWith(path.resolve(swarmRoot))
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

app.on('before-quit', () => {
  if (swarmWatcher) {
    swarmWatcher.close()
    swarmWatcher = null
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
