# Task: DAG Validation Feedback

**Phase:** 3 - Interactive Editing
**Priority:** High

## Goal

Add real-time validation to the DAG canvas that detects and visually highlights invalid graph states: dependency cycles and orphaned tasks (tasks with dependencies but not assigned to any pipeline). This implements the "Validation Feedback" feature from ELECTRON_PLAN.md Phase 3:

- Red highlighting on cycles
- Warnings for orphaned tasks (have dependencies but no pipeline)

## Files

### Create
- `electron/src/renderer/lib/dagValidation.ts` — Pure functions for cycle detection (topological sort / DFS) and orphaned task detection. Returns structured validation results.

### Modify
- `electron/src/renderer/components/DagCanvas.tsx` — Run validation on parsed compose data, pass validation state to TaskNode components (e.g. `isInCycle`, `isOrphan`), and highlight edges involved in cycles with red color.
- `electron/src/renderer/components/TaskNode.tsx` — Render red border/glow for tasks in cycles, yellow badge/border for orphaned tasks, and tooltip text explaining the issue.

## Dependencies

- DAG canvas with React Flow (completed — DagCanvas.tsx)
- Task node components (completed — TaskNode.tsx)
- YAML parser with compose file support (completed — yamlParser.ts)
- Pipeline configuration UI (in-progress — PipelineConfigBar.tsx, but validation does not depend on it being fully complete; it only needs access to the parsed `ComposeFile` which is already available)

## Implementation Notes

### dagValidation.ts

Create a validation module with two core functions:

```typescript
interface ValidationResult {
  cycleNodes: Set<string>    // task names involved in cycles
  cycleEdges: Set<string>    // edge IDs (format: "source->target") in cycles
  orphanedTasks: Set<string> // tasks with deps but not in any pipeline
}

function validateDag(compose: ComposeFile): ValidationResult
```

**Cycle detection:**
- Build an adjacency list from task `depends_on` fields
- Use Kahn's algorithm (topological sort via in-degree counting) or iterative DFS
- Any nodes remaining after topological sort are part of a cycle
- Track which edges participate in cycles for edge highlighting

**Orphaned task detection:**
- Collect all tasks that have `depends_on` entries
- Collect all tasks that appear in at least one pipeline's `tasks` array
- Tasks with dependencies that are NOT in any pipeline are "orphaned"

### DagCanvas.tsx changes

- Import and call `validateDag(compose)` in the existing `useMemo` that parses YAML
- Pass `isInCycle` and `isOrphan` boolean props into `TaskNode` via the node data
- For edges in cycles, override the edge style: `{ stroke: 'red', strokeWidth: 2 }` and add an animated property
- Add a small validation summary banner at the top of the canvas (e.g., "⚠ 2 tasks in cycle, 1 orphaned task") using React Flow's `<Panel position="top-left">`

### TaskNode.tsx changes

- If `isInCycle` is true: render a red border (`border-red-500`), add a red glow effect, and show a tooltip on hover: "This task is part of a dependency cycle"
- If `isOrphan` is true: render a yellow/amber border (`border-amber-500`), add a warning badge icon, and show a tooltip: "This task has dependencies but is not in any pipeline"
- Both states can be active simultaneously (a cycled task not in a pipeline)

### Styling

- Cycle highlighting: red border, red edge color, subtle red glow/shadow on the node
- Orphan highlighting: amber/yellow border, small warning triangle badge in corner
- Validation banner: dark background with warning text, dismissible or auto-hidden when valid

## Acceptance Criteria

1. Creating a dependency cycle (A→B→C→A) causes all three task nodes to show red borders/highlights
2. Edges forming the cycle are rendered in red (distinct from the default edge color)
3. A task that has `depends_on` entries but is not listed in any pipeline's `tasks` array shows a yellow/amber warning indicator
4. A validation summary appears at the top of the DAG canvas when issues are detected (e.g., "⚠ Cycle detected involving: A, B, C")
5. Validation updates in real-time as the YAML changes (no manual refresh needed)
6. When all validation issues are resolved, the highlighting and banner disappear
7. Validation does not affect task interaction (clicking, dragging, connecting still work normally)
8. The validation module has no side effects — it's a pure function of the ComposeFile
9. The project builds successfully: `cd electron && npx tsc --noEmit` passes with zero errors

## Completion Notes

Implemented by agent 5e4dd214 on iteration 6.

### What was implemented:
- **`electron/src/renderer/lib/dagValidation.ts`** — Created with `validateDag()` pure function using Kahn's algorithm for cycle detection and pipeline membership check for orphan detection. Returns `ValidationResult` with `cycleNodes`, `cycleEdges`, and `orphanedTasks` Sets.
- **`electron/src/renderer/components/DagCanvas.tsx`** — Integrated validation into the YAML parse useMemo. Injects `isInCycle`/`isOrphan` into node data, overrides cycle edge styles to red/animated, and renders a validation summary `<Panel position="top-left">` showing cycle and orphan warnings.
- **`electron/src/renderer/components/TaskNode.tsx`** — Added red border + glow for cycle nodes, amber border + warning badge for orphans, and native title tooltip explaining the issue.
- **`electron/src/renderer/lib/yamlParser.ts`** — Added `isInCycle` and `isOrphan` optional fields to `TaskNodeData` interface.
- Build passes successfully (`npm run build` succeeds). No new TypeScript errors introduced.
