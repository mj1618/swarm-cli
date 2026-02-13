# Task: Add menu:open-recent Listener to App.tsx

**Phase:** 5 - Polish (bug fix)
**Priority:** Medium

## Goal

The recent projects menu infrastructure is in place, but App.tsx is missing the listener for `menu:open-recent` events. When a user clicks a recent project from the menu, nothing happens.

## Files to Modify

### electron/src/renderer/App.tsx

Add a listener for `menu:open-recent` in the useEffect that handles menu events (around line 800, in the `cleanups` array):

```typescript
window.electronMenu.on('menu:open-recent', async (projectPath: string) => {
  if (!projectPath) return
  const result = await window.workspace.switch(projectPath)
  if (result.error === 'no-swarm-dir') {
    addToast('warning', `No swarm/ directory found in ${result.path}`)
    setProjectPath(result.path)
    localStorage.setItem('swarm-project-path', result.path)
    return
  }
  if (result.error) {
    addToast('error', `Failed to open project: ${result.error}`)
    return
  }
  setProjectPath(result.path)
  localStorage.setItem('swarm-project-path', result.path)
  setSelectedFile(null)
  setSelectedTask(null)
  setSelectedPipeline(null)
  const reloaded = await window.fs.readfile('swarm/swarm.yaml')
  if (reloaded.error) {
    setDefaultYamlError(reloaded.error)
    setDefaultYamlContent(null)
  } else {
    setDefaultYamlContent(reloaded.content)
    setDefaultYamlError(null)
  }
  addToast('success', `Switched to ${result.path}`)
}),
```

## Acceptance Criteria

1. Clicking a recent project in the menu opens it and switches the workspace
2. A success toast is shown when switching projects
3. Appropriate error handling for invalid/missing directories
4. App builds successfully with `npm run build`

## Notes

- The `workspace.switch` API and `menu:open-recent` channel are already set up
- This task just adds the missing event listener in the renderer
