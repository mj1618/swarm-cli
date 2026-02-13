# Task: Enable Draggable Nodes on DAG Canvas

**Phase:** 3 — Interactive Editing

## Goal

Enable drag-and-drop repositioning of task nodes on the DAG canvas. Currently, `nodesDraggable` is set to `false` in `DagCanvas.tsx`, locking all nodes in their dagre-computed positions. This task enables node dragging so users can manually arrange their pipeline layout, and persists custom positions so they survive re-renders and YAML reloads.

## Files

### Modify
- `electron/src/renderer/components/DagCanvas.tsx` — Set `nodesDraggable={true}` on the `<ReactFlow>` component. Add an `onNodesChange` handler that updates node positions when dragged. Track whether nodes have been manually positioned vs auto-laid-out by dagre. Add a "Reset Layout" button that re-applies the dagre auto-layout, clearing any manual positions.
- `electron/src/renderer/lib/yamlParser.ts` — Update `composeToFlow()` to accept an optional `savedPositions: Record<string, { x: number, y: number }>` parameter. If a node has a saved position, use it instead of the dagre-computed position. This allows manual positions to persist across YAML reloads.
- `electron/src/renderer/App.tsx` — Add state for `nodePositions: Record<string, { x: number, y: number }>`. Pass it to `DagCanvas` and update it when nodes are dragged. Store positions in localStorage keyed by the YAML file path so they persist across sessions.

## Dependencies

- Phase 2 complete (DAG canvas with React Flow) ✓
- Does NOT depend on the Task Configuration Drawer (in progress)

## Acceptance Criteria

1. Task nodes in the DAG canvas can be dragged to new positions
2. After dragging a node, it stays in its new position (does not snap back)
3. When the YAML file is reloaded/re-parsed, manually positioned nodes retain their positions
4. A "Reset Layout" button is visible that re-applies the automatic dagre layout
5. Clicking "Reset Layout" moves all nodes back to their dagre-computed positions
6. Node positions persist across page reloads via localStorage
7. Edges update in real-time as nodes are dragged (React Flow handles this natively)
8. The app builds successfully (`npm run build` in electron/)

## Notes

- React Flow's `onNodesChange` handler receives `NodeChange[]` events including position changes. Use `applyNodeChanges` from `@xyflow/react` to apply them.
- The dagre layout in `composeToFlow()` currently computes positions for all nodes. The change is to let saved positions override dagre positions, not to remove dagre entirely — dagre is still needed for initial layout and the reset button.
- For localStorage, use a key like `swarm-dag-positions:${yamlFilePath}` to support multiple projects.
- Keep `nodesConnectable={false}` for now — visual dependency creation is a separate task.
- The "Reset Layout" button should be placed in the DAG canvas toolbar area (top-right corner of the canvas, similar to React Flow's built-in controls).

## Completion Notes

**Completed by agent 736d1f57.**

### Changes Made

1. **`yamlParser.ts`** — `composeToFlow()` now accepts an optional `savedPositions` parameter. When a saved position exists for a node, it is used instead of the dagre-computed position.

2. **`App.tsx`** — Added `nodePositions` state with localStorage persistence keyed by `swarm-dag-positions:${yamlFilePath}`. Positions are loaded on mount and when the active YAML file changes. Added `handlePositionsChange` (saves to state + localStorage) and `handleResetLayout` (clears state + localStorage). These are passed to `DagCanvas`.

3. **`DagCanvas.tsx`** — Set `nodesDraggable={true}`. Added local `nodes` state managed via `onNodesChange` + `applyNodeChanges`. On drag-end events, positions are persisted via `onPositionsChange` callback. Added a "Reset Layout" button in the top-right panel using React Flow's `<Panel>` component.

### All Acceptance Criteria Met
- Nodes are draggable and stay in position after drag
- Positions persist across YAML reloads and page reloads (localStorage)
- Reset Layout button restores dagre auto-layout
- Edges update in real-time during drag (React Flow native)
- App builds successfully
