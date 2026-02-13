# Task: Add User-Facing Error Feedback for AgentPanel Control Actions

**Phase:** 5 - Polish
**Priority:** Medium

## Goal

The AgentPanel component's agent control actions (set iterations, set model, clone, pause, resume, kill) silently fail with `console.error` when operations fail. Users get no visual feedback. Add toast notifications for both success and error outcomes of these actions.

## Files to Modify

1. **`electron/src/renderer/components/AgentPanel.tsx`** — Accept an `onToast` prop (or use a shared toast context) and replace `console.error` calls with user-visible toast notifications. Also add success toasts for successful operations.
2. **`electron/src/renderer/App.tsx`** — Pass `addToast` to `AgentPanel` as an `onToast` prop.

## Current Behavior

In `AgentPanel.tsx`, lines 57-75:

```tsx
const handleSetIterations = async (agentId: string, iterations: number) => {
  const result = await window.swarm.run(['update', agentId, '--iterations', String(iterations)])
  if (result.code !== 0) {
    console.error('Failed to set iterations:', result.stderr)  // Silent!
  }
}

const handleSetModel = async (agentId: string, model: string) => {
  const result = await window.swarm.run(['update', agentId, '--model', model])
  if (result.code !== 0) {
    console.error('Failed to set model:', result.stderr)  // Silent!
  }
}

const handleClone = async (agentId: string) => {
  const result = await window.swarm.run(['clone', agentId, '-d'])
  if (result.code !== 0) {
    console.error('Failed to clone agent:', result.stderr)  // Silent!
  }
}
```

Also `handlePause`, `handleResume`, and `handleKill` (lines 45-55) have no error handling at all — if the IPC call fails the promise rejection is unhandled.

## Implementation Details

### 1. Add `onToast` prop to AgentPanel

```tsx
interface AgentPanelProps {
  onToast: (type: 'success' | 'error' | 'warning' | 'info', message: string) => void
}
```

### 2. Update handlers with error + success toasts

```tsx
const handleSetIterations = async (agentId: string, iterations: number) => {
  const result = await window.swarm.run(['update', agentId, '--iterations', String(iterations)])
  if (result.code !== 0) {
    onToast('error', `Failed to set iterations: ${result.stderr}`)
  } else {
    onToast('success', `Updated iterations to ${iterations}`)
  }
}

const handleSetModel = async (agentId: string, model: string) => {
  const result = await window.swarm.run(['update', agentId, '--model', model])
  if (result.code !== 0) {
    onToast('error', `Failed to set model: ${result.stderr}`)
  } else {
    onToast('success', `Updated model to ${model}`)
  }
}

const handleClone = async (agentId: string) => {
  const result = await window.swarm.run(['clone', agentId, '-d'])
  if (result.code !== 0) {
    onToast('error', `Failed to clone agent: ${result.stderr}`)
  } else {
    onToast('success', 'Agent cloned')
  }
}
```

### 3. Add try-catch to pause/resume/kill handlers

```tsx
const handlePause = async (agentId: string) => {
  try {
    await window.swarm.pause(agentId)
  } catch {
    onToast('error', 'Failed to pause agent')
  }
}
// Similarly for handleResume and handleKill
```

### 4. Update App.tsx to pass onToast

```tsx
<AgentPanel onToast={addToast} />
```

## Dependencies

- Toast notification system (completed)

## Acceptance Criteria

1. Setting iterations on an agent shows a success toast when it works, or an error toast with the error message when it fails
2. Setting model shows success/error toast feedback
3. Cloning an agent shows success/error toast feedback
4. Pause/resume/kill actions are wrapped in try-catch and show error toasts on failure
5. `console.error` calls are removed or supplemented with toasts
6. TypeScript compiles without errors
7. App builds successfully with `npm run build`

## Completion Notes

Implemented by agent 9ae424d6. Changes:
- Added optional `onToast` prop to `AgentPanelProps` interface
- Wrapped `handlePause`, `handleResume`, `handleKill` in try-catch blocks with error toasts
- Replaced `console.error` calls in `handleSetIterations`, `handleSetModel`, `handleClone` with toast notifications (both success and error)
- Passed `addToast` from App.tsx to `<AgentPanel onToast={addToast} />`
- Build verified successfully
