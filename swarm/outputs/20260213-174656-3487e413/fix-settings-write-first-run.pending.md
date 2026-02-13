# Task: Fix settings:write Failure on First Run (Missing Config File)

## Goal

The `settings:write` IPC handler in `electron/src/main/index.ts` fails silently when the config file (`~/.config/swarm/config.toml`) doesn't exist yet. This means users on a fresh install cannot save backend/model settings from the Settings panel because the handler tries to `readFile` first and throws ENOENT.

## Phase

Phase 5: Polish — Bug fix for first-run experience.

## Files to Modify

1. **`electron/src/main/index.ts`** — Update the `settings:write` handler to create the config file with defaults if it doesn't exist

## Dependencies

None

## Implementation Details

### In the `settings:write` handler (around line 432):

Handle the ENOENT case by creating a new config file with sensible defaults before applying updates:

```typescript
ipcMain.handle('settings:write', async (_event, updates: { backend?: string; model?: string }): Promise<{ error?: string }> => {
  try {
    let content: string
    try {
      content = await fs.readFile(configFilePath, 'utf-8')
    } catch (readErr: any) {
      if (readErr.code === 'ENOENT') {
        // Config file doesn't exist yet — create with defaults
        await fs.mkdir(path.dirname(configFilePath), { recursive: true })
        content = `backend = "claude-code"\nmodel = "sonnet"\n`
      } else {
        throw readErr
      }
    }
    if (updates.backend !== undefined) {
      content = content.replace(/^(backend\s*=\s*)"[^"]*"/m, `$1"${updates.backend}"`)
    }
    if (updates.model !== undefined) {
      content = content.replace(/^(model\s*=\s*)"[^"]*"/m, `$1"${updates.model}"`)
    }
    await fs.writeFile(configFilePath, content, 'utf-8')
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})
```

## Acceptance Criteria

1. On a fresh install (no `~/.config/swarm/config.toml`), saving settings from the Settings panel creates the file and persists the chosen backend/model
2. On an existing install, the existing behavior is preserved (read-modify-write)
3. The app builds successfully with `npm run build`
4. No regressions: settings panel read still returns defaults when file is missing
