# Task: Fix Pre-existing TypeScript Errors

## Goal

Fix TypeScript compiler errors across the electron app so that `tsc --noEmit` passes cleanly.

## Current Errors

1. `src/renderer/components/SettingsPanel.tsx(21)` — `systemNotifications` and `setSystemNotifications` are declared but never read (TS6133)

## Files

- **Modify**: `electron/src/renderer/components/SettingsPanel.tsx` — Remove or use unused variables

## Dependencies

- None

## Acceptance Criteria

1. `cd electron && npx tsc --noEmit` passes with no errors
2. `npm run build` (which runs `tsc && vite build`) succeeds
3. No functional regressions

## Notes

- These are pre-existing issues, not caused by recent feature work
- The SettingsPanel likely has a planned feature for system notifications that hasn't been wired up yet — either use the state or remove it
