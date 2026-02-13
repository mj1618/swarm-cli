# Task: Console Panel Log Search & Filter

## Goal

Add search and filter functionality to the ConsolePanel / LogView components. The ELECTRON_PLAN.md (line 265) specifies "Filter/search within logs" as a required feature for the bottom console panel. Currently, the console panel displays log content but provides no way to search or filter log lines.

## Files

- **Modify:** `electron/src/renderer/components/ConsolePanel.tsx` — Add a search input bar to the tab bar area
- **Modify:** `electron/src/renderer/components/LogView.tsx` — Accept a `searchQuery` prop, highlight matching lines, and optionally filter to only show matching lines

## Dependencies

None — the ConsolePanel and LogView components already exist and are fully functional. This builds on top of them.

## Acceptance Criteria

1. A search input field is visible in the ConsolePanel header (right side of tab bar, before the Clear button)
2. Typing in the search field highlights matching text within log lines (e.g., yellow background on matched substrings)
3. A "Filter" toggle button next to the search input, when active, hides non-matching lines (showing only lines that match the query)
4. Search is case-insensitive by default
5. The match count is displayed (e.g., "12 matches")
6. Pressing Escape while the search input is focused clears the search
7. The existing auto-scroll behavior continues to work correctly when search is active
8. Build passes cleanly (`npm run build` in electron/)

## Notes

- Keep the search simple — no regex support needed, just plain substring matching
- Use the existing Tailwind color classes for highlight styling (e.g., `bg-yellow-500/30` for highlights)
- The `LogView` component already splits content into lines and classifies them — the search highlight should layer on top of the existing `classifyLine` color coding
- Consider using `String.prototype.indexOf` for efficient substring matching rather than regex
- The filter toggle should be visually distinct when active (e.g., filled vs outline icon, or different background color)
- Don't over-engineer — a simple controlled input with state lifted to ConsolePanel is sufficient

## Completion Notes

Implemented by agent 1544d36c:

- **ConsolePanel.tsx**: Added search input (w-40, right side of tab bar), match count display, and Filter toggle button. Added Cmd+F/Ctrl+F keyboard shortcut to focus search. Escape clears search and blurs input.
- **LogView.tsx**: Added `searchQuery`, `filterMode`, and `onMatchCount` props. `highlightMatches()` splits text on case-insensitive matches and wraps them in `<mark>` with `bg-yellow-500/30`. `filteredLines` memo filters lines when filter mode is active. `matchCount` counts actual substring occurrences (not just matching lines). Auto-scroll behavior preserved.
- Build passes cleanly.
