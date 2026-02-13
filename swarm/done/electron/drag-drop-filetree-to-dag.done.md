# Task: Drag-and-Drop Prompt Files from File Tree to DAG Canvas

**Phase:** 3 - Interactive Editing
**Priority:** High (Phase 3 gap — all other Phase 3 features are implemented)

## Goal

Enable users to drag `.md` prompt files from the file tree sidebar and drop them onto the DAG canvas to create new tasks. This is explicitly called out in ELECTRON_PLAN.md under both "Panel 1: File Tree" ("Drag-and-drop prompt files to DAG editor to create tasks") and "Panel 2: DAG Editor / Creating Tasks" ("Click '+ Add Task' button or drag prompt from file tree").

## Files to Modify

1. **`electron/src/renderer/components/FileTreeItem.tsx`** — Add `draggable` attribute and `onDragStart` handler to `.md` files in the `prompts/` directory. Set drag data with the prompt name (filename without extension).

2. **`electron/src/renderer/components/DagCanvas.tsx`** — Add `onDragOver` and `onDrop` handlers to the ReactFlow container. On drop, extract the prompt name from the drag data and call a new `onDropCreateTask` callback prop with the prompt name and drop position.

3. **`electron/src/renderer/App.tsx`** — Implement the `onDropCreateTask` handler that:
   - Derives a task name from the prompt filename (e.g., `planner.md` → task name `planner`)
   - If the task name already exists in compose, append a numeric suffix (`planner-2`)
   - Creates a new task entry with `prompt: <name>` in the compose file
   - Writes the updated YAML via the existing `yamlWriter` utility
   - Opens the TaskDrawer for the newly created task so the user can configure it

## Implementation Details

### FileTreeItem.tsx

Add to the main clickable `<div>` for non-directory `.md` files inside `prompts/`:

```tsx
draggable={!entry.isDirectory && entry.path.includes('/prompts/') && entry.name.endsWith('.md')}
onDragStart={(e) => {
  const promptName = entry.name.replace(/\.md$/, '')
  e.dataTransfer.setData('application/swarm-prompt', promptName)
  e.dataTransfer.setData('text/plain', promptName)
  e.dataTransfer.effectAllowed = 'copy'
}}
```

Add a visual drag indicator (e.g., subtle opacity change via CSS `opacity: 0.5` while dragging).

### DagCanvas.tsx

Add drop zone handlers to the ReactFlow wrapper div:

```tsx
onDragOver={(e) => {
  if (e.dataTransfer.types.includes('application/swarm-prompt')) {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'copy'
  }
}}
onDrop={(e) => {
  e.preventDefault()
  const promptName = e.dataTransfer.getData('application/swarm-prompt')
  if (promptName && onDropCreateTask) {
    const position = screenToFlowPosition({ x: e.clientX, y: e.clientY })
    onDropCreateTask(promptName, position)
  }
}}
```

Add `onDropCreateTask` to the `DagCanvasProps` interface:

```tsx
onDropCreateTask?: (promptName: string, position: { x: number; y: number }) => void
```

Use `useReactFlow().screenToFlowPosition` to convert screen coordinates to flow coordinates for the new node position.

### App.tsx

Add handler:

```tsx
const handleDropCreateTask = useCallback(async (promptName: string, position: { x: number; y: number }) => {
  // Determine unique task name
  let taskName = promptName
  let counter = 2
  while (compose?.tasks?.[taskName]) {
    taskName = `${promptName}-${counter++}`
  }
  // Create task via yamlWriter (same pattern as existing task creation)
  // Save position to savedPositions
  // Open task drawer for the new task
}, [compose, /* other deps */])
```

## Dependencies

- File tree component (completed)
- DAG canvas with React Flow (completed)
- Task creation via yamlWriter (completed)
- TaskDrawer for editing new tasks (completed)

## Acceptance Criteria

1. `.md` files inside the `prompts/` directory show a drag cursor and can be dragged
2. Non-prompt files and directories are NOT draggable
3. Dropping a prompt file on the DAG canvas creates a new task with `prompt: <filename>` in `swarm.yaml`
4. The new task node appears at the drop position on the canvas
5. If a task with the same name already exists, a numeric suffix is appended (e.g., `planner-2`)
6. The TaskDrawer opens for the newly created task after drop
7. App builds successfully with `npm run build`

## Notes

- Use the HTML5 Drag and Drop API (native browser), not a library — keeps it simple and works well with React Flow's existing drag handling
- The custom MIME type `application/swarm-prompt` prevents accidental drops from other drag sources
- React Flow's `screenToFlowPosition` properly accounts for zoom/pan state

## Completion Notes

Implemented drag-and-drop from file tree to DAG canvas across three files:

- **FileTreeItem.tsx**: Added `draggable` attribute and `onDragStart`/`onDragEnd` handlers for `.md` files inside `prompts/` directories. Includes visual opacity feedback during drag.
- **DagCanvas.tsx**: Added `onDropCreateTask` prop, `onDragOver` handler (with custom MIME type check), and `onDrop` handler that converts screen coordinates to flow coordinates via `screenToFlowPosition`.
- **App.tsx**: Implemented `handleDropCreateTask` that derives a unique task name (with numeric suffix for duplicates), writes the new task to `swarm.yaml`, saves the drop position to localStorage, reloads the YAML, and opens the TaskDrawer for the new task.

All acceptance criteria met. Build passes successfully.
