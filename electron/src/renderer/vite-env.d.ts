/// <reference types="vite/client" />

interface DirEntry {
  name: string
  path: string
  isDirectory: boolean
}

interface Window {
  swarm: {
    list: () => Promise<{ stdout: string; stderr: string; code: number }>
    run: (args: string[]) => Promise<{ stdout: string; stderr: string; code: number }>
    kill: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
    pause: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
    resume: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
    logs: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
    inspect: (agentId: string) => Promise<{ stdout: string; stderr: string; code: number }>
  }
  fs: {
    readdir: (dirPath: string) => Promise<{ entries: DirEntry[]; error?: string }>
    readfile: (filePath: string) => Promise<{ content: string; error?: string }>
    writefile: (filePath: string, content: string) => Promise<{ error?: string }>
    listprompts: () => Promise<{ prompts: string[]; error?: string }>
    swarmRoot: () => Promise<string>
    watch: () => Promise<void>
    unwatch: () => Promise<void>
    onChanged: (callback: (data: { event: string; path: string }) => void) => () => void
  }
  settings: {
    read: () => Promise<{ config: { backend: string; model: string; statePath: string; logsDir: string }; error?: string }>
    write: (updates: { backend?: string; model?: string }) => Promise<{ error?: string }>
  }
}
