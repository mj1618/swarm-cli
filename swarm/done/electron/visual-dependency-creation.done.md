# Task: Visual Dependency Creation via Port-to-Port Dragging

**Phase:** 3 — Interactive Editing

## Goal

Enable users to create task dependencies by dragging from one task node's source handle (bottom port) to another task node's target handle (top port) on the DAG canvas. When a connection is made, prompt the user to select a condition (`success | failure | any | always`), then update the in-memory compose model and write the change back to `swarm.yaml`.

This is a core Phase 3 interaction feature described in ELECTRON_PLAN.md (lines 108-109): "Drag from one task's output port to another's input port → Creates edge with dropdown to select condition."

## Files

### Create
- `electron/src/renderer/components/ConnectionDialog.tsx` — A small popover/dialog that appears near the newly created edge asking the user to pick a dependency condition. Shows 4 buttons or a dropdown: `success`, `failure`, `any`, `always`. On selection, confirms the connection; on cancel/escape, removes the pending edge. Styled to match the dark theme.

### Modify
- `electron/src/renderer/components/DagCanvas.tsx`:
  - Change `nodesConnectable={false}` to `nodesConnectable={true}`
  - Add `onConnect` handler from React Flow that fires when a user drags between ports
  - When `onConnect` fires, show the `ConnectionDialog` to pick a condition
  - After condition is selected, call a new `onAddDependency` callback prop with `{ source: string, target: string, condition: string }`
  - If cancelled, do nothing (no edge added)
  - Add state for the pending connection (source/target) and dialog visibility/position

- `electron/src/renderer/components/TaskNode.tsx`:
  - Update the Handle components to be visually interactive (show a highlight/glow on hover when connecting)
  - The source handle (bottom) initiates connections, the target handle (top) receives them

- `electron/src/renderer/App.tsx`:
  - Add `onAddDependency` handler that:
    1. Updates the in-memory compose data by adding a dependency to the target task's `depends_on` array
    2. Uses `serializeCompose()` from `yamlWriter.ts` to generate updated YAML
    3. Writes the YAML back via `window.fs.writefile('swarm/swarm.yaml', updatedYaml)`
    4. Refreshes the YAML content state so the DAG re-renders with the new edge
  - Pass `onAddDependency` as a prop to `DagCanvas`

- `electron/src/renderer/lib/yamlWriter.ts`:
  - Add a function `addDependency(compose: ComposeFile, targetTask: string, sourceTask: string, condition: string): ComposeFile` that adds a dependency entry to `compose.tasks[targetTask].depends_on` (creating the array if it doesn't exist), avoiding duplicates

## Dependencies

- Phase 2 complete (DAG canvas with React Flow, task nodes with handles, edges) ✓
- `yamlWriter.ts` exists with `serializeCompose()` ✓
- `fs:writefile` IPC handler exists ✓
- Does NOT depend on the task configuration drawer save being wired up (independent feature)
- Does NOT depend on draggable nodes (independent feature)

## Acceptance Criteria

1. Task node handles (top/bottom ports) are visually interactive — they highlight on hover during connection mode
2. Dragging from a source handle (bottom) of one task to a target handle (top) of another initiates a connection
3. A condition selection dialog appears near the connection point
4. Selecting a condition (`success`, `failure`, `any`, `always`) creates the dependency edge
5. The new edge renders with the correct color coding and label matching existing edges
6. The dependency is written back to `swarm.yaml` in the correct `depends_on` format
7. Cancelling the dialog (Escape or clicking away) discards the pending connection
8. Duplicate dependencies (same source→target) are prevented
9. Self-references (connecting a task to itself) are prevented
10. The app builds successfully (`npm run build` in electron/)

## Notes

- React Flow's `onConnect` callback provides `{ source, target, sourceHandle, targetHandle }` — use `source` and `target` which are the node IDs (task names)
- The `ConnectionDialog` should appear as a floating element near the midpoint of the new edge or near the target node
- Use React Flow's `useReactFlow` hook if needed to get viewport coordinates for positioning the dialog
- For the condition dialog, a simple floating div with 4 styled buttons is sufficient — no need for a heavy modal
- The `addDependency` function in yamlWriter should use the simple string form (`"taskname"`) when condition is `success` (the default), and the object form (`{ task: "taskname", condition: "failure" }`) for other conditions — matching the existing convention in `applyTaskEdits`
- Edge colors should match the existing scheme in `yamlParser.ts`: green for success, red for failure, yellow for any, blue for always
- React Flow requires `nodesConnectable={true}` for `onConnect` to fire — this is currently set to `false`
