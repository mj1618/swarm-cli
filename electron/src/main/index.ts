import { app, BrowserWindow, ipcMain } from 'electron'
import * as path from 'path'
import { spawn, ChildProcess } from 'child_process'

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
