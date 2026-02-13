# Fix: TaskDrawer form resets continuously in creation mode

## Issue

In `TaskDrawer.tsx`, the `taskDef` variable on line 47 is computed as:
```ts
const taskDef = compose.tasks[taskName] ?? {}
```

In creation mode (`taskName === ''`), `compose.tasks['']` is `undefined`, so `taskDef` is always `{}` — a **new object reference on every render**.

The reset `useEffect` on lines 70-79 has `taskDef` in its dependency array:
```ts
useEffect(() => {
  setNewName('')
  setNameError(null)
  setPromptType(getPromptType(taskDef))
  // ... resets all form fields
}, [taskName, taskDef])
```

Since `{} !== {}` (Object.is comparison), this effect fires on every re-render, continuously resetting the form state. This means **any state change in creation mode (typing a name, changing prompt type, etc.) triggers the reset effect**, clearing the form.

## Which Files Need Changes

- `electron/src/renderer/components/TaskDrawer.tsx`

## How to Fix

Stabilize the `taskDef` reference in creation mode. Replace:

```ts
const taskDef = compose.tasks[taskName] ?? {}
```

With:

```ts
const EMPTY_TASK_DEF: TaskDef = {}
// ... outside the component, or use useMemo:
const taskDef = useMemo(
  () => compose.tasks[taskName] ?? EMPTY_TASK_DEF,
  [compose.tasks, taskName]
)
```

The simplest fix is to hoist an empty `TaskDef` constant outside the component so it's a stable reference:

```ts
const EMPTY_TASK: TaskDef = {}

export default function TaskDrawer(...) {
  const taskDef = compose.tasks[taskName] ?? EMPTY_TASK
  // ...
}
```

## Severity

**High** — this makes new task creation non-functional (form fields get reset on every keystroke).
