import { useState, useCallback, useRef, useEffect } from 'react'

export interface DirEntry {
  name: string
  path: string
  isDirectory: boolean
}

interface FileTreeItemProps {
  entry: DirEntry
  depth: number
  selectedPath: string | null
  onSelect: (path: string, isDirectory: boolean) => void
  onContextMenu: (e: React.MouseEvent, entry: DirEntry) => void
  renaming: string | null
  onRenameSubmit: (oldPath: string, newName: string) => void
  onRenameCancel: () => void
  creating: { parentPath: string; type: 'file' | 'dir' } | null
  onCreateSubmit: (parentPath: string, name: string, type: 'file' | 'dir') => void
  onCreateCancel: () => void
  filterQuery?: string
  onVisibleChange?: (path: string, visible: boolean) => void
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

function HighlightedName({ name, query }: { name: string; query: string }) {
  if (!query) return <>{name}</>
  const lower = name.toLowerCase()
  const qLower = query.toLowerCase()
  const idx = lower.indexOf(qLower)
  if (idx === -1) return <>{name}</>
  return (
    <>
      {name.slice(0, idx)}
      <span className="text-blue-400 font-semibold">{name.slice(idx, idx + query.length)}</span>
      {name.slice(idx + query.length)}
    </>
  )
}

function InlineInput({
  defaultValue,
  onSubmit,
  onCancel,
  depth,
  icon,
  iconColor,
}: {
  defaultValue: string
  onSubmit: (value: string) => void
  onCancel: () => void
  depth: number
  icon: string
  iconColor: string
}) {
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    inputRef.current?.focus()
    inputRef.current?.select()
  }, [])

  return (
    <div
      className="flex items-center py-0.5 px-1 text-sm"
      style={{ paddingLeft: `${depth * 12 + 4}px` }}
    >
      <span className={`w-4 text-center text-xs mr-1 ${iconColor}`}>{icon}</span>
      <input
        ref={inputRef}
        className="flex-1 bg-background border border-border rounded px-1 py-0 text-sm text-foreground outline-none focus:border-blue-500 min-w-0"
        defaultValue={defaultValue}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            const val = (e.target as HTMLInputElement).value.trim()
            if (val) onSubmit(val)
          } else if (e.key === 'Escape') {
            onCancel()
          }
        }}
        onBlur={(e) => {
          const val = e.target.value.trim()
          if (val && val !== defaultValue) {
            onSubmit(val)
          } else {
            onCancel()
          }
        }}
      />
    </div>
  )
}

