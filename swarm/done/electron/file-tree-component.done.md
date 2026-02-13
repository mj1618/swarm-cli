# Task: Build Dynamic File Tree Component

**Phase:** 1 - Core Foundation

## Goal

Replace the hardcoded file tree placeholder in the left sidebar with a real, dynamic file tree component that reads the `swarm/` directory from disk. The tree should display files and folders with expand/collapse, appropriate icons, and click-to-select behavior.

## Files

### Create
- `electron/src/renderer/components/FileTree.tsx` — The file tree React component with recursive directory rendering
- `electron/src/renderer/components/FileTreeItem.tsx` — Individual tree node (file or folder) with expand/collapse and icon logic

### Modify
- `electron/src/main/index.ts` — Add IPC handlers: `fs:readDir` (list directory contents), `fs:readFile` (read file content), `fs:watchDir` (watch for changes)
- `electron/src/preload/index.ts` — Expose new `fs` API methods to renderer via contextBridge
- `electron/src/renderer/App.tsx` — Replace the hardcoded file tree placeholder with the new `<FileTree>` component; add state for selected file and file content
- `electron/src/renderer/index.html` — Update CSP if needed (unlikely)

## Dependencies

- None — this is the first real feature task after the scaffold

## Acceptance Criteria

1. The left sidebar shows a real file tree rooted at the workspace's `swarm/` directory
2. Folders can be expanded/collapsed by clicking the folder row or a chevron icon
3. Files show appropriate icons based on extension (`.yaml` → config icon, `.md` → document icon, folders → folder icon)
4. Clicking a file selects it (highlighted state) and could emit a selection event (the center panel integration comes later)
5. The tree updates when files are added/removed on disk (via chokidar or fs.watch in the main process)
6. The IPC channel uses `contextIsolation: true` properly — no `nodeIntegration`
7. The app builds successfully with `npm run build` and runs with `npm run electron:dev`

## Notes

- **From ELECTRON_PLAN.md**: The plan specifies `react-arborist` as the file tree library and `chokidar` for file watching. However, for Phase 1 a simple custom recursive tree is fine — react-arborist can be introduced later if needed for drag-and-drop features in Phase 3.
- The file tree should be scoped to show the `swarm/` subdirectory of the current working directory (or a configurable project root).
- Use Tailwind classes consistent with the existing dark theme CSS variables.
- The IPC `fs:readDir` handler should return an array of `{ name: string, path: string, isDirectory: boolean }` entries, sorted with directories first.
- Consider adding a "Select Workspace" mechanism later, but for now default to `process.cwd()` or a hardcoded path.
- The `swarm/outputs/` subdirectories should show with timestamps as described in the plan.

## Completion Notes

Implemented by agent 4bb8c182. All acceptance criteria met. Both `npm run build` and `npm run build:electron` pass cleanly.

**Additional work by agent 74c9b46a:**
- Added chokidar-based file watching: `fs:watch`/`fs:unwatch` IPC handlers in main process
- Extended preload bridge with `watch`, `unwatch`, `onChanged` methods
- FileTree component now auto-refreshes when files change on disk (AC #5)
- Cleanup on app quit to properly close watcher
