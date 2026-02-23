import { useState, useEffect, useCallback } from 'react'

interface OutputFile {
  name: string
  path: string
  status: 'done' | 'pending' | 'processing' | 'other'
  taskName: string
}

interface OutputRunViewerProps {
  folderPath: string
  onOpenFile: (filePath: string) => void
}

function parseRunFolder(folderName: string): { timestamp: string; id: string } | null {
  // Format: YYYYMMDD-HHMMSS-<hex-id>
  const match = folderName.match(/^(\d{4})(\d{2})(\d{2})-(\d{2})(\d{2})(\d{2})-([a-f0-9]+)$/)
  if (!match) return null
  const [, year, month, day, hour, minute, second, id] = match
  const date = new Date(
    parseInt(year), parseInt(month) - 1, parseInt(day),
    parseInt(hour), parseInt(minute), parseInt(second),
  )
  const timestamp = date.toLocaleString('en-US', {
    month: 'short', day: 'numeric', year: 'numeric',
    hour: 'numeric', minute: '2-digit', second: '2-digit',
    hour12: true,
  })
  return { timestamp, id }
}

function classifyFile(name: string): { status: OutputFile['status']; taskName: string } {
  if (name.endsWith('.done.md')) {
    return { status: 'done', taskName: name.replace(/\.done\.md$/, '') }
  }
  if (name.endsWith('.pending.md')) {
    return { status: 'pending', taskName: name.replace(/\.pending\.md$/, '') }
  }
  if (name.includes('.processing.md')) {
    return { status: 'processing', taskName: name.replace(/\.[a-f0-9]+\.processing\.md$/, '') }
  }
  return { status: 'other', taskName: name }
}

const statusConfig = {
  done: { label: 'Done', color: 'text-green-400', bg: 'bg-green-400/10', icon: '✓' },
  pending: { label: 'Pending', color: 'text-yellow-400', bg: 'bg-yellow-400/10', icon: '○' },
  processing: { label: 'In Progress', color: 'text-blue-400', bg: 'bg-blue-400/10', icon: '◉' },
  other: { label: 'File', color: 'text-muted-foreground', bg: 'bg-muted/30', icon: '·' },
}

export default function OutputRunViewer({ folderPath, onOpenFile }: OutputRunViewerProps) {
  const [files, setFiles] = useState<OutputFile[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const folderName = folderPath.split('/').pop() || ''
  const parsed = parseRunFolder(folderName)

  const loadFiles = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const result = await window.fs.readdir(folderPath)
      if (result.error) {
        setError(result.error)
        setFiles([])
      } else {
        const mapped: OutputFile[] = result.entries
          .filter(e => !e.isDirectory)
          .map(e => {
            const { status, taskName } = classifyFile(e.name)
            return { name: e.name, path: e.path, status, taskName }
          })
          .sort((a, b) => {
            const order = { done: 0, processing: 1, pending: 2, other: 3 }
            const diff = order[a.status] - order[b.status]
            if (diff !== 0) return diff
            return a.taskName.localeCompare(b.taskName)
          })
        setFiles(mapped)
      }
    } catch {
      setError('Failed to read output folder')
      setFiles([])
    } finally {
      setLoading(false)
    }
  }, [folderPath])

  useEffect(() => {
    loadFiles()
  }, [loadFiles])

  const doneCount = files.filter(f => f.status === 'done').length
  const pendingCount = files.filter(f => f.status === 'pending').length
  const processingCount = files.filter(f => f.status === 'processing').length
  const totalTasks = doneCount + pendingCount + processingCount

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="p-4 border-b border-border">
        <div className="flex items-center gap-2 mb-1">
          <span className="text-lg font-semibold text-foreground">Pipeline Run</span>
          <button
            onClick={loadFiles}
            className="text-xs px-1.5 py-0.5 rounded hover:bg-accent text-muted-foreground"
            title="Refresh"
          >
            ↻
          </button>
        </div>
        {parsed ? (
          <div className="flex items-center gap-3 text-sm text-muted-foreground">
            <span>{parsed.timestamp}</span>
            <span className="font-mono text-xs bg-muted/50 px-1.5 py-0.5 rounded">{parsed.id}</span>
          </div>
        ) : (
          <div className="text-sm text-muted-foreground font-mono">{folderName}</div>
        )}
      </div>

      {/* Summary stats */}
      {!loading && !error && totalTasks > 0 && (
        <div className="px-4 py-3 border-b border-border flex items-center gap-4">
          <div className="flex items-center gap-1.5 text-sm">
            <span className="text-green-400 font-medium">{doneCount}</span>
            <span className="text-muted-foreground">done</span>
          </div>
          {processingCount > 0 && (
            <div className="flex items-center gap-1.5 text-sm">
              <span className="text-blue-400 font-medium">{processingCount}</span>
              <span className="text-muted-foreground">in progress</span>
            </div>
          )}
          <div className="flex items-center gap-1.5 text-sm">
            <span className="text-yellow-400 font-medium">{pendingCount}</span>
            <span className="text-muted-foreground">pending</span>
          </div>
          {totalTasks > 0 && (
            <div className="ml-auto text-xs text-muted-foreground">
              {doneCount}/{totalTasks} tasks
            </div>
          )}
        </div>
      )}

      {/* File list */}
      <div className="flex-1 overflow-auto p-2">
        {loading ? (
          <div className="text-sm text-muted-foreground p-4">Loading...</div>
        ) : error ? (
          <div className="text-sm text-red-400 p-4">{error}</div>
        ) : files.length === 0 ? (
          <div className="text-sm text-muted-foreground p-4">No files in this output folder.</div>
        ) : (
          <div className="space-y-1">
            {files.map(file => {
              const cfg = statusConfig[file.status]
              return (
                <button
                  key={file.path}
                  onClick={() => onOpenFile(file.path)}
                  className="w-full text-left px-3 py-2 rounded hover:bg-accent/50 transition-colors flex items-center gap-3 group"
                >
                  <span className={`text-sm ${cfg.color} w-4 text-center shrink-0`}>{cfg.icon}</span>
                  <span className="text-sm text-foreground truncate flex-1">{file.taskName}</span>
                  <span className={`text-xs px-1.5 py-0.5 rounded ${cfg.bg} ${cfg.color} shrink-0`}>
                    {cfg.label}
                  </span>
                </button>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

/** Check if a path is an output run folder */
export function isOutputRunFolder(filePath: string): boolean {
  // Match paths like .../outputs/YYYYMMDD-HHMMSS-hexid
  return /\/outputs\/\d{8}-\d{6}-[a-f0-9]+\/?$/.test(filePath)
}
