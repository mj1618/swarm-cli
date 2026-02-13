# Task: Add menu:open-recent Listener to App.tsx

**Phase:** 5 - Polish (bug fix)
**Priority:** Medium
**Status:** COMPLETED

## Goal

The recent projects menu infrastructure is in place (main process sends `menu:open-recent` events with the project path), but App.tsx is missing the listener. When a user clicks a recent project from File > Recent Projects, nothing happens.

## Completion Note

This task was already implemented. The following code is present in `App.tsx`:

1. **Listener** (line 835): `window.electronMenu.on('menu:open-recent', (path: string) => handleOpenRecentProject(path))`

2. **Handler** (lines 668-699): `handleOpenRecentProject` callback that:
   - Calls `window.workspace.switch(recentPath)` to change directories
   - Handles "Directory not found" error with error toast
   - Updates `projectPath` state and localStorage
   - Adds to recents (moves to top)
   - Resets UI state (selectedFile, selectedTask, selectedPipeline)
   - Reloads swarm.yaml for the new workspace
   - Shows warning toast if no swarm/swarm.yaml found
   - Shows success toast on successful switch

3. **Dependencies array** includes `handleOpenRecentProject`

All acceptance criteria are met:
- ✅ Clicking a recent project in File > Recent Projects opens it and switches the workspace
- ✅ The DAG canvas reloads with the new project's swarm.yaml
- ✅ A success toast is shown: "Switched to {path}"
- ✅ If the directory has no swarm/ folder, a warning toast is shown
- ✅ If the path doesn't exist, an error toast is shown
- ✅ App builds successfully with `npm run build`
