# Task: Add Expandable Agent Detail View

**Phase:** 4 - Agent Management
**Priority:** High

## Goal

Add a click-to-expand Agent Detail View to the Agent Panel. Currently, `AgentCard` shows a compact summary. The plan specifies a full detail view that appears when clicking on an agent card, showing comprehensive status, IDs, progress breakdown, token usage, and enhanced controls.

## Files

- `electron/src/renderer/components/AgentCard.tsx` — Modify to add expand/collapse toggle
- `electron/src/renderer/components/AgentDetailView.tsx` — **New file**: Full detail panel shown when an agent card is expanded

## Dependencies

- AgentPanel (completed)
- AgentCard (completed)
- State watching (completed)
- Pause/resume/stop controls (completed)

## What to Build

When a user clicks on an AgentCard, it should expand inline to show the full detail view (per ELECTRON_PLAN.md lines 192-223):

### Compact View (existing — keep as-is)
- Status dot + name + model badge
- Iteration progress bar
- Cost + duration
- Pause/Resume/Stop buttons

### Expanded Detail View (new)
Add a toggled expanded section below the compact card with:

1. **Identity Section**
   - Agent ID (full, copyable)
   - PID
   - Working directory (truncated)
   - Started at (human-readable time)

2. **Progress Section**
   - Iteration: X / Y with percentage
   - Progress bar (reuse existing)
   - Successful iterations count
   - Failed iterations count

3. **Usage Section**
   - Input tokens (formatted with commas)
   - Output tokens (formatted with commas)
   - Total cost

4. **Current Task**
   - Full current_task text (not truncated, unlike compact view)

5. **Controls Section** (for running agents)
   - Pause / Resume / Stop buttons (existing, move here)
   - Log file path (clickable — opens in console panel)

## Acceptance Criteria

1. Clicking an AgentCard toggles between compact and expanded views
2. Expanded view shows all fields listed above from `AgentState`
3. Only one agent card is expanded at a time (clicking another collapses the previous)
4. The expanded view updates in real-time as state changes
5. The app builds successfully with `npm run build`
6. Compact view remains unchanged when collapsed

## Notes

- The `AgentState` interface in `preload/index.ts` already has all needed fields: `id`, `pid`, `working_dir`, `started_at`, `successful_iterations`, `failed_iterations`, `input_tokens`, `output_tokens`, `total_cost_usd`, `current_task`, `log_file`
- Keep the expansion state in `AgentPanel` as a `expandedAgentId: string | null` state variable
- Use smooth CSS transitions for the expand/collapse animation
- Format large numbers with `toLocaleString()` for readability
- The plan shows a "Back" button in the detail view — implement this as a collapse toggle (click card header or a close button)

## Completion Notes

Implemented by agent cd4f15cd. Changes:

- **AgentDetailView.tsx** (new): Full detail panel with Info (ID, PID, model, started time, duration, working dir), Progress (iteration bar, succeeded/failed counts), Usage (input/output tokens, cost), Current Task, Result (exit reason/error), Log File path, and Controls (pause/resume/stop). Duration auto-ticks every second for running agents.
- **AgentCard.tsx**: Added `expanded` and `onToggleExpand` props. Card header is clickable with expand/collapse indicator. Compact-only content is hidden when expanded; detail view rendered inline below compact stats.
- **AgentPanel.tsx**: Added `expandedAgentId` state. Only one card can be expanded at a time (clicking another collapses the previous). Both running and history sections pass expand props.
- Build verified with `npm run build`.
