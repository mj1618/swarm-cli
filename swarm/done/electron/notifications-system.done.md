# Task: Implement Notifications System

**Phase:** 5 - Polish
**Priority:** Medium (Phase 5 item, after command palette)

## Goal

Add a toast notification system that alerts the user when agents complete, fail, or change status. This provides passive feedback so users don't have to constantly watch the agent panel. The ELECTRON_PLAN.md (lines 319-322) specifies:
- Toast notifications when agents complete/fail
- System notifications (optional) for long-running tasks
- Sound alerts (configurable)

This task focuses on the in-app toast notifications. System notifications and sound alerts can be follow-ups.

## Current State

- The AgentPanel already watches `state.json` via chokidar and receives real-time `state:changed` events
- Agent state includes `status` (running/terminated), `exit_reason` (completed/killed/crashed), and `last_error`
- There is no notification or toast system in the app

## What to Build

### 1. Toast Component (`ToastContainer.tsx`)

A fixed-position container (bottom-right corner) that renders a stack of toast notifications:

- Each toast has: icon (success/error/info), message text, optional agent name, auto-dismiss timer (5 seconds), manual dismiss button (X)
- Toasts stack vertically with newest at the bottom
- Max 5 visible toasts at once (oldest dismissed when limit exceeded)
- Slide-in animation from the right, fade-out on dismiss
- Styling consistent with app dark theme (zinc/slate palette)

### 2. Toast Types

| Event | Icon | Message | Color |
|-------|------|---------|-------|
| Agent completed | checkmark | "{name} completed successfully" | Green |
| Agent failed/crashed | X mark | "{name} failed: {exit_reason}" | Red |
| Agent killed | stop icon | "{name} was stopped" | Yellow |
| Agent started | play icon | "{name} started" | Blue |

### 3. Integration with App.tsx

- Track previous agent states to detect transitions (running → terminated, etc.)
- When `state:changed` fires, compare new state against previous state to detect:
  - New agents that weren't in the previous list (→ "started" toast)
  - Agents that changed from `running` to `terminated` (→ completed/failed/killed toast based on `exit_reason`)
- Expose `addToast` function via React state or a simple context

## Files to Create/Modify

- `electron/src/renderer/components/ToastContainer.tsx` (NEW) — Toast container and individual toast components
- `electron/src/renderer/App.tsx` (MODIFY) — Add toast state, render ToastContainer, wire up agent state change detection

## Dependencies

- Phase 4 state watching — already implemented (AgentPanel watches state.json)
- Agent state types — already defined in preload (`AgentState` interface)

## Acceptance Criteria

1. When an agent transitions from `running` to `terminated` with `exit_reason: "completed"`, a green success toast appears
2. When an agent terminates with `exit_reason: "crashed"` or `"killed"`, an appropriate toast appears (red for crash, yellow for kill)
3. Toasts auto-dismiss after 5 seconds
4. Toasts can be manually dismissed by clicking an X button
5. Multiple toasts stack correctly without overlapping
6. No more than 5 toasts are visible at once
7. The app builds successfully with `npm run build` in electron/

## Notes

- Reference ELECTRON_PLAN.md lines 319-322 for the notification spec
- The key challenge is detecting state transitions — need to compare previous agent list against new agent list on each `state:changed` event
- Use `useRef` to store the previous agent state map (keyed by ID) for comparison
- Keep the toast system self-contained — a simple array of `{ id, type, message, timestamp }` in state is sufficient
- Use CSS transitions or simple keyframes for slide-in/fade-out animations — no animation library needed
- System-level notifications (via Electron `Notification` API) can be a separate follow-up task
- Do NOT install any additional npm packages

## Completion Notes

Implemented by agent e9cfa7c7.

### What was built:
- **`ToastContainer.tsx`** — Self-contained toast notification system with:
  - `useToasts()` hook managing toast state, auto-dismiss timers (5s), and max 5 visible limit
  - `ToastContainer` component rendering fixed-position toast stack in bottom-right
  - `ToastItem` component with slide-in animation (translate-x + opacity transition), dismiss button
  - Four toast types: success (green), error (red), warning (yellow), info (blue)
  - Each type has distinct icon, color scheme, and border styling
- **`App.tsx`** modifications:
  - Added `useToasts()` hook and `prevAgentsRef` for tracking previous agent states
  - Added `useEffect` that compares new agents against previous agents on every `state:changed` event
  - Detects: new running agents → "started" toast; running→terminated transitions → completed/failed/stopped toasts based on `exit_reason`
  - Renders `<ToastContainer>` as last child in root div (fixed positioning, z-50)

All acceptance criteria met. Build passes successfully.
