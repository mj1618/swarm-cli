# Task: Add React Error Boundary Component

## Goal

Add a React Error Boundary component that wraps the main application panels, preventing a crash in any single panel (File Tree, DAG Canvas, Agent Panel, Console) from taking down the entire app. Display a user-friendly fallback UI with error details and a "Retry" button.

## Files

- **Create:** `electron/src/renderer/components/ErrorBoundary.tsx` — The error boundary class component
- **Modify:** `electron/src/renderer/App.tsx` — Wrap major panel sections with `<ErrorBoundary>`

## Dependencies

None — all 5 phases are complete. This is a Phase 5 (Polish) quality improvement.

## Acceptance Criteria

1. `ErrorBoundary.tsx` is a React class component (error boundaries require class components) that:
   - Catches errors from child component trees via `componentDidCatch` / `getDerivedStateFromError`
   - Renders a styled fallback UI matching the dark theme (`bg-slate-800`, `text-slate-300`)
   - Shows the error message and a "Retry" button that resets the error state
   - Accepts an optional `fallback` prop for custom fallback UI
   - Accepts an optional `name` prop to identify which panel crashed (e.g., "DAG Canvas")

2. `App.tsx` wraps these sections with `<ErrorBoundary>`:
   - Left sidebar (File Tree)
   - Center panel (DAG Canvas / File Viewer / Settings)
   - Right sidebar (Task Drawer / Agent Panel / Pipeline Panel)
   - Bottom panel (Console)

3. The app continues working when one panel crashes — other panels remain interactive.

4. The project builds successfully: `cd electron && npm run build`

## Notes

- React error boundaries **must** be class components — functional components cannot catch render errors.
- Keep the component minimal: no external dependencies needed.
- Match the existing dark theme styling used throughout the app (slate-800/900 backgrounds, slate-300 text).
- The "Retry" button should call `this.setState({ hasError: false })` to attempt re-rendering the children.
- Consider logging the error to console for debugging: `console.error('[ErrorBoundary]', name, error)`.
