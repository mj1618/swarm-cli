# Task: Add Run Pipeline Button to Pipeline Config Bar

## Goal

Add a prominent "Run" button to the `PipelineConfigBar` component so users can launch a pipeline directly from the DAG view without needing the command palette. The ELECTRON_PLAN.md mockup explicitly shows a `[▶ Run]` button in the pipeline header area. Currently, running pipelines is only possible via the command palette, making the core workflow hard to discover.

## Files

- **Modify**: `electron/src/renderer/components/PipelineConfigBar.tsx` — Add a Run button that invokes `window.swarm.run()` with the selected pipeline
- **Modify**: `electron/src/renderer/App.tsx` — Pass an `onRunPipeline` callback to `PipelineConfigBar` and wire it to the swarm CLI

## Dependencies

- PipelineConfigBar component (exists)
- `window.swarm.run()` IPC handler (exists — already used by command palette)
- Pipeline selection state in App.tsx (exists — `activePipeline`)

## Acceptance Criteria

1. When a pipeline is selected in the PipelineConfigBar dropdown, a "Run" button appears next to the pipeline settings
2. Clicking the Run button calls `window.swarm.run(['pipeline', '--name', pipelineName])` (or `window.swarm.run(['pipeline'])` for the default pipeline)
3. While the command is executing, the button shows a loading/disabled state to prevent double-clicks
4. On success, a toast notification confirms the pipeline was started
5. On failure (non-zero exit code), a toast notification shows the error from stderr
6. The Run button is NOT shown when "All Tasks" is selected (no pipeline context)
7. The button styling matches the existing UI (small, compact, uses primary color)

## Notes

- The command palette already has this wiring at `App.tsx:387`: `window.swarm.run(['pipeline', '--name', name])` — reuse the same pattern
- The PipelineConfigBar already receives the `activePipeline` name as a prop, so it knows which pipeline to run
- Keep it simple: a single button with loading state and toast feedback. No confirmation dialog needed since running a pipeline is not destructive.
- Consider adding a keyboard shortcut hint (e.g., shown in tooltip) for discoverability

## Completion Notes

Implemented by agent edf6400c on iteration 6.

**Changes made:**

1. **`PipelineConfigBar.tsx`**: Added `onRunPipeline` optional prop and a `running` state. When a pipeline is selected and `onRunPipeline` is provided, a styled "▶ Run" button appears after the Configure button. The button shows "Running…" with disabled state while the async operation is in progress.

2. **`App.tsx`**: Added `handleRunPipeline` callback that calls `window.swarm.run(['pipeline', '--name', pipelineName])` and shows toast notifications for success/failure. Passed this callback as `onRunPipeline` to `PipelineConfigBar`.

**Acceptance criteria met:**
- ✓ Run button appears only when a specific pipeline is selected (inside `{current && activePipeline && ...}` block)
- ✓ Calls `window.swarm.run(['pipeline', '--name', pipelineName])` on click
- ✓ Shows loading/disabled state during execution
- ✓ Success toast on code 0, error toast with stderr on non-zero exit
- ✓ Not shown when "All Tasks" is selected (button is inside the activePipeline guard)
- ✓ Compact styling with primary color, matching existing UI
