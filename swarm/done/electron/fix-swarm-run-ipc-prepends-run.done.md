# Fix: swarm:run IPC handler incorrectly prepends "run" to all commands

## Goal

Fix the `swarm:run` IPC handler in the Electron main process that unconditionally prepends `"run"` to all arguments. This causes pipeline execution from the UI to be completely broken — calling `window.swarm.run(['pipeline', '--name', 'main'])` results in the shell command `swarm run pipeline --name main` instead of the correct `swarm pipeline --name main`.

## Files

- `electron/src/main/index.ts` — Replace the `swarm:run` handler with a generic `swarm:exec` handler (or rename it) that passes args directly to `runSwarmCommand` without prepending `"run"`
- `electron/src/preload/index.ts` — Update the IPC channel name if renamed, and update types
- `electron/src/renderer/App.tsx` — Update all call sites:
  - `handleRunPipeline`: `window.swarm.run(['pipeline', '--name', pipelineName])` — these args should be passed through directly
  - `handleRunTask`: `window.swarm.run(['run', ...])` — these already include `'run'` as the first arg, so they're correct after the fix
  - Command palette pipeline commands: same issue as `handleRunPipeline`

## Dependencies

None — this is a standalone critical bug fix.

## Acceptance Criteria

1. `handleRunPipeline('main')` executes `swarm pipeline --name main` (not `swarm run pipeline --name main`)
2. `handleRunTask(...)` still executes `swarm run -p <prompt> -n 1 -d` correctly
3. Command palette "Run pipeline: ..." commands execute the correct `swarm pipeline` command
4. All existing `window.swarm.run(...)` call sites pass the correct first argument (`run`, `pipeline`, etc.)
5. The preload types and IPC channel are consistent

## Notes

The simplest fix is to change the main process handler from:
```typescript
ipcMain.handle('swarm:run', async (_event, args: string[]) => {
  return runSwarmCommand(['run', ...args])  // BUG: always prepends "run"
})
```
to:
```typescript
ipcMain.handle('swarm:run', async (_event, args: string[]) => {
  return runSwarmCommand(args)  // Pass args through directly
})
```

Then verify all call sites already include the correct subcommand as their first arg:
- `handleRunPipeline`: `['pipeline', '--name', name]` — correct
- `handleRunTask`: `['run', '-p', prompt, ...]` — already has `'run'`
- Command palette: `['pipeline', '--name', name]` and `['pipeline']` — correct

This is the cleanest fix since `handleRunTask` already includes `'run'` as the first argument in its args array (line 414: `const args: string[] = ['run']`).

## Completion Notes

**Fixed** by removing the `'run'` prefix from the `swarm:run` IPC handler in `electron/src/main/index.ts:61`. Changed `runSwarmCommand(['run', ...args])` to `runSwarmCommand(args)`. All call sites already pass the correct subcommand as the first argument, so no renderer or preload changes were needed. Build verified passing.
