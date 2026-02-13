/// <reference types="vite/client" />

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
}
