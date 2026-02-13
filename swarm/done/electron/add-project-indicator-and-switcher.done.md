# Task: Add Project Indicator and Workspace Switcher to Title Bar

**Phase:** 5 - Polish
**Priority:** Medium

## Goal

The ELECTRON_PLAN.md mockup shows `[Project: ~/code/myapp]` in the title bar, but the current implementation only displays "Swarm Desktop" with no workspace information. Add a project path indicator to the title bar and a button to open/switch to a different project directory.

This makes it clear which project the user is working in (especially when running multiple instances) and provides the ability to switch workspaces without restarting the app.

## Files

- **`electron/src/main/index.ts`** — Add an IPC handler for opening a directory picker dialog and for getting/setting the current working directory. When the workspace changes, update all file watchers (state.json watcher, swarm.yaml watcher, logs watcher) to point at the new directory.
- **`electron/src/preload/index.ts`** — Expose `workspace.getCwd()` and `workspace.open()` APIs via contextBridge.
- **`electron/src/renderer/App.tsx`** — Add state for the current project path, display it in the title bar, wire up the switcher button. When the workspace changes, reload the swarm.yaml and file tree.

## Dependencies

None — this is a standalone feature that enhances the existing title bar.

## Acceptance Criteria

1. The title bar displays the current working directory as a shortened path (e.g., `~/code/myapp`) next to "Swarm Desktop"
2. A button or clickable area in the title bar opens a native OS directory picker dialog
3. After selecting a new directory, the app:
   - Updates the displayed project path
   - Reloads the file tree for the new `swarm/` subdirectory
   - Reloads `swarm.yaml` from the new directory
   - Re-initializes file watchers for the new paths
   - Updates the Electron window title to include the project name
4. If the selected directory has no `swarm/` subdirectory, show a toast warning
5. The initial project path is determined from the CWD the Electron app was launched from

## Notes

- From ELECTRON_PLAN.md main layout mockup: `Swarm Desktop                                          [Project: ~/code/myapp]`
- Use `electron.dialog.showOpenDialog` with `properties: ['openDirectory']` for the native folder picker
- Shorten the home directory prefix to `~` for display (e.g., `/Users/matt/code/myapp` → `~/code/myapp`)
- Consider persisting the last-opened workspace path in localStorage so the app can re-open it on next launch
- The main process already tracks the working directory via `process.cwd()` — the new IPC handler should allow changing this and updating all watchers accordingly

## Completion Notes

Implemented by agent 36d489c3:

- **Main process** (`electron/src/main/index.ts`): Added `workspace:getCwd` and `workspace:open` IPC handlers. `workspace:open` uses `dialog.showOpenDialog` with `openDirectory` property, checks for `swarm/` subdirectory, updates the working directory and swarm root, restarts all three file watchers (swarm, state, logs), and updates the Electron window title.
- **Preload** (`electron/src/preload/index.ts`): Exposed `workspace.getCwd()` and `workspace.open()` via contextBridge with `WorkspaceAPI` type definition.
- **Renderer** (`electron/src/renderer/App.tsx`): Added `projectPath` state persisted to localStorage, `shortenHomePath` utility to abbreviate home directory prefix to `~`, a clickable button in the title bar showing the current project path, `handleOpenProject` callback that opens the directory picker, reloads workspace state, and shows toast notifications. Also added "Open project" command to the command palette.
