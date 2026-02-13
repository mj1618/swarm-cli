# Task: Add Toast Notifications for Save Failures

## Goal

When task editing or dependency creation fails to save, the user currently gets no feedback — errors are only logged to `console.error`. Add toast notifications so the user knows when save operations fail.

## Phase

Phase 2: Bug Fix / UX Improvement

## Files to Modify

1. **`electron/src/renderer/App.tsx`** — Add `addToast('error', ...)` calls in `handleSaveTask` and `handleAddDependency` when `result.error` is truthy

## Dependencies

- Toast notification system exists (done)

## Implementation Details

### 1. handleSaveTask (App.tsx ~line 96-100)

Currently:
```tsx
if (result.error) {
  console.error('Failed to save:', result.error)
  return
}
```

Should add:
```tsx
if (result.error) {
  addToast('error', `Failed to save task: ${result.error}`)
  return
}
```

### 2. handleAddDependency (App.tsx ~line 136-140)

Currently:
```tsx
if (result.error) {
  console.error('Failed to save dependency:', result.error)
  return
}
```

Should add:
```tsx
if (result.error) {
  addToast('error', `Failed to save dependency: ${result.error}`)
  return
}
```

Note: `addToast` is already available in the App component scope from the `useToasts()` hook, but `handleSaveTask` and `handleAddDependency` don't include it in their dependency arrays. They'll need `addToast` added to their `useCallback` dependency arrays.

## Acceptance Criteria

1. When saving a task fails, a red error toast appears with the error message
2. When saving a dependency fails, a red error toast appears with the error message
3. No regressions to existing toast behavior
4. TypeScript compiles without errors
