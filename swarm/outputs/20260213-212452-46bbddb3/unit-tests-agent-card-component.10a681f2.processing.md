# Unit Tests for AgentCard Component

## Goal

Add comprehensive unit tests for the `AgentCard` component, which displays agent status, progress, and controls in the Agent Panel. This component has multiple states and interactions that should be tested to ensure reliability.

## Files

- **Create**: `electron/src/renderer/components/__tests__/AgentCard.test.tsx`
- **Reference**: `electron/src/renderer/components/AgentCard.tsx` (the component to test)
- **Reference**: `electron/src/renderer/components/__tests__/ErrorBoundary.test.tsx` (follow test patterns)

## Dependencies

- Requires: Vitest test setup (already complete)
- Requires: `@testing-library/react` (already installed)

## Acceptance Criteria

1. **Helper function tests**:
   - `formatDuration()`: Test seconds, minutes, and hours formatting
   - `formatCost()`: Test zero, sub-cent, and normal cost formatting

2. **Render state tests**:
   - Running agent: Shows green pulsing dot, pause/stop buttons, current task
   - Paused agent: Shows yellow dot, "Paused" label, resume/stop buttons
   - Terminated agent: Shows gray dot, exit reason (completed/killed/crashed)
   - Agent with no iterations: Progress bar should not render

3. **Progress bar tests**:
   - Shows correct percentage (current/total)
   - Handles edge cases (0/0, current > total)

4. **User interaction tests**:
   - Click on card calls `onClick` with agent data
   - Pause button calls `onPause` with agent ID
   - Resume button calls `onResume` with agent ID  
   - Stop button calls `onKill` with agent ID
   - Button clicks don't propagate to card click

5. **Display tests**:
   - Shows agent name or truncated ID if no name
   - Shows model badge
   - Shows cost and duration

6. All tests pass with `npm test` in the electron directory

## Notes

From `AgentCard.tsx`:
- Helper functions are not exported, so test them indirectly through component renders
- Use `data-testid` attributes already present: `agent-card-{id}`, `agent-controls-{id}`, `agent-pause-{id}`, `agent-resume-{id}`, `agent-stop-{id}`
- StatusDot and ProgressBar are internal components rendered conditionally

### Test Fixtures

Create mock agent data for different states:

```typescript
const runningAgent: AgentState = {
  id: 'abc12345',
  name: 'planner',
  pid: 12345,
  status: 'running',
  paused: false,
  model: 'opus',
  started_at: '2026-02-13T14:30:00Z',
  iterations: 20,
  current_iteration: 5,
  input_tokens: 1000,
  output_tokens: 500,
  total_cost_usd: 0.42,
  current_task: 'Reading src/app.ts',
  working_dir: '/tmp/test'
}

const pausedAgent: AgentState = { ...runningAgent, paused: true }

const terminatedAgent: AgentState = {
  ...runningAgent,
  status: 'terminated',
  exit_reason: 'completed',
  terminated_at: '2026-02-13T15:00:00Z'
}
```

### Key Assertions

- Use `@testing-library/react` render and screen utilities
- Use `vi.fn()` for callback mocks
- Use `fireEvent.click()` for interaction testing
- Check for presence/absence of UI elements based on agent state
