# Fix: TaskDrawer form resets continuously in creation mode

## Goal

Fix a high-severity bug in `TaskDrawer.tsx` where the form continuously resets in creation mode (when creating a new task). The user cannot type a task name, select a prompt, or fill in any field because every state change triggers a useEffect that resets all form fields back to defaults.

## Root Cause

In `TaskDrawer.tsx` line 47:
```ts
const taskDef = compose.tasks[taskName] ?? {}
```

When `taskName === ''` (creation mode), `compose.tasks['']` is `undefined`, so `taskDef` becomes `{}` — a **new object reference on every render**.

The reset useEffect on line 70-79 includes `taskDef` in its dependency array:
```ts
useEffect(() => {
  setNewName('')
  setPromptType(getPromptType(taskDef))
  setPromptValue(getPromptValue(taskDef))
  // ... resets all fields
}, [taskName, taskDef])
```

Since `{} !== {}` by reference, this effect fires on every render, resetting all form state.

## Files

- `electron/src/renderer/components/TaskDrawer.tsx` — the only file that needs changes

## Dependencies

None — this is a standalone bug fix.

## Implementation

1. Hoist a stable empty `TaskDef` constant outside the component:
   ```ts
   const EMPTY_TASK: TaskDef = {}
   ```

2. Replace line 47:
   ```ts
   // Before:
   const taskDef = compose.tasks[taskName] ?? {}
   // After:
   const taskDef = compose.tasks[taskName] ?? EMPTY_TASK
   ```

This ensures `taskDef` is a stable reference in creation mode, so the reset useEffect only fires when `taskName` actually changes.

## Acceptance Criteria

1. Opening the "New Task" drawer allows the user to type a task name without it being cleared
2. All form fields (prompt type, prompt value, model, prefix, suffix, dependencies) retain their values while editing
3. The reset effect still works correctly when switching between different existing tasks (taskName changes)
4. The app builds without TypeScript errors: `cd electron && npx tsc --noEmit`

## Notes

- This is referenced in the earlier pending task at `swarm/outputs/20260213-170423-bc56bcd6/fix-taskdrawer-reset-loop.pending.md`
- Severity: **High** — new task creation is effectively non-functional without this fix
- The fix is a one-line change plus one constant declaration

## Completion

Implemented as specified: added `const EMPTY_TASK: TaskDef = {}` outside the component and changed the nullish coalescing fallback from `{}` to `EMPTY_TASK`. This gives a stable reference in creation mode so the reset useEffect only fires when `taskName` actually changes. No TypeScript errors in TaskDrawer.tsx (pre-existing DagCanvas.tsx errors are unrelated).
