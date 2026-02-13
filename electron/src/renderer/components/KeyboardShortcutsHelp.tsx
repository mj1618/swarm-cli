import { useEffect, useCallback, useRef } from 'react'

interface KeyboardShortcutsHelpProps {
  open: boolean
  onClose: () => void
}

const isMac = navigator.platform.toUpperCase().includes('MAC')
const mod = isMac ? '⌘' : 'Ctrl'

interface ShortcutEntry {
  keys: string[]
  action: string
}

interface ShortcutGroup {
  title: string
  shortcuts: ShortcutEntry[]
}

const groups: ShortcutGroup[] = [
  {
    title: 'General',
    shortcuts: [
      { keys: [`${mod}+K`], action: 'Open command palette' },
      { keys: [`${mod}+J`], action: 'Toggle console panel' },
      { keys: ['?'], action: 'Show keyboard shortcuts' },
    ],
  },
  {
    title: 'Console / Logs',
    shortcuts: [
      { keys: [`${mod}+F`], action: 'Search in console logs' },
      { keys: ['Esc'], action: 'Clear search' },
    ],
  },
  {
    title: 'File Editor',
    shortcuts: [
      { keys: [`${mod}+S`], action: 'Save file' },
    ],
  },
  {
    title: 'DAG Canvas',
    shortcuts: [
      { keys: ['N'], action: 'Create new task' },
      { keys: ['F'], action: 'Fit DAG to view' },
      { keys: ['R'], action: 'Reset DAG layout' },
      { keys: ['Delete', 'Backspace'], action: 'Delete selected task or edge' },
      { keys: ['Esc'], action: 'Deselect all' },
    ],
  },
  {
    title: 'Panels & Dialogs',
    shortcuts: [
      { keys: ['Esc'], action: 'Close open drawer / dialog / panel' },
    ],
  },
]

export default function KeyboardShortcutsHelp({ open, onClose }: KeyboardShortcutsHelpProps) {
  const backdropRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown, true)
    return () => document.removeEventListener('keydown', handleKeyDown, true)
  }, [open, onClose])

  const handleBackdropClick = useCallback((e: React.MouseEvent) => {
    if (e.target === backdropRef.current) {
      onClose()
    }
  }, [onClose])

  if (!open) return null

  return (
    <div
      ref={backdropRef}
      className="fixed inset-0 z-[100] flex items-start justify-center pt-[15vh] bg-black/50"
      onClick={handleBackdropClick}
    >
      <div className="w-full max-w-lg bg-card border border-border rounded-lg shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border">
          <h2 className="text-sm font-semibold text-foreground">Keyboard Shortcuts</h2>
          <button
            onClick={onClose}
            className="text-muted-foreground hover:text-foreground text-lg leading-none"
          >
            ×
          </button>
        </div>

        {/* Body */}
        <div className="px-4 py-3 max-h-[60vh] overflow-y-auto space-y-4">
          {groups.map((group) => (
            <div key={group.title}>
              <h3 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                {group.title}
              </h3>
              <div className="space-y-1.5">
                {group.shortcuts.map((shortcut) => (
                  <div
                    key={shortcut.action}
                    className="flex items-center justify-between text-sm"
                  >
                    <span className="text-foreground">{shortcut.action}</span>
                    <div className="flex items-center gap-1 ml-4 shrink-0">
                      {shortcut.keys.map((key, i) => (
                        <span key={key}>
                          {i > 0 && <span className="text-muted-foreground mx-0.5">/</span>}
                          <kbd className="inline-block px-1.5 py-0.5 text-xs font-mono bg-secondary border border-border rounded text-muted-foreground">
                            {key}
                          </kbd>
                        </span>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
