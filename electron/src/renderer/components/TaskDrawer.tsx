import { useState, useEffect, useRef, useCallback } from 'react'
import type { ComposeFile, TaskDef, TaskDependency } from '../lib/yamlParser'

type PromptType = 'prompt' | 'prompt-file' | 'prompt-string'

interface TaskDrawerProps {
  taskName: string
  compose: ComposeFile
  onSave: (taskName: string, updatedDef: TaskDef) => void
  onClose: () => void
}

function normalizeDep(dep: string | TaskDependency): TaskDependency {
  if (typeof dep === 'string') return { task: dep, condition: 'success' }
  return { ...dep }
}

function getPromptType(def: TaskDef): PromptType {
  if (def['prompt-string'] !== undefined) return 'prompt-string'
  if (def['prompt-file'] !== undefined) return 'prompt-file'
  return 'prompt'
}

function getPromptValue(def: TaskDef): string {
  if (def['prompt-string'] !== undefined) return def['prompt-string']
  if (def['prompt-file'] !== undefined) return def['prompt-file']
  return def.prompt || ''
}

function conditionBadgeClass(condition: string): string {
  switch (condition) {
    case 'success': return 'bg-green-500/20 text-green-400 border-green-500/30'
    case 'failure': return 'bg-red-500/20 text-red-400 border-red-500/30'
    case 'any': return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30'
    case 'always': return 'bg-blue-500/20 text-blue-400 border-blue-500/30'
    default: return 'bg-muted text-muted-foreground border-border'
  }
}

const MODELS = ['opus', 'sonnet', 'haiku']
const CONDITIONS: TaskDependency['condition'][] = ['success', 'failure', 'any', 'always']

