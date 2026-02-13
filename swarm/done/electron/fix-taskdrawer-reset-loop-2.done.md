# Fix TaskDrawer Form Reset Loop in Creation Mode

## Goal

Fix the TaskDrawer component so that creating a new task works correctly. Currently, when the TaskDrawer opens in creation mode (no existing task selected), the form resets in an infinite loop because a new empty `taskDef` object is created on every render, causing the `useEffect` that syncs form state from `taskDef` to fire repeatedly.

## Files

- `electron/src/renderer/components/TaskDrawer.tsx` — Main fix location

## Dependencies

None — this is a standalone bug fix.

## Acceptance Criteria

1. Opening the TaskDrawer in "create new task" mode (no existing task) displays a stable form that does not flicker or reset
2. The user can type a task name, select a prompt source, configure model/prefix/suffix, and save without the form resetting mid-edit
3. Opening the TaskDrawer with an existing task still correctly populates the form fields
4. Switching between different existing tasks still correctly updates the form fields
5. TypeScript compiles with no errors (`npx tsc --noEmit` passes)
6. The app runs without console warnings about re-renders or state update loops

## Notes

The root cause is that the `taskDef` prop (or a derived default object) is a new reference on every render when no task is selected. This causes any `useEffect` that depends on `taskDef` to re-fire, resetting form state.

**Fix approach**: Hoist the empty/default TaskDef to a module-level constant (outside the component), or wrap it in `useMemo` with stable dependencies, so the reference stays the same across renders when in creation mode. For example:

```tsx
// At module level, outside the component:
const EMPTY_TASK_DEF: TaskDef = {
  prompt: '',
  'prompt-file': '',
  'prompt-string': '',
  model: '',
  prefix: '',
  suffix: '',
  depends_on: [],
}

// Then inside the component, use it as the fallback:
const effectiveTaskDef = taskDef ?? EMPTY_TASK_DEF
```

This ensures referential stability when no task is selected, preventing the reset loop.

Reference: This was identified as a **high severity** bug in iteration 7 of the pipeline — it makes the "Add Task" feature effectively non-functional.
