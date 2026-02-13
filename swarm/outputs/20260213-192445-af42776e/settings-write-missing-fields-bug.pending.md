# Bug: settings:write fails silently when config file lacks expected fields

**Phase:** 5 - Polish (bug fix)
**Priority:** Low
**Discovered by:** Agent 69cc7de2

## Problem

The `settings:write` IPC handler in `electron/src/main/index.ts` uses regex replacement to update the `backend` and `model` fields in the TOML config file. However, if the config file exists but doesn't contain these fields, the regex won't match and the update silently fails.

## Files

### Modify
- `electron/src/main/index.ts` (lines 662-686)

## Current Behavior

```typescript
if (updates.backend !== undefined) {
  content = content.replace(/^(backend\s*=\s*)"[^"]*"/m, `$1"${updates.backend}"`)
}
```

If the config file contains:
```toml
some_other_setting = "value"
```

And you try to set `backend = "cursor"`, the regex won't match, so the file stays unchanged. No error is returned.

## Expected Behavior

If the field doesn't exist, it should be added to the file. Alternatively, the handler should return an error or warning.

## Suggested Fix

```typescript
ipcMain.handle('settings:write', async (_event, updates: { backend?: string; model?: string }): Promise<{ error?: string }> => {
  try {
    let content: string
    try {
      content = await fs.readFile(getConfigFilePath(), 'utf-8')
    } catch (readErr: any) {
      if (readErr.code === 'ENOENT') {
        content = ''
      } else {
        throw readErr
      }
    }
    
    if (updates.backend !== undefined) {
      if (/^backend\s*=/m.test(content)) {
        content = content.replace(/^(backend\s*=\s*)"[^"]*"/m, `$1"${updates.backend}"`)
      } else {
        content = `backend = "${updates.backend}"\n` + content
      }
    }
    
    if (updates.model !== undefined) {
      if (/^model\s*=/m.test(content)) {
        content = content.replace(/^(model\s*=\s*)"[^"]*"/m, `$1"${updates.model}"`)
      } else {
        content = `model = "${updates.model}"\n` + content
      }
    }
    
    await fs.writeFile(getConfigFilePath(), content, 'utf-8')
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})
```

## Acceptance Criteria

1. Setting `backend` when it's missing from the config file adds it
2. Setting `model` when it's missing from the config file adds it
3. Existing fields are still updated correctly (no regression)
4. App builds with `npm run build`

## Notes

- This is a low-priority bug because the default config template includes both fields
- However, it could cause confusion if users manually edit the config file
