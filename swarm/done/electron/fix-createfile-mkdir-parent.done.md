# Task: Fix createFile IPC to create parent directories

**Phase:** 4 - Agent Management (bug fix)
**Priority:** Medium (UX)
**Status:** COMPLETED

## Completion Notes

- Added `fs.mkdir(path.dirname(resolved), { recursive: true })` before `fs.writeFile` in the `fs:createfile` IPC handler
- Security validation (`isWithinSwarmDir`) still runs before creating directories
- Build verified with `npm run build` - no errors
- This now matches the behavior of `fs:createdir` which already uses `{ recursive: true }`

---

## Goal

Fix the `fs:createfile` IPC handler to automatically create parent directories when they don't exist. Currently, creating a file in a non-existent directory fails with ENOENT, which breaks the "New Prompt" quick-create button when `swarm/prompts/` doesn't exist yet.

## Files

### Modify
- `electron/src/main/index.ts`

## What to Change

### Fix `fs:createfile` handler (around line 326-337)

**Before:**
```typescript
ipcMain.handle('fs:createfile', async (_event, filePath: string): Promise<{ error?: string }> => {
  try {
    const resolved = path.resolve(filePath)
    if (!isWithinSwarmDir(resolved)) {
      return { error: 'Access denied: path outside swarm/ directory' }
    }
    await fs.writeFile(resolved, '', 'utf-8')
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})
```

**After:**
```typescript
ipcMain.handle('fs:createfile', async (_event, filePath: string): Promise<{ error?: string }> => {
  try {
    const resolved = path.resolve(filePath)
    if (!isWithinSwarmDir(resolved)) {
      return { error: 'Access denied: path outside swarm/ directory' }
    }
    // Ensure parent directory exists
    await fs.mkdir(path.dirname(resolved), { recursive: true })
    await fs.writeFile(resolved, '', 'utf-8')
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})
```

## Dependencies

- None (standalone bug fix)

## Acceptance Criteria

1. Creating a file in a non-existent directory succeeds (parent dirs are auto-created)
2. "New Prompt" quick-create button works even when `swarm/prompts/` doesn't exist
3. Security validation still runs before creating directories
4. Error handling still works for actual errors (permissions, disk full, etc.)
5. App builds with `npm run build`

## Notes

- This matches the behavior of `fs:createdir` which already uses `{ recursive: true }`
- The fix is a single line addition before the `writeFile` call
- `fs.mkdir` with `recursive: true` is idempotent (no error if dir already exists)
