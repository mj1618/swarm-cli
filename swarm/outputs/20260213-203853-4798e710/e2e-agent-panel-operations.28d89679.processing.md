# E2E Tests for Agent Panel Operations

## Goal

Add Playwright E2E tests for agent panel operations including:
- Viewing agent details
- Pause/Resume functionality  
- Stop/Kill agent functionality
- Prune history feature

These are critical user flows that currently lack E2E test coverage.

## Files

- **Create**: `electron/e2e/agent-panel.spec.ts`
- **Modify**: None (standalone test file)

## Dependencies

- Unit test setup (vitest) complete
- E2E test infrastructure exists (app.spec.ts, dag-editing.spec.ts)
- Agent panel component exists with data-testid attributes

## Acceptance Criteria

1. New test file `electron/e2e/agent-panel.spec.ts` exists
2. Tests cover:
   - Agent list displays correctly (running vs history sections)
   - Agent search/filter functionality
   - Status filter dropdown works
   - Clicking agent expands detail view
   - Pause button calls correct IPC handler (mock or verify UI state)
   - Resume button calls correct IPC handler
   - Stop button calls correct IPC handler
   - Prune history button clears terminated agents
3. Tests use proper data-testid selectors
4. Tests handle cases where no agents exist gracefully
5. All tests pass: `npm run test:e2e`

## Notes

From ELECTRON_PLAN.md Agent Panel section:
- Running agents show: iteration progress, token counts, cost, controls
- History section shows completed/failed agents
- Agent detail view expands on click with "Back" button
- Controls: Pause, Resume, Stop, Clone buttons

Existing data-testid attributes to use:
- `data-testid="agent-panel"` - main panel container
- `data-testid="agent-card-{id}"` - individual agent cards (if they exist)
- Agent panel has search input and status filter dropdown

Consider mocking the IPC layer for pause/resume/stop since these require actual CLI interaction:
- Can verify the button click triggers the expected IPC call
- Or verify UI state changes (e.g., pause button becomes resume)

The prune history feature removes terminated agents from state.json - verify the UI updates accordingly.
