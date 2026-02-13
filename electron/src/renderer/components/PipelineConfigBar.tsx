import { useState, useCallback } from 'react'
import type { ComposeFile, PipelineDef } from '../lib/yamlParser'

interface PipelineConfigBarProps {
  compose: ComposeFile
  activePipeline: string | null
  onSelectPipeline: (name: string | null) => void
  onUpdatePipeline: (name: string, updates: { iterations?: number; parallelism?: number }) => void
  onEditPipeline?: (name: string) => void
  onCreatePipeline?: () => void
  onRunPipeline?: (name: string) => Promise<void>
}

export default function PipelineConfigBar({
  compose,
  activePipeline,
  onSelectPipeline,
  onUpdatePipeline,
  onEditPipeline,
  onCreatePipeline,
  onRunPipeline,
}: PipelineConfigBarProps) {
  const [running, setRunning] = useState(false)
  const pipelines = compose.pipelines ?? {}
  const pipelineNames = Object.keys(pipelines)

  // Always show the bar if there are pipelines, or if we can create new ones
  if (pipelineNames.length === 0 && !onCreatePipeline) return null

  const current: PipelineDef | null = activePipeline ? pipelines[activePipeline] ?? null : null

  return (
    <div className="px-3 py-2 border-b border-border bg-secondary/40 flex items-center gap-4 text-sm">
      {/* Pipeline selector */}
      <div className="flex items-center gap-2">
        <label className="text-xs font-semibold text-muted-foreground whitespace-nowrap">Pipeline</label>
        <select
          value={activePipeline ?? '__all__'}
          onChange={e => onSelectPipeline(e.target.value === '__all__' ? null : e.target.value)}
          className="bg-background border border-border rounded px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
        >
          <option value="__all__">All Tasks</option>
          {pipelineNames.map(name => (
            <option key={name} value={name}>{name}</option>
          ))}
        </select>
      </div>

      {/* Pipeline settings (only when a specific pipeline is selected) */}
      {current && activePipeline && (
        <>
          <div className="w-px h-5 bg-border" />
          <NumberField
            label="Iterations"
            value={current.iterations ?? 1}
            onCommit={val => onUpdatePipeline(activePipeline, { iterations: val })}
          />
          <div className="w-px h-5 bg-border" />
          <NumberField
            label="Parallelism"
            value={current.parallelism ?? 1}
            onCommit={val => onUpdatePipeline(activePipeline, { parallelism: val })}
          />
          <div className="w-px h-5 bg-border" />
          <div className="flex items-center gap-2 min-w-0">
            <span className="text-xs font-semibold text-muted-foreground whitespace-nowrap">Tasks</span>
            <span className="text-xs text-foreground truncate">
              {current.tasks && current.tasks.length > 0
                ? current.tasks.join(', ')
                : <span className="text-muted-foreground italic">all</span>}
            </span>
          </div>
          {onEditPipeline && (
            <>
              <div className="w-px h-5 bg-border" />
              <button
                onClick={() => onEditPipeline(activePipeline)}
                className="text-[10px] px-2 py-1 rounded border border-border text-muted-foreground hover:text-foreground hover:border-primary/50 transition-colors whitespace-nowrap"
              >
                Configure
              </button>
            </>
          )}
          {onRunPipeline && (
            <>
              <div className="w-px h-5 bg-border" />
              <button
                disabled={running}
                onClick={async () => {
                  setRunning(true)
                  try {
                    await onRunPipeline(activePipeline)
                  } finally {
                    setRunning(false)
                  }
                }}
                className="text-[10px] px-2.5 py-1 rounded bg-primary text-primary-foreground hover:bg-primary/90 transition-colors whitespace-nowrap font-medium disabled:opacity-50 disabled:cursor-not-allowed"
                title="Run this pipeline"
              >
                {running ? 'Running\u2026' : '\u25B6 Run'}
              </button>
            </>
          )}
        </>
      )}

      {/* New pipeline button */}
      <div className="ml-auto">
        {onCreatePipeline && (
          <button
            onClick={onCreatePipeline}
            className="text-[10px] px-2 py-1 rounded border border-border text-primary hover:bg-primary/10 transition-colors whitespace-nowrap font-medium"
          >
            + New Pipeline
          </button>
        )}
      </div>
    </div>
  )
}

function NumberField({
  label,
  value,
  onCommit,
}: {
  label: string
  value: number
  onCommit: (val: number) => void
}) {
  const [draft, setDraft] = useState(String(value))
  const [focused, setFocused] = useState(false)

  // Keep draft in sync when value changes externally (and field is not focused)
  if (!focused && String(value) !== draft) {
    setDraft(String(value))
  }

  const commit = useCallback(() => {
    const parsed = parseInt(draft, 10)
    if (!isNaN(parsed) && parsed > 0 && parsed !== value) {
      onCommit(parsed)
    } else {
      setDraft(String(value))
    }
  }, [draft, value, onCommit])

  return (
    <div className="flex items-center gap-2">
      <label className="text-xs font-semibold text-muted-foreground whitespace-nowrap">{label}</label>
      <input
        type="number"
        min={1}
        value={draft}
        onChange={e => setDraft(e.target.value)}
        onFocus={() => setFocused(true)}
        onBlur={() => { setFocused(false); commit() }}
        onKeyDown={e => { if (e.key === 'Enter') commit() }}
        className="w-16 bg-background border border-border rounded px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
      />
    </div>
  )
}
