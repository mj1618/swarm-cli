import { contextBridge, ipcRenderer } from 'electron'

contextBridge.exposeInMainWorld('swarm', {
  list: () => ipcRenderer.invoke('swarm:list'),
  run: (args: string[]) => ipcRenderer.invoke('swarm:run', args),
  kill: (agentId: string) => ipcRenderer.invoke('swarm:kill', agentId),
  pause: (agentId: string) => ipcRenderer.invoke('swarm:pause', agentId),
  resume: (agentId: string) => ipcRenderer.invoke('swarm:resume', agentId),
  logs: (agentId: string) => ipcRenderer.invoke('swarm:logs', agentId),
  inspect: (agentId: string) => ipcRenderer.invoke('swarm:inspect', agentId),
})

contextBridge.exposeInMainWorld('state', {
  read: () => ipcRenderer.invoke('state:read'),
  watch: () => ipcRenderer.invoke('state:watch'),
  unwatch: () => ipcRenderer.invoke('state:unwatch'),
  onChanged: (callback: (data: { agents: any[] }) => void) => {
    const listener = (_event: any, data: { agents: any[] }) => callback(data)
    ipcRenderer.on('state:changed', listener)
    return () => { ipcRenderer.removeListener('state:changed', listener) }
  },
})

contextBridge.exposeInMainWorld('logs', {
  list: () => ipcRenderer.invoke('logs:list'),
  read: (filePath: string) => ipcRenderer.invoke('logs:read', filePath),
  watch: () => ipcRenderer.invoke('logs:watch'),
  unwatch: () => ipcRenderer.invoke('logs:unwatch'),
  onChanged: (callback: (data: { event: string; path: string }) => void) => {
    const listener = (_event: any, data: { event: string; path: string }) => callback(data)
    ipcRenderer.on('logs:changed', listener)
    return () => { ipcRenderer.removeListener('logs:changed', listener) }
  },
})

contextBridge.exposeInMainWorld('settings', {
  read: () => ipcRenderer.invoke('settings:read'),
  write: (updates: { backend?: string; model?: string }) => ipcRenderer.invoke('settings:write', updates),
})

contextBridge.exposeInMainWorld('promptResolver', {
  resolve: (filePath: string) => ipcRenderer.invoke('prompt:resolve', filePath),
})

contextBridge.exposeInMainWorld('notify', {
  send: (payload: { title: string; body: string }) => ipcRenderer.invoke('notify:send', payload),
})

contextBridge.exposeInMainWorld('dialog', {
  saveFile: (options: { defaultName: string; content: string }) =>
    ipcRenderer.invoke('dialog:saveFile', options),
  saveImage: (options: { defaultName: string; dataUrl: string; format: 'png' | 'svg' }) =>
    ipcRenderer.invoke('dialog:saveImage', options),
})

contextBridge.exposeInMainWorld('workspace', {
  getCwd: () => ipcRenderer.invoke('workspace:getCwd'),
  open: () => ipcRenderer.invoke('workspace:open'),
  switch: (dirPath: string) => ipcRenderer.invoke('workspace:switch', dirPath),
})

contextBridge.exposeInMainWorld('recent', {
  get: () => ipcRenderer.invoke('recent:get'),
  add: (projectPath: string) => ipcRenderer.invoke('recent:add', projectPath),
  clear: () => ipcRenderer.invoke('recent:clear'),
})

contextBridge.exposeInMainWorld('electronMenu', {
  on: (channel: string, callback: (data?: any) => void) => {
    const allowed = ['menu:settings', 'menu:toggle-console', 'menu:command-palette', 'menu:open-project', 'menu:keyboard-shortcuts', 'menu:about', 'menu:open-recent']
    if (!allowed.includes(channel)) return () => {}
    const listener = (_event: any, data?: any) => callback(data)
    ipcRenderer.on(channel, listener)
    return () => { ipcRenderer.removeListener(channel, listener) }
  },
})

contextBridge.exposeInMainWorld('fs', {
  readdir: (dirPath: string) => ipcRenderer.invoke('fs:readdir', dirPath),
  readfile: (filePath: string) => ipcRenderer.invoke('fs:readfile', filePath),
  writefile: (filePath: string, content: string) => ipcRenderer.invoke('fs:writefile', filePath, content),
  rename: (oldPath: string, newPath: string) => ipcRenderer.invoke('fs:rename', oldPath, newPath),
  delete: (targetPath: string) => ipcRenderer.invoke('fs:delete', targetPath),
  duplicate: (filePath: string) => ipcRenderer.invoke('fs:duplicate', filePath),
  createFile: (filePath: string) => ipcRenderer.invoke('fs:createfile', filePath),
  createDir: (dirPath: string) => ipcRenderer.invoke('fs:createdir', dirPath),
  listprompts: () => ipcRenderer.invoke('fs:listprompts'),
  swarmRoot: () => ipcRenderer.invoke('fs:swarmroot'),
  watch: () => ipcRenderer.invoke('fs:watch'),
  unwatch: () => ipcRenderer.invoke('fs:unwatch'),
  onChanged: (callback: (data: { event: string; path: string }) => void) => {
    const listener = (_event: any, data: { event: string; path: string }) => callback(data)
    ipcRenderer.on('fs:changed', listener)
    return () => { ipcRenderer.removeListener('fs:changed', listener) }
  },
})

