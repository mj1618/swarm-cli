import { useState, useEffect, useCallback, useRef } from 'react'
import LogView from './LogView'

interface LogFile {
  name: string
  path: string
  modifiedAt: number
}

export default function ConsolePanel() {
  const [logFiles, setLogFiles] = useState<LogFile[]>([])
  const [activeTab, setActiveTab] = useState<string>('console')
  const [logContents, setLogContents] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
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

  const fetchLogContent = useCallback(async (filePath: string) => {
    const result = await window.logs.read(filePath)
    if (!result.error) {
      setLogContents(prev => ({ ...prev, [filePath]: result.content }))
    }
  }, [])

  // Initial load
  useEffect(() => {
    fetchLogFiles()
  }, [fetchLogFiles])

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
      fetchLogFiles()
    })
    cleanupRef.current = cleanup

    return () => {
      if (cleanupRef.current) {
        cleanupRef.current()
      }
      window.logs.unwatch()
    }
  }, [fetchLogFiles])

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
        />
      )}
    </div>
  )
}
