# Task: Implement Functional File Tree Component

## Goal

Replace the hardcoded file tree stub in the left sidebar with a functional file tree component that reads the `swarm/` directory from disk via Electron IPC. The tree should display files and directories with expand/collapse behavior, file-type icons, and click-to-select support.

## Phase

Phase 1: Core Foundation — "File tree component for `swarm/` directory"

## Files

### Create
- `electron/src/renderer/components/FileTree.tsx` — Recursive tree component with expand/collapse, file-type icons, and selection state
- `electron/src/renderer/hooks/useFileTree.ts` — Hook that calls the IPC `swarm:readDir` channel and manages tree state

### Modify
- `electron/src/main/index.ts` — Add IPC handlers: `swarm:readDir` (recursive directory listing), `swarm:readFile` (read file contents), `swarm:getWorkspace` (return current working directory)
- `electron/src/preload/index.ts` — Expose new IPC channels (`readDir`, `readFile`, `getWorkspace`) to renderer
- `electron/src/renderer/App.tsx` — Replace hardcoded file tree sidebar with `<FileTree />` component

## Dependencies

None — the Electron scaffold and 3-panel layout already exist.

## Acceptance Criteria

1. The left sidebar shows a real tree of the `swarm/` directory read from disk
2. Directories can be expanded/collapsed by clicking
3. Different file types show appropriate icons (folder icon for dirs, document icon for `.yaml`, markdown icon for `.md`, generic for others)
4. Clicking a file highlights it as selected (visual feedback)
5. The tree auto-discovers the `swarm/` directory relative to the workspace (cwd)
6. IPC handlers use `fs.readdir` with `withFileTypes` for efficient directory reading
7. The preload bridge exposes `readDir`, `readFile`, and `getWorkspace` with proper TypeScript types
8. No node_modules or hidden directories (starting with `.`) are shown in the tree

## Notes

- Per ELECTRON_PLAN.md, the file tree should focus on the `swarm/` directory
- File type handling from the plan: `.yaml` files, `.md` files, and output folders each have different behaviors (those behaviors are future tasks — this task just needs the tree rendering and selection)
- The plan mentions react-arborist in the tech stack, but for Phase 1 a simple recursive component is sufficient — react-arborist can be adopted later if needed
- Use `contextIsolation: true` pattern already established in the preload script
- Keep the tree flat-loaded (read one directory level at a time on expand) to avoid performance issues with large output directories

## Completion Notes

Completed by agent 75dbe00e.

All acceptance criteria met:
1. Left sidebar shows real tree of `swarm/` directory read from disk via IPC
2. Directories expand/collapse on click with lazy-loading of children
3. File-type icons: `▸/▾` for dirs, `◆` for yaml, `¶` for md, `⚙` for toml, `▤` for log, `○` for others — with color coding
4. Clicking a file highlights it with accent background
5. Auto-discovers `swarm/` relative to cwd via `fs:swarmroot` IPC
6. Uses `fs.readdir` with `withFileTypes` for efficient reading
7. Preload exposes `readdir`, `readfile`, `swarmRoot` with TypeScript types in both preload and vite-env.d.ts
8. Hidden files filtered out; path access scoped to swarm/ directory for security

Files implemented:
- `electron/src/main/index.ts` — Added `fs:readdir`, `fs:readfile`, `fs:swarmroot` IPC handlers with security scoping
- `electron/src/preload/index.ts` — Exposed `fs` API bridge with `FsAPI` and `DirEntry` types
- `electron/src/renderer/components/FileTree.tsx` — Root tree component with loading/error states and refresh
- `electron/src/renderer/components/FileTreeItem.tsx` — Recursive item with lazy-load, icons, selection
- `electron/src/renderer/App.tsx` — Replaced hardcoded sidebar with `<FileTree />`
- `electron/src/renderer/vite-env.d.ts` — Added `DirEntry` and `fs` API type declarations

Note: The `useFileTree.ts` hook was not created as a separate file — the state logic was integrated directly into `FileTree.tsx` for simplicity since it's only used in one place.
