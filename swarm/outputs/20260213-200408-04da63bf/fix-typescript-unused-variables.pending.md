# Task: Fix TypeScript Unused Variable Errors

**Phase:** 5 - Polish
**Priority:** Medium

## Goal

Fix TypeScript compilation errors caused by unused variables that prevent `npm run build` from completing when strict mode is enforced.

## Current Behavior

Running `tsc` reports the following unused variable errors:

```
src/renderer/App.tsx(88,10): error TS6133: 'triggerSave' is declared but its value is never read.
src/renderer/App.tsx(548,9): error TS6133: 'handleDirtyChange' is declared but its value is never read.
src/renderer/App.tsx(561,9): error TS6133: 'handleSaveComplete' is declared but its value is never read.
src/renderer/components/AgentPanel.tsx(28,10): error TS6133: 'pruning' is declared but its value is never read.
src/renderer/components/AgentPanel.tsx(144,9): error TS6133: 'handlePrune' is declared but its value is never read.
```

## Files to Modify

- `electron/src/renderer/App.tsx` — Remove or use `triggerSave`, `handleDirtyChange`, `handleSaveComplete`
- `electron/src/renderer/components/AgentPanel.tsx` — Remove or use `pruning`, `handlePrune`

## Dependencies

None

## Implementation Notes

For each unused variable:
1. Determine if the variable is intended for future use
2. If no longer needed, remove the declaration
3. If intended for future use, either:
   - Prefix with underscore (`_triggerSave`) to indicate intentionally unused
   - Use the variable in the code as originally intended
   - Add a `// TODO:` comment explaining the planned usage

## Acceptance Criteria

1. `npm run build` completes without TypeScript errors
2. No functional regressions introduced
3. Code intent is preserved (don't remove logic that should be wired up)

## Notes

These appear to be remnants of incomplete feature implementations (save functionality, agent pruning). Consider whether these features should be completed or removed.
