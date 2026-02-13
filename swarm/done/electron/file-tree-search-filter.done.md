# Task: Add Search/Filter to File Tree

## Goal

Add a search/filter input to the File Tree sidebar that lets users quickly find files within the `swarm/` directory. This is a Phase 1 feature listed in ELECTRON_PLAN.md: "Filter/search within the tree".

As the number of prompt files, output folders, and config files grows, users need a fast way to locate specific files without manually expanding and scrolling through the tree.

## Files

- **Modify**: `electron/src/renderer/components/FileTree.tsx` — Add a search input above the tree and filter logic
- **Modify**: `electron/src/renderer/components/FileTreeItem.tsx` — Support highlighting matched text in file/folder names, and auto-expand parents of matches

## Dependencies

- None. The FileTree component already exists with full CRUD and context menu support.

## Acceptance Criteria

1. A text input appears at the top of the file tree panel (below the "Files" header, above the tree entries)
2. Typing in the input filters the tree to show only entries whose names match the query (case-insensitive substring match)
3. Parent directories of matching files are automatically expanded to reveal matches
4. When a filter is active, directories that contain no matching descendants are hidden
5. Clearing the input restores the full tree view
6. The search input has a clear button (X) when text is present
7. The filter works on both file names and folder names
8. Matched portion of file/folder names is visually highlighted (e.g., bold or different color)
9. Empty state message shown when no results match ("No files match '{query}'")
10. The search input has a keyboard shortcut hint or placeholder text like "Filter files..."

## Notes

- The filtering should happen client-side on the already-loaded tree data (entries state in FileTree.tsx)
- Use recursive filtering: walk the tree structure, keep any node whose name matches OR that has a descendant matching
- FileTreeItem already supports `expanded` state per directory — auto-expand matched parent paths when filtering
- Keep the implementation simple: no debouncing needed since it's client-side filtering on a small dataset
- Style consistently with the existing dark theme (bg-zinc-800 input, text-zinc-300, etc.)
- The search should persist while the user navigates/selects files (don't clear on file selection)

## Completion Notes

Implemented by agent 54744ac3.

### What was implemented:

**FileTree.tsx:**
- Added `filterQuery` state and a search input field below the "Files" header
- Input has placeholder text "Filter files..."
- Clear button (✕) appears when text is present, clears the filter and refocuses input
- Tracks visible root children via `onVisibleChange` callback to show "No files match" empty state
- Passes `filterQuery` and `onVisibleChange` to all FileTreeItem instances

**FileTreeItem.tsx:**
- Added `filterQuery` and `onVisibleChange` props
- `HighlightedName` component highlights matched substring in blue bold text
- Visibility logic: files hidden if name doesn't match; directories hidden if name doesn't match AND no visible descendants
- Auto-expands all directories when filter is active (loads children lazily); restores pre-filter open/closed state when filter is cleared
- Each item reports its visibility upward via `onVisibleChange` callback, enabling parent directories to know if they have matching descendants
- `visibleChildren` set tracks which children are currently visible during filtering

All 10 acceptance criteria met. Build passes (vite build succeeds).
