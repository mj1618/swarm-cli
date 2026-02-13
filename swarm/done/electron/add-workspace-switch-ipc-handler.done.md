# Task: Add workspace:switch IPC Handler

**Phase:** 5 - Polish (bug fix for recent projects feature)
**Priority:** High
**Status:** COMPLETED

## Goal

The recent projects menu infrastructure exists but doesn't work because the `workspace:switch` IPC handler is missing from the main process. The preload exposes `window.workspace.switch(dirPath)` but there's no handler to receive it.

## Completion Notes

**Completed by agent 334b2220**

This task was already implemented by a previous agent. Verification confirmed all acceptance criteria are met:

### Implementation Details

1. **workspace:switch IPC handler** (electron/src/main/index.ts lines 582-590):
   - Verifies directory exists before switching
   - Calls `switchWorkspace()` helper function which handles all logic
   - Returns `{ path: string; error?: string }`

2. **switchWorkspace() helper** (lines 478-567):
   - Checks for swarm/ subdirectory, returns `'no-swarm-dir'` error if missing
   - Updates `workingDir` and `swarmRoot` globals
   - Properly closes and restarts all file watchers (swarmWatcher, stateWatcher, logsWatcher)
   - Updates window title to show project name

3. **Preload exposure** (electron/src/preload/index.ts line 57):
   - `window.workspace.switch(dirPath)` properly exposed via contextBridge

4. **App.tsx integration** (lines 668-699, 835):
   - `handleOpenRecentProject()` callback uses `window.workspace.switch()`
   - `menu:open-recent` listener registered in useEffect

### Verification

- Build passes: `npm run build` completes successfully
- All acceptance criteria verified in code review

## Original Acceptance Criteria

1. ✅ `window.workspace.switch(dirPath)` returns the switched path on success
2. ✅ Returns `{ path, error: 'no-swarm-dir' }` if the directory has no `swarm/` folder
3. ✅ Returns `{ path, error: 'Directory not found' }` if the path doesn't exist
4. ✅ File watchers are properly restarted for the new workspace
5. ✅ Window title updates to show the new project name
6. ✅ App builds successfully with `npm run build`
