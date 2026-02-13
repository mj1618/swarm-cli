# Task: Persist DAG Viewport Zoom and Pan State

**Phase:** 5 - Polish (Enhancement)
**Priority:** Low

## Goal

Persist the DAG canvas viewport state (zoom level and pan position) so users don't lose their view when switching files or reopening the app. Currently, node positions are saved to localStorage, but the viewport resets to a default zoom/pan on every load.

## Files to Modify

- **`electron/src/renderer/components/DagCanvas.tsx`**
  - Save viewport state (zoom, x, y) to localStorage on `onMoveEnd` or debounced `onMove` events
  - Restore viewport state on mount via React Flow's `defaultViewport` prop or `setViewport()` from `useReactFlow`
  - Use the same localStorage key pattern as node positions: `swarm-dag-viewport:{filePath}`
  
- **`electron/src/renderer/App.tsx`** (possibly)
  - Pass the active YAML file path to DagCanvas if not already available for the localStorage key

## Dependencies

- DAG Canvas with React Flow (completed)
- Node position persistence (completed - uses `swarm-dag-positions:{filePath}` pattern)

## Implementation Notes

### Saving Viewport State

React Flow provides an `onMoveEnd` callback that fires when the user finishes panning/zooming. Use this to persist:

```typescript
const { getViewport, setViewport } = useReactFlow()

const handleMoveEnd = useCallback((event, viewport) => {
  localStorage.setItem(
    `swarm-dag-viewport:${activeYamlPath ?? 'swarm/swarm.yaml'}`,
    JSON.stringify({ x: viewport.x, y: viewport.y, zoom: viewport.zoom })
  )
}, [activeYamlPath])
```

### Restoring Viewport State

On component mount, check localStorage for saved viewport and apply it:

```typescript
const savedViewport = useMemo(() => {
  try {
    const raw = localStorage.getItem(`swarm-dag-viewport:${activeYamlPath ?? 'swarm/swarm.yaml'}`)
    if (raw) return JSON.parse(raw)
  } catch { /* ignore */ }
  return null
}, [activeYamlPath])

// Then use defaultViewport prop on ReactFlow if savedViewport exists
// Or call setViewport() in a useEffect after initial render
```

### Key Considerations

- Don't persist viewport when the user explicitly clicks "Fit to View" or "Reset Layout" - these should also clear the saved viewport
- Consider debouncing the save to avoid excessive localStorage writes during smooth panning
- The existing `fitView` functionality should take precedence over restored viewport when nodes change significantly (e.g., new tasks added)

## Acceptance Criteria

1. Zooming and panning the DAG canvas saves the viewport state to localStorage
2. Reopening the same YAML file restores the previous zoom level and pan position
3. Switching between different YAML files maintains separate viewport states
4. Clicking "Reset Layout" clears both node positions AND viewport state
5. The "Fit to View" (F key) command works regardless of saved viewport
6. App builds successfully with `npm run build`
7. No performance degradation (viewport saves are debounced or use onMoveEnd)

## Notes

- This follows the existing pattern for node position persistence in App.tsx (`loadPositions`, `handlePositionsChange`, `handleResetLayout`)
- The localStorage key should include the file path to support different viewports for different YAML files
- React Flow's `onMoveEnd` is preferred over `onMove` for performance (fewer writes)
- Default to `fitView` behavior if no saved viewport exists (current behavior)
