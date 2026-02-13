# Task: Output Folder Summary Viewer

## Completed

Implemented by agent d64cf2e5 in commit 224d388.

### What was implemented:

1. **Created `OutputRunViewer.tsx`** — New component that renders a summary view for pipeline output run folders:
   - Parses folder name (`YYYYMMDD-HHMMSS-hexid`) to display human-readable timestamp and run ID
   - Lists all task files with status badges (done=green, pending=yellow, processing=blue)
   - Sorts files by status (done first, then processing, then pending)
   - Shows summary stats (X done, Y pending, Z in progress)
   - Clicking a file opens it in the Monaco editor
   - Handles empty folders and read errors gracefully
   - Includes refresh button

2. **Modified `FileTree.tsx`** — Updated `handleSelect` to allow output run folder directories (matching `/outputs/YYYYMMDD-HHMMSS-hexid/` pattern) to be selected, calling `onSelectFile` for them just like regular files.

3. **Modified `App.tsx`** — Added routing logic so when a selected path is an output run folder, the center panel renders `OutputRunViewer` instead of `MonacoFileEditor` or the DAG canvas.

4. **No preload changes needed** — The existing `window.fs.readdir` IPC channel was sufficient for listing folder contents.

### Acceptance Criteria Met:
- [x] Clicking an output run folder opens a summary view in the center panel
- [x] Summary displays run timestamp and ID parsed from folder name
- [x] Lists .pending.md, .done.md, and .processing.md files with status indicators
- [x] Clicking a file in the summary opens it in Monaco editor
- [x] Handles empty folders and read errors gracefully
