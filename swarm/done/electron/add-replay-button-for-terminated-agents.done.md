# Task: Add Replay and Clone buttons for terminated agents in Agent Detail View

## Goal

Add "Replay" and "Clone" action buttons to the Agent Detail View for terminated agents. Currently, action buttons (Pause/Resume/Stop/Clone) are only visible for active/running agents — once an agent terminates, there are no action buttons available. The swarm CLI supports `replay` (re-run with same config) and `clone` (duplicate) for terminated agents, but the Electron UI doesn't expose these.

## Files

- **Modify**: `electron/src/renderer/components/AgentDetailView.tsx` — Add a "Replay" button and "Clone" button that appear for terminated agents (in addition to the existing buttons for active agents)
- **Modify**: `electron/src/renderer/components/AgentPanel.tsx` — Add `handleReplay` handler that calls `window.swarm.run(['replay', agentId, '-d'])` and wire it to AgentDetailView via a new `onReplay` prop

## Dependencies

None — the `window.swarm.run()` API already supports arbitrary swarm CLI commands including `replay`.

## Acceptance Criteria

1. When viewing a **terminated** agent's detail view, "Replay" and "Clone" buttons are visible below the Result section
2. Clicking "Replay" calls `window.swarm.run(['replay', agentId, '-d'])` and shows a success/error toast
3. Clicking "Clone" calls `window.swarm.run(['clone', agentId, '-d'])` and shows a success/error toast (same as existing clone for active agents)
4. The buttons for **active** agents remain unchanged (Pause/Resume/Stop/Clone still work as before)
5. The app builds successfully with `npm run build`

## Implementation Notes

- The `isActive` guard on line 224 and 272 of AgentDetailView.tsx hides all controls and action buttons for terminated agents — add a separate section below for terminated agent actions
- Use the same button styling as existing action buttons for visual consistency
- The replay command is: `swarm replay <agent-id> -d` (detached mode)
- Toast messages should indicate the action taken, e.g. "Replaying agent planner" / "Cloned agent planner"
- AgentDetailView props need a new `onReplay: (id: string) => void` callback

## Completion Notes

Implemented by agent 88d28f83:

- Added `onReplay` prop to `AgentDetailViewProps` interface
- Added a "Replay" and "Clone" button section for terminated agents (`!isActive`) below the existing active agent action buttons
- Added `handleReplay` handler in `AgentPanel.tsx` that calls `window.swarm.run(['replay', agentId, '-d'])` with toast feedback including agent name
- Wired `onReplay={handleReplay}` to the `AgentDetailView` component
- All existing active agent buttons (Pause/Resume/Stop/Clone) remain unchanged
- Build verified successfully with `npm run build`
