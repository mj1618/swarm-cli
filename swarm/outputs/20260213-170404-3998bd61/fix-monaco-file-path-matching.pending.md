# Task: Fix Monaco Editor File Change Detection Path Matching

## Goal

The MonacoFileEditor's external file change watcher matches files by filename only, not by full path. This can cause unnecessary reloads when a different file with the same name changes in another directory.

## Phase

Phase 5: Polish / Bug Fix

## Files to Modify

1. **`electron/src/renderer/components/MonacoFileEditor.tsx`** — Fix the `onChanged` callback to compare full paths instead of just filenames

## Implementation Details

### Current code (lines 128-141):

```tsx
const unsubscribe = window.fs.onChanged((data) => {
  const changedName = data.path.split('/').pop()
  const currentName = filePath.split('/').pop()
  if (changedName && currentName && changedName === currentName) {
```

The `data.path` from the `fs:changed` event is a full absolute path emitted by chokidar (see `electron/src/main/index.ts:185`). The `filePath` prop is also a full path. Comparing just the filename (`.split('/').pop()`) can cause false matches.

### Fix:

Compare the full paths directly, or normalize both paths before comparison:

```tsx
const unsubscribe = window.fs.onChanged((data) => {
  if (data.path === filePath) {
    if (!isDirtyRef.current) {
      window.fs.readfile(filePath).then((result) => {
        if (!result.error) {
          setContent(result.content)
          setSavedContent(result.content)
        }
      })
    }
  }
})
```

If the paths might differ in format (e.g., relative vs absolute), use `data.path.endsWith('/' + filePath)` or normalize both. But given both are full paths from the same system, direct comparison should work.

## Acceptance Criteria

1. External file changes only trigger a reload when the exact same file path changes, not just a file with the same name
2. No regressions — file reload still works correctly when the actual file changes
3. TypeScript compiles without errors
