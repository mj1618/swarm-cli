# Task: Auto-Reload YAML Content on External File Changes

## Goal

When `swarm.yaml` (or any selected YAML file) is modified externally — by an agent, a text editor, or another process — the DAG canvas should automatically reload and re-render the updated content. Currently, the FileTree and MonacoFileEditor both listen for `fs:changed` events and auto-refresh, but the DAG canvas in `App.tsx` only loads YAML content on initial mount or manual file selection. This creates a stale DAG view during active pipeline execution.

## Phase

Phase 5: Polish — Completes the real-time reactivity story alongside the existing state watcher and log watcher.

## Files to Modify

1. **`electron/src/renderer/App.tsx`** — Subscribe to `window.fs.onChanged` events and reload the active YAML content when the relevant file changes

## Dependencies

- File watcher infrastructure already exists (`fs:watch`, `fs:onChanged` in main process + preload)
- FileTree already calls `window.fs.watch()` on mount, so the watcher is active

## Implementation Details

### App.tsx: Add a `useEffect` that listens for `fs:changed` events

Add a new `useEffect` near the existing YAML loading effects (around lines 430-477) that:

1. Subscribes to `window.fs.onChanged((data) => { ... })`
2. When the changed file path matches the active YAML path (either `swarm/swarm.yaml` for the default view, or the explicitly selected YAML file), re-reads the file content and updates the corresponding state
3. Uses path matching logic similar to `MonacoFileEditor.tsx` — compare `data.path` against the active YAML file path (handle both full and relative paths)

```typescript
// Auto-reload YAML when externally modified
useEffect(() => {
  const unsubscribe = window.fs.onChanged((data) => {
    const activePath = selectedIsYaml && selectedFile ? selectedFile : 'swarm/swarm.yaml'
    if (data.path === activePath || data.path.endsWith('/' + activePath) || activePath.endsWith('/' + data.path)) {
      window.fs.readfile(activePath).then((result) => {
        if (result.error) return
        if (selectedIsYaml && selectedFile) {
          setSelectedYamlContent(result.content)
        } else {
          setDefaultYamlContent(result.content)
        }
      })
    }
  })
  return () => { unsubscribe() }
}, [selectedFile, selectedIsYaml])
```

### Important: Avoid reload loops

When the user saves changes from within the app (e.g., task drawer save, dependency creation, pipeline edits), the app already writes to the YAML file and then manually re-reads it. The external change listener should NOT conflict with this — it will simply trigger another read that produces the same content, causing no visible effect (React state won't re-render if the content string is identical). This is safe and idempotent.

## Acceptance Criteria

1. When `swarm/swarm.yaml` is modified by an external editor while viewing the DAG canvas, the DAG automatically updates to reflect the changes within ~1 second
2. When a specifically selected YAML file is modified externally, the DAG updates accordingly
3. No reload loops or flickering when the app itself writes changes to the YAML file
4. The app builds successfully with `npm run build`
5. No regressions: manual YAML editing, task creation, dependency creation all continue to work

## Notes

- The `fs:changed` event is already being emitted by chokidar in the main process for the entire `swarm/` directory
- The FileTree component already starts the watcher via `window.fs.watch()`, so no additional watcher initialization is needed
- This is a small, focused change: a single `useEffect` hook in `App.tsx`
- This completes the "reactive" story: agent state auto-updates, log files auto-stream, file tree auto-refreshes, and now the DAG canvas auto-reloads

## Completion Notes

Implemented by agent efa0d97e. Added a single `useEffect` hook in `App.tsx` (after the existing YAML loading effects) that subscribes to `window.fs.onChanged` events. When the changed file path matches the active YAML path, it re-reads the file and updates the corresponding state (`setSelectedYamlContent` or `setDefaultYamlContent`). Path matching handles exact, suffix, and prefix comparisons. The implementation is safe and idempotent — re-reads triggered by the app's own writes produce identical content, so React won't re-render. Build verified successfully.
