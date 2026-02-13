import { useRef, useEffect, useState, useCallback } from 'react'

interface LogViewProps {
  content: string
  loading?: boolean
  error?: string | null
}

function classifyLine(line: string): 'error' | 'tool' | 'normal' {
  if (/\b[Ee]rror\b/.test(line) || /\b[Ff]ailed\b/.test(line) || /\bpanic\b/.test(line)) {
    return 'error'
  }
  if (/\b(Read|Write|Edit|Bash|Glob|Grep|WebFetch|Task)\b/.test(line)) {
    return 'tool'
  }
  return 'normal'
}

function lineClass(kind: 'error' | 'tool' | 'normal'): string {
  switch (kind) {
    case 'error':
      return 'text-red-400'
    case 'tool':
      return 'text-muted-foreground/60'
    default:
      return 'text-foreground/80'
  }
}

export default function LogView({ content, loading, error }: LogViewProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [autoScroll, setAutoScroll] = useState(true)

  const scrollToBottom = useCallback(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight
    }
  }, [])

  useEffect(() => {
    if (autoScroll) {
      scrollToBottom()
    }
  }, [content, autoScroll, scrollToBottom])

  const handleScroll = useCallback(() => {
    if (!containerRef.current) return
    const { scrollTop, scrollHeight, clientHeight } = containerRef.current
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 40
    setAutoScroll(isAtBottom)
  }, [])

  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center text-sm text-muted-foreground">
        Loading logs...
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center text-sm text-red-400">
        {error}
      </div>
    )
  }

  if (!content) {
    return (
      <div className="flex-1 flex items-center justify-center text-sm text-muted-foreground">
        No log content
      </div>
    )
  }

  const lines = content.split('\n')

  return (
    <div className="flex-1 flex flex-col min-h-0 relative">
      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-auto p-2 font-mono text-xs leading-relaxed"
      >
        {lines.map((line, i) => {
          const kind = classifyLine(line)
          return (
            <div key={i} className={`whitespace-pre-wrap break-all ${lineClass(kind)}`}>
              {line || '\u00A0'}
            </div>
          )
        })}
      </div>
      {!autoScroll && (
        <button
          onClick={() => {
            setAutoScroll(true)
            scrollToBottom()
          }}
          className="absolute bottom-2 right-4 px-2 py-1 rounded bg-primary text-primary-foreground text-xs hover:bg-primary/90 shadow-lg"
        >
          ↓ Scroll to bottom
        </button>
      )}
    </div>
  )
}
