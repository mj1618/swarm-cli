# Task: Add Task Creation UI to DAG Canvas

## Goal

Add an "+ Add Task" button to the DAG canvas and fix the TaskDrawer to support creating new tasks (not just editing existing ones). Currently, the DAG canvas has no way for users to create tasks visually — the "Create new task" command palette action passes an empty task name which doesn't work properly. This completes a core Phase 3 (Interactive Editing) feature described in the ELECTRON_PLAN.md design spec (lines 97-98, 104-105, 429).

## Phase

Phase 3: Interactive Editing — "Drag-and-drop task creation" / "+ Add Task" button

## Files to Modify

1. **`electron/src/renderer/components/DagCanvas.tsx`** — Add an "+ Add Task" button using React Flow's `<Panel>` component (bottom-left position, next to the existing "Reset Layout" button at top-right)
2. **`electron/src/renderer/components/TaskDrawer.tsx`** — Support new task creation mode:
   - When `taskName` is empty, show an editable text input for the task name
   - Validate that the new task name is non-empty, uses valid characters (lowercase alphanumeric + hyphens), and doesn't conflict with existing task names
   - Header should say "New Task" instead of an empty string
3. **`electron/src/renderer/App.tsx`** — Wire up the "+ Add Task" button from DagCanvas to open the TaskDrawer in creation mode. The `handleSaveTask` function needs to handle the case where `taskName` differs from `selectedTask.name` (i.e., a new name was provided)

## Dependencies

- DagCanvas component exists and works (done)
- TaskDrawer component exists with full edit form (done)
- YAML serialization and write-back works (done)
- Command palette "Create new task" action exists but needs the same TaskDrawer fix (will be fixed by this task)

## Implementation Details

### 1. DagCanvas: "+ Add Task" button

Add a React Flow `<Panel position="bottom-left">` with an "+ Add Task" button. Expose a new prop `onCreateTask?: () => void` that fires when clicked. Style it consistently with the existing "Reset Layout" button.

```tsx
{onCreateTask && (
  <Panel position="bottom-left">
    <button
      onClick={onCreateTask}
      className="px-3 py-1.5 text-xs font-medium rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
    >
      + Add Task
    </button>
  </Panel>
)}
```

### 2. TaskDrawer: New task name input

When `taskName` is empty (creation mode):
- Show a text input field for the task name at the top of the form, with label "Task Name"
- Track the name in local state: `const [newName, setNewName] = useState('')`
- Validate: non-empty, matches `/^[a-z][a-z0-9-]*$/`, not already in `compose.tasks`
- Show inline validation error text below the input if invalid
- On save, pass the new name to `onSave(newName, updatedDef)` instead of the empty `taskName`

### 3. App.tsx: Wire up creation flow

- Add `onCreateTask` prop to DagCanvas, triggered by a callback that creates the empty task selection:
  ```tsx
  const handleCreateTask = useCallback(() => {
    const yamlContent = selectedIsYaml && selectedFile ? selectedYamlContent : defaultYamlContent
    if (!yamlContent) return
    const compose = parseComposeFile(yamlContent)
    setSelectedTask({ name: '', def: { prompt: '' }, compose })
  }, [selectedIsYaml, selectedFile, selectedYamlContent, defaultYamlContent])
  ```
- Pass `onCreateTask={handleCreateTask}` to `<DagCanvas>`
- In `handleSaveTask`, use the `taskName` param from `onSave(taskName, def)` directly as the key (this already works since TaskDrawer will now pass the user-entered name)

## Acceptance Criteria

1. A blue "+ Add Task" button appears in the bottom-left corner of the DAG canvas
2. Clicking the button opens the TaskDrawer with an editable task name field
3. The task name field validates input (required, lowercase alphanumeric + hyphens, no duplicates)
4. Saving a new task writes it to swarm.yaml and the DAG updates to show the new node
5. The command palette "Create new task" action also opens the drawer with the name field
6. Editing an existing task (by clicking a node) does NOT show the name field — it remains read-only in the header
7. No regressions: existing task editing, dependency creation, and DAG interactions still work
8. TypeScript compiles without errors

## Notes

- The ELECTRON_PLAN.md also shows a "+ Add Pipeline" button. That's a separate, more complex task involving pipeline configuration UI. Keep this task focused on task creation only.
- The task name validation pattern (`/^[a-z][a-z0-9-]*$/`) matches what swarm-cli expects for task names in swarm.yaml
- Consider auto-focusing the task name input when in creation mode for better UX
- The existing "Create new task" command palette entry (App.tsx:301-309) already passes `name: ''`, so it will automatically benefit from the TaskDrawer fix

## Completion Notes

Implemented by agent 9621a8e0.

**Changes made:**
1. **DagCanvas.tsx**: Added `onCreateTask` prop and a blue "+ Add Task" button in a `<Panel position="bottom-left">` that fires the callback when clicked.
2. **TaskDrawer.tsx**: Added creation mode (`isCreating = taskName === ''`) with:
   - Editable task name input field with auto-focus
   - Validation: required, must match `/^[a-z][a-z0-9-]*$/`, no duplicate names
   - Inline error messages below the input
   - Header shows "New Task" instead of empty string
   - Save passes the user-entered name to `onSave()`
3. **App.tsx**: Added `handleCreateTask` callback that parses current YAML and opens TaskDrawer with `name: ''`. Passed `onCreateTask={handleCreateTask}` to DagCanvas. The existing command palette "Create new task" action also benefits from the TaskDrawer fix.

All acceptance criteria met. TypeScript compiles without errors.
