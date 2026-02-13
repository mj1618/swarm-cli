# Fix: Spurious toasts on app startup

## Issue

When the app first loads, `App.tsx` reads agent state via `window.state.read()` and sets `agents` state. This triggers the toast detection `useEffect` (line 215), which compares against `prevAgentsRef` — which is initially an empty Map. Every running agent is treated as "new" and fires a "started" toast.

If 5 agents are already running when the user opens the app, they'd see 5 spurious "Agent started" toasts immediately.

## Files to Change

- `electron/src/renderer/App.tsx`

## How to Fix

Add a `isInitialLoadRef` flag (useRef, starts `true`) to skip toast generation on the first agent state update:

```tsx
const isInitialLoadRef = useRef(true)

// In the toast detection useEffect:
useEffect(() => {
  if (isInitialLoadRef.current) {
    // On first load, just seed the previous state map without firing toasts
    prevAgentsRef.current = new Map(agents.map(a => [a.id, a]))
    if (agents.length > 0) {
      isInitialLoadRef.current = false
    }
    return
  }
  // ... rest of existing transition detection logic
}, [agents, addToast])
```

This ensures:
1. First state load seeds `prevAgentsRef` silently (no toasts)
2. Subsequent updates detect real transitions and fire appropriate toasts
3. The flag only flips after agents are actually loaded (handles empty initial state)
