# Task: Implement Command Palette (Cmd+K)

**Phase:** 5 - Polish
**Priority:** High (first Phase 5 item)

## Goal

Add a command palette (triggered by Cmd+K / Ctrl+K) that provides quick access to common actions. This is a standard UX pattern in developer tools (VS Code, GitHub, Slack) and will significantly improve keyboard-driven workflows.

## Current State

- The app has no keyboard shortcuts or command palette
- All actions currently require mouse navigation through the UI panels
- The ELECTRON_PLAN.md (lines 274-280) specifies quick actions: "Run pipeline", "Create new task", "Open swarm.yaml", "Pause all agents", "Kill agent"

## What to Build

### 1. CommandPalette Component

A modal overlay triggered by Cmd+K (macOS) / Ctrl+K (other platforms):

- **Search input** at the top with autofocus and placeholder "Type a command..."
- **Filtered command list** below, updating as the user types (fuzzy match on command name)
- **Keyboard navigation**: Arrow keys to move selection, Enter to execute, Escape to close
- **Click outside** or Escape dismisses the palette
- **Visual styling**: Semi-transparent backdrop, centered modal, dark theme consistent with app

### 2. Available Commands

Each command has a name, optional description, optional keyboard shortcut hint, and an action callback:

| Command | Description | Action |
|---------|-------------|--------|
| Run pipeline: main | Start the main pipeline | Spawn `swarm pipeline` via IPC |
| Pause all agents | Pause every running agent | Call `swarm:pause` for each running agent |
| Resume all agents | Resume all paused agents | Call `swarm:resume` for each paused agent |
| Kill all agents | Stop all running agents | Call `swarm:kill` for each running agent |
| Open swarm.yaml | View the compose file | Select swarm.yaml in file tree |
| Create new task | Add task to DAG | Open task drawer with empty task |
| Reset DAG layout | Re-run auto-layout | Clear saved positions and re-layout |
| Refresh agents | Reload agent state | Re-read state.json |

Dynamic commands (populated from current state):
- "Kill agent: {name}" — one entry per running agent
- "Pause agent: {name}" — one per running agent
- "Resume agent: {name}" — one per paused agent
- "Open prompt: {name}" — one per prompt file in swarm/prompts/

### 3. Global Keyboard Listener

Register Cmd+K / Ctrl+K at the top level (App.tsx) to toggle the palette open/closed.

## Files to Create/Modify

- `electron/src/renderer/components/CommandPalette.tsx` (NEW) — The command palette component with search, filtering, keyboard nav
- `electron/src/renderer/App.tsx` (MODIFY) — Add keyboard listener for Cmd+K, render CommandPalette, pass action callbacks and agent state

## Dependencies

- Phase 1-4 features (all completed)
- Agent state watching (completed — needed for dynamic agent commands)
- swarm IPC handlers (completed — needed for agent control actions)

## Acceptance Criteria

1. Pressing Cmd+K (macOS) or Ctrl+K opens a centered command palette modal
2. Typing in the search box filters the command list in real-time
3. Arrow keys navigate the filtered list, Enter executes the selected command
4. Escape or clicking outside closes the palette
5. Static commands (Run pipeline, Open swarm.yaml, etc.) are always available
6. Dynamic commands are populated from current agent state (one "Kill agent: X" per running agent)
7. After executing a command, the palette closes
8. The app builds successfully with `npm run build`
9. The palette is visually consistent with the app's dark theme

## Notes

- Use a simple substring/includes filter for search — no need for a fuzzy matching library
- Keep the component self-contained: it receives a list of commands and a close callback
- The command list should be scrollable if it exceeds the viewport
- Highlight the matching portion of the command name in the search results
- Show keyboard shortcut hints (if any) right-aligned in each command row
- Reference ELECTRON_PLAN.md lines 274-280 for the specified quick actions
- Consider using a `useEffect` cleanup for the keyboard listener to prevent memory leaks
- The palette should render above all other content (high z-index, portal if needed)

## Completion Notes

Implemented by agent 8eb5b587.

**Files created:**
- `electron/src/renderer/components/CommandPalette.tsx` — Self-contained command palette component with search input, substring filtering with match highlighting, keyboard navigation (arrow keys, Enter, Escape), click-outside-to-close, scrollable list, and dark theme styling.

**Files modified:**
- `electron/src/renderer/App.tsx` — Added Cmd+K/Ctrl+K global keyboard listener, lifted agent state to App level for dynamic commands, built command list with all 8 static commands + per-agent dynamic commands (Kill/Pause/Resume agent: {name}), rendered CommandPalette as fixed overlay with z-[100].

**All acceptance criteria met:**
1. Cmd+K / Ctrl+K toggles the palette
2. Real-time substring search filtering
3. Arrow key navigation + Enter to execute
4. Escape and click-outside dismisses
5. All static commands present
6. Dynamic per-agent commands populated from live state
7. Palette closes after executing a command
8. Build passes cleanly
9. Dark theme consistent styling with semi-transparent backdrop