export interface DirEntry {
  name: string
  path: string
  isDirectory: boolean
}

export type SwarmAPI = {
  list: () => Promise<{ stdout: string; stderr: string; code: number }>
  run: (args: string[]) => Promise<{ stdout: string; stderr: string; code: number }>
  kill: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
  pause: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
  resume: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
  logs: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
  inspect: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
}

export interface LogEntry {
  name: string
  path: string
  modifiedAt: number
}

export type LogsAPI = {
  list: () => Promise<{ entries: LogEntry[]; error?: string }>
  read: (filePath: string) => Promise<{ content: string; error?: string }>
  watch: () => Promise<void>
  unwatch: () => Promise<void>
  onChanged: (callback: (data: { event: string; path: string }) => void) => () => void
}

export type FsAPI = {
  readdir: (dirPath: string) => Promise<{ entries: DirEntry[]; error?: string }>
  readfile: (filePath: string) => Promise<{ content: string; error?: string }>
  writefile: (filePath: string, content: string) => Promise<{ error?: string }>
  rename: (oldPath: string, newPath: string) => Promise<{ error?: string }>
  delete: (targetPath: string) => Promise<{ error?: string }>
  duplicate: (filePath: string) => Promise<{ error?: string }>
  createFile: (filePath: string) => Promise<{ error?: string }>
  createDir: (dirPath: string) => Promise<{ error?: string }>
  listprompts: () => Promise<{ prompts: string[]; error?: string }>
  swarmRoot: () => Promise<string>
  watch: () => Promise<void>
  unwatch: () => Promise<void>
  onChanged: (callback: (data: { event: string; path: string }) => void) => () => void
}

export interface SwarmConfig {
  backend: string
  model: string
  statePath: string
  logsDir: string
}

export type SettingsAPI = {
  read: () => Promise<{ config: SwarmConfig; error?: string }>
  write: (updates: { backend?: string; model?: string }) => Promise<{ error?: string }>
}

export type PromptAPI = {
  resolve: (filePath: string) => Promise<{ content: string; error?: string }>
}

export type NotifyAPI = {
  send: (payload: { title: string; body: string }) => Promise<void>
}

export type DialogAPI = {
  saveFile: (options: { defaultName: string; content: string }) =>
    Promise<{ error?: string; canceled?: boolean }>
  saveImage: (options: { defaultName: string; dataUrl: string; format: 'png' | 'svg' }) =>
    Promise<{ error?: string; canceled?: boolean }>
}

export type WorkspaceAPI = {
  getCwd: () => Promise<string>
  open: () => Promise<{ path: string | null; error?: string }>
  switch: (dirPath: string) => Promise<{ path: string; error?: string }>
}

export type RecentAPI = {
  get: () => Promise<string[]>
  add: (projectPath: string) => Promise<string[]>
  clear: () => Promise<void>
}

export type ElectronMenuAPI = {
  on: (channel: string, callback: (data?: any) => void) => () => void
}

export type StateAPI = {
  read: () => Promise<{ agents: AgentState[]; error?: string }>
  watch: () => Promise<void>
  unwatch: () => Promise<void>
  onChanged: (callback: (data: { agents: AgentState[] }) => void) => () => void
}

export interface AgentState {
  id: string
  name: string
  parent_id?: string
  labels?: Record<string, string>
  pid: number
  prompt: string
  model: string
  started_at: string
  iterations: number
  current_iteration: number
  status: string
  terminate_mode?: string
  paused: boolean
  paused_at?: string
  log_file: string
  working_dir: string
  terminated_at?: string
  exit_reason?: string
  successful_iterations: number
  failed_iterations: number
  last_error?: string
  input_tokens: number
  output_tokens: number
  total_cost_usd: number
  current_task?: string
}

declare global {
  interface Window {
    swarm: SwarmAPI
    fs: FsAPI
    state: StateAPI
    logs: LogsAPI
    settings: SettingsAPI
    promptResolver: PromptAPI
    notify: NotifyAPI
    dialog: DialogAPI
    workspace: WorkspaceAPI
    recent: RecentAPI
    electronMenu: ElectronMenuAPI
  }
}
