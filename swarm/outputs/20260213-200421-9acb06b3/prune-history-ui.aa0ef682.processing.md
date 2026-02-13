# Prune History UI

## Goal

Add a "Clear History" or "Prune" feature to the Agent Panel that allows users to remove terminated agents from the history. This exposes the existing `swarm prune` CLI functionality in the GUI.

## Files

- `electron/src/renderer/components/AgentPanel.tsx` - Add prune button and confirmation dialog
- `electron/src/preload/index.ts` - Expose prune IPC handler (use existing `swarm:run` with prune args)

## Dependencies

None - the `swarm prune` CLI command already exists.

## Acceptance Criteria

1. A "Clear History" button appears in the Agent Panel header when there are terminated agents
2. Clicking the button shows a confirmation dialog with options:
   - Checkbox: "Also delete log files" (maps to `--logs` flag)
   - Optional: Age filter dropdown (e.g., "Older than 1 day", "Older than 7 days", "All")
3. After confirmation, calls `swarm prune --force` (with optional `--logs` and `--older-than` flags)
4. Shows a toast notification with the result (e.g., "Removed 5 terminated agents")
5. The agent list automatically refreshes (already happens via state:changed watcher)
6. Button is disabled when there are no terminated agents

## Notes

From ELECTRON_PLAN.md, the Agent Panel shows a "History" section with terminated agents. As this list grows, users need a way to clean it up.

The CLI command supports these flags:
- `--force` / `-f`: Skip confirmation prompt
- `--logs`: Also delete log files
- `--older-than <duration>`: Only prune agents older than duration (e.g., "7d", "24h")
- `--outputs`: Clean up pipeline output directories

For the initial implementation, expose at minimum:
- Basic prune (remove terminated agents)
- Option to include logs

The confirmation dialog should warn users that this action is irreversible.
