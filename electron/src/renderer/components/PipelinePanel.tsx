import { useState, useEffect, useRef, useCallback } from 'react'
import type { ComposeFile, PipelineDef } from '../lib/yamlParser'

interface PipelinePanelProps {
  pipelineName: string // empty string = creation mode
  compose: ComposeFile
  onSave: (pipelineName: string, pipelineDef: PipelineDef) => void
  onDelete: (pipelineName: string) => void
  onClose: () => void
}

export default function PipelinePanel({ pipelineName, compose, onSave, onDelete, onClose }: PipelinePanelProps) {
  const nameInputRef = useRef<HTMLInputElement>(null)
  const isCreating = pipelineName === ''
  const pipelineDef = compose.pipelines?.[pipelineName] ?? {}
  const allTaskNames = Object.keys(compose.tasks ?? {})
  const allPipelineNames = Object.keys(compose.pipelines ?? {})

  const [newName, setNewName] = useState('')
  const [nameError, setNameError] = useState<string | null>(null)
  const [iterations, setIterations] = useState(pipelineDef.iterations ?? 1)
  const [parallelism, setParallelism] = useState(pipelineDef.parallelism ?? 1)
  const [selectedTasks, setSelectedTasks] = useState<Set<string>>(
    new Set(pipelineDef.tasks ?? []),
  )
  const [saving, setSaving] = useState(false)

  // Reset form when pipeline changes
  useEffect(() => {
    const def = compose.pipelines?.[pipelineName] ?? {}
    setNewName('')
    setNameError(null)
    setIterations(def.iterations ?? 1)
    setParallelism(def.parallelism ?? 1)
    setSelectedTasks(new Set(def.tasks ?? []))
  }, [pipelineName, compose])

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
    if (!name) return 'Pipeline name is required'
    if (!/^[a-z][a-z0-9-]*$/.test(name)) return 'Must start with a letter, use only lowercase letters, numbers, and hyphens'
    if (allPipelineNames.includes(name)) return 'A pipeline with this name already exists'
    return null
  }, [allPipelineNames])

  const toggleTask = useCallback((taskName: string) => {
    setSelectedTasks(prev => {
      const next = new Set(prev)
      if (next.has(taskName)) {
        next.delete(taskName)
      } else {
        next.add(taskName)
      }
      return next
    })
  }, [])

  const selectAll = useCallback(() => {
    setSelectedTasks(new Set(allTaskNames))
  }, [allTaskNames])

  const selectNone = useCallback(() => {
    setSelectedTasks(new Set())
  }, [])

  const handleSave = useCallback(() => {
    const saveName = isCreating ? newName : pipelineName
    if (isCreating) {
      const error = validateName(newName)
      if (error) {
        setNameError(error)
        return
      }
    }

    const def: PipelineDef = {}
    if (iterations > 1) def.iterations = iterations
    if (parallelism > 1) def.parallelism = parallelism
    const tasks = Array.from(selectedTasks)
    if (tasks.length > 0) def.tasks = tasks

    setSaving(true)
    onSave(saveName, def)
    setTimeout(() => setSaving(false), 500)
  }, [pipelineName, isCreating, newName, validateName, iterations, parallelism, selectedTasks, onSave])

  const handleDelete = useCallback(() => {
    if (isCreating) return
    onDelete(pipelineName)
  }, [isCreating, pipelineName, onDelete])

  const inputClass = 'w-full bg-background border border-border rounded px-2 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-primary'
  const labelClass = 'text-xs font-semibold text-muted-foreground mb-1.5 block'

  return (
    <div className="bg-card flex flex-col h-full animate-in slide-in-from-right duration-200">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border">
        <h2 className="text-sm font-semibold text-card-foreground truncate">
          {isCreating ? 'New Pipeline' : `Pipeline: ${pipelineName}`}
        </h2>
        <button
          onClick={onClose}
          className="text-muted-foreground hover:text-foreground transition-colors text-lg leading-none px-1"
          aria-label="Close panel"
        >
          &times;
        </button>
      </div>

      {/* Form */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Pipeline Name (creation mode only) */}
        {isCreating && (
          <div>
            <label className={labelClass}>Pipeline Name</label>
            <input
              ref={nameInputRef}
              type="text"
              value={newName}
              onChange={e => {
                setNewName(e.target.value)
                setNameError(null)
              }}
              className={`${inputClass} font-mono text-xs ${nameError ? 'border-red-500 focus:ring-red-500' : ''}`}
              placeholder="my-pipeline"
            />
            {nameError && (
              <p className="text-[10px] text-red-400 mt-1">{nameError}</p>
            )}
          </div>
        )}

        {/* Iterations */}
        <div>
          <label className={labelClass}>Iterations</label>
          <input
            type="number"
            min={1}
            value={iterations}
            onChange={e => setIterations(Math.max(1, parseInt(e.target.value) || 1))}
            className={inputClass + ' font-mono text-xs'}
          />
          <p className="text-[10px] text-muted-foreground mt-1">Number of times to run the pipeline</p>
        </div>

        {/* Parallelism */}
        <div>
          <label className={labelClass}>Parallelism</label>
          <input
            type="number"
            min={1}
            value={parallelism}
            onChange={e => setParallelism(Math.max(1, parseInt(e.target.value) || 1))}
            className={inputClass + ' font-mono text-xs'}
          />
          <p className="text-[10px] text-muted-foreground mt-1">Max concurrent agents</p>
        </div>

        {/* Tasks */}
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className={labelClass + ' !mb-0'}>Tasks</label>
            <div className="flex gap-2">
              <button
                onClick={selectAll}
                className="text-[10px] text-primary hover:text-primary/80 font-medium"
              >
                All
              </button>
              <button
                onClick={selectNone}
                className="text-[10px] text-primary hover:text-primary/80 font-medium"
              >
                None
              </button>
            </div>
          </div>
          {allTaskNames.length === 0 ? (
            <p className="text-xs text-muted-foreground italic">No tasks defined</p>
          ) : (
            <div className="space-y-1">
              {allTaskNames.map(name => (
                <label key={name} className="flex items-center gap-2 cursor-pointer group">
                  <input
                    type="checkbox"
                    checked={selectedTasks.has(name)}
                    onChange={() => toggleTask(name)}
                    className="rounded border-border bg-background text-primary focus:ring-primary focus:ring-offset-0"
                  />
                  <span className="text-xs text-foreground group-hover:text-primary transition-colors font-mono">
                    {name}
                  </span>
                </label>
              ))}
            </div>
          )}
          <p className="text-[10px] text-muted-foreground mt-1.5">
            {selectedTasks.size} of {allTaskNames.length} tasks selected
          </p>
        </div>
      </div>

      {/* Footer */}
      <div className="px-4 py-3 border-t border-border flex gap-2">
        {!isCreating && (
          <button
            onClick={handleDelete}
            className="text-xs px-3 py-1.5 border border-red-500/30 rounded text-red-400 hover:bg-red-500/10 transition-colors"
          >
            Delete
          </button>
        )}
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
