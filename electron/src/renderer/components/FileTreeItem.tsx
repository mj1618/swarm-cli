import { useState, useCallback } from 'react'

interface DirEntry {
  name: string
  path: string
  isDirectory: boolean
}

interface FileTreeItemProps {
  entry: DirEntry
  depth: number
  selectedPath: string | null
  onSelect: (path: string) => void
}

function getFileIcon(name: string, isDirectory: boolean, isOpen: boolean): string {
  if (isDirectory) {
    return isOpen ? '▾' : '▸'
  }
  const ext = name.split('.').pop()?.toLowerCase()
  switch (ext) {
    case 'yaml':
    case 'yml':
      return '◆'
    case 'md':
      return '¶'
    case 'toml':
      return '⚙'
    case 'log':
      return '▤'
    default:
      return '○'
  }
}

function getFileIconColor(name: string, isDirectory: boolean): string {
  if (isDirectory) return 'text-blue-400'
  const ext = name.split('.').pop()?.toLowerCase()
  switch (ext) {
    case 'yaml':
    case 'yml':
      return 'text-yellow-400'
    case 'md':
      return 'text-green-400'
    case 'toml':
      return 'text-orange-400'
    case 'log':
      return 'text-gray-400'
    default:
      return 'text-muted-foreground'
  }
}

export default function FileTreeItem({ entry, depth, selectedPath, onSelect }: FileTreeItemProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [children, setChildren] = useState<DirEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const isSelected = selectedPath === entry.path

  const loadChildren = useCallback(async () => {
    if (!entry.isDirectory) return
    setLoading(true)
    setError(null)
    try {
      const result = await window.fs.readdir(entry.path)
      if (result.error) {
        setError(result.error)
        setChildren([])
      } else {
        setChildren(result.entries)
      }
    } catch {
      setError('Failed to read directory')
      setChildren([])
    } finally {
      setLoading(false)
    }
  }, [entry.path, entry.isDirectory])

  const handleClick = useCallback(() => {
    if (entry.isDirectory) {
      const willOpen = !isOpen
      setIsOpen(willOpen)
      if (willOpen && children.length === 0) {
        loadChildren()
      }
    }
    onSelect(entry.path)
  }, [entry.isDirectory, entry.path, isOpen, children.length, loadChildren, onSelect])

  const icon = getFileIcon(entry.name, entry.isDirectory, isOpen)
  const iconColor = getFileIconColor(entry.name, entry.isDirectory)

  return (
    <div>
      <div
        className={`flex items-center py-0.5 px-1 rounded cursor-pointer text-sm select-none ${
          isSelected
            ? 'bg-accent text-accent-foreground'
            : 'hover:bg-accent/50 text-muted-foreground'
        }`}
        style={{ paddingLeft: `${depth * 12 + 4}px` }}
        onClick={handleClick}
      >
        <span className={`w-4 text-center text-xs mr-1 ${iconColor}`}>{icon}</span>
        <span className="truncate">{entry.name}</span>
      </div>
      {entry.isDirectory && isOpen && (
        <div>
          {loading && (
            <div
              className="text-xs text-muted-foreground py-0.5"
              style={{ paddingLeft: `${(depth + 1) * 12 + 4}px` }}
            >
              Loading...
            </div>
          )}
          {error && (
            <div
              className="text-xs text-red-400 py-0.5"
              style={{ paddingLeft: `${(depth + 1) * 12 + 4}px` }}
            >
              {error}
            </div>
          )}
          {children.map((child) => (
            <FileTreeItem
              key={child.path}
              entry={child}
              depth={depth + 1}
              selectedPath={selectedPath}
              onSelect={onSelect}
            />
          ))}
        </div>
      )}
    </div>
  )
}
