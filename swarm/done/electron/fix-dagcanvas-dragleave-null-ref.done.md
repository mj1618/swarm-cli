# Fix DagCanvas handleDragLeave Null Reference

## Goal

Fix a potential runtime crash in `DagCanvas.tsx` when `e.relatedTarget` is `null` during drag-leave events. This occurs when the user drags a file outside the browser window.

## Files

- **Modify**: `electron/src/renderer/components/DagCanvas.tsx`

## Dependencies

None

## Implementation Details

In the `handleDragLeave` callback at line 630-635, the code casts `e.relatedTarget` to `globalThis.Node` and passes it to `.contains()`. When the user drags off the window, `e.relatedTarget` is `null`, causing a runtime error.

### Current Code (line 632)
```typescript
if (e.currentTarget === e.target || !e.currentTarget.contains(e.relatedTarget as globalThis.Node)) {
```

### Fix
Add a null check before the `.contains()` call:
```typescript
if (e.currentTarget === e.target || !e.relatedTarget || !e.currentTarget.contains(e.relatedTarget as globalThis.Node)) {
```

## Acceptance Criteria

1. No runtime error when dragging a file from the file tree and moving it outside the window
2. Drag-and-drop to create tasks still works correctly within the DAG canvas
3. The drag-over visual indicator clears appropriately when leaving the canvas area
4. TypeScript compiles without errors (`npm run typecheck`)
5. App builds successfully (`npm run build`)

## Notes

This is a Phase 5 polish fix. The bug is subtle but can cause unexpected crashes during normal drag-and-drop workflows.

---

## Completion Notes

**Completed by agent 92672fc3**

Added null check for `e.relatedTarget` in the `handleDragLeave` callback in `DagCanvas.tsx` at line 632. When `e.relatedTarget` is null (which happens when dragging outside the browser window), the condition now short-circuits before calling `.contains()`, preventing a runtime error.

- TypeScript compiles without errors
- App builds successfully
