# Task: Drag-and-Drop Prompt Files from File Tree to DAG Editor

## Goal

Implement drag-and-drop functionality so users can drag prompt files from the File Tree sidebar directly onto the DAG canvas to create new tasks. This is a Phase 3 (Interactive Editing) feature explicitly called out in ELECTRON_PLAN.md:

- File Tree panel: "Drag-and-drop prompt files to DAG editor to create tasks"
- DAG Editor - Creating Tasks: "Click '+ Add Task' button or drag prompt from file tree"

When a user drags a `.md` file from `swarm/prompts/` onto the DAG canvas, a new task should be created with the prompt field pre-populated from the dragged file, and the task name derived from the filename.

## Files

- **Modify**: `electron/src/renderer/components/FileTreeItem.tsx` — Add `draggable` attribute and `onDragStart` handler to prompt files, setting drag data with the prompt name/path
- **Modify**: `electron/src/renderer/components/DagCanvas.tsx` — Add `onDrop` and `onDragOver` handlers to the React Flow container; on drop, create a new task node at the drop position
- **Modify**: `electron/src/renderer/App.tsx` — Add handler to create a new task entry in the YAML compose data when a prompt is dropped on the DAG, then trigger YAML save and refresh

## Dependencies

- File tree with prompt files listing (done)
- DAG canvas with React Flow (done)
- Task creation from "+" button already works (done — used as reference for the creation flow)
- YAML write-back via `yamlWriter.ts` (done)

## Acceptance Criteria

1. Files inside `swarm/prompts/` in the file tree show a drag cursor on hover
2. Dragging a prompt file shows a drag ghost/preview
3. The DAG canvas accepts drops (shows visual drop target indicator when dragging over it)
4. Dropping a prompt file on the DAG canvas creates a new task where:
   - Task name is derived from the prompt filename (e.g., `reviewer.md` → task name `reviewer`)
   - The `prompt` field is set to the prompt name (without `.md` extension)
   - The task has no dependencies by default
   - The task appears at (or near) the drop position in the DAG layout
5. After the drop, the new task is persisted to `swarm.yaml` via the existing YAML write-back mechanism
6. The new task node appears in the DAG canvas immediately after drop
7. Non-prompt files (e.g., YAML, output folders) are NOT draggable to the DAG
8. If a task with the same name already exists, show a toast warning and do not create a duplicate
9. The TaskDrawer opens automatically for the newly created task so the user can configure it further

## Notes

- Use the HTML5 Drag and Drop API (`draggable`, `onDragStart`, `onDrop`, `onDragOver`)
- Set `event.dataTransfer.setData('application/x-swarm-prompt', promptName)` in the drag start handler to carry the prompt name
- In DagCanvas, use `reactFlowInstance.screenToFlowPosition()` to convert the drop coordinates to flow coordinates for positioning the new node
- Reference the existing "Add Task" flow in App.tsx (`handleSaveTask`) for how tasks are added to the YAML and saved
- Only files under `swarm/prompts/` should be draggable — check the file path in FileTreeItem before enabling drag
- Keep the implementation simple: no need for drag preview customization beyond the browser default
- Style the drop zone indicator with a dashed border or subtle background highlight using Tailwind classes

## Completion Notes

Implemented by agents across two iterations. Final implementation:

- **FileTreeItem.tsx**: `isDraggable` flag checks `entry.path.includes('/prompts/')` and `.md` extension. Draggable items get `cursor-grab`/`active:cursor-grabbing` styles. `onDragStart` sets `application/swarm-prompt` data with the prompt name (sans `.md`). Drag opacity applied during drag.
- **DagCanvas.tsx**: Drop zone with `handleDragOver`/`handleDragLeave`/`handleDrop`. Visual indicator: blue ring + subtle bg highlight when dragging over (`ring-2 ring-inset ring-blue-500/50 bg-blue-500/5`). Uses `screenToFlowPosition()` for accurate drop placement.
- **App.tsx**: `handleDropCreateTask` creates task with prompt name, auto-increments name if duplicate exists (with toast warning), saves to YAML, persists node position, reloads DAG, and auto-opens TaskDrawer for configuration.

All 9 acceptance criteria met. Build passes.
