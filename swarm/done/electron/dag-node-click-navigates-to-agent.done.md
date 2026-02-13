# Task: Click running DAG node to navigate to its agent in Agent Panel

## Goal

When a DAG task node shows a running/paused/completed agent status overlay, clicking that node should navigate the right sidebar to the agent's detail view instead of opening the task configuration drawer. Currently, clicking any task node always opens the TaskDrawer for editing — even when the task has an active agent with visible status. This means users must manually find the agent in the Agent Panel to see details or control it.

## Phase

Phase 4/5 — UX improvement linking DAG visualization to agent management.

## Files to Modify

1. **`electron/src/renderer/components/DagCanvas.tsx`** — Update `handleNodeClick` to check if the clicked node has an active agent, and if so, call a new `onNavigateToAgent` callback instead of `onSelectTask`
2. **`electron/src/renderer/App.tsx`** — Pass a new `onNavigateToAgent` prop to `DagCanvas`, which selects the agent in the AgentPanel (by setting a `selectedAgentId` state) and closes any open TaskDrawer/PipelinePanel
3. **`electron/src/renderer/components/AgentPanel.tsx`** — Accept an optional `selectedAgentId` prop that auto-opens the agent detail view for that agent when provided

## Dependencies

None — all prerequisite components exist.

## Implementation Details

### DagCanvas.tsx changes
- In `handleNodeClick`, check if the clicked node's `data.agentStatus` is defined (meaning an agent is mapped to this task)
- If an agent is active, find the matching agent from the `agents` prop by name/task_id
- Call `onNavigateToAgent(agentId)` if available; otherwise fall back to `onSelectTask`
- Add a visual hint: when a node has an active agent, show a small "click to view agent" tooltip or change the cursor

### App.tsx changes
- Add `selectedAgentId` state (string | null)
- Pass `onNavigateToAgent` to DagCanvas that sets `selectedAgentId` and clears `selectedTask`/`selectedPipeline`
- Pass `selectedAgentId` to AgentPanel
- Clear `selectedAgentId` when user navigates back from agent detail

### AgentPanel.tsx changes
- Accept `selectedAgentId` prop
- When `selectedAgentId` changes to a non-null value, auto-select that agent (show its detail view)
- When user clicks "Back", clear the parent's `selectedAgentId` via a callback

## Acceptance Criteria

1. Clicking a DAG task node that has a running agent navigates to that agent's detail view in the right panel
2. Clicking a DAG task node with no active agent still opens the TaskDrawer as before
3. The agent detail view shows the correct agent info (iterations, cost, controls)
4. Clicking "Back" in the agent detail view returns to the agent list (not the task drawer)
5. No TypeScript errors (`tsc --noEmit` passes)

## Notes

- The agent-to-task matching logic already exists in `DagCanvas.tsx` `enrichedNodes` useMemo (line 136-152) — reuse the same matching logic
- This creates a natural workflow: user sees a running task on the DAG → clicks it → sees real-time agent stats and controls

## Completion Notes

Implemented all three file changes:

1. **DagCanvas.tsx**: Added `onNavigateToAgent` prop to interface and destructuring. Modified `handleNodeClick` to check `node.data.agentStatus` — if present, finds the matching agent by name/task_id/current_task and calls `onNavigateToAgent(agent.id)` instead of opening the task drawer.

2. **App.tsx**: Added `selectedAgentId` state, `handleNavigateToAgent` callback (sets agent ID, clears task/pipeline selections), and `handleClearSelectedAgent` callback. Passed `onNavigateToAgent` to DagCanvas, and `selectedAgentId`/`onClearSelectedAgent` to AgentPanel. Also clears `selectedAgentId` when selecting a task via `handleSelectTask`.

3. **AgentPanel.tsx**: Added `selectedAgentId` and `onClearSelectedAgent` props. Uses external prop with internal state fallback pattern. Syncs internal state when external selection changes. The "Back" button and agent-disappears cleanup both call `onClearSelectedAgent` to clear parent state.

Build passes with no TypeScript errors.
