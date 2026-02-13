# Add Duplicate Task to DAG Context Menu

## Goal

Add a "Duplicate Task" option to the DAG canvas right-click context menu. This allows users to quickly create copies of existing tasks, which is useful when building similar tasks with slight variations.

## Files

- `electron/src/renderer/components/DagCanvas.tsx` - Add duplicate handler and menu item
- `electron/src/renderer/App.tsx` - Add `onDuplicateTask` prop and handler

## Dependencies

None - all prerequisite infrastructure exists:
- Context menu already implemented (lines 380-415 in DagCanvas.tsx)
- YAML serialization works (`serializeCompose` in yamlParser.ts)
- Task creation flow exists (`handleSaveTask` pattern)

## Implementation

1. Add `onDuplicateTask?: (taskName: string, taskDef: TaskDef) => void` prop to DagCanvas
2. Add "Duplicate Task" button to the context menu (between Run and Delete)
3. In App.tsx, implement `handleDuplicateTask`:
   - Parse current compose
   - Generate unique name: `{taskName}-copy` or `{taskName}-copy-2` if exists
   - Add new task with same definition
   - Write back to YAML
   - Show toast notification
   - Optionally open the task drawer for the new task

## Acceptance Criteria

- [x] Right-click on a task node shows "Duplicate Task" option
- [x] Clicking duplicate creates a new task with suffix `-copy` (or `-copy-N`)
- [x] New task has identical configuration (prompt, model, prefix, suffix)
- [x] New task does NOT inherit dependencies (clean slate)
- [x] Toast confirms duplication: "Duplicated task 'planner' as 'planner-copy'"
- [x] DAG refreshes to show the new task node
- [x] Works for tasks created via any method (YAML, drag-drop, drawer)

## Notes

- The file tree already has a duplicate feature for files (fs:duplicate IPC handler)
- Keep dependencies empty on the copy to avoid creating cycles
- The duplicate should appear near the original in the DAG (could use same position + offset)

## Completion Notes

Implemented by agent 6b55d926 on iteration 5.

Changes made:
1. **DagCanvas.tsx**:
   - Added `onDuplicateTask?: (taskName: string, taskDef: TaskDef) => void` prop to `DagCanvasProps` interface
   - Added `onDuplicateTask` to props destructuring
   - Updated `handleNodeContextMenu` to check for `onDuplicateTask`
   - Added `handleContextMenuDuplicate` callback handler
   - Added "Duplicate Task" button in context menu between "Run Task" and "Delete Task"

2. **App.tsx**:
   - Added `handleDuplicateTask` callback function that:
     - Parses current compose file
     - Generates unique name (`taskName-copy` or `taskName-copy-N`)
     - Clones task definition without dependencies
     - Writes updated YAML to file
     - Positions new task near original (offset by 50px)
     - Shows success toast notification
     - Opens task drawer for the new task
   - Passed `onDuplicateTask={handleDuplicateTask}` to DagCanvas component

Build verified: `npm run build` passes successfully.
