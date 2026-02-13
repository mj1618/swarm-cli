# Task: Fix DagCanvas handleDragLeave null reference

**Phase:** 5 - Polish
**Priority:** High

## Goal

Fix a potential runtime crash in `DagCanvas.tsx` when `e.relatedTarget` is `null` during drag-leave events. This happens when the user drags a file out of the browser window entirely.

## Files to Modify

1. **`electron/src/renderer/components/DagCanvas.tsx`** — Add null check for `e.relatedTarget` in the drag leave handler

## Dependencies

- None

## Implementation Details

In the `handleDragLeave` handler (around line 404), the code casts `e.relatedTarget` as `globalThis.Node` and passes it to `.contains()`. When the user drags outside the window, `e.relatedTarget` is `null`, which can cause a runtime error.

### Current Code
```typescript
if (e.currentTarget === e.target || !e.currentTarget.contains(e.relatedTarget as globalThis.Node)) {
```

### Fix
Add a null check before calling `.contains()`:
```typescript
if (e.currentTarget === e.target || !e.relatedTarget || !e.currentTarget.contains(e.relatedTarget as globalThis.Node)) {
```

## Acceptance Criteria

1. No runtime error when dragging a file out of the DagCanvas and off the window
2. TypeScript compiles without errors
3. App builds successfully with `npm run build`
