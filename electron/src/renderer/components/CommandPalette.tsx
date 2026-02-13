import { useState, useEffect, useRef, useCallback } from 'react'

export interface Command {
  id: string
  name: string
  description?: string
  shortcut?: string
  action: () => void
}

interface CommandPaletteProps {
  open: boolean
  onClose: () => void
  commands: Command[]
}

function highlightMatch(text: string, query: string): React.ReactNode {
  if (!query) return text
  const lower = text.toLowerCase()
  const idx = lower.indexOf(query.toLowerCase())
  if (idx === -1) return text
  return (
    <>
      {text.slice(0, idx)}
      <span className="text-blue-400 font-semibold">{text.slice(idx, idx + query.length)}</span>
      {text.slice(idx + query.length)}
    </>
  )
}

export default function CommandPalette({ open, onClose, commands }: CommandPaletteProps) {
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const backdropRef = useRef<HTMLDivElement>(null)

  const filtered = commands.filter(cmd =>
    cmd.name.toLowerCase().includes(query.toLowerCase()),
  )

  // Reset state when opening
  useEffect(() => {
    if (open) {
      setQuery('')
      setSelectedIndex(0)
      // Focus input after a tick so the element is rendered
      requestAnimationFrame(() => inputRef.current?.focus())
    }
  }, [open])

  // Keep selected index in bounds when filter changes
  useEffect(() => {
    setSelectedIndex(prev => Math.min(prev, Math.max(filtered.length - 1, 0)))
  }, [filtered.length])

  // Scroll selected item into view
  useEffect(() => {
    if (!listRef.current) return
    const item = listRef.current.children[selectedIndex] as HTMLElement | undefined
    item?.scrollIntoView({ block: 'nearest' })
  }, [selectedIndex])

  const execute = useCallback((cmd: Command) => {
    onClose()
    // Run action after close to avoid stale state
    requestAnimationFrame(() => cmd.action())
  }, [onClose])

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        setSelectedIndex(prev => (prev + 1) % Math.max(filtered.length, 1))
        break
      case 'ArrowUp':
        e.preventDefault()
        setSelectedIndex(prev => (prev - 1 + filtered.length) % Math.max(filtered.length, 1))
        break
      case 'Enter':
        e.preventDefault()
        if (filtered[selectedIndex]) {
          execute(filtered[selectedIndex])
        }
        break
      case 'Escape':
        e.preventDefault()
        onClose()
        break
    }
  }, [filtered, selectedIndex, execute, onClose])

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
      <div className="w-full max-w-md bg-card border border-border rounded-lg shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        {/* Search input */}
        <div className="flex items-center border-b border-border px-3">
          <svg className="w-4 h-4 text-muted-foreground shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Type a command..."
            className="flex-1 bg-transparent text-sm text-foreground placeholder:text-muted-foreground py-3 px-2 outline-none"
          />
          <kbd className="text-[10px] text-muted-foreground bg-zinc-800 px-1.5 py-0.5 rounded border border-zinc-700">
            ESC
          </kbd>
        </div>

        {/* Command list */}
        <div ref={listRef} className="max-h-72 overflow-auto py-1">
          {filtered.length === 0 ? (
            <div className="px-3 py-6 text-sm text-muted-foreground text-center">
              No commands found
            </div>
          ) : (
            filtered.map((cmd, i) => (
              <button
                key={cmd.id}
                onClick={() => execute(cmd)}
                onMouseEnter={() => setSelectedIndex(i)}
                className={`w-full text-left px-3 py-2 flex items-center justify-between gap-2 transition-colors ${
                  i === selectedIndex
                    ? 'bg-zinc-700/60 text-foreground'
                    : 'text-zinc-300 hover:bg-zinc-800/50'
                }`}
              >
                <div className="min-w-0">
                  <div className="text-sm truncate">{highlightMatch(cmd.name, query)}</div>
                  {cmd.description && (
                    <div className="text-[11px] text-muted-foreground truncate">{cmd.description}</div>
                  )}
                </div>
                {cmd.shortcut && (
                  <kbd className="shrink-0 text-[10px] text-muted-foreground bg-zinc-800 px-1.5 py-0.5 rounded border border-zinc-700">
                    {cmd.shortcut}
                  </kbd>
                )}
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
