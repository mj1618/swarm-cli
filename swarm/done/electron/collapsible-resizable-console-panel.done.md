# Task: Make Console Panel Collapsible and Resizable

## Goal

The console panel at the bottom of the app has a fixed `h-48` height and cannot be collapsed or resized. Add the ability to:
1. Collapse/expand the console panel (toggle visibility)
2. Resize the console panel by dragging its top border
3. Persist the collapsed state and height to localStorage

This is a Phase 5 (Polish) improvement that directly enables the "Toggle console visibility" command palette command referenced in the `command-palette-missing-commands.pending.md` task.

## Files

### Modify
- `electron/src/renderer/App.tsx` — Replace the fixed `h-48` console container with a collapsible, resizable wrapper. Add `consoleCollapsed` state and `toggleConsole` callback. Expose `toggleConsole` to the command palette. Add a collapse/expand button to the console border area.
- `electron/src/renderer/components/ConsolePanel.tsx` — (Minor) Ensure the component handles zero-height gracefully when collapsed (optional, may already work).

## Dependencies

- Console panel already exists and is fully functional (`ConsolePanel.tsx`)
- Command palette exists (`CommandPalette.tsx`) — the toggle command will be wired as a follow-up or inline

## Acceptance Criteria

1. A small toggle button (chevron icon) appears on the console panel's top border bar, allowing collapse/expand
2. When collapsed, the console panel shrinks to just its top border (~28px) showing "Console" label and the toggle button
3. When expanded, the console panel shows at the last-used height
4. The user can drag the top border of the console panel to resize it (drag handle cursor on hover)
5. The minimum expanded height is 100px; maximum is 60% of the window height
6. Collapsed state and panel height persist to localStorage across app restarts (keys: `swarm-console-collapsed`, `swarm-console-height`)
7. A "Toggle console" command is available in the command palette (Cmd+K)
8. Keyboard shortcut: Cmd+J / Ctrl+J toggles the console panel

## Notes

- Use `mousedown`/`mousemove`/`mouseup` event handlers for the drag-to-resize interaction. Set `cursor: row-resize` on the drag handle.
- Store the height as a pixel value in localStorage. Default to 192px (the current h-48 = 12rem = 192px).
- When collapsed, render the console section as a thin bar with just a header. The `ConsolePanel` component itself can simply not be rendered (or rendered with `display: none`) when collapsed.
- The toggle button can be a simple `ChevronDown` / `ChevronUp` character (▼/▲) to avoid adding icon dependencies.
- Add the command palette entry inline in `App.tsx` where other commands are defined (around line 548-672).
