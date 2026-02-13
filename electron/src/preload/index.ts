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

contextBridge.exposeInMainWorld('fs', {
  readdir: (dirPath: string) => ipcRenderer.invoke('fs:readdir', dirPath),
  readfile: (filePath: string) => ipcRenderer.invoke('fs:readfile', filePath),
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

export type FsAPI = {
  readdir: (dirPath: string) => Promise<{ entries: DirEntry[]; error?: string }>
  readfile: (filePath: string) => Promise<{ content: string; error?: string }>
  swarmRoot: () => Promise<string>
  watch: () => Promise<void>
  unwatch: () => Promise<void>
  onChanged: (callback: (data: { event: string; path: string }) => void) => () => void
}

declare global {
  interface Window {
    swarm: SwarmAPI
    fs: FsAPI
  }
}