export default function FileTreeItem({
  entry,
  depth,
  selectedPath,
  onSelect,
  onContextMenu,
  renaming,
  onRenameSubmit,
  onRenameCancel,
  creating,
  onCreateSubmit,
  onCreateCancel,
  filterQuery,
  onVisibleChange,
}: FileTreeItemProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [children, setChildren] = useState<DirEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [visibleChildren, setVisibleChildren] = useState<Set<string>>(new Set())
  const [isDragging, setIsDragging] = useState(false)

  const isDraggable = !entry.isDirectory && entry.path.includes('/prompts/') && entry.name.endsWith('.md')

  const isSelected = selectedPath === entry.path
  const isRenaming = renaming === entry.path
  const isCreatingHere = creating && creating.parentPath === entry.path
  const isFiltering = !!filterQuery
  const nameMatches = isFiltering
    ? entry.name.toLowerCase().includes(filterQuery!.toLowerCase())
    : false
  const preFilterOpenRef = useRef(false)

  // Determine visibility
  const isVisible = !isFiltering
    || nameMatches
    || (entry.isDirectory && (loading || visibleChildren.size > 0))

  // Report visibility to parent
  useEffect(() => {
    onVisibleChange?.(entry.path, isVisible)
  }, [isVisible, entry.path, onVisibleChange])

  const handleChildVisibleChange = useCallback((path: string, visible: boolean) => {
    setVisibleChildren((prev) => {
      const next = new Set(prev)
      if (visible) {
        next.add(path)
      } else {
        next.delete(path)
      }
      // Only update if actually changed
      if (next.size === prev.size) {
        let same = true
        for (const p of next) {
          if (!prev.has(p)) { same = false; break }
        }
        if (same) return prev
      }
      return next
    })
  }, [])

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

  // Auto-expand directories when filter becomes active; restore when cleared
  useEffect(() => {
    if (!entry.isDirectory) return
    if (isFiltering) {
      preFilterOpenRef.current = isOpen
      if (!isOpen) {
        setIsOpen(true)
        if (children.length === 0) {
          loadChildren()
        }
      }
    } else {
      setIsOpen(preFilterOpenRef.current)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isFiltering])

  // Auto-open directory when a create action targets it
  useEffect(() => {
    if (isCreatingHere && entry.isDirectory && !isOpen) {
      setIsOpen(true)
      if (children.length === 0) {
        loadChildren()
      }
    }
  }, [isCreatingHere, entry.isDirectory, isOpen, children.length, loadChildren])

  const handleClick = useCallback(() => {
    if (entry.isDirectory) {
      const willOpen = !isOpen
      setIsOpen(willOpen)
      if (willOpen && children.length === 0) {
        loadChildren()
      }
    }
    onSelect(entry.path, entry.isDirectory)
  }, [entry.isDirectory, entry.path, isOpen, children.length, loadChildren, onSelect])

  const handleContextMenu = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault()
      e.stopPropagation()
      onContextMenu(e, entry)
    },
    [onContextMenu, entry],
  )

  const icon = getFileIcon(entry.name, entry.isDirectory, isOpen)
  const iconColor = getFileIconColor(entry.name, entry.isDirectory)

  if (!isVisible) return null

  return (
    <div>
      {isRenaming ? (
        <InlineInput
          defaultValue={entry.name}
          onSubmit={(newName) => onRenameSubmit(entry.path, newName)}
          onCancel={onRenameCancel}
          depth={depth}
          icon={icon}
          iconColor={iconColor}
        />
      ) : (
        <div
          className={`flex items-center py-0.5 px-1 rounded cursor-pointer text-sm select-none ${
            isSelected
              ? 'bg-accent text-accent-foreground'
              : 'hover:bg-accent/50 text-muted-foreground'
          }`}
          style={{ paddingLeft: `${depth * 12 + 4}px`, opacity: isDragging ? 0.5 : undefined }}
          onClick={handleClick}
          onContextMenu={handleContextMenu}
          draggable={isDraggable}
          onDragStart={isDraggable ? (e) => {
            const promptName = entry.name.replace(/\.md$/, '')
            e.dataTransfer.setData('application/swarm-prompt', promptName)
            e.dataTransfer.setData('text/plain', promptName)
            e.dataTransfer.effectAllowed = 'copy'
            setIsDragging(true)
          } : undefined}
          onDragEnd={isDraggable ? () => setIsDragging(false) : undefined}
        >
          <span className={`w-4 text-center text-xs mr-1 ${iconColor}`}>{icon}</span>
          <span className="truncate">
            <HighlightedName name={entry.name} query={filterQuery || ''} />
          </span>
        </div>
      )}
      {entry.isDirectory && isOpen && (
        <div>
          {isCreatingHere && (
            <InlineInput
              defaultValue=""
              onSubmit={(name) => onCreateSubmit(entry.path, name, creating.type)}
              onCancel={onCreateCancel}
              depth={depth + 1}
              icon={creating.type === 'dir' ? '▸' : '○'}
              iconColor={creating.type === 'dir' ? 'text-blue-400' : 'text-muted-foreground'}
            />
          )}
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
              onContextMenu={onContextMenu}
              renaming={renaming}
              onRenameSubmit={onRenameSubmit}
              onRenameCancel={onRenameCancel}
              creating={creating}
              onCreateSubmit={onCreateSubmit}
              onCreateCancel={onCreateCancel}
              filterQuery={filterQuery}
              onVisibleChange={handleChildVisibleChange}
            />
          ))}
        </div>
      )}
    </div>
  )
}
