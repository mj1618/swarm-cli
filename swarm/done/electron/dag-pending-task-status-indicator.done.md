# Task: DAG Pending Task Status Indicator

## Goal

Add a "pending" status indicator (gray dot) to DAG task nodes during active pipeline execution. When a pipeline is running, tasks that are part of the pipeline but haven't started yet should visually show as "pending" rather than having no status indicator.

## Context

From ELECTRON_PLAN.md, the Live Execution Overlay section specifies:

| Status | Visual |
|--------|--------|
| Pending | ⚪ Gray |
| Running | 🔵 Blue (animated pulse) |
| Succeeded | ✅ Green checkmark |
| Failed | ❌ Red X |
| Skipped | ⏭️ Gray with skip icon |

Currently, TaskNode only implements running, paused, succeeded, and failed states. The "pending" state is missing.

## Files to Modify

1. `electron/src/renderer/lib/yamlParser.ts`
   - Add `'pending'` to the `AgentDisplayStatus` type

2. `electron/src/renderer/components/TaskNode.tsx`
   - Add pending case to `StatusIndicator` component (gray dot without animation)

3. `electron/src/renderer/components/DagCanvas.tsx`
   - In `enrichedNodes` useMemo, detect when a pipeline is actively running
   - For tasks in `pipelineTasks` that don't have associated agents, set `agentStatus: 'pending'`

## Implementation Details

### Detecting Active Pipeline Execution

A pipeline is considered "actively running" when:
1. `activePipeline` is set (a pipeline is selected)
2. At least one agent exists with `status === 'running'` that matches a task in `pipelineTasks`

### Setting Pending Status

For each task node:
1. If the task has an associated running/terminated agent → use that agent's status
2. If the task is in `pipelineTasks` AND the pipeline is actively running AND no agent exists → status is 'pending'
3. Otherwise → no status indicator

### Visual Design

Pending indicator should be:
- Small gray dot (`bg-zinc-500` or similar)
- Same size as running indicator (`w-2 h-2 rounded-full`)
- No animation (static, unlike running which pulses)

## Dependencies

None - this is a UI enhancement that builds on existing infrastructure.

## Acceptance Criteria

1. [x] `AgentDisplayStatus` type includes `'pending'`
2. [x] TaskNode's StatusIndicator renders a gray dot for pending status
3. [x] When a pipeline is actively running:
   - Tasks with running agents show blue pulsing dot
   - Tasks with terminated agents (completed/failed) show appropriate icons
   - Tasks in the pipeline without agents show gray pending dot
4. [x] When no pipeline is running or no agents are running, no pending indicators are shown
5. [x] The app builds without TypeScript errors
6. [x] Visual indicator matches the plan's design (gray dot, similar to other status indicators)

## Notes

- This completes the "Live Execution Overlay" feature from Phase 2 of ELECTRON_PLAN.md
- The "skipped" status is more complex (requires tracking dependency condition results) and could be a separate task

---

## Completion Notes

**Completed by agent ae6bddf6**

### Changes Made:

1. **yamlParser.ts**: Added `'pending'` to the `AgentDisplayStatus` type union.

2. **TaskNode.tsx**: Added a new case in `StatusIndicator` for pending status that renders a static gray dot (`bg-zinc-500`, `w-2 h-2 rounded-full`) without animation.

3. **DagCanvas.tsx**: Updated the `enrichedNodes` useMemo to:
   - Build a set of pipeline tasks for quick lookup
   - Detect if the pipeline is actively running (at least one running agent matches a pipeline task)
   - For tasks in the pipeline without agents, set `agentStatus: 'pending'` when the pipeline is running
   - Added `activePipeline` and `pipelineTasks` to the dependency array

### Testing:
- Build passes with `npm run build` (no TypeScript errors)
- The implementation follows the exact specification from ELECTRON_PLAN.md
