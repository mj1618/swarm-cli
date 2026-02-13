# Task: DAG Canvas Foundation (Phase 2 Start)

## Goal

Set up the DAG visualization canvas in the center panel using React Flow. This includes installing dependencies, creating a YAML-to-graph parser that converts `swarm.yaml` tasks and dependencies into React Flow nodes/edges, and building the initial read-only DAG canvas component. When no file is selected in the file tree, the center panel should show the DAG view of the current `swarm.yaml`.

## Files

### Create
- `electron/src/renderer/lib/yamlParser.ts` — Parses `swarm.yaml` content (loaded via IPC) into a structured TypeScript type (`ComposeFile` with tasks/pipelines), then converts tasks + `depends_on` into React Flow `Node[]` and `Edge[]` with automatic layout (top-to-bottom using dagre or manual level-based positioning)
- `electron/src/renderer/components/DagCanvas.tsx` — React Flow canvas component that displays the parsed DAG. Shows task nodes with name, prompt, and model info. Edges show dependency conditions (`success`, `failure`, `any`, `always`). Read-only for now (no drag-to-create).
- `electron/src/renderer/components/TaskNode.tsx` — Custom React Flow node component for rendering a task card (task name, prompt source, model badge, dependency condition)

### Modify
- `electron/src/renderer/App.tsx` — Import DagCanvas and show it in the center panel as the default view (when no file is selected). Load `swarm.yaml` content on mount via `window.fs.readfile('swarm.yaml')` and pass parsed data to DagCanvas.

## Dependencies

- Electron app scaffold (completed)
- File tree component (completed)
- `fs:readfile` IPC handler (completed)
- No dependency on the File Content Viewer task (these can coexist — FileViewer shows when a file is selected, DagCanvas shows as the default/home view)

## NPM Packages to Install

- `reactflow` — DAG visualization library (specified in ELECTRON_PLAN.md tech stack)
- `js-yaml` + `@types/js-yaml` — YAML parsing (specified in ELECTRON_PLAN.md tech stack)
- `dagre` + `@types/dagre` — Automatic graph layout for positioning nodes

## Acceptance Criteria

1. `npm install` succeeds with new dependencies added to `package.json`
2. `swarm.yaml` is loaded and parsed into a `ComposeFile` TypeScript type on app mount
3. Tasks from `swarm.yaml` render as nodes on the React Flow canvas
4. `depends_on` relationships render as directed edges between nodes
5. Edge labels show the dependency condition (success/failure/any/always)
6. Nodes display: task name, prompt source, and model (if specified)
7. Nodes are automatically laid out in a top-to-bottom DAG arrangement (no overlapping)
8. Canvas supports zoom and pan (built into React Flow)
9. The DAG view is the default center panel content (shown when no file is selected in the tree)
10. The project still builds successfully (`npm run build`)

## Notes

- Use `@xyflow/react` (the current React Flow package name) — the plan says "React Flow" which is now published as `@xyflow/react`
- For automatic layout, use dagre to compute x/y positions based on the dependency graph, then pass positioned nodes to React Flow
- Custom node type `TaskNode` should follow the dark theme (use existing CSS variables: `--card`, `--card-foreground`, `--primary`, etc.)
- Edge condition labels should be color-coded: green for `success`, red for `failure`, yellow for `any`, blue for `always`
- Keep the canvas read-only — interactive editing (drag-to-create nodes/edges) comes in Phase 3
- The `ComposeFile` type should match the YAML structure documented in ELECTRON_PLAN.md (version, tasks with prompt/model/prefix/suffix/depends_on, pipelines with iterations/parallelism/tasks)
- Reference `swarm/swarm.yaml` in this repo as a real-world example for testing

## Completion Notes

**Completed by agent 4fb886f4**

### What was implemented:

1. **Installed dependencies**: `@xyflow/react`, `js-yaml`, `@types/js-yaml`, `dagre`, `@types/dagre`

2. **`electron/src/renderer/lib/yamlParser.ts`**: YAML-to-graph parser with:
   - `ComposeFile`, `TaskDef`, `PipelineDef`, `TaskDependency` TypeScript types
   - `parseComposeFile()` function using js-yaml
   - `composeToFlow()` function that converts tasks/deps into React Flow nodes/edges with dagre auto-layout (top-to-bottom)
   - Color-coded edges: green=success, red=failure, yellow=any, blue=always
   - Animated edges for `any`/`always` conditions

3. **`electron/src/renderer/components/TaskNode.tsx`**: Custom React Flow node component with:
   - Task name in a header bar with primary color accent
   - Prompt source display
   - Model badge (when specified)
   - Top/bottom handles for edge connections
   - Dark theme using CSS variables (--card, --primary, etc.)

4. **`electron/src/renderer/components/DagCanvas.tsx`**: React Flow canvas with:
   - Loading/error/empty states
   - Dot-pattern background, zoom controls
   - Read-only mode (no dragging/connecting)
   - Dark color mode
   - `fitView` for automatic centering

5. **`electron/src/renderer/App.tsx`**: Modified to:
   - Load `swarm/swarm.yaml` on mount via `window.fs.readfile()`
   - Show DagCanvas as default center panel (when no file selected)
   - Wrapped in ReactFlowProvider

### Build status: Passing (`npm run build` succeeds)
