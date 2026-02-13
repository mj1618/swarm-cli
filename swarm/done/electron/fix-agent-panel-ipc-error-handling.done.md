# Task: Fix AgentPanel IPC Error Handling

**Phase:** 4 - Agent Management (bug fix)
**Priority:** Medium

## Goal

Fix incorrect error handling in `AgentPanel.tsx` where `handlePause`, `handleResume`, and `handleKill` use try/catch to handle errors, but the IPC calls (`window.swarm.pause`, `window.swarm.resume`, `window.swarm.kill`) always resolve with a result object `{ code, stdout, stderr }` - they never reject. This means errors are silently swallowed.

## Files

### Modify
- `electron/src/renderer/components/AgentPanel.tsx`

## What to Change

### Fix `handlePause` (around line 69)

**Before:**
```typescript
const handlePause = async (agentId: string) => {
  try {
    await window.swarm.pause(agentId)
  } catch {
    onToast?.('error', 'Failed to pause agent')
  }
}
```

**After:**
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

### Fix `handleResume` (around line 77)

**Before:**
```typescript
const handleResume = async (agentId: string) => {
  try {
    await window.swarm.resume(agentId)
  } catch {
    onToast?.('error', 'Failed to resume agent')
  }
}
```

**After:**
```typescript
const handleResume = async (agentId: string) => {
  const result = await window.swarm.resume(agentId)
  if (result.code !== 0) {
    onToast?.('error', `Failed to resume agent: ${result.stderr}`)
  } else {
    onToast?.('success', 'Agent resumed')
  }
}
```

### Fix `handleKill` (around line 85)

**Before:**
```typescript
const handleKill = async (agentId: string) => {
  try {
    await window.swarm.kill(agentId)
  } catch {
    onToast?.('error', 'Failed to kill agent')
  }
}
```

**After:**
```typescript
const handleKill = async (agentId: string) => {
  const result = await window.swarm.kill(agentId)
  if (result.code !== 0) {
    onToast?.('error', `Failed to stop agent: ${result.stderr}`)
  } else {
    onToast?.('success', 'Agent stopped')
  }
}
```

## Dependencies

- None (standalone bug fix)

## Acceptance Criteria

1. `handlePause` checks `result.code !== 0` and shows error toast with stderr on failure, success toast otherwise
2. `handleResume` checks `result.code !== 0` and shows error toast with stderr on failure, success toast otherwise
3. `handleKill` checks `result.code !== 0` and shows error toast with stderr on failure, success toast otherwise
4. Error messages include the stderr output for debugging
5. Success toasts confirm the action completed
6. App builds with `npm run build`

## Notes

- This follows the same pattern already used by `handleSetIterations` and `handleSetModel` (lines 93-109) which correctly check `result.code`
- The `handleClone` and `handleReplay` functions already use the correct pattern
- The IPC types in `preload/index.ts` confirm these methods return `Promise<{ stdout, stderr, code }>` (lines 98-100)

---

## Completion Note

**Completed by:** 68438a14
**Date:** 2026-02-13

All three handlers (`handlePause`, `handleResume`, `handleKill`) were updated to:
1. Capture the result object from the IPC call instead of using try/catch
2. Check `result.code !== 0` to detect errors
3. Show error toast with stderr message on failure
4. Show success toast on successful operations

The fix now matches the pattern used by `handleSetIterations`, `handleSetModel`, `handleClone`, and `handleReplay`.

Build verified with `npm run build` - no errors.
