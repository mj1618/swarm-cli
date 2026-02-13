# Task: Fix createFile to handle missing parent directories

## Goal

When creating a new file via the File Tree, if the parent directory doesn't exist, the operation fails with "ENOENT: no such file or directory". This affects the "New Prompt" quick-create button when `swarm/prompts/` doesn't exist yet.

## Problem

In `electron/src/main/index.ts`, the `fs:createfile` handler uses:

```typescript
await fs.writeFile(resolved, '', 'utf-8')
```

This fails if the parent directory doesn't exist. In contrast, `fs:createdir` correctly uses `{ recursive: true }`.

## Files

- `electron/src/main/index.ts` - Fix the `fs:createfile` IPC handler

## Dependencies

None

## Acceptance Criteria

1. Creating a file in a non-existent directory automatically creates the parent directories
2. "New Prompt" works even when `swarm/prompts/` doesn't exist
3. Error handling still works for actual errors (permissions, etc.)

## Suggested Fix

Before writing the file, ensure the parent directory exists:

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

## Notes

This is a minor UX issue but will cause confusion when users try to create prompts in a fresh project.
