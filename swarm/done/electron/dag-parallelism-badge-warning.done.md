# Task: Add Parallelism Warning Badge to DAG Task Nodes

**Phase:** 3 - Interactive Editing (Validation Feedback)
**Priority:** Low

## Goal

Add yellow warning badges to task nodes in the DAG canvas when they belong to a pipeline with `parallelism > 1`. This completes the validation feedback feature from ELECTRON_PLAN.md:

> "Yellow badges for tasks with parallelism inside pipelines"

This visual indicator warns users that tasks may run concurrently, which is important for tasks that modify shared resources or depend on execution order.

## Files

### Modify
- **`electron/src/renderer/lib/dagValidation.ts`** — Add `parallelTasks: Set<string>` to the `ValidationResult` interface. Implement `detectParallelTasks()` function that identifies tasks belonging to pipelines where `parallelism > 1`.

- **`electron/src/renderer/lib/yamlParser.ts`** — Add `isInParallelPipeline?: boolean` to the `TaskNodeData` interface.

- **`electron/src/renderer/components/DagCanvas.tsx`** — Pass `isInParallelPipeline` to node data based on the new validation result. Update the validation summary panel to show parallel task warnings if needed.

- **`electron/src/renderer/components/TaskNode.tsx`** — Render a small yellow "⚡" or "∥" badge in the corner when `isInParallelPipeline` is true. Add tooltip: "This task is in a pipeline with parallelism > 1 and may run concurrently".

## Dependencies

- DAG validation module (completed — `dagValidation.ts`)
- TaskNode validation styling (completed — already has `isInCycle` and `isOrphan` badge patterns)
- Pipeline configuration (completed — pipelines already have `parallelism` field)

## Implementation Notes

### dagValidation.ts

Add a new detection function:

```typescript
function detectParallelTasks(compose: ComposeFile): Set<string> {
  const parallelTasks = new Set<string>()
  const pipelines = compose.pipelines ?? {}
  
  for (const [, pipeline] of Object.entries(pipelines)) {
    if ((pipeline.parallelism ?? 1) > 1) {
      for (const taskName of pipeline.tasks ?? []) {
        parallelTasks.add(taskName)
      }
    }
  }
  
  return parallelTasks
}
```

Update `validateDag()` to call this and include the result.

### TaskNode.tsx

Add the badge next to or below the orphan badge position. Use a distinct icon (⚡ for parallel/concurrent or ∥ for parallel lines) and a different shade of yellow/orange to distinguish from the orphan warning:

```tsx
{isInParallelPipeline && (
  <div 
    className="absolute -top-1.5 -left-1.5 w-4 h-4 rounded-full bg-yellow-400 flex items-center justify-center text-[8px] font-bold text-black z-10"
    title="This task may run concurrently (pipeline has parallelism > 1)"
  >
    ⚡
  </div>
)}
```

### Visual Distinction

- **Orphan badge** (existing): amber/orange "!" in top-right corner
- **Parallel badge** (new): yellow "⚡" in top-left corner

This allows both badges to be visible simultaneously if a task is both orphaned and in a parallel pipeline.

## Acceptance Criteria

1. Tasks in a pipeline with `parallelism: 2` or higher show a yellow badge icon
2. Tasks in pipelines with `parallelism: 1` (or unspecified) do not show the badge
3. Hovering over the badge shows a tooltip explaining concurrent execution
4. The badge does not appear for tasks not assigned to any pipeline
5. The badge coexists properly with cycle (red) and orphan (amber) indicators
6. Changing a pipeline's parallelism setting updates the badges in real-time
7. The project builds successfully: `cd electron && npm run build`

## Notes

- This is a low-priority polish feature — the UI is fully functional without it
- The warning is informational only (not an error) — parallel execution is valid, just noteworthy
- Consider making the badge subtle to avoid visual clutter — smaller size or lower opacity than error badges
- Reference the existing orphan badge implementation in TaskNode.tsx for styling patterns

---

## Completion Notes

**Completed by:** Agent a36c577e  
**Date:** 2026-02-13

### Implementation Summary

All acceptance criteria have been met:

1. **dagValidation.ts**: Added `parallelTasks: Set<string>` to `ValidationResult` interface and implemented `detectParallelTasks()` function that scans pipelines for `parallelism > 1` and collects their tasks.

2. **yamlParser.ts**: Added `isInParallelPipeline?: boolean` to `TaskNodeData` interface.

3. **DagCanvas.tsx**: Updated the validation injection to include `isInParallelPipeline: validation.parallelTasks.has(node.id)` in node data.

4. **TaskNode.tsx**: Added yellow "⚡" badge in the top-left corner (distinct from the amber orphan badge in top-right) with a tooltip explaining concurrent execution.

### Build Verification

- `npm run build` completed successfully
- `tsc --noEmit` passed with no errors
