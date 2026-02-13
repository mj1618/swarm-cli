# Task: Agent Detail View with Expandable Panel

**Phase:** 4 - Agent Management
**Priority:** High (next uncompleted Phase 4 item)

## Goal

Add a click-to-expand agent detail view in the Agent Panel. Currently, `AgentCard` shows a compact summary (name, progress, cost, controls). The plan (ELECTRON_PLAN.md lines 192-223) specifies a full detail view that appears when clicking an agent card, showing comprehensive stats, the current task description, and editable controls for iterations and model.

## Current State

- `AgentPanel.tsx` renders a list of `AgentCard` components with running/history sections
- `AgentCard.tsx` shows: status dot, name, model badge, iteration progress bar, cost, duration, current task, pause/resume/stop buttons
- The preload already exposes `window.swarm.inspect(agentId)` for fetching detailed agent info
- `AgentState` interface in `preload/index.ts` already has all needed fields (tokens, cost, iterations, working_dir, etc.)

## What to Build

An `AgentDetailView` component that replaces the agent list when a user clicks on an `AgentCard`. It should include:

1. **Back button** — returns to the agent list view
2. **Full status section** — ID, PID, model, started time, working directory
3. **Progress section** — iteration progress with bar, successful/failed iteration counts
4. **Usage section** — input tokens, output tokens, total cost (formatted)
5. **Current task** — what the agent is currently working on (if running)
6. **Controls section**:
   - Pause / Resume / Stop buttons (same as card but larger)
   - Clone button that calls `window.swarm.run()` with the same args
7. **Auto-refresh** — detail view should update in real-time when `state:changed` fires

## Files to Create/Modify

- `electron/src/renderer/components/AgentDetailView.tsx` (NEW) — The expanded detail component
- `electron/src/renderer/components/AgentPanel.tsx` (MODIFY) — Add selected agent state, toggle between list and detail views
- `electron/src/renderer/components/AgentCard.tsx` (MODIFY) — Add `onClick` prop to navigate to detail view

## Dependencies

- Phase 4 state watching — already implemented
- Pause/resume/stop IPC — already implemented
- `AgentState` type — already defined in preload

## Acceptance Criteria

1. Clicking an `AgentCard` navigates to a detail view showing all fields from the plan spec (ID, PID, model, start time, iteration progress with success/fail counts, token usage, cost, current task)
2. A "Back" button in the detail view returns to the agent list
3. The detail view updates in real-time when the agent state changes (not stale)
4. Pause/Resume/Stop buttons work from the detail view
5. The app builds successfully (`npm run build` in electron/)

## Notes

- Reference the Agent Detail View mockup in ELECTRON_PLAN.md (lines 192-223)
- Keep the dark theme consistent with existing zinc/slate palette
- Use the same formatting functions already in `AgentCard.tsx` (`formatDuration`, `formatCost`)
- The Clone button can be deferred to a follow-up task if it requires additional IPC work to reconstruct the original run arguments; the priority is the view itself
- Do NOT add the editable iterations/model controls yet — those require new IPC handlers (`swarm update-iterations`, `swarm update-model`) that don't exist. Just display the values read-only for now.

## Completion Notes

**Completed by agent 626e9653**

Implemented the full-view navigation agent detail view:

- **AgentDetailView.tsx**: Rewrote as a full-panel replacement view with Back button header, status indicator, and sections for Info (ID, PID, model, start time, duration, working dir), Progress (iteration bar with success/fail counts), Usage (input/output tokens, total cost), Current Task, Result (exit reason + error), Log File, and Controls (Pause/Resume/Stop). Duration auto-ticks every second for running agents.
- **AgentCard.tsx**: Simplified to a compact clickable card with `onClick` prop. Control buttons use `e.stopPropagation()` to prevent navigation when clicking Pause/Resume/Stop.
- **AgentPanel.tsx**: Added `selectedAgentId` state. When an agent is selected, the entire panel renders the `AgentDetailView` instead of the card list. The detail view auto-updates via the existing `state:changed` listener (agents state is looked up by ID each render). Handles agent disappearance gracefully.
- Clone button deferred (needs IPC for original run args reconstruction).
- Editable iterations/model controls deferred (needs new CLI handlers).
- Build passes successfully.
