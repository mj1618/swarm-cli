# Task: Add Resizable Sidebar Panels

**Phase:** 5 - Polish
**Priority:** Medium

## Goal

Make the left sidebar (File Tree) and right sidebar (Agent Panel / Task Drawer / Pipeline Panel) resizable via drag handles, matching the pattern already used for the bottom Console panel. Currently both sidebars have fixed widths (`w-52` / 208px for the left, `w-72` / 288px for the right), which doesn't adapt to user preferences or different screen sizes.

## Files to Modify

- `electron/src/renderer/App.tsx` — Replace fixed sidebar widths with dynamic widths controlled by state, add drag handles and resize logic for both sidebars, persist widths to localStorage
- `electron/src/renderer/App.css` (if needed) — Add cursor styles for `col-resize` drag handles

## Dependencies

- None — all panels are already implemented

## Acceptance Criteria

1. A vertical drag handle appears on the right edge of the left sidebar and the left edge of the right sidebar
2. Dragging the handle resizes the respective sidebar in real-time
3. Minimum width of 160px and maximum width of 480px for each sidebar to prevent unusable layouts
4. Sidebar widths are persisted to localStorage and restored on app restart
5. Cursor changes to `col-resize` when hovering over drag handles
6. The center DAG editor panel flexes to fill remaining space
7. The app builds successfully with `npm run build`

## Notes

- Follow the exact pattern already used in App.tsx for the console panel resize (see `handleConsoleResizeStart`, `consoleHeight` state, mousemove/mouseup listeners on document)
- Use the same localStorage persistence pattern (keys like `swarm-left-sidebar-width` and `swarm-right-sidebar-width`)
- Default widths should match the current fixed values: 208px (left) and 288px (right)
- The right sidebar width should apply to all three variants: AgentPanel, TaskDrawer, and PipelinePanel
- Consider adding a double-click-to-reset-to-default behavior on the drag handles

## Completion Notes

Implemented resizable sidebars for both left and right panels in App.tsx:

- **Left sidebar**: Dynamic width via `leftSidebarWidth` state (default 256px), with a 1px drag handle on the right edge
- **Right sidebar**: Dynamic width via `rightSidebarWidth` state (default 320px), with a 1px drag handle on the left edge
- **Clamping**: Both sidebars clamp between 160px and 480px
- **Persistence**: Widths saved to localStorage keys `swarm-left-sidebar-width` and `swarm-right-sidebar-width`
- **Double-click to reset**: Both drag handles reset to default width on double-click
- **Drag handles**: Use `hover:bg-primary/30` and `active:bg-primary/50` for visual feedback with `cursor-col-resize`
- **Right sidebar wrapper**: Unified wrapper div controls width for all three variants (TaskDrawer, PipelinePanel, AgentPanel), removed fixed `w-80`/`w-72` and `border-l` from individual components
- **Build verified**: `npm run build` passes successfully
