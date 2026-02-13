# Fix: ConsolePanel file watcher leak on effect re-run

## Goal

Fix a resource leak in `ConsolePanel.tsx` where the `useEffect` that sets up the log file watcher has `fetchLogFiles` in its dependency array. Since `fetchLogFiles` is a `useCallback` that may change identity on re-renders, the effect re-runs, calling `window.logs.watch()` multiple times and creating duplicate chokidar watchers in the main process. Only the last cleanup runs `unwatch()`, leaking all previous watchers.

## Files

- **Modify**: `electron/src/renderer/components/ConsolePanel.tsx` — Stabilize the watcher useEffect to run only once

## Dependencies

None — standalone bug fix.

## Acceptance Criteria

1. The watcher `useEffect` (currently at ~line 63) only calls `window.logs.watch()` once during the component lifetime
2. The `fetchLogFiles` dependency is removed from the watcher effect by using a ref
3. File change notifications still trigger log file list refresh correctly
4. Cleanup properly calls `unwatch()` on unmount
5. TypeScript compiles cleanly: `cd electron && npx tsc --noEmit`
6. No functional regressions — console panel still shows logs, tabs work, search works

## Notes

The fix should use a ref to hold the current `fetchLogFiles` callback, breaking the effect dependency:

```typescript
const fetchLogFilesRef = useRef(fetchLogFiles)
fetchLogFilesRef.current = fetchLogFiles

useEffect(() => {
  window.logs.watch()
  const cleanup = window.logs.onChanged(() => {
    fetchLogFilesRef.current()
  })
  cleanupRef.current = cleanup

  return () => {
    if (cleanupRef.current) {
      cleanupRef.current()
    }
    window.logs.unwatch()
  }
}, []) // empty deps — only run once
```

Also remove `fetchLogFiles` from the "Initial load" effect at ~line 51-53 — it should use the ref pattern or just call directly since it only needs to run once on mount.
