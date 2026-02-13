# Task: Console Log Viewer Foundation

**Phase:** 4 (partial) — Bottom Panel Enhancement
**Status:** COMPLETED

## What Was Implemented

### IPC Handlers (main/index.ts)
- `logs:list` — Lists log files in `~/swarm/logs/` with name, path, and modifiedAt
- `logs:read` — Reads a specific log file by path (with path traversal protection)
- `logs:watch` — Watches the logs directory with chokidar, pushes `logs:changed` events to renderer
- `logs:unwatch` — Stops the watcher
- Cleanup integrated into `app.on('before-quit')`

### Preload API (preload/index.ts)
- Exposed `window.logs` API: `list()`, `read(path)`, `watch()`, `unwatch()`, `onChanged(callback)`
- Added `LogEntry` interface and `LogsAPI` type
- Extended `Window` interface declaration

### Components
- **LogView.tsx** — Reusable log content display with:
  - Line-by-line rendering in monospace font
  - Line classification: errors (red), tool calls (dimmed), normal
  - Auto-scroll to bottom by default
  - "Scroll to bottom" button when user scrolls up
- **ConsolePanel.tsx** — Tabbed console panel with:
  - "Console" tab showing combined view of all logs
  - Per-agent tabs (one per log file, labeled with truncated filename)
  - Real-time watching via `logs:watch` / `onChanged`
  - "Clear" / refresh button
  - Graceful "No logs yet" state when directory is empty

### Integration (App.tsx)
- Replaced static console placeholder with `<ConsolePanel />` component

## Acceptance Criteria Met
1. Bottom panel shows tab bar with "Console" tab
2. Log files get individual tabs
3. Tab switching shows corresponding log content
4. Console tab shows combined view
5. Auto-scroll to bottom by default
6. Scroll-to-bottom toggle button when scrolled up
7. Directory watching for real-time updates
8. "No logs yet" shown when directory is empty
9. App builds successfully
