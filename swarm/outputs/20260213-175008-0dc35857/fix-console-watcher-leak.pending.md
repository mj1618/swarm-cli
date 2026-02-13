# Fix: ConsolePanel file watcher leak on effect re-run

## Goal

Fix a resource leak in `ConsolePanel.tsx` where the `useEffect` that sets up the log file watcher depends on `fetchLogFiles`, causing it to re-run when that callback changes. Each re-run calls `window.logs.watch()` which creates a new chokidar watcher in the main process, but only one `unwatch()` is called on cleanup, leaking file watchers.

## Files

- `electron/src/renderer/components/ConsolePanel.tsx` — Stabilize the watcher useEffect to prevent multiple `watch()` calls

## Dependencies

None — this is a standalone bug fix.

## Acceptance Criteria

1. The watcher `useEffect` only calls `window.logs.watch()` once during the component lifetime
2. The `fetchLogFiles` dependency is removed from the watcher effect (use a ref instead)
3. File change notifications still trigger log file list refresh
4. Cleanup properly calls `unwatch()` on unmount
5. No leaked chokidar watchers in the main process

## Notes

The fix should separate the watcher setup from the fetch callback. Use a ref for the fetch function:

```typescript
const fetchLogFilesRef = useRef(fetchLogFiles)
fetchLogFilesRef.current = fetchLogFiles

useEffect(() => {
  window.logs.watch()
  const cleanup = window.logs.onChanged(() => {
    fetchLogFilesRef.current()
  })
  return () => {
    cleanup()
    window.logs.unwatch()
  }
}, []) // empty deps - only run once
```
