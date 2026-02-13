# Task: System Notifications for Agent Completion/Failure

## Goal

Add native OS notifications (via the Electron `Notification` API) when agents complete or fail. This implements the "Notifications" feature from Phase 5 of ELECTRON_PLAN.md, which specifies:

- Toast notifications when agents complete/fail (already done via ToastContainer)
- **System notifications (optional) for long-running tasks** (this task)
- Sound alerts (configurable) (out of scope — separate task)

The system notification should fire when:
1. An agent transitions from `running` → `terminated` (success or failure)
2. The app window is **not focused** (no point notifying if user is already looking)

## Files

- `electron/src/main/index.ts` — Add IPC handler for `notify:send` that creates an `Electron.Notification`
- `electron/src/renderer/App.tsx` — Add logic in agent state change detection to call `window.notify.send()` when agents complete/fail while window is blurred
- `electron/src/main/preload.ts` (or wherever preload bridge is defined) — Expose `notify:send` IPC channel
- `electron/src/renderer/components/SettingsPanel.tsx` — Add a toggle for enabling/disabling system notifications (default: enabled)

## Dependencies

- Agent state watching is already implemented (chokidar watcher on state.json)
- Toast notifications already fire on agent state changes in App.tsx
- The pattern for detecting agent completion is already in place (comparing previous vs current agent states)

## Acceptance Criteria

1. When an agent finishes (success or failure) and the Electron window is NOT focused, a native OS notification appears with:
   - Title: "Agent completed" or "Agent failed"
   - Body: agent name, iteration count, and cost (e.g., "planner — 20/20 iterations — $1.23")
2. Clicking the notification brings the Electron window to the foreground
3. A "System Notifications" toggle exists in SettingsPanel (persisted in localStorage)
4. When the toggle is OFF, no system notifications fire
5. No notification fires if the window IS focused (to avoid redundancy with toasts)
6. The feature uses Electron's built-in `Notification` class (no external dependencies)

## Notes

- Use `Notification` from the main process (more reliable cross-platform than the renderer's web `Notification` API)
- The renderer should send an IPC message like `notify:send({ title, body })` and the main process creates the notification
- Use `mainWindow.isFocused()` check in the main process to skip notifications when window is active
- Clicking the notification should call `mainWindow.show()` and `mainWindow.focus()`
- For the settings toggle, use `localStorage.getItem('swarm-system-notifications')` — default to `'true'`
- The existing agent state diffing logic in App.tsx (which already fires toasts) is the right place to add the notification call

## Completion Notes

Implemented by agent 69cc7a9e. All acceptance criteria met:

1. **Main process** (`electron/src/main/index.ts`): Added `notify:send` IPC handler using Electron's `Notification` class. Checks `mainWindow.isFocused()` to skip notifications when the window is active. Click handler calls `mainWindow.show()` and `mainWindow.focus()`.
2. **Preload bridge** (`electron/src/preload/index.ts`): Exposed `notify` API with `send()` method. Added `NotifyAPI` type and updated `Window` interface.
3. **Renderer** (`electron/src/renderer/App.tsx`): Added system notification call in the existing agent state transition effect. Checks `localStorage.getItem('swarm-system-notifications')` and only fires for completed/crashed agents (not killed). Notification body includes agent name, iteration count, and cost.
4. **Settings toggle** (`electron/src/renderer/components/SettingsPanel.tsx`): Added toggle switch for "System notifications when agents complete or fail" with helper text. Persisted in localStorage, defaults to enabled.
