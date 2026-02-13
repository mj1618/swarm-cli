# Task: Add Missing Command Palette Commands

**Phase:** 5 - Polish
**Priority:** Medium

## Goal

The command palette (Cmd+K) is missing two commands specified in the ELECTRON_PLAN.md:

1. **"Toggle console"** — show/hide the bottom console panel
2. **"Fit DAG to view"** — triggers React Flow's fitView to center the DAG

## What to Build

### 1. Toggle Console

Add state in `App.tsx` to track whether the console panel is visible:
```tsx
const [consoleVisible, setConsoleVisible] = useState(true)
```

Conditionally render the console panel based on this state, and add a command:
```tsx
{ id: 'toggle-console', name: 'Toggle console', description: 'Show/hide the bottom console panel', action: () => setConsoleVisible(prev => !prev) }
```

### 2. Fit DAG to View

This requires accessing the React Flow `fitView` function from outside the `DagCanvas` component. Options:
- Use a ref or callback prop on `DagCanvas` to expose `fitView`
- Use a shared event pattern (e.g., a simple custom event)

Add a command:
```tsx
{ id: 'fit-dag', name: 'Fit DAG to view', description: 'Center and fit the DAG in the viewport', action: () => { /* trigger fitView */ } }
```

## Files

- `electron/src/renderer/App.tsx` — Add console toggle state and both commands
- `electron/src/renderer/components/DagCanvas.tsx` — Expose fitView via ref/callback

## Acceptance Criteria

1. "Toggle console" command hides/shows the bottom console panel
2. "Fit DAG to view" command centers the DAG in the viewport
3. Both commands appear in the command palette search results
4. App builds with `npm run build`
