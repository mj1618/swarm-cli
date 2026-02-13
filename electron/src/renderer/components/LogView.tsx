import { useRef, useEffect, useState, useCallback, useMemo } from 'react'

interface LogViewProps {
  content: string
  loading?: boolean
  error?: string | null
  searchQuery?: string
  filterMode?: 'highlight' | 'filter'
  onMatchCount?: (count: number) => void
}

const AGENT_COLORS = [
  'text-cyan-400',
  'text-green-400',
  'text-yellow-400',
  'text-pink-400',
  'text-orange-400',
  'text-purple-400',
  'text-blue-400',
  'text-emerald-400',
  'text-rose-400',
  'text-teal-400',
]

const agentColorMap = new Map<string, string>()

function getAgentColor(agentName: string): string {
  const existing = agentColorMap.get(agentName)
  if (existing) return existing
  // Deterministic hash to pick a color
  let hash = 0
  for (let i = 0; i < agentName.length; i++) {
    hash = ((hash << 5) - hash + agentName.charCodeAt(i)) | 0
  }
  const color = AGENT_COLORS[Math.abs(hash) % AGENT_COLORS.length]
  agentColorMap.set(agentName, color)
  return color
}

// Pattern to match agent tags like [planner], [coder], [evaluator]
const AGENT_TAG_RE = /^(.*?)(\[[a-zA-Z][a-zA-Z0-9_-]*\])(.*)$/

function parseAgentTag(line: string): { before: string; tag: string; after: string; agentName: string } | null {
  const match = AGENT_TAG_RE.exec(line)
  if (!match) return null
  return {
    before: match[1],
    tag: match[2],
    after: match[3],
    agentName: match[2].slice(1, -1),
  }
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

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function highlightMatches(text: string, query: string): React.ReactNode {
  if (!query) return text
  const regex = new RegExp(`(${escapeRegex(query)})`, 'gi')
  const parts = text.split(regex)
  if (parts.length === 1) return text
  const lowerQuery = query.toLowerCase()
  return parts.map((part, i) =>
    part.toLowerCase() === lowerQuery
      ? <mark key={i} className="bg-yellow-500/30 text-yellow-200 rounded-sm px-0.5">{part}</mark>
      : part
  )
}

function renderLineContent(line: string, query: string): React.ReactNode {
  const parsed = parseAgentTag(line)
  if (!parsed) {
    return query ? highlightMatches(line || '\u00A0', query) : (line || '\u00A0')
  }
  const colorClass = getAgentColor(parsed.agentName)
  if (query) {
    return (
      <>
        {highlightMatches(parsed.before, query)}
        <span className={colorClass}>{highlightMatches(parsed.tag, query)}</span>
        {highlightMatches(parsed.after, query)}
      </>
    )
  }
  return (
    <>
      {parsed.before}
      <span className={colorClass}>{parsed.tag}</span>
      {parsed.after}
    </>
  )
}

export default function LogView({ content, loading, error, searchQuery, filterMode = 'highlight', onMatchCount }: LogViewProps) {
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

  const query = searchQuery?.trim() || ''

  const filteredLines = useMemo(() => {
    if (!query) return lines.map((line, i) => ({ line, index: i }))
    const lowerQuery = query.toLowerCase()
    const all = lines.map((line, i) => ({ line, index: i, matches: line.toLowerCase().includes(lowerQuery) }))
    if (filterMode === 'filter') {
      return all.filter(l => l.matches)
    }
    return all
  }, [lines, query, filterMode])

  const matchCount = useMemo(() => {
    if (!query) return 0
    const lowerQuery = query.toLowerCase()
    let count = 0
    for (const line of lines) {
      const lowerLine = line.toLowerCase()
      let idx = 0
      while ((idx = lowerLine.indexOf(lowerQuery, idx)) !== -1) {
        count++
        idx += lowerQuery.length
      }
    }
    return count
  }, [lines, query])

  useEffect(() => {
    onMatchCount?.(matchCount)
  }, [matchCount, onMatchCount])

  return (
    <div className="flex-1 flex flex-col min-h-0 relative">
      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-auto p-2 font-mono text-xs leading-relaxed"
      >
        {filteredLines.map(({ line, index }) => {
          const kind = classifyLine(line)
          return (
            <div key={index} className={`whitespace-pre-wrap break-all ${lineClass(kind)}`}>
              {renderLineContent(line, query)}
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
