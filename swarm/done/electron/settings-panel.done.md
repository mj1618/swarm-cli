# Task: Settings Panel

## Goal

Add a Settings panel to the Electron app that displays and allows editing of swarm configuration. The panel should be accessible from the command palette and show the current backend, default model, state path, and logs directory as described in ELECTRON_PLAN.md Phase 5.

## Files

### Create
- `electron/src/renderer/components/SettingsPanel.tsx` — Settings panel component with form fields

### Modify
- `electron/src/main/index.ts` — Add IPC handler to read `swarm/swarm.toml` config and expose paths (state file, logs dir)
- `electron/src/preload/index.ts` — Add `settings` API bridge (readConfig, getPaths)
- `electron/src/renderer/App.tsx` — Add settings panel state, toggle from command palette, render SettingsPanel when active
- `electron/src/renderer/vite-env.d.ts` — Add SettingsAPI type to Window interface (if types are declared here)
- `electron/src/renderer/components/CommandPalette.tsx` — No changes needed (command is added in App.tsx's paletteCommands)

## Dependencies

- Phase 1-4 complete (all done)
- Command palette complete (done — `command-palette.done.md`)
- Toast notifications integrated (done — toasts are wired into App.tsx)

## Acceptance Criteria

1. A "Settings" or "Open settings" command exists in the command palette (Cmd+K)
2. Selecting it opens a settings panel in the center area (replacing DAG/file viewer)
3. The panel displays:
   - **Backend**: Current backend value from `swarm/swarm.toml` (e.g., "claude-code" or "cursor")
   - **Default Model**: Current model from config (e.g., "opus", "sonnet")
   - **State Path**: The path to `state.json` (read-only display, e.g., `~/.swarm/state.json`)
   - **Logs Directory**: The logs directory path (read-only display, e.g., `~/swarm/logs/`)
4. Backend and model fields are editable (dropdown or text input) and save back to `swarm/swarm.toml`
5. A save action triggers a toast notification (success or error) using the existing toast system
6. The panel has a close/back button that returns to the previous view
7. TypeScript compiles without errors (`npx tsc --noEmit` from electron/ dir)

## Notes

- The config file is `swarm/swarm.toml` (TOML format). Use a simple TOML parser or read/write as text with regex for the two editable fields (backend, model).
- State path is `~/.swarm/state.json` (hardcoded in main process as `path.join(os.homedir(), '.swarm', 'state.json')`)
- Logs dir is `~/swarm/logs/` (hardcoded in main process as `path.join(os.homedir(), 'swarm', 'logs')`)
- The IPC handler for reading config should parse the TOML file. For writing, update only the changed fields. Consider using the `@iarna/toml` package or simple string replacement to avoid adding heavy dependencies.
- Follow the existing dark theme styling (Tailwind classes like `bg-secondary/30`, `text-foreground`, `border-border`, etc.)
- The settings panel should use a similar layout pattern to the existing TaskDrawer or AgentDetailView for visual consistency.
- From ELECTRON_PLAN.md Settings Panel spec:
  ```
  Backend: ○ Claude Code / ○ Cursor
  Default Model: [dropdown]
  State Path: ~/.swarm/state.json
  Logs Directory: ~/.swarm/logs/
  ```

## Completion Notes

Implemented by agent cf38fc45.

### What was done:
- **Created** `electron/src/renderer/components/SettingsPanel.tsx` — Settings panel with radio buttons for backend (Claude Code / Cursor), dropdown for model (opus/sonnet/haiku), and read-only displays for state path and logs directory. Save button with dirty detection and toast notifications.
- **Modified** `electron/src/main/index.ts` — Added `settings:read` and `settings:write` IPC handlers that parse `swarm/.swarm.toml` using regex and write back changes via string replacement.
- **Modified** `electron/src/preload/index.ts` — Added `settings` context bridge API with `read()` and `write()` methods, plus `SwarmConfig` and `SettingsAPI` types.
- **Modified** `electron/src/renderer/vite-env.d.ts` — Added `settings` property to Window interface.
- **Modified** `electron/src/renderer/App.tsx` — Added `settingsOpen` state, "Open settings" command in palette, and renders SettingsPanel in center area when active.

All acceptance criteria met. Both renderer and main process TypeScript compile without errors.
