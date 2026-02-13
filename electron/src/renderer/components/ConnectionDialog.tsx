import { useEffect, useRef } from 'react'

type Condition = 'success' | 'failure' | 'any' | 'always'

interface ConnectionDialogProps {
  sourceTask: string
  targetTask: string
  position: { x: number; y: number }
  onSelect: (condition: Condition) => void
  onCancel: () => void
}

const CONDITIONS: { value: Condition; label: string; color: string }[] = [
  { value: 'success', label: 'success', color: 'bg-green-500/20 text-green-400 border-green-500/40 hover:bg-green-500/30' },
  { value: 'failure', label: 'failure', color: 'bg-red-500/20 text-red-400 border-red-500/40 hover:bg-red-500/30' },
  { value: 'any', label: 'any', color: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/40 hover:bg-yellow-500/30' },
  { value: 'always', label: 'always', color: 'bg-blue-500/20 text-blue-400 border-blue-500/40 hover:bg-blue-500/30' },
]

export default function ConnectionDialog({ sourceTask, targetTask, position, onSelect, onCancel }: ConnectionDialogProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as HTMLElement)) {
        onCancel()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    document.addEventListener('mousedown', handleClickOutside)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      document.removeEventListener('mousedown', handleClickOutside)
    }
  }, [onCancel])

  return (
    <div
      ref={ref}
      className="fixed z-50 bg-card border border-border rounded-lg shadow-xl p-3 animate-in fade-in zoom-in-95 duration-150"
      style={{ left: position.x, top: position.y, transform: 'translate(-50%, -50%)' }}
    >
      <p className="text-[10px] text-muted-foreground mb-2 text-center">
        {sourceTask} &rarr; {targetTask}
      </p>
      <div className="flex gap-1.5">
        {CONDITIONS.map(({ value, label, color }) => (
          <button
            key={value}
            onClick={() => onSelect(value)}
            className={`text-[11px] font-medium px-2.5 py-1 rounded border transition-colors ${color}`}
          >
            {label}
          </button>
        ))}
      </div>
    </div>
  )
}
