import { useState, useEffect, useCallback, useRef } from 'react'
import LogView from './LogView'

type FilterMode = 'highlight' | 'filter'

interface LogFile {
  name: string
  path: string
  modifiedAt: number
}

interface ConsolePanelProps {
  activeTab?: string
  onActiveTabChange?: (tab: string) => void
}

export default function ConsolePanel({ activeTab: controlledTab, onActiveTabChange }: ConsolePanelProps = {}) {
  const [logFiles, setLogFiles] = useState<LogFile[]>([])
  const [internalTab, setInternalTab] = useState<string>('console')
  const activeTab = controlledTab ?? internalTab
  const setActiveTab = onActiveTabChange ?? setInternalTab
  const [logContents, setLogContents] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [filterMode, setFilterMode] = useState<FilterMode>('highlight')
  const [matchCount, setMatchCount] = useState(0)
  const [autoScroll, setAutoScroll] = useState(true)
  const searchInputRef = useRef<HTMLInputElement>(null)
  const cleanupRef = useRef<(() => void) | null>(null)

  const fetchLogFiles = useCallback(async () => {
    const result = await window.logs.list()
    if (result.error) {
      setError(result.error)
      setLogFiles([])
    } else {
      setError(null)
      setLogFiles(result.entries)
    }
    setLoading(false)
  }, [])
  const fetchLogFilesRef = useRef(fetchLogFiles)
  fetchLogFilesRef.current = fetchLogFiles

  const fetchLogContent = useCallback(async (filePath: string) => {
    const result = await window.logs.read(filePath)
    if (!result.error) {
      setLogContents(prev => ({ ...prev, [filePath]: result.content }))
    }
  }, [])

  // Initial load
  useEffect(() => {
    fetchLogFilesRef.current()
  }, [])

  // Load content for all log files when list changes
  useEffect(() => {
    for (const file of logFiles) {
      fetchLogContent(file.path)
    }
  }, [logFiles, fetchLogContent])

  // Watch for changes
  useEffect(() => {
    window.logs.watch()
    const cleanup = window.logs.onChanged(() => {
      fetchLogFilesRef.current()
    })
    cleanupRef.current = cleanup

    return () => {
      if (cleanupRef.current) {
        cleanupRef.current()
      }
      window.logs.unwatch()
    }
  }, [])

  // Cmd+F / Ctrl+F to focus search input
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'f') {
        e.preventDefault()
        searchInputRef.current?.focus()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  // Derive tab label from filename (strip .log extension)
  const tabLabel = (name: string) => name.replace(/\.log$/, '').slice(0, 12)

  // Build combined console content from all log files (most recent first)
  const combinedContent = logFiles
    .map(f => logContents[f.path] || '')
    .filter(Boolean)
    .join('\n--- --- ---\n')

  const activeContent = activeTab === 'console'
    ? combinedContent
    : logContents[activeTab] || ''

  const activeFile = logFiles.find(f => f.path === activeTab)
  const activeLoading = loading
  const activeError = activeTab === 'console' ? error : (activeFile ? null : error)

  return (
    <div className="h-full flex flex-col">
      {/* Tab bar */}
      <div className="flex items-center border-b border-border px-1 gap-0.5 shrink-0">
        <button
          onClick={() => setActiveTab('console')}
          className={`px-3 py-1.5 text-xs font-medium border-b-2 transition-colors ${
            activeTab === 'console'
              ? 'border-primary text-foreground'
              : 'border-transparent text-muted-foreground hover:text-foreground'
          }`}
        >
          Console
        </button>
        {logFiles.map(file => (
          <button
            key={file.path}
            onClick={() => setActiveTab(file.path)}
            className={`px-3 py-1.5 text-xs font-medium border-b-2 transition-colors ${
              activeTab === file.path
                ? 'border-primary text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {tabLabel(file.name)}
          </button>
        ))}
        <div className="flex-1" />
        {/* Search bar */}
        <div className="flex items-center gap-1.5">
          <div className="relative">
            <input
              ref={searchInputRef}
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Escape') {
                  setSearchQuery('')
                  searchInputRef.current?.blur()
                }
              }}
              placeholder="Search logs..."
              className="h-6 w-40 rounded border border-border bg-background px-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          {searchQuery && (
            <span className="text-xs text-muted-foreground whitespace-nowrap">
              {matchCount} {matchCount === 1 ? 'match' : 'matches'}
            </span>
          )}
          <button
            onClick={() => setFilterMode(prev => prev === 'highlight' ? 'filter' : 'highlight')}
            className={`px-1.5 py-0.5 text-xs rounded transition-colors ${
              filterMode === 'filter'
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:text-foreground'
            }`}
            title={filterMode === 'filter' ? 'Showing matching lines only' : 'Highlighting matches'}
          >
            Filter
          </button>
          <button
            onClick={() => setAutoScroll(prev => !prev)}
            className={`px-1.5 py-0.5 text-xs rounded transition-colors ${
              autoScroll
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:text-foreground'
            }`}
            title={autoScroll ? 'Auto-scroll enabled (click to disable)' : 'Auto-scroll disabled (click to enable)'}
          >
            {autoScroll ? '↓ Auto' : '↓ Manual'}
          </button>
        </div>
        <button
          onClick={async () => {
            const tabName = activeTab === 'console' ? 'console' : tabLabel(activeFile?.name || 'log')
            const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
            const defaultName = `swarm-${tabName}-${timestamp}.log`
            await window.dialog.saveFile({ defaultName, content: activeContent })
          }}
          disabled={!activeContent}
          className="px-2 py-1 text-xs text-muted-foreground hover:text-foreground disabled:opacity-40"
          title="Export logs to file"
        >
          Export
        </button>
        <button
          onClick={() => {
            if (activeTab === 'console') {
              setLogContents({})
              fetchLogFiles()
            } else {
              fetchLogContent(activeTab)
            }
          }}
          className="px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
          title="Refresh"
        >
          Clear
        </button>
      </div>

      {/* Log content */}
      {logFiles.length === 0 && !loading && !error ? (
        <div className="flex-1 flex items-center justify-center text-sm text-muted-foreground">
          No logs yet
        </div>
      ) : (
        <LogView
          content={activeContent}
          loading={activeLoading}
          error={activeError}
          searchQuery={searchQuery}
          filterMode={filterMode}
          onMatchCount={setMatchCount}
          autoScroll={autoScroll}
          onAutoScrollChange={setAutoScroll}
        />
      )}
    </div>
  )
}
