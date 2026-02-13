# Task: Fix null dereference in SettingsPanel when settings fail to load

## Goal

Fix a critical runtime error in `SettingsPanel.tsx` where `result.config` is accessed even when `window.settings.read()` returns an error, causing a null/undefined property access crash.

## Files

- `electron/src/renderer/components/SettingsPanel.tsx` — Add early return after error check in the settings loading effect

## Dependencies

None

## Acceptance Criteria

1. When `window.settings.read()` returns an error, the component shows the error toast and does NOT attempt to access `result.config`
2. Loading state is still set to false after an error so the component doesn't show a permanent loading spinner
3. Normal (non-error) settings loading continues to work as before

## Bug Details

In `SettingsPanel.tsx` lines 28-41, the `useEffect` calls `window.settings.read()` and checks for `result.error`, but then proceeds to access `result.config.backend`, `result.config.model`, etc. without an `else` or early `return`. When an error occurs, `result.config` is undefined and accessing its properties throws a runtime TypeError.

Fix: Add `return` after the error toast call, but still set `setLoading(false)` before returning.
