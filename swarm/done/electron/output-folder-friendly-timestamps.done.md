# Output Folder Friendly Timestamps

## Goal

Enhance the file tree to display output folder timestamps in a human-readable format. Currently, output folders like `20260213-142305-abc123` show only the raw folder name. This task adds friendly timestamp display to improve UX when browsing pipeline outputs.

From ELECTRON_PLAN.md File Tree section:
> "Shows recent outputs with timestamps (`swarm/outputs/20260213-142305-abc123/`)"

## Files

- `electron/src/renderer/components/FileTreeItem.tsx` - Modify to detect output folders and format timestamps
- `electron/src/renderer/lib/outputFolderUtils.ts` (new) - Helper functions for parsing/formatting output folder names

## Dependencies

None - standalone UX improvement. File tree component is already complete.

## Acceptance Criteria

1. Output run folders (matching pattern `/outputs/YYYYMMDD-HHMMSS-[hash]/`) display with human-readable timestamps
2. Format example: "Feb 13, 2:43 PM" or "Today 2:43 PM" shown alongside or below the folder hash
3. Folders are still identifiable (show partial hash like `abc123`) but timestamp is prominent
4. Non-output folders display normally (no change to other directories)
5. Relative time display for recent runs (e.g., "5 min ago") is a bonus but not required
6. Build passes with `npm run build`

## Notes

### Implementation Approach

1. Create a helper function `parseOutputFolderName(name: string)` that:
   - Matches pattern `^(\d{8})-(\d{6})-([a-f0-9]+)$`
   - Extracts date (YYYYMMDD), time (HHMMSS), and hash
   - Returns `{ date: Date, hash: string } | null`

2. Create a formatter `formatOutputTimestamp(date: Date)` that:
   - For today: "Today 2:43 PM"
   - For yesterday: "Yesterday 2:43 PM"
   - For this year: "Feb 13, 2:43 PM"
   - For older: "Feb 13, 2025 2:43 PM"

3. In `FileTreeItem.tsx`:
   - Check if entry is in `/outputs/` directory and matches the pattern
   - If so, render a custom display showing the friendly timestamp and short hash
   - Use a slightly different layout: timestamp on first line, hash muted below (or inline with separator)

### Visual Example

Instead of:
```
▸ 20260213-194305-abc12345
```

Show:
```
▸ Feb 13, 7:43 PM (abc123)
```

Or stacked:
```
▸ Feb 13, 7:43 PM
    abc12345
```

Keep it subtle and consistent with the existing dark theme styling.

---

## Completion Notes

**Completed by:** Agent 0626ad93  
**Date:** Feb 13, 2026

### What was implemented:

1. **Created `electron/src/renderer/lib/outputFolderUtils.ts`** with:
   - `parseOutputFolderName()` - Parses folder names matching `YYYYMMDD-HHMMSS-[hash]` pattern
   - `isInOutputsDirectory()` - Checks if a path is inside `/outputs/`
   - `formatOutputTimestamp()` - Formats dates with context-aware display (Today/Yesterday/Month Day/Full date)
   - `getOutputFolderDisplay()` - Combined helper returning timestamp and short hash

2. **Modified `electron/src/renderer/components/FileTreeItem.tsx`** to:
   - Import the new utility functions
   - Detect output folders during rendering
   - Display friendly timestamps like "Today 7:43 PM (abc123)" instead of raw folder names
   - Short hash (6 chars) shown in muted monospace style for identification

### All acceptance criteria met:
- [x] Output folders show human-readable timestamps
- [x] Format matches spec: "Feb 13, 2:43 PM" or "Today 2:43 PM"
- [x] Hash shown in parentheses for identification
- [x] Non-output folders unchanged
- [x] Build passes with `npm run build`
