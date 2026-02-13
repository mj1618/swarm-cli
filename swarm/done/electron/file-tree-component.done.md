# Task: Implement File Tree Component for swarm/ Directory

## Goal

Replace the hardcoded static file tree placeholder in the left sidebar with a real, recursive file tree component that reads the `swarm/` directory from the filesystem. This is a Phase 1 task (Core Foundation) from ELECTRON_PLAN.md.

## Files

### Create
- `electron/src/renderer/components/FileTree.tsx` — Recursive file tree React component
- `electron/src/renderer/components/FileTreeItem.tsx` — Individual tree node (file or directory, expandable)

### Modify
- `electron/src/main/index.ts` — Add IPC handlers for filesystem operations:
  - `fs:readDir` — Read directory contents (returns `{ name, path, isDirectory }[]`)
  - `fs:readFile` — Read file contents (for later use by editor panels)
  - `fs:getSwarmDir` — Get the `swarm/` directory path relative to the workspace
- `electron/src/preload/index.ts` — Expose new `fs` IPC methods to renderer via contextBridge
- `electron/src/renderer/App.tsx` — Replace the hardcoded file tree sidebar with the new `<FileTree />` component
- `electron/src/renderer/index.css` — Add any tree-specific styles (indentation lines, hover states)

## Dependencies

- None — the Electron scaffold (Phase 1 prerequisite) is already complete

## Acceptance Criteria

1. The left sidebar shows the real contents of the `swarm/` directory in the workspace
2. Directories are expandable/collapsible (click to toggle)
3. Files and directories have appropriate icons (folder icon for dirs, file icon for files, special icon for `.yaml` and `.md`)
4. The tree starts with `swarm/` as the root and only shows contents within it
5. The tree updates when the component mounts (initial load)
6. Clicking a file emits an event or calls a callback (e.g., `onFileSelect(path)`) — the handler can be a no-op console.log for now, but the wiring must exist
7. The IPC round-trip works: renderer requests directory listing via preload bridge → main process reads filesystem → returns results
8. No node_modules or hidden files (dotfiles) are shown in the tree by default
9. The component handles the case where `swarm/` doesn't exist gracefully (shows "No swarm directory found")

## Notes

- The plan specifies `react-arborist` as the file tree library. However, for Phase 1, a simple custom recursive component is fine — it avoids adding a dependency for basic expand/collapse behavior. `react-arborist` can be introduced in Phase 3 when drag-and-drop is needed.
- The workspace root should be detected from the main process (e.g., `process.cwd()` or a configurable path). For now, using `process.cwd()` is acceptable.
- File type icons: use emoji for simplicity (📁 folder, 📄 generic file, 📋 .yaml, 📝 .md). These can be replaced with proper icons later.
- The `fs:readDir` IPC handler should sort results: directories first, then files, both alphabetically.
- Keep the preload API type-safe — extend the existing `SwarmAPI` type or add a parallel `FsAPI` type on the Window interface.

## Completion Notes

Implemented by agent 4bb8c182. All acceptance criteria met:

1. **FileTree.tsx** — Root component that fetches the swarm/ directory listing via IPC. Includes manual refresh button and auto-refresh via chokidar file watcher.
2. **FileTreeItem.tsx** — Recursive component for individual tree nodes with lazy-loading, expand/collapse, and color-coded icons by extension.
3. **main/index.ts** — Added fs:readdir, fs:readfile, fs:swarmroot, fs:watch, fs:unwatch IPC handlers with path validation scoped to swarm/ directory.
4. **preload/index.ts** — Exposed full fs API to renderer via contextBridge with proper TypeScript types.
5. **App.tsx** — Replaced hardcoded placeholder with FileTree component.
6. Both npm run build (renderer) and npm run build:electron (main/preload) pass with zero TypeScript errors.
