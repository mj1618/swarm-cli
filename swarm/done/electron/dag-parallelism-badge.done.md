# Task: Add Yellow Parallelism Badge to DAG Task Nodes

**Phase:** 2 - DAG Visualization (validation feedback enhancement)
**Priority:** Low
**Status:** COMPLETED

## Goal

Add a yellow badge to task nodes in the DAG canvas when the task belongs to a pipeline with `parallelism > 1`. This visual indicator helps users understand that tasks in this pipeline may run concurrently with other tasks.

From ELECTRON_PLAN.md validation feedback section:
> "Yellow badges for tasks with parallelism inside pipelines"

## Implementation Summary

The feature was already implemented across the following files:

### electron/src/renderer/lib/dagValidation.ts

Added `detectParallelTasks()` function (lines 110-123) that:
- Iterates through all pipelines
- Checks if `parallelism > 1`
- Adds all tasks in parallel pipelines to a Set

The `ValidationResult` interface includes `parallelTasks: Set<string>`.

### electron/src/renderer/lib/yamlParser.ts

Added `isInParallelPipeline?: boolean` to `TaskNodeData` interface (line 44).

### electron/src/renderer/components/DagCanvas.tsx

Injects validation data into nodes (line 107):
```typescript
isInParallelPipeline: validation.parallelTasks.has(node.id),
```

### electron/src/renderer/components/TaskNode.tsx

Renders the badge (lines 116-123):
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

## Acceptance Criteria - All Met

1. ✅ Tasks that belong to a pipeline with `parallelism > 1` show a small yellow badge in the top-left corner
2. ✅ The badge has a tooltip: "This task may run concurrently (pipeline has parallelism > 1)"
3. ✅ Tasks in pipelines with `parallelism: 1` or no parallelism specified do NOT show the badge
4. ✅ Tasks not in any pipeline do NOT show the badge
5. ✅ The parallelism badge does not overlap with the orphan warning badge (orphan is on right, parallelism is on left)
6. ✅ The badge is visible but not distracting (small yellow-400 circle with ⚡ icon)
7. ✅ App builds successfully with `npm run build`

## Notes

- Uses ⚡ symbol instead of ⫽ for better visibility at small sizes
- Uses `bg-yellow-400` instead of `bg-yellow-500` for slightly better contrast
- The current swarm.yaml has `parallelism: 6`, so all tasks (planner, implementer, reviewer) will display this badge
