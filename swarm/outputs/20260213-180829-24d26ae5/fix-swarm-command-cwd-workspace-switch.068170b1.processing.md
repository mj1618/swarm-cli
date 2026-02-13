# Fix: swarm CLI commands ignore workspace switch (missing cwd)

## Goal

Fix `runSwarmCommand()` in the Electron main process so that swarm CLI commands (`run`, `kill`, `pause`, `resume`, `list`, etc.) execute in the correct working directory after a workspace switch. Currently, `spawn('swarm', args)` uses the Electron app's original `process.cwd()` regardless of which project directory the user has switched to via the workspace picker.

This is a critical functional bug: after switching workspaces via the project picker (File > Open Project), all swarm CLI operations still target the original project directory.

## Files

- `electron/src/main/index.ts` — Update `runSwarmCommand()` to pass `{ cwd: workingDir }` to `spawn()`

## Dependencies

None — the workspace switching feature is already implemented (commits c66e62d, e329416). This fixes a gap in that feature.

## Acceptance Criteria

1. `runSwarmCommand()` passes `{ cwd: workingDir }` as a spawn option
2. After switching workspaces via the project picker, `swarm list`, `swarm run`, `swarm kill`, etc. operate against the new project directory
3. On initial launch (no workspace switch), behavior is unchanged — `workingDir` defaults to `process.cwd()`
4. TypeScript compiles cleanly (`tsc --noEmit` passes)

## Notes

The fix is a one-line change at `electron/src/main/index.ts:764`:

```typescript
// Before:
const proc = spawn('swarm', args)

// After:
const proc = spawn('swarm', args, { cwd: workingDir })
```

The `workingDir` variable is already maintained and updated by the `workspace:open` handler (line 373). It just isn't passed to `spawn()`.
