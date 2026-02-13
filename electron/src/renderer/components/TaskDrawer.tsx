import { useEffect, useRef } from 'react'
import type { TaskDef, TaskDependency } from '../lib/yamlParser'

interface TaskDrawerProps {
  taskName: string
  taskDef: TaskDef
  onClose: () => void
}

function normalizeDep(dep: string | TaskDependency): TaskDependency {
  if (typeof dep === 'string') return { task: dep, condition: 'success' }
  return dep
}

function getPromptSourceInfo(def: TaskDef): { type: string; value: string } {
  if (def.prompt) return { type: 'prompt', value: def.prompt }
  if (def['prompt-file']) return { type: 'prompt-file', value: def['prompt-file'] }
  if (def['prompt-string']) return { type: 'prompt-string', value: def['prompt-string'] }
  return { type: 'none', value: 'No prompt configured' }
}

function conditionBadgeClass(condition: string): string {
  switch (condition) {
    case 'success': return 'bg-green-500/20 text-green-400'
    case 'failure': return 'bg-red-500/20 text-red-400'
    case 'any': return 'bg-yellow-500/20 text-yellow-400'
    case 'always': return 'bg-blue-500/20 text-blue-400'
    default: return 'bg-muted text-muted-foreground'
  }
}

export default function TaskDrawer({ taskName, taskDef, onClose }: TaskDrawerProps) {
  const drawerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (drawerRef.current && !drawerRef.current.contains(e.target as globalThis.Node)) {
        onClose()
      }
    }
    function handleEscape(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [onClose])

  const promptInfo = getPromptSourceInfo(taskDef)
  const deps = taskDef.depends_on?.map(normalizeDep) ?? []

  return (
    <div
      ref={drawerRef}
      className="w-80 border-l border-border bg-card flex flex-col h-full animate-in slide-in-from-right duration-200"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold text-card-foreground truncate">
          Task: {taskName}
        </h2>
        <button
          onClick={onClose}
          className="text-muted-foreground hover:text-foreground transition-colors text-lg leading-none px-1"
          aria-label="Close drawer"
        >
          &times;
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-5">
        {/* Prompt Source */}
        <Section title="Prompt Source">
          <div className="space-y-1">
            <span className="inline-block text-[10px] uppercase tracking-wider text-muted-foreground font-medium">
              {promptInfo.type}
            </span>
            <p className="text-xs text-foreground break-all bg-secondary/50 rounded px-2 py-1.5 font-mono">
              {promptInfo.value}
            </p>
          </div>
        </Section>

        {/* Model */}
        <Section title="Model">
          {taskDef.model ? (
            <span className="text-xs px-2 py-1 rounded bg-primary/20 text-primary font-medium">
              {taskDef.model}
            </span>
          ) : (
            <span className="text-xs text-muted-foreground italic">inherited</span>
          )}
        </Section>

        {/* Prefix */}
        <Section title="Prefix">
          {taskDef.prefix ? (
            <p className="text-xs text-foreground break-all bg-secondary/50 rounded px-2 py-1.5 font-mono">
              {taskDef.prefix}
            </p>
          ) : (
            <span className="text-xs text-muted-foreground italic">none</span>
          )}
        </Section>

        {/* Suffix */}
        <Section title="Suffix">
          {taskDef.suffix ? (
            <p className="text-xs text-foreground break-all bg-secondary/50 rounded px-2 py-1.5 font-mono">
              {taskDef.suffix}
            </p>
          ) : (
            <span className="text-xs text-muted-foreground italic">none</span>
          )}
        </Section>

        {/* Dependencies */}
        <Section title="Dependencies">
          {deps.length === 0 ? (
            <span className="text-xs text-muted-foreground italic">none</span>
          ) : (
            <div className="space-y-1.5">
              {deps.map((dep) => (
                <div key={dep.task} className="flex items-center gap-2">
                  <span className="text-xs text-foreground font-medium">{dep.task}</span>
                  <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${conditionBadgeClass(dep.condition)}`}>
                    {dep.condition}
                  </span>
                </div>
              ))}
            </div>
          )}
        </Section>
      </div>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h3 className="text-xs font-semibold text-muted-foreground mb-1.5">{title}</h3>
      {children}
    </div>
  )
}
