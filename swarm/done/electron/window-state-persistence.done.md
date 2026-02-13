# Window State Persistence

## Goal

Persist and restore the main window's size, position, and maximized state across app restarts. This is a standard Electron UX improvement that makes the app feel more native.

## Files

- `electron/src/main/index.ts` - Add window state loading/saving logic

## Dependencies

None - this is a standalone enhancement.

## Acceptance Criteria

1. When the app closes, save the current window bounds (x, y, width, height) and maximized state
2. When the app launches, restore the saved window state instead of using fixed defaults
3. Handle edge cases:
   - First launch (no saved state) uses sensible defaults (1400x900, centered)
   - Saved position is off-screen (display was removed) - reset to centered
   - Validate saved dimensions are reasonable (within min/max constraints)
4. State is stored in `app.getPath('userData')` as `window-state.json`

## Notes

Implementation pattern:
1. Create `loadWindowState()` function that reads from `window-state.json`
2. Create `saveWindowState()` function that writes current bounds
3. Call `saveWindowState()` on `mainWindow.on('close')` event (before window is destroyed)
4. In `createWindow()`, use loaded state for initial dimensions or fall back to defaults
5. Handle `mainWindow.isMaximized()` to restore maximized state after showing

Example state structure:
```json
{
  "x": 100,
  "y": 100,
  "width": 1400,
  "height": 900,
  "isMaximized": false
}
```

Use `screen.getPrimaryDisplay().workAreaSize` to validate bounds are reasonable.

---

## Completion Notes

**Implemented by:** Agent 08fe5c7f
**Date:** 2026-02-13

### Implementation Summary

Added window state persistence to `electron/src/main/index.ts`:

1. **Added `screen` import** from Electron for display bounds checking
2. **Created `WindowState` interface** with x, y, width, height, and isMaximized fields
3. **Implemented `loadWindowState()`** function that:
   - Reads from `window-state.json` in userData directory
   - Validates required fields exist and are correct types
   - Enforces minimum dimensions (1000x600)
   - Checks if saved position is visible on any connected display
   - Falls back to centered window if position is off-screen
4. **Implemented `saveWindowState()`** function that:
   - Skips saving if window is minimized
   - Uses `getNormalBounds()` when maximized to preserve pre-maximized size
   - Saves x, y, width, height, and isMaximized state
5. **Modified `createWindow()`** to be async and:
   - Load saved window state on startup
   - Apply saved position/size or center on screen
   - Restore maximized state after window creation
   - Save state on window close event

All acceptance criteria met. Build verified successful.
