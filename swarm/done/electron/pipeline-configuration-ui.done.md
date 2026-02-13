# Task: Add Pipeline Configuration UI

**Phase:** 3 - Interactive Editing
**Priority:** Medium

## Goal

Add a pipeline configuration panel to the DAG editor that allows users to view and edit pipeline settings (iterations, parallelism) and see which tasks belong to each pipeline. Currently, pipelines are parsed from `swarm.yaml` by the YAML parser but there is no UI for viewing or editing pipeline configuration.

The ELECTRON_PLAN.md specifies:
- Dropdown to switch between pipelines (or "All Tasks" view)
- Edit pipeline settings: iterations, parallelism
- Select which tasks belong to pipeline (checkboxes or drag into group)

## Files to Modify

- `electron/src/renderer/App.tsx` — Add pipeline selection state, pass pipeline data to DAG canvas and a new PipelineConfig component
- `electron/src/renderer/components/PipelineConfigBar.tsx` (new) — Horizontal bar above the DAG canvas showing pipeline selector dropdown, iterations input, parallelism input, and task membership
- `electron/src/renderer/components/DagCanvas.tsx` — Accept active pipeline prop, optionally highlight/filter tasks belonging to selected pipeline
- `electron/src/renderer/lib/yamlParser.ts` — Already parses pipelines; may need minor additions
- `electron/src/renderer/lib/yamlWriter.ts` — Add function to update pipeline settings in YAML

## Dependencies

- dag-canvas-foundation (completed)
- task-configuration-drawer (completed)
- yaml-viewer-editor (completed)

## Implementation Notes

### PipelineConfigBar Component

A horizontal toolbar rendered above the DagCanvas:

```
┌──────────────────────────────────────────────────────────────────┐
│  Pipeline: [main ▼]  │  Iterations: [20  ]  │  Parallelism: [1  ]  │  Tasks: planner, coder, evaluator, tester  │
└──────────────────────────────────────────────────────────────────┘
```

- **Pipeline dropdown**: Lists all pipelines from `swarm.yaml`, plus an "All Tasks" option
- **Iterations input**: Number input, updates on blur or Enter
- **Parallelism input**: Number input, updates on blur or Enter
- **Task list**: Shows which tasks belong to the selected pipeline (read-only in v1, editable later)

### Data Flow

1. `yamlParser.ts` already parses the `pipelines` section — use this data
2. When user changes iterations/parallelism, call a new `yamlWriter` function to update the YAML
3. Save via existing `window.fs.writefile()` IPC
4. File watcher picks up the change and refreshes the UI

### Styling

- Use Tailwind classes consistent with existing components
- Dark theme (bg-slate-800/900, text-slate-200)
- Input fields styled like existing TaskDrawer inputs

## Acceptance Criteria

1. A pipeline selector dropdown appears above the DAG canvas listing all pipelines from `swarm.yaml`
2. Selecting a pipeline shows its iterations and parallelism values in editable inputs
3. Changing iterations or parallelism saves the change back to `swarm.yaml`
4. The task list shows which tasks belong to the selected pipeline
5. "All Tasks" option shows all tasks without pipeline filtering
6. App builds successfully with `npm run build`

## Completion Notes

Implemented by agent 56ab1d22.

### What was implemented:
- **PipelineConfigBar component** (`electron/src/renderer/components/PipelineConfigBar.tsx`): Horizontal toolbar above the DAG canvas with pipeline selector dropdown, editable iterations/parallelism number inputs (commit on blur/Enter), task membership display, Configure button, + New Pipeline button, and Run pipeline button.
- **applyPipelineEdits function** (`electron/src/renderer/lib/yamlWriter.ts`): New function to update pipeline iterations, parallelism, and tasks in the compose file, with a `deletePipeline` helper.
- **PipelinePanel component** (`electron/src/renderer/components/PipelinePanel.tsx`): Full pipeline editing panel in the right sidebar with name input (creation mode), iterations/parallelism inputs, task checkboxes with select all/none, save and delete functionality.
- **App.tsx integration**: Added `activePipeline` state, `currentCompose` memo, pipeline update/save/delete/edit/create handlers, PipelineConfigBar rendered above DAG, PipelinePanel in right sidebar, dynamic pipeline commands in command palette.
- **DagCanvas pipeline filtering**: Nodes not in the active pipeline are dimmed to 35% opacity when a pipeline is selected.
- All acceptance criteria met. Build passes with `npm run build`.
