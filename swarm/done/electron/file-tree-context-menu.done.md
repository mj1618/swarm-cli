# Task: File Tree Context Menu

## Goal

Add a right-click context menu to the file tree sidebar with operations: Open, Rename, Delete, Duplicate, and New File/New Folder. This is a Phase 1 feature from ELECTRON_PLAN.md ("Right-click context menu: Edit, Rename, Delete, Duplicate") that has not been implemented yet.

## Files

### Create
- `electron/src/renderer/components/ContextMenu.tsx` — Reusable context menu component (positioned at cursor, closes on click-outside/Escape)

### Modify
- `electron/src/renderer/components/FileTreeItem.tsx` — Add `onContextMenu` handler to each tree item, pass context menu state up
- `electron/src/renderer/components/FileTree.tsx` — Manage context menu state (open/close, target file, position), wire up menu actions
- `electron/src/main/index.ts` — Add IPC handlers for `fs:rename`, `fs:delete`, `fs:duplicate`, `fs:createFile`, `fs:createDir` (all scoped to swarm/ directory with path validation)
- `electron/src/preload/index.ts` — Expose new IPC methods on `window.fs`: `rename`, `delete`, `duplicate`, `createFile`, `createDir`

## Dependencies

- File tree component (completed — FileTree.tsx, FileTreeItem.tsx exist)
- File watcher (completed — chokidar watches swarm/ and auto-refreshes tree)

## Acceptance Criteria

1. Right-clicking a **file** in the tree shows a context menu with: Open, Rename, Delete, Duplicate
2. Right-clicking a **directory** in the tree shows: New File, New Folder, Rename, Delete
3. Right-clicking the **root area** (empty space) shows: New File, New Folder
4. **Open** selects the file (same as clicking it)
5. **Rename** shows an inline text input replacing the file name; pressing Enter confirms, Escape cancels. Renames the file on disk via IPC
6. **Delete** shows a confirmation prompt ("Delete {filename}?") before removing the file/directory on disk via IPC
7. **Duplicate** copies the file with a `-copy` suffix (e.g., `planner.md` → `planner-copy.md`) via IPC
8. **New File** / **New Folder** creates an entry with an inline text input for the name; pressing Enter creates it on disk via IPC
9. All IPC handlers validate that paths are within the `swarm/` directory (same security pattern as existing handlers)
10. The file tree auto-refreshes after any mutation (already handled by chokidar watcher)
11. Errors show a toast notification (use existing `onToast` or `addToast` mechanism)
12. The project builds successfully: `cd electron && npx tsc --noEmit` passes with zero errors

## Notes

- Follow the existing path validation pattern in `main/index.ts` (the `isWithinSwarmDir()` helper)
- The context menu should be positioned at the mouse cursor coordinates and close when clicking outside or pressing Escape
- Use the same dark theme styling as the rest of the app (bg-card, border-border, text-foreground classes)
- For inline rename/create inputs, render a small `<input>` in place of the file name text, styled to match the tree
- The duplicate operation for directories is not required (only files) — directories can be complex with nested content
- Keep the IPC handlers simple: use `fs.promises.rename`, `fs.promises.rm`, `fs.promises.copyFile`, `fs.promises.writeFile`, `fs.promises.mkdir`

## Completion Notes

**Status: Completed** — Commit `61e8d13`

All acceptance criteria implemented:
- Created `ContextMenu.tsx` — reusable context menu positioned at cursor, closes on click-outside/Escape, supports danger-styled items
- Updated `FileTreeItem.tsx` — added `onContextMenu` handler, inline `InlineInput` component for rename/create, auto-opens directories when creating inside them
- Updated `FileTree.tsx` — manages context menu state, rename/create state, delete confirmation dialog, and all IPC actions (rename, delete, duplicate, createFile, createDir) with toast error notifications
- IPC handlers in `main/index.ts` and preload bridge in `preload/index.ts` were already added (by prior pipeline work)
- File context menu: Open, Rename, Duplicate, Delete
- Directory context menu: New File, New Folder, Rename, Delete
- Root area context menu: New File, New Folder
- Delete confirmation modal before removing files/directories
- Build passes with zero TypeScript errors
