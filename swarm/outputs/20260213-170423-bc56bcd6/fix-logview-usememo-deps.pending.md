# Fix: LogView useMemo dependencies use unstable references

## Issue

In `LogView.tsx`, line 97:
```ts
const lines = content.split('\n')
```

This creates a new array on every render. The `filteredLines` and `matchCount` useMemo hooks (lines 101 and 111) both depend on `lines`, making the memoization ineffective since `lines` is always a new reference.

## Which Files Need Changes

- `electron/src/renderer/components/LogView.tsx`

## How to Fix

Memoize the `lines` array:

```ts
const lines = useMemo(() => content.split('\n'), [content])
```

This ensures `filteredLines` and `matchCount` only recompute when `content` actually changes.

## Severity

**Low** — This is a performance optimization, not a functional bug. The component renders correctly, just does unnecessary recomputation on re-renders. With large log files this could cause jank.
