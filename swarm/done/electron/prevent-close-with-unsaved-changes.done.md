# Task: Prevent Window Close with Unsaved Changes

## Goal

Add a confirmation dialog when the user attempts to close the app window while there are unsaved changes in the Monaco file editor. This prevents accidental data loss.

## Files

- `electron/src/main/index.ts` - Add IPC handler for dirty state and `close` event handling
- `electron/src/preload/index.ts` - Expose IPC methods for dirty state communication
- `electron/src/renderer/App.tsx` - Track global dirty state and communicate with main process
- `electron/src/renderer/components/MonacoFileEditor.tsx` - Report dirty state changes

## Dependencies

None - all prerequisite features are complete.

## Acceptance Criteria

1. When user has unsaved changes in the file editor and tries to close the window:
   - A native dialog appears: "You have unsaved changes. Are you sure you want to close?"
   - Options: "Don't Save" (closes anyway), "Cancel" (stays open), "Save" (saves then closes)
2. When there are no unsaved changes, window closes immediately
3. The dirty state tracking works across file switches
4. Works with both Cmd+Q and clicking the window close button

## Implementation Notes

From Electron best practices:
- Use `win.on('close', (e) => { e.preventDefault(); ... })` in main process
- Use `dialog.showMessageBoxSync()` for the confirmation
- Use IPC to query renderer for dirty state
- Consider using `app.on('before-quit')` for Cmd+Q handling

Example flow:
1. MonacoFileEditor notifies App when dirty state changes via callback
2. App tracks which files have unsaved changes
3. App exposes this via IPC: `window.editor.hasDirtyFiles()`
4. Main process queries this on close and shows dialog if needed

## Phase

Phase 5: Polish (enhancement to existing functionality)

---

## Completion Notes

Implemented by agent defee9f0 on 2026-02-13.

### Changes Made

1. **Main process (`electron/src/main/index.ts`)**:
   - Added `hasDirtyFiles` and `isQuitting` state variables
   - Modified the window `close` event handler to check dirty state and show a native confirmation dialog with three options: "Don't Save", "Cancel", "Save"
   - Updated `app.on('before-quit')` to handle Cmd+Q by delegating to the close handler
   - Added IPC handlers `editor:dirty-state` and `editor:save-complete` for renderer communication

2. **Preload (`electron/src/preload/index.ts`)**:
   - Added `editor` API exposing:
     - `setDirtyState(dirty: boolean)` - renderer reports dirty state
     - `notifySaveComplete()` - renderer signals save is complete
     - `onSaveAndClose(callback)` - listen for save-and-close requests
   - Added TypeScript types for the new `EditorAPI`

3. **MonacoFileEditor (`electron/src/renderer/components/MonacoFileEditor.tsx`)**:
   - Added `onDirtyChange` callback prop to report dirty state changes
   - Added `triggerSave` prop to trigger saves from parent
   - Added `onSaveComplete` callback prop to notify when save completes
   - Reports dirty state on change and cleans up on unmount

4. **App.tsx (`electron/src/renderer/App.tsx`)**:
   - Added `dirtyFiles` state to track which files have unsaved changes
   - Added effect to report dirty state to main process via `window.editor.setDirtyState()`
   - Added listener for `editor:save-and-close` to trigger saves and notify completion
   - Passed dirty state callbacks to MonacoFileEditor

### Flow

1. MonacoFileEditor tracks content changes and calls `onDirtyChange(filePath, isDirty)`
2. App.tsx aggregates dirty files and reports to main via `window.editor.setDirtyState()`
3. On close attempt with dirty files, main shows dialog:
   - Cancel: Window stays open
   - Don't Save: Window closes immediately
   - Save: Main sends `editor:save-and-close` → renderer saves → renderer sends `editor:save-complete` → window closes
