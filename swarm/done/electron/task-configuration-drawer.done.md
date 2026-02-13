# Task: Task Configuration Drawer

**Phase:** 3 — Interactive Editing (first sub-task)

## Goal

Add a slide-out configuration drawer that opens when a user clicks on a task node in the DAG canvas. The drawer displays the task's full configuration (prompt source, model, prefix, suffix, dependencies) in a read-only view. This is the foundation for Phase 3's interactive editing — later iterations will make the fields editable and write changes back to `swarm.yaml`.

## Files

### Create
- `electron/src/renderer/components/TaskDrawer.tsx` — A slide-out panel that appears on the right side (overlaying or pushing the Agent Panel). Displays:
  - Task name (header)
  - Prompt source: shows which of `prompt`, `prompt-file`, or `prompt-string` is set, with the value
  - Model: the model override if set, or "inherited" if not
  - Prefix: the prefix text if set
  - Suffix: the suffix text if set
  - Dependencies: list of `depends_on` entries with task name and condition
  - A close button (X) in the top-right corner

### Modify
- `electron/src/renderer/components/DagCanvas.tsx` — Make task nodes clickable. When a node is clicked, call a callback (`onSelectTask`) with the task name and its `TaskDef` data.
- `electron/src/renderer/components/TaskNode.tsx` — Update node data type to include the full `TaskDef` so the drawer has access to all task fields (not just `label`, `promptSource`, `model`).
- `electron/src/renderer/lib/yamlParser.ts` — Extend `TaskNodeData` to include the full `TaskDef` (or pass it through as `taskDef` property). Update `composeToFlow` to include the full task definition in each node's data.
- `electron/src/renderer/App.tsx` — Add state for `selectedTask: { name: string, def: TaskDef } | null`. Pass `onSelectTask` callback down to `DagCanvas`. Render `<TaskDrawer>` when a task is selected, positioned over or adjacent to the Agent Panel.

## Dependencies

- DAG canvas with React Flow (completed — Phase 2)
- YAML parser with TaskDef types (completed)

## Acceptance Criteria

1. Clicking a task node in the DAG canvas opens a slide-out drawer on the right side
2. The drawer displays: task name, prompt source (type + value), model (or "inherited"), prefix, suffix, and dependencies list
3. Each dependency shows the task name and condition (success/failure/any/always)
4. Clicking the close button (or clicking outside the drawer) dismisses it
5. Clicking a different task node switches the drawer to show that task's config
6. The drawer does not break the existing Agent Panel layout — it either overlays it or temporarily replaces it
7. The app builds successfully (`npm run build` in electron/)

## Notes

- The Task Configuration Panel spec from ELECTRON_PLAN.md shows the target design (search for "Task Configuration Panel (Right Drawer)"). For this task, implement it as read-only — editable fields come in a follow-up task.
- The `TaskDef` interface already exists in `yamlParser.ts` — reuse it rather than creating a new type.
- The `TaskNodeData` currently only has `label`, `promptSource`, and `model`. Extend it to carry the full `TaskDef` so the drawer has all the data it needs without re-parsing.
- For the drawer animation, a simple CSS transition (`transform: translateX`) sliding in from the right is sufficient. No need for a full animation library.
- Use the same Tailwind dark theme classes as the rest of the app (`bg-card`, `text-card-foreground`, `border-border`).
- React Flow supports `onNodeClick` — use this on the `<ReactFlow>` component to detect task node clicks.

## Completion Notes

Implemented by agent 4d916b14 on iteration 3.

### Changes Made
- **`yamlParser.ts`**: Added `taskDef: TaskDef` field to `TaskNodeData` interface; updated `composeToFlow` to pass full task definition in node data
- **`DagCanvas.tsx`**: Added `onSelectTask` callback prop; wired `onNodeClick` handler on `<ReactFlow>` to emit task selection events; enabled `elementsSelectable`
- **`TaskNode.tsx`**: Added cursor pointer, hover highlight (`border-primary/50`), and selected state styling (`border-primary ring-2 ring-primary/30`) using the `selected` prop from React Flow
- **`TaskDrawer.tsx`** (new): Read-only slide-out drawer displaying task name, prompt source (type + value), model (or "inherited"), prefix, suffix, and dependencies with color-coded condition badges. Dismissible via close button, click-outside, or Escape key
- **`App.tsx`**: Added `selectedTask` state; passes `onSelectTask` to `DagCanvas`; conditionally renders `TaskDrawer` in place of `AgentPanel` when a task is selected

All acceptance criteria met. Build passes cleanly.
