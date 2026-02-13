# Task: Add Delete Tasks and Edges from DAG Canvas

**Phase:** 3 - Interactive Editing
**Priority:** High

## Goal

Users should be able to delete tasks and dependency edges directly from the DAG canvas. Currently, there is no way to remove tasks or edges visually — the only option is to manually edit the YAML. This is a core missing piece of Phase 3 (Interactive Editing).

## What to Build

### 1. Delete Tasks via Keyboard (Backspace/Delete)

When a node is selected on the canvas, pressing Backspace or Delete should:
- Show a confirmation dialog ("Delete task 'coder'? This will also remove all its dependencies.")
- Remove the task from the compose file
- Remove all edges (dependencies) that reference this task
- Write the updated YAML back to disk
- Show a success toast

### 2. Delete Edges via Keyboard (Backspace/Delete)

When an edge is selected on the canvas, pressing Backspace or Delete should:
- Remove the dependency from the target task's `depends_on` array
- Write the updated YAML back to disk
- Show a success toast (no confirmation needed for edges — they're easy to recreate)

### 3. Make Edges Selectable

Currently the ReactFlow instance has `elementsSelectable={true}` but edges may not be interactable. Ensure edges can be clicked/selected by setting appropriate edge interaction props.

### 4. Context Menu (Right-Click) on Nodes

Add a right-click context menu on task nodes with a "Delete Task" option as an alternative to the keyboard shortcut.

## Files

### Modify
- **`electron/src/renderer/components/DagCanvas.tsx`**
  - Add `onDeleteTask` and `onDeleteEdge` callback props
  - Add `onNodesDelete` / `onEdgesDelete` handlers (or use `onDelete` callback from ReactFlow)
  - Make edges selectable: add `edgesSelectable` or set `selectable` on edge objects
  - Add right-click context menu for nodes with "Delete" option
  - Add a simple confirmation dialog component (inline or extracted)

- **`electron/src/renderer/App.tsx`**
  - Add `handleDeleteTask` function: removes task from compose, removes references from other tasks' `depends_on`, serializes and saves YAML, reloads
  - Add `handleDeleteEdge` function: removes specific dependency from target task's `depends_on`, serializes and saves YAML, reloads
  - Pass both handlers as props to `DagCanvas`

## Dependencies

- task-configuration-drawer (completed) — establishes the YAML write-back pattern
- visual-dependency-creation (completed) — establishes the edge/dependency model

## Acceptance Criteria

1. Selecting a task node and pressing Delete/Backspace removes it from the YAML (with confirmation)
2. Selecting an edge and pressing Delete/Backspace removes the dependency from the YAML
3. Right-clicking a task node shows a context menu with "Delete Task"
4. After deletion, the DAG re-renders correctly without the removed element
5. Deleting a task also removes all `depends_on` references to it from other tasks
6. A toast notification confirms the deletion
7. App builds with `npm run build`

## Notes

- Follow the existing YAML write-back pattern in `handleSaveTask` (App.tsx:84-114): modify compose object → `serializeCompose()` → `window.fs.writefile()` → reload
- The `serializeCompose` utility from `../lib/yamlParser` is already used for saving
- ReactFlow supports `onNodesDelete` and `onEdgesDelete` callbacks, or the unified `onDelete` prop
- For the confirmation dialog, a simple inline modal is sufficient — no need for a full dialog library
- Edge IDs follow the format `{source}->{target}` (set in `composeToFlow`)
- Be careful to also remove the task from any `pipelines[].tasks` arrays in the compose file
