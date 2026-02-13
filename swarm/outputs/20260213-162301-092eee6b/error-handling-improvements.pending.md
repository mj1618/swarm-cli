# Task: Add Error Handling and React Error Boundary

**Phase:** 1 - Core Foundation
**Priority:** Medium
**Reporter:** Agent 74c9b46a (code review)

## Goal

Improve error handling across the Electron renderer to prevent silent failures and app crashes.

## Issues Found

### 1. Missing try-catch in agent control handlers (App.tsx:44-56)

`handleKill`, `handlePause`, and `handleResume` are async but have no error handling. If the IPC call fails, the error is silently swallowed and `fetchAgents()` runs anyway.

### 2. Unhandled promise from `window.fs.watch()` (FileTree.tsx:40)

The file watcher initialization has no error handling. If `fs.watch()` fails (e.g. swarm/ directory doesn't exist), the promise rejection is unhandled.

### 3. No React Error Boundary

The app has no error boundary component. Any unhandled component error crashes the entire UI.

## Files to Modify

- `electron/src/renderer/App.tsx` — Wrap agent handlers in try-catch, show toast/error on failure
- `electron/src/renderer/components/FileTree.tsx` — Add try-catch around `window.fs.watch()`
- `electron/src/renderer/components/ErrorBoundary.tsx` (new) — Basic React error boundary component
- `electron/src/renderer/main.tsx` — Wrap App with ErrorBoundary

## Acceptance Criteria

1. Agent control buttons (Kill/Pause/Resume) show an error message when the operation fails
2. File watcher initialization failure is handled gracefully
3. React error boundary catches component render errors and shows a fallback UI
4. The app builds successfully

## Dependencies

- file-tree-component (completed)
