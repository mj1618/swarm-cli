# Fix: Null dereference crash in SettingsPanel on settings read error

## Goal

Fix a critical runtime crash in `SettingsPanel.tsx` where accessing `result.config` properties after a failed `settings:read` IPC call causes a null dereference error. When the settings file doesn't exist or is malformed, the error toast fires but execution continues to access `result.config.backend`, `result.config.model`, etc., which are undefined.

## Files

- `electron/src/renderer/components/SettingsPanel.tsx` — Add early return after error check in the `useEffect` that loads settings (lines 28-41)

## Dependencies

None — this is a standalone bug fix.

## Acceptance Criteria

1. When `window.settings.read()` returns `{ error: "..." }` (i.e., no `.config` property), the component does **not** crash
2. The error toast is shown to the user
3. The loading state is cleared (set to `false`) so the UI doesn't hang on "Loading settings..."
4. The component gracefully shows empty/default values instead of crashing
5. The app still loads settings correctly in the happy path (no regression)

## Notes

The fix is straightforward — add an early return after the error toast on line 31:

```typescript
useEffect(() => {
  window.settings.read().then(result => {
    if (result.error) {
      onToast('error', `Failed to load settings: ${result.error}`)
      setLoading(false)
      return  // <-- ADD THIS: prevent null dereference on result.config
    }
    setBackend(result.config.backend)
    setModel(result.config.model)
    // ... rest unchanged
  })
}, [onToast])
```

This ensures that when settings fail to load, the component shows the loading-complete state with empty defaults rather than crashing. The user sees the error toast and can close/retry.