export default function TaskDrawer({ taskName, compose, onSave, onClose }: TaskDrawerProps) {
  const drawerRef = useRef<HTMLDivElement>(null)
  const nameInputRef = useRef<HTMLInputElement>(null)
  const isCreating = taskName === ''
  const taskDef = compose.tasks[taskName] ?? {}
  const allTaskNames = Object.keys(compose.tasks)
  const [newName, setNewName] = useState('')
  const [nameError, setNameError] = useState<string | null>(null)
  const [promptType, setPromptType] = useState<PromptType>(getPromptType(taskDef))
  const [promptValue, setPromptValue] = useState(getPromptValue(taskDef))
  const [model, setModel] = useState(taskDef.model || '')
  const [prefix, setPrefix] = useState(taskDef.prefix || '')
  const [suffix, setSuffix] = useState(taskDef.suffix || '')
  const [deps, setDeps] = useState<TaskDependency[]>(
    (taskDef.depends_on || []).map(normalizeDep)
  )
  const [saving, setSaving] = useState(false)
  const [prompts, setPrompts] = useState<string[]>([])

  // Load available prompt files
  useEffect(() => {
    window.fs.listprompts().then(result => {
      if (result.prompts) setPrompts(result.prompts)
    })
  }, [])

  // Reset form when task changes
  useEffect(() => {
    setNewName('')
    setNameError(null)
    setPromptType(getPromptType(taskDef))
    setPromptValue(getPromptValue(taskDef))
    setModel(taskDef.model || '')
    setPrefix(taskDef.prefix || '')
    setSuffix(taskDef.suffix || '')
    setDeps((taskDef.depends_on || []).map(normalizeDep))
  }, [taskName, taskDef])

  // Auto-focus name input in creation mode
  useEffect(() => {
    if (isCreating && nameInputRef.current) {
      nameInputRef.current.focus()
    }
  }, [isCreating])

  // Close on Escape
  useEffect(() => {
    function handleEscape(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [onClose])

  const validateName = useCallback((name: string): string | null => {
    if (!name) return 'Task name is required'
    if (!/^[a-z][a-z0-9-]*$/.test(name)) return 'Must start with a letter, use only lowercase letters, numbers, and hyphens'
    if (allTaskNames.includes(name)) return 'A task with this name already exists'
    return null
  }, [allTaskNames])

  const handleSave = useCallback(() => {
    const saveName = isCreating ? newName : taskName
    if (isCreating) {
      const error = validateName(newName)
      if (error) {
        setNameError(error)
        return
      }
    }

    const updated: TaskDef = {}

    if (promptType === 'prompt' && promptValue) updated.prompt = promptValue
    if (promptType === 'prompt-file' && promptValue) updated['prompt-file'] = promptValue
    if (promptType === 'prompt-string' && promptValue) updated['prompt-string'] = promptValue

    if (model) updated.model = model
    if (prefix) updated.prefix = prefix
    if (suffix) updated.suffix = suffix
    if (deps.length > 0) updated.depends_on = deps

    setSaving(true)
    onSave(saveName, updated)
    setTimeout(() => setSaving(false), 500)
  }, [taskName, isCreating, newName, validateName, promptType, promptValue, model, prefix, suffix, deps, onSave])

  const addDep = useCallback(() => {
    const currentName = isCreating ? newName : taskName
    const available = allTaskNames.filter(
      n => n !== currentName && !deps.some(d => d.task === n)
    )
    if (available.length === 0) return
    setDeps(prev => [...prev, { task: available[0], condition: 'success' }])
  }, [allTaskNames, taskName, isCreating, newName, deps])

  const removeDep = useCallback((index: number) => {
    setDeps(prev => prev.filter((_, i) => i !== index))
  }, [])

  const updateDep = useCallback((index: number, field: 'task' | 'condition', value: string) => {
    setDeps(prev => prev.map((d, i) =>
      i === index ? { ...d, [field]: value } : d
    ))
  }, [])

  const inputClass = 'w-full bg-background border border-border rounded px-2 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-primary'
  const labelClass = 'text-xs font-semibold text-muted-foreground mb-1.5 block'

  return (
    <div
      ref={drawerRef}
      className="w-80 border-l border-border bg-card flex flex-col h-full animate-in slide-in-from-right duration-200"
    >
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold text-card-foreground truncate">
          {isCreating ? 'New Task' : taskName}
        </h2>
        <button
          onClick={onClose}
          className="text-muted-foreground hover:text-foreground transition-colors text-lg leading-none px-1"
          aria-label="Close drawer"
        >
          &times;
        </button>
      </div>

      {/* Form */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Task Name (creation mode only) */}
        {isCreating && (
          <div>
            <label className={labelClass}>Task Name</label>
            <input
              ref={nameInputRef}
              type="text"
              value={newName}
              onChange={e => {
                setNewName(e.target.value)
                setNameError(null)
              }}
              className={`${inputClass} font-mono text-xs ${nameError ? 'border-red-500 focus:ring-red-500' : ''}`}
              placeholder="my-task-name"
            />
            {nameError && (
              <p className="text-[10px] text-red-400 mt-1">{nameError}</p>
            )}
          </div>
        )}
        {/* Prompt Source */}
        <div>
          <label className={labelClass}>Prompt Source</label>
          <div className="flex gap-1 mb-2">
            {(['prompt', 'prompt-file', 'prompt-string'] as PromptType[]).map(pt => (
              <button
                key={pt}
                onClick={() => setPromptType(pt)}
                className={`text-[10px] px-2 py-1 rounded border transition-colors ${
                  promptType === pt
                    ? 'bg-primary/20 border-primary text-primary'
                    : 'border-border text-muted-foreground hover:text-foreground'
                }`}
              >
                {pt}
              </button>
            ))}
          </div>
          {promptType === 'prompt-string' ? (
            <textarea
              value={promptValue}
              onChange={e => setPromptValue(e.target.value)}
              rows={4}
              className={inputClass + ' resize-y font-mono text-xs'}
              placeholder="Inline prompt text..."
            />
          ) : promptType === 'prompt' && prompts.length > 0 ? (
            <select
              value={promptValue}
              onChange={e => setPromptValue(e.target.value)}
              className={inputClass + ' font-mono text-xs'}
            >
              <option value="">Select a prompt...</option>
              {prompts.map(p => {
                const name = p.replace(/\.(md|txt)$/, '')
                return <option key={p} value={name}>{p}</option>
              })}
              {promptValue && !prompts.some(p => p.replace(/\.(md|txt)$/, '') === promptValue) && (
                <option value={promptValue}>{promptValue}</option>
              )}
            </select>
          ) : (
            <input
              type="text"
              value={promptValue}
              onChange={e => setPromptValue(e.target.value)}
              className={inputClass + ' font-mono text-xs'}
              placeholder={promptType === 'prompt' ? 'prompt-name' : 'path/to/prompt-file.md'}
            />
          )}
        </div>

        {/* Model */}
        <div>
          <label className={labelClass}>Model</label>
          <select
            value={model}
            onChange={e => setModel(e.target.value)}
            className={inputClass}
          >
            <option value="">inherit</option>
            {MODELS.map(m => (
              <option key={m} value={m}>{m}</option>
            ))}
          </select>
        </div>

        {/* Prefix */}
        <div>
          <label className={labelClass}>Prefix</label>
          <textarea
            value={prefix}
            onChange={e => setPrefix(e.target.value)}
            rows={2}
            className={inputClass + ' resize-y font-mono text-xs'}
            placeholder="Prefix text..."
          />
        </div>

        {/* Suffix */}
        <div>
          <label className={labelClass}>Suffix</label>
          <textarea
            value={suffix}
            onChange={e => setSuffix(e.target.value)}
            rows={2}
            className={inputClass + ' resize-y font-mono text-xs'}
            placeholder="Suffix text..."
          />
        </div>

        {/* Dependencies */}
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className={labelClass + ' !mb-0'}>Dependencies</label>
            <button
              onClick={addDep}
              className="text-[10px] text-primary hover:text-primary/80 font-medium"
            >
              + Add
            </button>
          </div>
          {deps.length === 0 ? (
            <p className="text-xs text-muted-foreground italic">No dependencies</p>
          ) : (
            <div className="space-y-2">
              {deps.map((dep, i) => (
                <div key={i} className="flex items-center gap-1.5">
                  <select
                    value={dep.task}
                    onChange={e => updateDep(i, 'task', e.target.value)}
                    className="flex-1 bg-background border border-border rounded px-1.5 py-1 text-xs text-foreground"
                  >
                    {allTaskNames
                      .filter(n => n !== taskName)
                      .map(n => (
                        <option key={n} value={n}>{n}</option>
                      ))}
                  </select>
                  <select
                    value={dep.condition}
                    onChange={e => updateDep(i, 'condition', e.target.value)}
                    className={`w-[76px] border rounded px-1.5 py-1 text-[10px] font-medium ${conditionBadgeClass(dep.condition)}`}
                  >
                    {CONDITIONS.map(c => (
                      <option key={c} value={c}>{c}</option>
                    ))}
                  </select>
                  <button
                    onClick={() => removeDep(i)}
                    className="text-red-400 hover:text-red-300 text-sm px-0.5"
                  >
                    &times;
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Footer */}
      <div className="px-4 py-3 border-t border-border flex gap-2">
        <button
          onClick={onClose}
          className="flex-1 text-xs px-3 py-1.5 border border-border rounded text-muted-foreground hover:text-foreground transition-colors"
        >
          Cancel
        </button>
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex-1 text-xs px-3 py-1.5 bg-primary text-primary-foreground rounded hover:bg-primary/90 disabled:opacity-50 font-medium transition-colors"
        >
          {saving ? 'Saving...' : 'Save'}
        </button>
      </div>
    </div>
  )
}
