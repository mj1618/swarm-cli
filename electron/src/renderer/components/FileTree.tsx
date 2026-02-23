import { useState, useEffect, useCallback, useRef } from 'react'
import FileTreeItem from './FileTreeItem'
import type { DirEntry } from './FileTreeItem'
import ContextMenu from './ContextMenu'
import type { ContextMenuItem } from './ContextMenu'
import type { ToastType } from './ToastContainer'

interface FileTreeProps {
  selectedPath: string | null
  onSelectFile: (filePath: string) => void
  onToast?: (type: ToastType, message: string) => void
}

interface ContextMenuState {
  x: number
  y: number
  entry: DirEntry | null // null = root area right-click
}

export default function FileTree({ selectedPath, onSelectFile, onToast }: FileTreeProps) {
  const [entries, setEntries] = useState<DirEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [swarmRootPath, setSwarmRootPath] = useState<string | null>(null)
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null)
  const [renaming, setRenaming] = useState<string | null>(null)
  const [creating, setCreating] = useState<{ parentPath: string; type: 'file' | 'dir' } | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<DirEntry | null>(null)
  const [filterQuery, setFilterQuery] = useState('')
  const filterInputRef = useRef<HTMLInputElement>(null)
  const [visibleRootChildren, setVisibleRootChildren] = useState<Set<string>>(new Set())
  const [quickCreateMenu, setQuickCreateMenu] = useState<{ x: number; y: number } | null>(null)
  const quickCreateButtonRef = useRef<HTMLButtonElement>(null)

  const toast = useCallback(
    (type: ToastType, message: string) => {
      onToast?.(type, message)
    },
    [onToast],
  )

  const loadRoot = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const root = await window.fs.swarmRoot()
      setSwarmRootPath(root)
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
    } else if (/\/outputs\/\d{8}-\d{6}-[a-f0-9]+\/?$/.test(filePath)) {
      // Allow selecting output run folders to open the summary viewer
      onSelectFile(filePath)
    }
  }, [onSelectFile])

  // Context menu handlers
  const handleContextMenu = useCallback((e: React.MouseEvent, entry: DirEntry) => {
    setContextMenu({ x: e.clientX, y: e.clientY, entry })
  }, [])

  const handleRootContextMenu = useCallback((e: React.MouseEvent) => {
    // Only fire if the click is on the root area, not on a tree item
    if (e.target === e.currentTarget) {
      e.preventDefault()
      setContextMenu({ x: e.clientX, y: e.clientY, entry: null })
    }
  }, [])

  const closeContextMenu = useCallback(() => {
    setContextMenu(null)
  }, [])

  // Actions
  const handleOpen = useCallback((entry: DirEntry) => {
    if (!entry.isDirectory) {
      onSelectFile(entry.path)
    }
  }, [onSelectFile])

  const handleStartRename = useCallback((entry: DirEntry) => {
    setRenaming(entry.path)
  }, [])

  const handleRenameSubmit = useCallback(
    async (oldPath: string, newName: string) => {
      const parts = oldPath.split('/')
      parts[parts.length - 1] = newName
      const newPath = parts.join('/')
      const result = await window.fs.rename(oldPath, newPath)
      if (result.error) {
        toast('error', `Rename failed: ${result.error}`)
      }
      setRenaming(null)
    },
    [toast],
  )

  const handleRenameCancel = useCallback(() => {
    setRenaming(null)
  }, [])

  const handleDelete = useCallback(
    async (entry: DirEntry) => {
      const result = await window.fs.delete(entry.path)
      if (result.error) {
        toast('error', `Delete failed: ${result.error}`)
      }
      setConfirmDelete(null)
    },
    [toast],
  )

  const handleDuplicate = useCallback(
    async (entry: DirEntry) => {
      const result = await window.fs.duplicate(entry.path)
      if (result.error) {
        toast('error', `Duplicate failed: ${result.error}`)
      }
    },
    [toast],
  )

  const handleStartCreate = useCallback(
    (parentPath: string, type: 'file' | 'dir') => {
      setCreating({ parentPath, type })
    },
    [],
  )

  const handleCreateSubmit = useCallback(
    async (parentPath: string, name: string, type: 'file' | 'dir') => {
      const fullPath = `${parentPath}/${name}`
      const result =
        type === 'dir'
          ? await window.fs.createDir(fullPath)
          : await window.fs.createFile(fullPath)
      if (result.error) {
        toast('error', `Create failed: ${result.error}`)
      }
      setCreating(null)
    },
    [toast],
  )

  const handleCreateCancel = useCallback(() => {
    setCreating(null)
  }, [])

  const handleRootChildVisibleChange = useCallback((path: string, visible: boolean) => {
    setVisibleRootChildren((prev) => {
      const next = new Set(prev)
      if (visible) {
        next.add(path)
      } else {
        next.delete(path)
      }
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

  // Build context menu items based on target
  const contextMenuItems: ContextMenuItem[] = []
  if (contextMenu) {
    const { entry } = contextMenu
    if (entry === null) {
      // Right-click on root area
      if (swarmRootPath) {
        contextMenuItems.push({
          label: 'New File',
          action: () => handleStartCreate(swarmRootPath, 'file'),
        })
        contextMenuItems.push({
          label: 'New Folder',
          action: () => handleStartCreate(swarmRootPath, 'dir'),
        })
      }
    } else if (entry.isDirectory) {
      contextMenuItems.push({
        label: 'New File',
        action: () => handleStartCreate(entry.path, 'file'),
      })
      contextMenuItems.push({
        label: 'New Folder',
        action: () => handleStartCreate(entry.path, 'dir'),
      })
      contextMenuItems.push({
        label: 'Rename',
        action: () => handleStartRename(entry),
      })
      contextMenuItems.push({
        label: 'Delete',
        action: () => setConfirmDelete(entry),
        danger: true,
      })
    } else {
      // File
      contextMenuItems.push({
        label: 'Open',
        action: () => handleOpen(entry),
      })
      contextMenuItems.push({
        label: 'Rename',
        action: () => handleStartRename(entry),
      })
      contextMenuItems.push({
        label: 'Duplicate',
        action: () => handleDuplicate(entry),
      })
      contextMenuItems.push({
        label: 'Delete',
        action: () => setConfirmDelete(entry),
        danger: true,
      })
    }
  }

  return (
    <div className="flex flex-col h-full" data-testid="file-tree">
      <div className="p-3 border-b border-border flex items-center justify-between">
        <h2 className="text-sm font-semibold text-foreground">Files</h2>
        <div className="flex items-center gap-1">
          <button
            ref={quickCreateButtonRef}
            onClick={(e) => {
              if (quickCreateMenu) {
                setQuickCreateMenu(null)
              } else {
                const rect = e.currentTarget.getBoundingClientRect()
                setQuickCreateMenu({ x: rect.left, y: rect.bottom + 4 })
              }
            }}
            className="text-xs px-1.5 py-0.5 rounded hover:bg-accent text-muted-foreground"
            title="Create new file or folder"
            aria-haspopup="true"
            aria-expanded={quickCreateMenu !== null}
            data-testid="file-tree-create-button"
          >
            +
          </button>
          <button
            onClick={loadRoot}
            className="text-xs px-1.5 py-0.5 rounded hover:bg-accent text-muted-foreground"
            title="Refresh file tree"
            data-testid="file-tree-refresh-button"
          >
            ↻
          </button>
        </div>
      </div>
      {!loading && !error && entries.length > 0 && (
        <div className="px-3 pb-2 pt-1">
          <div className="relative">
            <input
              ref={filterInputRef}
              type="text"
              value={filterQuery}
              onChange={(e) => setFilterQuery(e.target.value)}
              placeholder="Filter files..."
              className="w-full bg-background border border-border rounded px-2 py-1 text-xs text-foreground outline-none focus:border-blue-500 placeholder:text-muted-foreground/50"
              data-testid="file-tree-filter-input"
            />
            {filterQuery && (
              <button
                onClick={() => {
                  setFilterQuery('')
                  filterInputRef.current?.focus()
                }}
                className="absolute right-1.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground text-xs leading-none"
                title="Clear filter"
              >
                ✕
              </button>
            )}
          </div>
        </div>
      )}
      <div
        className="flex-1 overflow-auto p-1 text-sm"
        onContextMenu={handleRootContextMenu}
      >
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
            {/* Root-level create input */}
            {creating && swarmRootPath && creating.parentPath === swarmRootPath && (
              <div className="flex items-center py-0.5 px-1 text-sm" style={{ paddingLeft: '16px' }}>
                <span className={`w-4 text-center text-xs mr-1 ${creating.type === 'dir' ? 'text-blue-400' : 'text-muted-foreground'}`}>
                  {creating.type === 'dir' ? '▸' : '○'}
                </span>
                <input
                  className="flex-1 bg-background border border-border rounded px-1 py-0 text-sm text-foreground outline-none focus:border-blue-500 min-w-0"
                  autoFocus
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      const val = (e.target as HTMLInputElement).value.trim()
                      if (val) handleCreateSubmit(swarmRootPath, val, creating.type)
                    } else if (e.key === 'Escape') {
                      handleCreateCancel()
                    }
                  }}
                  onBlur={(e) => {
                    const val = e.target.value.trim()
                    if (val) {
                      handleCreateSubmit(swarmRootPath, val, creating.type)
                    } else {
                      handleCreateCancel()
                    }
                  }}
                />
              </div>
            )}
            {entries.map((entry) => (
              <FileTreeItem
                key={entry.path}
                entry={entry}
                depth={1}
                selectedPath={selectedPath}
                onSelect={handleSelect}
                onContextMenu={handleContextMenu}
                renaming={renaming}
                onRenameSubmit={handleRenameSubmit}
                onRenameCancel={handleRenameCancel}
                creating={creating}
                onCreateSubmit={handleCreateSubmit}
                onCreateCancel={handleCreateCancel}
                filterQuery={filterQuery || undefined}
                onVisibleChange={handleRootChildVisibleChange}
              />
            ))}
            {filterQuery && visibleRootChildren.size === 0 && (
              <div className="text-xs text-muted-foreground p-2 italic">
                No files match &apos;{filterQuery}&apos;
              </div>
            )}
          </>
        )}
      </div>

      {/* Context menu */}
      {contextMenu && contextMenuItems.length > 0 && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          items={contextMenuItems}
          onClose={closeContextMenu}
        />
      )}

      {/* Quick-create dropdown menu */}
      {quickCreateMenu && swarmRootPath && (
        <ContextMenu
          x={quickCreateMenu.x}
          y={quickCreateMenu.y}
          items={[
            {
              label: 'New Prompt',
              action: () => {
                handleStartCreate(`${swarmRootPath}/prompts`, 'file')
              },
            },
            {
              label: 'New File',
              action: () => {
                handleStartCreate(swarmRootPath, 'file')
              },
            },
            {
              label: 'New Folder',
              action: () => {
                handleStartCreate(swarmRootPath, 'dir')
              },
            },
          ]}
          onClose={() => setQuickCreateMenu(null)}
        />
      )}

      {/* Delete confirmation dialog */}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" data-testid="file-tree-delete-dialog">
          <div className="bg-card border border-border rounded-lg p-4 shadow-xl max-w-sm mx-4">
            <p className="text-sm text-foreground mb-4">
              Delete <span className="font-semibold">{confirmDelete.name}</span>?
              {confirmDelete.isDirectory && ' This will delete all contents.'}
            </p>
            <div className="flex gap-2 justify-end">
              <button
                className="px-3 py-1.5 text-sm rounded border border-border hover:bg-accent text-foreground"
                onClick={() => setConfirmDelete(null)}
                data-testid="file-tree-delete-cancel"
              >
                Cancel
              </button>
              <button
                className="px-3 py-1.5 text-sm rounded bg-red-600 hover:bg-red-700 text-white"
                onClick={() => handleDelete(confirmDelete)}
                data-testid="file-tree-delete-confirm"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
