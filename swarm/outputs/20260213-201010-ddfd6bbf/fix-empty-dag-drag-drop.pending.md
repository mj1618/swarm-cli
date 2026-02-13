# Fix: Empty DAG State Does Not Support Drag-and-Drop

## Bug Description

When the DAG canvas has no tasks (empty state), the drag-and-drop functionality to create the first task is broken. The empty state UI explicitly tells users:

> "Drag & drop: Drag a prompt from the File Tree on the left to create a task"

However, dragging a prompt onto the empty state does nothing because the drag event handlers are defined after the early return for the empty state.

## Root Cause

In `electron/src/renderer/components/DagCanvas.tsx`:

1. Lines 580-641 render the empty state and return early when `nodes.length === 0`
2. Lines 644-667 define `handleDragOver`, `handleDragLeave`, and `handleDrop`
3. Lines 669-875 contain the main DAG render with drag handlers attached

Because the handlers are defined after the early return, the empty state component never has access to them.

## Fix Required

Move the drag-and-drop handler definitions before the early returns, and add the drag event handlers to the empty state container element.

The fix should:
1. Move `handleDragOver`, `handleDragLeave`, and `handleDrop` useCallback definitions earlier in the component (before the early returns)
2. Add `onDragOver={handleDragOver}`, `onDragLeave={handleDragLeave}`, and `onDrop={handleDrop}` to the empty state container div
3. Add the same `isDragOver` visual indicator styling to the empty state

## Affected File

- `electron/src/renderer/components/DagCanvas.tsx`

## Testing

1. Start with a swarm.yaml that has no tasks defined
2. Open the electron app
3. Try dragging a prompt file from the File Tree onto the DAG canvas
4. Verify a new task is created at the drop location
5. Verify the drag-over visual indicator (blue ring) appears when dragging over the empty state

## Priority

Medium - This affects first-time user experience when starting with an empty project.
