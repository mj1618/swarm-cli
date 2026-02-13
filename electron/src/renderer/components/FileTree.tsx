import { useState, useEffect, useCallback } from 'react'
import FileTreeItem from './FileTreeItem'

interface DirEntry {
  name: string
  path: string
  isDirectory: boolean
}

interface FileTreeProps {
  selectedPath: string | null
  onSelectFile: (filePath: string) => void
}

export default function FileTree({ selectedPath, onSelectFile }: FileTreeProps) {
  const [entries, setEntries] = useState<DirEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadRoot = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const root = await window.fs.swarmRoot()
      const result = await window.fs.readdir(root)
      if (result.error) {
        setError(result.error)
        setEntries([])
      } else {
        setEntries(result.entries)
      }
    } catch {
      setError('Failed to load swarm/ directory')
      setEntries([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadRoot()

    // Start watching for filesystem changes and auto-refresh the tree
    window.fs.watch()
    const unsubscribe = window.fs.onChanged(() => {
      loadRoot()
    })

    return () => {
      unsubscribe()
      window.fs.unwatch()
    }
  }, [loadRoot])

  const handleSelect = useCallback((filePath: string, isDirectory: boolean) => {
    if (!isDirectory) {
      onSelectFile(filePath)
    }
  }, [onSelectFile])

  return (
    <div className="flex flex-col h-full">
      <div className="p-3 border-b border-border flex items-center justify-between">
        <h2 className="text-sm font-semibold text-foreground">Files</h2>
        <button
          onClick={loadRoot}
          className="text-xs px-1.5 py-0.5 rounded hover:bg-accent text-muted-foreground"
          title="Refresh file tree"
        >
          ↻
        </button>
      </div>
      <div className="flex-1 overflow-auto p-1 text-sm">
        {loading ? (
          <div className="text-xs text-muted-foreground p-2">Loading...</div>
        ) : error ? (
          <div className="text-xs text-muted-foreground p-2">
            {error.includes('ENOENT') ? 'No swarm directory found' : error}
          </div>
        ) : entries.length === 0 ? (
          <div className="text-xs text-muted-foreground p-2">No files found in swarm/</div>
        ) : (
          <>
            <div className="px-1 py-0.5 text-xs text-muted-foreground font-medium mb-1">
              swarm/
            </div>
            {entries.map((entry) => (
              <FileTreeItem
                key={entry.path}
                entry={entry}
                depth={1}
                selectedPath={selectedPath}
                onSelect={handleSelect}
              />
            ))}
          </>
        )}
      </div>
    </div>
  )
}
