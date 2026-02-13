# Task: Fix Command Palette IPC Error Handling

**Phase:** 4 - Agent Management (bug fix)
**Priority:** Medium

## Goal

Fix fire-and-forget IPC calls in `App.tsx` command palette actions. The `window.swarm.pause`, `window.swarm.resume`, and `window.swarm.kill` calls don't check the result and don't provide user feedback. Since these IPC calls return `{ code, stdout, stderr }` and never reject, errors are silently ignored.

## Files

### Modify
- `electron/src/renderer/App.tsx`

## What to Change

### Fix "Pause all agents" command (around line 853-855)

**Before:**
```typescript
action: () => {
  agents.filter(a => a.status === 'running' && !a.paused).forEach(a => window.swarm.pause(a.id))
},
```

**After:**
```typescript
action: async () => {
  const toPause = agents.filter(a => a.status === 'running' && !a.paused)
  const results = await Promise.all(toPause.map(a => window.swarm.pause(a.id)))
  const failed = results.filter(r => r.code !== 0).length
  if (failed > 0) {
    addToast('error', `Failed to pause ${failed} agent(s)`)
  } else if (toPause.length > 0) {
    addToast('success', `Paused ${toPause.length} agent(s)`)
  }
},
```

### Fix "Resume all agents" command (around line 861-863)

**Before:**
```typescript
action: () => {
  agents.filter(a => a.paused).forEach(a => window.swarm.resume(a.id))
},
```

**After:**
```typescript
action: async () => {
  const toResume = agents.filter(a => a.paused)
  const results = await Promise.all(toResume.map(a => window.swarm.resume(a.id)))
  const failed = results.filter(r => r.code !== 0).length
  if (failed > 0) {
    addToast('error', `Failed to resume ${failed} agent(s)`)
  } else if (toResume.length > 0) {
    addToast('success', `Resumed ${toResume.length} agent(s)`)
  }
},
```

### Fix "Kill all agents" command (around line 869-871)

**Before:**
```typescript
action: () => {
  agents.filter(a => a.status === 'running').forEach(a => window.swarm.kill(a.id))
},
```

**After:**
```typescript
action: async () => {
  const toKill = agents.filter(a => a.status === 'running')
  const results = await Promise.all(toKill.map(a => window.swarm.kill(a.id)))
  const failed = results.filter(r => r.code !== 0).length
  if (failed > 0) {
    addToast('error', `Failed to stop ${failed} agent(s)`)
  } else if (toKill.length > 0) {
    addToast('success', `Stopped ${toKill.length} agent(s)`)
  }
},
```

### Fix per-agent kill command (around line 939)

**Before:**
```typescript
action: () => { window.swarm.kill(a.id) },
```

**After:**
```typescript
action: async () => {
  const result = await window.swarm.kill(a.id)
  if (result.code !== 0) {
    addToast('error', `Failed to stop agent: ${result.stderr}`)
  } else {
    addToast('success', 'Agent stopped')
  }
},
```

### Fix per-agent pause command (around line 945)

**Before:**
```typescript
action: () => { window.swarm.pause(a.id) },
```

**After:**
```typescript
action: async () => {
  const result = await window.swarm.pause(a.id)
  if (result.code !== 0) {
    addToast('error', `Failed to pause agent: ${result.stderr}`)
  } else {
    addToast('success', 'Agent paused')
  }
},
```

### Fix per-agent resume command (around line 953)

**Before:**
```typescript
action: () => { window.swarm.resume(a.id) },
```

**After:**
```typescript
action: async () => {
  const result = await window.swarm.resume(a.id)
  if (result.code !== 0) {
    addToast('error', `Failed to resume agent: ${result.stderr}`)
  } else {
    addToast('success', 'Agent resumed')
  }
},
```

## Dependencies

- None (standalone bug fix)

## Acceptance Criteria

1. Bulk pause/resume/kill actions show toast feedback with count of affected agents
2. Individual agent pause/resume/kill from command palette show success/error toasts
3. Error messages include stderr when available
4. No silent failures for any agent control actions
5. App builds with `npm run build`

## Notes

- This is a follow-up to the fix in AgentPanel.tsx (same pattern)
- The `addToast` function is already available in the scope where these commands are defined
- The `Command` interface action type is `() => void`, so the async actions should still work (they return Promise<void> which is compatible)
