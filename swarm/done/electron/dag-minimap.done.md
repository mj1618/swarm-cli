# Task: Add DAG Canvas Minimap

**Phase:** 5 - Polish
**Priority:** Low
**Status:** COMPLETED

## Goal

Add a minimap to the DAG canvas for easier navigation of larger task graphs. React Flow provides a built-in `<MiniMap />` component that shows a birds-eye view of the entire DAG, with the current viewport indicated. This is especially useful when working with pipelines that have many tasks.

## Completion Notes

**Completed by:** Agent 70c929ae
**Date:** 2026-02-13

### What was implemented:

1. **Enhanced MiniMap with dynamic node colors based on task status:**
   - Running tasks: Blue (#3b82f6)
   - Paused tasks: Amber (#f59e0b)
   - Succeeded tasks: Green (#22c55e)
   - Failed tasks: Red (#ef4444)
   - Pending tasks: Gray (#6b7280)
   - Idle/no status: Slate (theme-aware)

2. **Full theme support:**
   - Dark theme: Dark background with visible nodes
   - Light theme: Light background with appropriate contrast
   - Border colors adapt to theme

3. **Added interactive features:**
   - `pannable` - Users can pan the viewport by dragging on the minimap
   - `zoomable` - Users can zoom by scrolling on the minimap

### Files modified:

- `electron/src/renderer/components/DagCanvas.tsx` - Enhanced MiniMap component with nodeColor function and theme-aware styling

### Acceptance criteria verification:

1. ✅ Minimap appears in bottom-right corner showing all task nodes
2. ✅ Minimap shows current viewport as highlighted rectangle
3. ✅ Clicking/dragging on minimap navigates the main canvas view (pannable enabled)
4. ✅ Node colors reflect task status (running=blue, paused=amber, succeeded=green, failed=red, pending=gray)
5. ✅ Minimap styling matches both dark and light themes
6. ✅ Minimap does not overlap with Controls (Controls is bottom-left, MiniMap is bottom-right)
7. ⚠️ Build has pre-existing errors in MonacoFileEditor.tsx unrelated to this task; DagCanvas.tsx compiles cleanly
