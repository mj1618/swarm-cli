# Task: Agent Panel Reading from state.json (Phase 1 Completion)

## Goal

Upgrade the right sidebar Agent Panel to read agent data from `~/swarm/state.json` (or the configured state path) instead of polling `swarm list --json` via CLI. Add file watching so the panel updates in real-time when agent state changes. This completes the last Phase 1 item: "Agent list panel reading from `state.json`".

Currently, `App.tsx` polls `window.swarm.list()` every 5 seconds, which spawns `swarm list --json` as a subprocess. This is slow and doesn't provide real-time updates. The plan specifies watching `state.json` for real-time state updates.

## Files

### Create
- `electron/src/renderer/components/AgentPanel.tsx` — Extract the agent list UI from App.tsx into its own component. Display agents with richer data from state.json (id, name, status, model, iterations, current_iteration, cost, pid, started_at, current_task). Show running agents and history sections as specified in ELECTRON_PLAN.md.
- `electron/src/renderer/components/AgentCard.tsx` — Individual agent card component showing status indicator, name/id, iteration progress bar, cost, and control buttons (Pause/Resume/Stop).

### Modify
- `electron/src/main/index.ts` — Add IPC handlers: `state:read` to read and parse `~/swarm/state.json`, `state:watch`/`state:unwatch` to watch the state file for changes using chokidar, and `state:changed` event to push updates to the renderer.
- `electron/src/preload/index.ts` — Expose `state` API: `read()`, `watch()`, `unwatch()`, `onChanged(callback)` — same pattern as the existing `fs` watcher.
- `electron/src/renderer/App.tsx` — Replace inline agent list with `<AgentPanel />` component. Remove the `fetchAgents`/`setInterval` polling logic.

## Dependencies

- Electron app scaffold (completed)
- File tree component (completed)
- IPC handlers for CLI commands (completed — swarm:kill, swarm:pause, swarm:resume already exist)

## Acceptance Criteria

1. Agent panel reads agent data from `~/swarm/state.json` (not from CLI polling)
2. State file is watched via chokidar — panel updates within ~1 second of state changes
3. Each agent card shows: status indicator (colored dot), name or ID, model, iteration progress (X of Y with progress bar), cost, and duration
4. Running agents section shows agents with status "running" or "paused"
5. Control buttons (Pause/Resume/Kill) still work via existing `swarm:pause`/`swarm:resume`/`swarm:kill` IPC
6. Graceful handling when state.json doesn't exist or is empty (show "No agents" message)
7. Graceful handling when state.json contains malformed JSON
8. The `AgentPanel` is a self-contained component (not inline in App.tsx)
9. The app builds successfully (`npm run build` in electron/)

## Notes

- The state file path is `~/swarm/state.json` by default (see `internal/state/` in the Go codebase). Use `os.homedir()` + `/swarm/state.json` in the main process to resolve it.
- State file format is documented in ELECTRON_PLAN.md under "AgentState" — it's an array of agent objects (or could be a single object per the Go code; read the actual state file format from `internal/state/state.go` if unsure).
- Use chokidar to watch the state file (already a dependency). When the file changes, read + parse it and send the updated array to the renderer via IPC.
- The agent status colors from ELECTRON_PLAN.md: running = green, paused = yellow, completed = gray.
- Show iteration progress as a visual bar: `current_iteration / iterations` (e.g., "3 of 20" with a filled bar).
- Keep the existing `swarm:kill`/`swarm:pause`/`swarm:resume` IPC handlers for control actions — those correctly go through the CLI.
- The AgentCard component should match the dark theme using existing Tailwind CSS variables.

## Completion Notes

Implemented by agent 65f6eefa. All acceptance criteria met:

1. **Main process** (`electron/src/main/index.ts`): Added `state:read`, `state:watch`, `state:unwatch` IPC handlers. Reads `~/.swarm/state.json` directly. Watches with chokidar using `awaitWriteFinish` for stability. Converts the `{ agents: { id: {...} } }` map format to an array for the renderer.
2. **Preload** (`electron/src/preload/index.ts`): Exposed `window.state` API with `read()`, `watch()`, `unwatch()`, `onChanged()`. Added full `AgentState` TypeScript interface matching the Go struct fields.
3. **AgentCard** (`electron/src/renderer/components/AgentCard.tsx`): Shows status dot (green=running, yellow=paused, gray=terminated), name/ID, model badge, iteration progress bar with percentage, cost, duration, current task, and Pause/Resume/Stop controls.
4. **AgentPanel** (`electron/src/renderer/components/AgentPanel.tsx`): Self-contained component. Reads state on mount, watches for real-time updates via chokidar. Splits agents into Running and History (collapsible) sections. Handles empty/missing state gracefully.
5. **App.tsx**: Replaced inline agent list and `fetchAgents`/`setInterval` polling with `<AgentPanel />`. Preserved DAG editor and YAML loading logic added by concurrent agents.
6. Build passes (`npm run build` and `npm run build:electron`).
