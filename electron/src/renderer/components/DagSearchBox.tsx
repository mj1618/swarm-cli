import { useState, useCallback, useRef, useEffect, useMemo } from 'react'

interface DagSearchBoxProps {
  taskNames: string[]
  onSelectTask: (taskName: string) => void
  disabled?: boolean
}

export default function DagSearchBox({ taskNames, onSelectTask, disabled }: DagSearchBoxProps) {
  const [query, setQuery] = useState('')
  const [isOpen, setIsOpen] = useState(false)
  const [highlightedIndex, setHighlightedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  // Filter tasks case-insensitively
  const filteredTasks = useMemo(() => {
    if (!query.trim()) return taskNames
    const lowerQuery = query.toLowerCase()
    return taskNames.filter(name => name.toLowerCase().includes(lowerQuery))
  }, [query, taskNames])

  // Reset highlighted index when filtered results change
  useEffect(() => {
    setHighlightedIndex(0)
  }, [filteredTasks])

  // Global `/` key handler to focus search
  useEffect(() => {
    function handleGlobalKeyDown(e: KeyboardEvent) {
      // Don't intercept when typing in other inputs
      const tag = (e.target as HTMLElement)?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return

      if (e.key === '/') {
        e.preventDefault()
        inputRef.current?.focus()
        setIsOpen(true)
      }
    }
    document.addEventListener('keydown', handleGlobalKeyDown)
    return () => document.removeEventListener('keydown', handleGlobalKeyDown)
  }, [])

  // Close dropdown on outside click
  useEffect(() => {
    if (!isOpen) return
    function handleClick(e: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(e.target as Node)
      ) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [isOpen])

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setQuery(e.target.value)
    setIsOpen(true)
  }, [])

  const handleInputFocus = useCallback(() => {
    setIsOpen(true)
  }, [])

  const handleSelect = useCallback((taskName: string) => {
    onSelectTask(taskName)
    setQuery('')
    setIsOpen(false)
    inputRef.current?.blur()
  }, [onSelectTask])

  const handleClear = useCallback(() => {
    setQuery('')
    setIsOpen(false)
    inputRef.current?.blur()
  }, [])

  const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setHighlightedIndex(prev => 
        prev < filteredTasks.length - 1 ? prev + 1 : prev
      )
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setHighlightedIndex(prev => (prev > 0 ? prev - 1 : 0))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (filteredTasks.length > 0 && highlightedIndex < filteredTasks.length) {
        handleSelect(filteredTasks[highlightedIndex])
      }
    } else if (e.key === 'Escape') {
      e.preventDefault()
      setQuery('')
      setIsOpen(false)
      inputRef.current?.blur()
    }
  }, [filteredTasks, highlightedIndex, handleSelect])

  // Scroll highlighted item into view
  useEffect(() => {
    if (!isOpen || filteredTasks.length === 0) return
    const dropdown = dropdownRef.current
    if (!dropdown) return
    const items = dropdown.querySelectorAll('[data-search-item]')
    const highlighted = items[highlightedIndex]
    if (highlighted) {
      highlighted.scrollIntoView({ block: 'nearest' })
    }
  }, [highlightedIndex, isOpen, filteredTasks.length])

  return (
    <div className="relative">
      <div className="relative">
        {/* Search icon */}
        <svg
          className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground pointer-events-none"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
          />
        </svg>
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={handleInputChange}
          onFocus={handleInputFocus}
          onKeyDown={handleKeyDown}
          disabled={disabled}
          placeholder="Search tasks..."
          className="w-48 pl-8 pr-7 py-1.5 text-xs rounded-md bg-secondary/80 border border-border text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary focus:border-primary disabled:opacity-50 disabled:cursor-not-allowed"
        />
        {/* Clear button or "/" hint */}
        {query ? (
          <button
            type="button"
            onClick={handleClear}
            className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
            title="Clear search"
          >
            <svg className="h-3.5 w-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        ) : (
          <span className="absolute right-2 top-1/2 -translate-y-1/2 text-[10px] text-muted-foreground/60 font-mono pointer-events-none">
            /
          </span>
        )}
      </div>

      {/* Dropdown */}
      {isOpen && (
        <div
          ref={dropdownRef}
          className="absolute top-full left-0 mt-1 w-56 max-h-48 overflow-y-auto rounded-md border border-border bg-popover shadow-lg z-50"
        >
          {filteredTasks.length === 0 ? (
            <div className="px-3 py-2 text-xs text-muted-foreground">
              No matching tasks
            </div>
          ) : (
            <ul className="py-1">
              {filteredTasks.map((task, index) => (
                <li key={task}>
                  <button
                    type="button"
                    data-search-item
                    onClick={() => handleSelect(task)}
                    onMouseEnter={() => setHighlightedIndex(index)}
                    className={`w-full px-3 py-1.5 text-left text-xs transition-colors flex items-center gap-2 ${
                      index === highlightedIndex
                        ? 'bg-secondary text-foreground'
                        : 'text-foreground/90 hover:bg-secondary/50'
                    }`}
                  >
                    {/* Task icon */}
                    <svg
                      className="h-3.5 w-3.5 text-muted-foreground shrink-0"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
                      />
                    </svg>
                    <span className="truncate">{task}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}
