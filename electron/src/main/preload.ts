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

export type SwarmAPI = {
  list: () => Promise<{ stdout: string; stderr: string; code: number }>
  run: (args: string[]) => Promise<{ stdout: string; stderr: string; code: number }>
  kill: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
  pause: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
  resume: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
  logs: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
  inspect: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
}

declare global {
  interface Window {
    swarm: SwarmAPI
  }
}
