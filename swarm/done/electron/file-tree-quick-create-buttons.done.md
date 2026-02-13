# Task: Add Quick-Create Buttons to File Tree Header

## Goal

Add dedicated quick-create buttons to the File Tree panel header for rapidly creating new prompt files and folders. Currently, users must right-click to access the context menu for file creation. The ELECTRON_PLAN.md specifies "Quick-create buttons for new prompts, tasks" as a feature.

## Files

- `electron/src/renderer/components/FileTree.tsx` - Add quick-create buttons to the header

## Dependencies

None - the file creation functionality already exists via context menu handlers (`handleStartCreate`).

## Acceptance Criteria

1. The File Tree header displays "+" button(s) next to the refresh button
2. Clicking the button shows a small dropdown/menu with options:
   - "New Prompt" - creates a new `.md` file in `swarm/prompts/`
   - "New File" - creates a new file in `swarm/`
   - "New Folder" - creates a new folder in `swarm/`
3. Selecting an option triggers the inline rename input at the appropriate location
4. The buttons match the existing UI style (small, muted, hover state)
5. Keyboard accessibility: buttons are focusable and activatable with Enter/Space

## Notes

From ELECTRON_PLAN.md File Tree section:
> "Quick-create buttons for new prompts, tasks"

The existing implementation has:
- `handleStartCreate(parentPath, type)` function for triggering inline creation
- `creating` state for showing the inline input
- Full file/folder creation via `window.fs.createFile` and `window.fs.createDir`

Implementation approach:
1. Add a "+" button in the header next to the refresh "↻" button
2. Use a small dropdown (similar to ContextMenu component) or a simple popover
3. For "New Prompt", automatically target `swarm/prompts/` directory
4. Reuse the existing creation flow with the inline input

Consider using the existing `ContextMenu` component for the dropdown, or a simpler inline button group.

---

## Completion Notes

**Completed by agent 5ed13857**

### Implementation Summary

Added a "+" quick-create button to the File Tree header with a dropdown menu featuring three options:

1. **New Prompt** - Creates a new file in `swarm/prompts/` directory
2. **New File** - Creates a new file in the `swarm/` root
3. **New Folder** - Creates a new folder in the `swarm/` root

### Changes Made

- **`electron/src/renderer/components/FileTree.tsx`**:
  - Added `quickCreateMenu` state to track dropdown visibility and position
  - Added `quickCreateButtonRef` for the button reference
  - Added a "+" button in the header next to the refresh button
  - Button has proper ARIA attributes (`aria-haspopup`, `aria-expanded`) for accessibility
  - Reused existing `ContextMenu` component for the dropdown
  - Wired up menu options to call `handleStartCreate` with appropriate paths

### Testing

- Build passes successfully (`npm run build`)
- TypeScript compilation included in build passes

### Acceptance Criteria Met

- [x] "+" button displayed next to refresh button
- [x] Dropdown menu with New Prompt, New File, New Folder options
- [x] Options trigger inline rename input at appropriate location
- [x] UI matches existing style (small, muted, hover state)
- [x] Keyboard accessible (button is focusable, activatable with Enter/Space)
