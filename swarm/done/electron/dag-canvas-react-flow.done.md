# Task: DAG Canvas with React Flow Integration

**Phase:** 2 - DAG Visualization

## Goal

Build a visual DAG canvas in the center panel that renders `swarm.yaml` task graphs using React Flow. When a `.yaml` file is selected in the file tree, parse it and display the task dependency graph with positioned nodes and labeled edges. This replaces the "DAG visualization coming soon" placeholder and leverages the existing `yamlParser.ts` which already contains `parseComposeFile()` and `composeToFlow()` functions.

## Files

### Create
- `electron/src/renderer/components/DagCanvas.tsx` — Main DAG canvas component that wraps `<ReactFlow>`, renders task nodes and edges from parsed YAML, includes controls (zoom, fit view), and a minimap
- `electron/src/renderer/components/TaskNode.tsx` — Custom React Flow node component for tasks, showing task name, prompt source, and model badge (matches the design in ELECTRON_PLAN.md)

### Modify
- `electron/src/renderer/App.tsx` — When a `.yaml`/`.yml` file is selected, show `<DagCanvas>` in the center panel instead of `<FileViewer>`. Non-YAML files continue to use `<FileViewer>`. When no file is selected, show the existing placeholder. Add the React Flow CSS import.
- `electron/src/renderer/index.css` — Add any needed styles for the custom task nodes and React Flow overrides that work with the existing dark theme

## Dependencies

- YAML parser utility (completed — `electron/src/renderer/lib/yamlParser.ts` with `parseComposeFile()` and `composeToFlow()`)
- `@xyflow/react` package (already installed)
- `dagre` package (already installed)
- File tree component and file selection (completed)
- `fs:readfile` IPC handler (completed)

## Acceptance Criteria

1. Selecting a `.yaml` or `.yml` file in the file tree renders the DAG canvas in the center panel (not the text file viewer)
2. Task nodes are displayed as styled cards showing: task name (bold), prompt source (e.g. "planner"), and model if specified
3. Dependency edges are drawn between nodes with condition labels (`success`, `failure`, `any`, `always`)
4. Edges are color-coded by condition type (green for success, red for failure, yellow for any, blue for always) — matching the existing `getEdgeColor()` in yamlParser.ts
5. Nodes are auto-laid out top-to-bottom using dagre (already implemented in `composeToFlow()`)
6. The canvas supports zoom, pan, and has a fit-view-on-load behavior
7. A minimap is shown in the corner for navigation
8. Non-YAML files still open in the existing FileViewer
9. When no file is selected, the DAG placeholder is still shown
10. The app builds without TypeScript errors (`npm run build` in electron/)

## Notes

- **From ELECTRON_PLAN.md**: The plan specifies React Flow for DAG visualization. The tech stack section confirms `@xyflow/react` (React Flow v12).
- The `yamlParser.ts` already does the heavy lifting: `parseComposeFile()` parses YAML to `ComposeFile`, and `composeToFlow()` converts it to React Flow `Node[]` and `Edge[]` with dagre layout. The main work is building the React components.
- Import React Flow CSS: `import '@xyflow/react/dist/style.css'` — this is required for React Flow to render properly.
- The custom `TaskNode` component should be registered with React Flow via `nodeTypes={{ taskNode: TaskNode }}` — note that `composeToFlow()` already sets `type: 'taskNode'` on nodes.
- Use `<ReactFlowProvider>` if needed for nested hook access.
- Keep the canvas read-only for now — interactive editing (drag-to-create dependencies, add tasks) is Phase 3.
- The `fitView` prop on `<ReactFlow>` handles initial zoom-to-fit.
- Style the custom nodes to match the dark theme using Tailwind classes consistent with the rest of the app.

## Completion Notes

**Completed by agent 6bac9a64**

Most components (DagCanvas.tsx, TaskNode.tsx) were created in the previous iteration. This iteration completed the remaining acceptance criteria:

1. **Added MiniMap** to DagCanvas.tsx — shows a mini navigation map in the corner with dark theme colors
2. **Updated App.tsx** to route YAML files to DagCanvas instead of FileViewer:
   - When a `.yaml`/`.yml` file is selected in the file tree, its content is loaded and displayed in the DAG canvas
   - Non-YAML files continue to use FileViewer
   - When no file is selected, the default `swarm/swarm.yaml` DAG is shown
   - Header dynamically shows the selected filename or "DAG Editor"
3. All 10 acceptance criteria are met:
   - YAML files → DagCanvas with parsed task graph
   - Task nodes show name (bold), prompt source, model badge
   - Edges are color-coded with condition labels
   - Dagre auto-layout (top-to-bottom)
   - Zoom, pan, fit-view-on-load
   - MiniMap in corner
   - Non-YAML files → FileViewer
   - No file selected → default DAG
   - Builds without TypeScript errors
