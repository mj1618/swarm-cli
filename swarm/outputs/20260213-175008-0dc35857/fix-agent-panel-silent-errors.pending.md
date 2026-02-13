# Fix: AgentPanel IPC error handling is silently swallowed

## Goal

Fix several agent control actions in `AgentPanel.tsx` that silently swallow IPC errors. The `window.swarm.*` IPC calls always resolve (never reject), returning a `{code, stderr, stdout}` result object. The current try/catch pattern never catches errors because the promise never rejects. Instead, the code should check `result.code !== 0`.

## Files

- `electron/src/renderer/components/AgentPanel.tsx` — Fix error handling in `handlePause`, `handleResume`, `handleKill`, `handleClone`, `handleReplay`

## Dependencies

None — this is a standalone bug fix.

## Acceptance Criteria

1. `handlePause` checks `result.code !== 0` and shows an error toast with stderr info
2. `handleResume` checks `result.code !== 0` and shows an error toast with stderr info
3. `handleKill` checks `result.code !== 0` and shows an error toast with stderr info
4. `handleClone` checks `result.code !== 0` and shows an error toast with stderr info (currently missing)
5. `handleReplay` checks `result.code !== 0` and shows an error toast with stderr info (currently missing)
6. Success toasts still fire on code === 0
7. No regression for `handleSetIterations` and `handleSetModel` which already check correctly

## Notes

The pattern should follow `handleSetIterations` / `handleSetModel` which already correctly check `result.code`:

```typescript
const handlePause = async (agentId: string) => {
  const result = await window.swarm.pause(agentId)
  if (result.code !== 0) {
    onToast?.('error', `Failed to pause agent: ${result.stderr}`)
  } else {
    onToast?.('success', 'Agent paused')
  }
}
```
