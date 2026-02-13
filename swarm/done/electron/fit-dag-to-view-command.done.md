# Task: Add "Fit DAG to View" Command Palette Command

**Phase:** 5 - Polish
**Priority:** Medium

## Goal

Add a "Fit DAG to view" command to the command palette (Cmd+K) that centers and fits the DAG in the viewport. This is explicitly specified in ELECTRON_PLAN.md under "Command Palette (Cmd+K)" as a quick action. The "Toggle console" command from the same missing-commands list is already implemented, but "Fit DAG to view" is not.

## Files

### Modify
- **`electron/src/renderer/components/DagCanvas.tsx`** — Expose a `fitView` callback to the parent via a new `onFitViewReady` prop (or via `useImperativeHandle` + `forwardRef`). The component already imports `useReactFlow` and can destructure `fitView` from it.
- **`electron/src/renderer/App.tsx`** — Store the `fitView` callback in a ref. Add a new command to `paletteCommands` with `id: 'fit-dag'`.

### Implementation Approach

1. In `DagCanvas.tsx`, add an optional prop `onFitViewReady?: (fitView: () => void) => void`. Inside the component, destructure `fitView` from `useReactFlow()` and call `onFitViewReady(() => fitView({ padding: 0.3 }))` in a `useEffect`.

2. In `App.tsx`, create a `fitViewRef = useRef<(() => void) | null>(null)` and pass `onFitViewReady={(fn) => { fitViewRef.current = fn }}` to `DagCanvas`.

3. Add command to `paletteCommands`:
```tsx
{
  id: 'fit-dag',
  name: 'Fit DAG to view',
  description: 'Center and fit the DAG in the viewport',
  action: () => { fitViewRef.current?.() },
}
```

## Dependencies

- DagCanvas component exists with React Flow (done)
- Command palette exists (done)

## Acceptance Criteria

1. "Fit DAG to view" appears when searching in the command palette (Cmd+K)
2. Selecting the command centers and fits all DAG nodes within the viewport with padding
3. The command works correctly when viewing both the default `swarm.yaml` and any selected YAML file
4. The command is a no-op (doesn't crash) when no DAG is displayed (e.g., when viewing a non-YAML file or settings)
5. App builds successfully with `npm run build`

## Notes

- Use `fitView({ padding: 0.3 })` to match the existing `fitViewOptions` prop on the `ReactFlow` component (DagCanvas.tsx line 434)
- The `useReactFlow` hook is already imported and used in DagCanvas (line 11, 236) — just add `fitView` to the destructured variables
- This is a small, self-contained change touching only two files

## Completion Notes

Implemented as specified. Changes:
- `DagCanvas.tsx`: Added `onFitViewReady` optional prop, destructured `fitView` from `useReactFlow()`, and exposed it via a `useEffect` callback with `padding: 0.3`.
- `App.tsx`: Added `fitViewRef` to store the callback, passed `onFitViewReady` to `DagCanvas`, and added "Fit DAG to view" command to `paletteCommands` with id `fit-dag`.
- The command is a safe no-op when no DAG is displayed since `fitViewRef.current` will be null.
- Build verified successfully.
