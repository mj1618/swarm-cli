# Fix: Empty DAG State Does Not Support Drag-and-Drop

## Goal

Fix the bug where dragging a prompt file from the File Tree onto an empty DAG canvas does nothing, even though the empty state UI explicitly instructs users to do this.

## Files to Modify

- `electron/src/renderer/components/DagCanvas.tsx`

## Problem Analysis

In `DagCanvas.tsx`:
1. The drag handlers (`handleDragOver`, `handleDragLeave`, `handleDrop`) are defined at lines 622-645
2. The empty state early return happens at lines 666-728 when `nodes.length === 0`
3. The empty state container (line 671) does NOT have the drag event handlers attached
4. The main DAG container (lines 731-737) correctly has all drag handlers

The empty state UI at line 715 explicitly tells users:
> "Drag & drop: Drag a prompt from the File Tree on the left to create a task"

But this functionality is broken because the handlers aren't attached.

## Required Changes

1. Add drag event handlers to the empty state container div (line 671):
   - `onDragOver={handleDragOver}`
   - `onDragLeave={handleDragLeave}`
   - `onDrop={handleDrop}`

2. Add the `isDragOver` visual indicator styling to the empty state container (same as line 732):
   - `className={...${isDragOver ? 'ring-2 ring-inset ring-blue-500/50 bg-blue-500/5' : ''}...}`

## Acceptance Criteria

1. Start with a `swarm.yaml` that has no tasks (or `tasks: {}`)
2. Open the Electron app and navigate to the DAG canvas
3. Drag a `.md` prompt file from the File Tree onto the empty DAG canvas
4. Verify the blue ring visual indicator appears during drag
5. Verify a new task is created when the file is dropped
6. Verify the task configuration drawer opens for the new task

## Notes

- The `isDragOver` state is already being tracked (line 225)
- The `onDropCreateTask` callback already exists and handles the task creation logic
- This is a UX regression - the instructional text promises functionality that doesn't work
- Priority: Medium - affects first-time user onboarding experience

---

## Completion Notes

**Completed by:** c78d4d16
**Date:** 2026-02-13

### Changes Made

Modified `electron/src/renderer/components/DagCanvas.tsx`:

Changed the empty state container from:
```tsx
<div className="flex-1 flex items-center justify-center text-muted-foreground">
```

To:
```tsx
<div
  className={`flex-1 flex items-center justify-center text-muted-foreground transition-colors ${isDragOver ? 'ring-2 ring-inset ring-blue-500/50 bg-blue-500/5' : ''}`}
  onDragOver={handleDragOver}
  onDragLeave={handleDragLeave}
  onDrop={handleDrop}
>
```

This adds:
1. The three drag event handlers (`onDragOver`, `onDragLeave`, `onDrop`) to enable drag-and-drop functionality
2. The `isDragOver` visual indicator (blue ring + tinted background) to provide feedback during drag
3. A `transition-colors` class for smooth visual transitions

### Verification

- Build passes: `npm run build` completes successfully
- The fix matches the pattern used by the main DAG container (lines 731-737)
