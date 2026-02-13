# Task: Task Configuration Drawer

**Phase:** 3 — Interactive Editing
**Priority:** High (first item in Phase 3)

## Goal

Add a slide-out drawer component that appears when a user clicks on a task node in the DAG canvas. The drawer displays the task's full configuration and allows editing fields. Changes are written back to the `swarm.yaml` file.

This is the foundational piece for Phase 3 — interactive editing must start with a way to view and modify task properties before drag-and-drop creation or visual dependency wiring.

## Files

### Create
- `electron/src/renderer/components/TaskDrawer.tsx` — Slide-out drawer panel with form fields for task configuration:
  - Task name (read-only header)
  - Prompt source: radio group selecting between `prompt` (from prompts/), `prompt-file` (file path), or `prompt-string` (inline textarea)
  - Model: dropdown with options `inherit`, `opus`, `sonnet`, `haiku`
  - Prefix: textarea
  - Suffix: textarea
  - Dependencies: list of `{task, condition}` pairs with condition dropdowns (`success | failure | any | always`)

### Modify
- `electron/src/renderer/components/DagCanvas.tsx` — Add `onNodeClick` handler that opens the TaskDrawer with the clicked task's data
- `electron/src/renderer/components/TaskNode.tsx` — Make nodes visually indicate they are clickable (cursor pointer, hover effect)
- `electron/src/renderer/App.tsx` — Add state for selected task and drawer visibility; render TaskDrawer component
- `electron/src/renderer/lib/yamlParser.ts` — Export a `serializeCompose()` function that converts the in-memory task graph back to valid YAML
- `electron/src/main/index.ts` — Add `fs:writefile` IPC handler (scoped to swarm/ directory) so the renderer can save changes
- `electron/src/preload/index.ts` — Expose `fs.writeFile` via context bridge

## Dependencies

- Phase 1 complete (file tree, YAML viewer, agent panel) ✅
- Phase 2 complete (DAG canvas with ReactFlow, task nodes, edges) ✅
- DAG canvas and yamlParser already parse swarm.yaml into nodes — this task adds the reverse: editing and writing back

## Acceptance Criteria

1. Clicking a task node in the DAG canvas opens a slide-out drawer on the right side
2. The drawer displays all current task properties (prompt source, model, prefix, suffix, dependencies)
3. Each field is editable with appropriate form controls (dropdowns, textareas, radio buttons)
4. A "Save" button serializes the updated task config back to valid `swarm.yaml` and writes it via IPC
5. The DAG canvas re-renders after save to reflect changes
6. A "Close" / "Cancel" button dismisses the drawer without saving
7. The drawer has smooth slide-in/slide-out animation
8. The `fs:writefile` IPC handler validates the path is within the swarm/ directory

## Notes

- Reference the Task Configuration Panel design in ELECTRON_PLAN.md (lines 113-138)
- The drawer should overlay the right side of the DAG canvas, not replace the Agent Panel
- Use Tailwind for styling, consistent with existing dark theme (`bg-[hsl(222,84%,5%)]` palette)
- The yamlParser.ts `serializeCompose()` should use `js-yaml`'s `dump()` to produce clean YAML
- Condition dropdown values: `success`, `failure`, `any`, `always` — matching `internal/compose/` Go types
- Keep the drawer width around 320-360px to match the design spec
- Consider using `position: fixed` or `absolute` with z-index to overlay properly

## Completion Notes

Implemented the Task Configuration Drawer (Phase 3 - Interactive Editing):

- **TaskDrawer.tsx**: Full editable form with:
  - Prompt source selector (prompt/prompt-file/prompt-string) with tab-style toggle buttons
  - For `prompt` type: dropdown populated from `fs:listprompts` IPC for available prompt files
  - Model dropdown (inherit/opus/sonnet/haiku)
  - Prefix and suffix textareas
  - Dependencies editor with task and condition dropdowns, add/remove controls
  - Save and Cancel buttons with loading state
  - Escape key to close, slide-in animation
  - Accepts `compose: ComposeFile` prop to derive task names and task definition
- **yamlParser.ts**: Added `serializeCompose()` using `js-yaml` `dump()` to convert compose back to YAML; `composeToFlow` now includes `taskDef` in node data and supports `savedPositions`
- **yamlWriter.ts**: Created with `applyTaskEdits()` and `serializeCompose()` utilities
- **App.tsx**: Implemented `handleSaveTask` that serializes updated task config, writes via IPC, and reloads YAML to refresh the DAG; passes `compose` object to TaskDrawer
- **DagCanvas.tsx**: Passes full `ComposeFile` in `onSelectTask` callback; supports node dragging with position persistence
- **TaskNode.tsx**: Shows selected state with ring highlight; hover effects for interactive handles
- **main/index.ts**: Added `fs:writefile` (scoped to swarm/ dir) and `fs:listprompts` IPC handlers
- **preload/index.ts**: Exposed `writefile` and `listprompts` in context bridge with updated `FsAPI` type
- **vite-env.d.ts**: Updated Window type with new IPC methods
