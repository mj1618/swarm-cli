# Task: Add Agent Detail Controls (Iterations, Model, Clone)

## Goal

Enhance the `AgentDetailView` component with inline controls to adjust a running agent's iterations count and model, and add a "Clone" button. The ELECTRON_PLAN.md specifies a "Controls" section in the agent detail view with:

- **Iterations setter**: Number input with a "Set" button to update the agent's max iterations via CLI
- **Model selector**: Dropdown with a "Set" button to change the agent's model via CLI
- **Clone button**: Duplicates the agent with the same configuration

These controls should only appear for active (running or paused) agents.

## Files

### Modify
- `electron/src/renderer/components/AgentDetailView.tsx` — Add a "Controls" section above the existing Pause/Resume/Stop buttons with:
  - Iterations input (number field) + "Set" button
  - Model dropdown (opus, sonnet, haiku) + "Set" button
  - Clone button alongside the existing control buttons
- `electron/src/renderer/App.tsx` — Add handler functions for `onSetIterations`, `onSetModel`, and `onClone`, and pass them as props to `AgentDetailView`
- `electron/src/preload/index.ts` — Ensure the `swarm.run` API supports the necessary CLI args (likely already sufficient since it takes arbitrary string arrays)

### Reference (no changes needed)
- `electron/src/main/index.ts` — The `swarm:run` IPC handler already spawns arbitrary swarm CLI commands

## Dependencies

- Phase 4 (Agent Management) is complete — AgentDetailView exists with basic controls
- The swarm CLI must support `swarm set-iterations <id> <n>` and `swarm set-model <id> <model>` commands (or equivalent). If these CLI commands don't exist, the UI should still be built but the handlers should log a TODO/warning.

## Acceptance Criteria

1. **Iterations setter**: A number input pre-filled with the agent's current `iterations` value and a "Set" button that calls the swarm CLI to update it. Only shown for active agents.
2. **Model selector**: A `<select>` dropdown with model options (opus, sonnet, haiku, inherit) pre-filled with the agent's current model, and a "Set" button. Only shown for active agents.
3. **Clone button**: A button labeled "Clone" that spawns a new agent with the same configuration (name, model, prompt, iterations). Shown for active agents.
4. All three controls appear in a new "Controls" `<Section>` in AgentDetailView, positioned between the "Current Task" / "Result" section and the action buttons.
5. Each control shows a brief toast notification on success or failure.
6. The component still compiles with `npx tsc --noEmit` from the electron directory.

## Notes

- From ELECTRON_PLAN.md, the Controls section mockup shows:
  ```
  Controls
  Iterations: [20    ] [Set]
  Model: [opus ▼]     [Set]

  [Pause] [Stop] [Clone]
  ```
- The Clone button should be added to the existing button row (alongside Pause/Resume and Stop).
- Use the same styling patterns already in AgentDetailView (zinc-700 buttons, 11px text, Section/DetailRow components).
- The swarm CLI commands for setting iterations/model may be `swarm run --set-iterations <id> <n>` or similar — check `cmd/` directory for available commands. If no such commands exist, wire up the UI anyway and use `console.warn('Not yet implemented')` as placeholder.

## Completion Notes

Implemented by agent cd6da9a7.

### Changes Made:

1. **AgentDetailView.tsx**: Added a "Controls" `<Section>` with:
   - Iterations number input pre-filled with current value, synced via useEffect, with "Set" button
   - Model `<select>` dropdown (opus, sonnet, haiku) pre-filled with current model, with "Set" button
   - Clone button added to the action buttons row alongside Pause/Resume and Stop
   - All controls only shown for active agents (`isActive`)

2. **AgentPanel.tsx**: Added three handler functions:
   - `handleSetIterations` — calls `swarm update <id> --iterations <n>` via `window.swarm.run`
   - `handleSetModel` — calls `swarm update <id> --model <model>` via `window.swarm.run`
   - `handleClone` — calls `swarm clone <id> -d` via `window.swarm.run`
   - All three passed as props to `AgentDetailView`

3. **No changes needed to preload/index.ts** — the existing `swarm.run` API accepts arbitrary string arrays

### CLI Commands Used:
- `swarm update <id> --iterations <n>` — sets iteration count on running agent
- `swarm update <id> --model <model>` — sets model for next iteration
- `swarm clone <id> -d` — clones agent in detached mode
