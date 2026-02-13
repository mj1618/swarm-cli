# Fix CLI Command Mismatches in Electron App

## Goal

The Electron app uses non-existent or wrong CLI subcommand names when invoking the `swarm` CLI, causing pipeline execution, agent pause, and agent resume to be completely broken. Fix all command invocations to match the actual CLI.

## Problem

The Electron app was built against an assumed CLI interface that doesn't match the real `swarm` commands:

| Feature | Electron App Uses (WRONG) | Real CLI Command |
|---|---|---|
| Run pipeline | `swarm pipeline --name <name>` | `swarm up <name>` |
| Run all pipelines | `swarm pipeline` | `swarm up` |
| Pause agent | `swarm pause <id>` | `swarm stop <id>` |
| Resume agent | `swarm resume <id>` | `swarm start <id>` |

Commands that are already correct (no change needed): `kill`, `list`, `run`, `logs`, `inspect`, `update`, `clone`, `replay`.

## Files to Modify

### 1. `electron/src/main/index.ts`

**IPC handler `swarm:pause` (line ~68-70):**
- Change `runSwarmCommand(['pause', agentId])` to `runSwarmCommand(['stop', agentId])`

**IPC handler `swarm:resume` (line ~72-74):**
- Change `runSwarmCommand(['resume', agentId])` to `runSwarmCommand(['start', agentId])`

### 2. `electron/src/renderer/App.tsx`

**`handleRunPipeline` callback (line ~405):**
- Change `window.swarm.run(['pipeline', '--name', pipelineName])` to `window.swarm.run(['up', pipelineName])`

**Command palette "Run pipeline: X" commands (line ~794):**
- Change `window.swarm.run(['pipeline', '--name', name])` to `window.swarm.run(['up', name])`

**Command palette "Run all pipelines" command (line ~802):**
- Change `window.swarm.run(['pipeline'])` to `window.swarm.run(['up'])`

**Command palette "Pause all agents" (line ~828):**
- Change `window.swarm.pause(a.id)` — this calls the IPC handler which is fixed in main/index.ts, so no renderer change needed here.

**Command palette "Resume all agents" (line ~836):**
- Same as above — calls IPC handler, no change needed here.

### 3. `electron/src/renderer/components/AgentPanel.tsx`

The `handlePause` (line ~52) and `handleResume` (line ~60) call `window.swarm.pause()` and `window.swarm.resume()` respectively. These delegate to IPC handlers in main/index.ts, so fixing the main process handlers is sufficient.

## Dependencies

None — this is a standalone bug fix.

## Acceptance Criteria

1. Clicking "Run" on a pipeline in PipelineConfigBar invokes `swarm up <pipeline-name>`
2. "Run pipeline: X" from command palette invokes `swarm up <name>`
3. "Run all pipelines" from command palette invokes `swarm up`
4. Pause button on agents invokes `swarm stop <id>`
5. Resume button on agents invokes `swarm start <id>`
6. The app compiles without TypeScript errors after changes

## Notes

- The `swarm stop` command pauses a running agent (Short: "Pause a running agent")
- The `swarm start` command resumes a paused agent (Short: "Resume a paused agent")
- The `swarm up` command runs tasks/pipelines from a compose file, with optional task/pipeline name arguments
- There is no `swarm pipeline` subcommand in the CLI at all
- There is no `swarm pause` or `swarm resume` subcommand — the analogous commands are `stop` and `start`
- The `swarm up` command uses positional args (not `--name` flag) to specify which pipeline/tasks to run

## Completion Notes

All 5 command mismatches have been fixed:

1. **`electron/src/main/index.ts`**: Changed `swarm:pause` IPC handler from `['pause', agentId]` to `['stop', agentId]`, and `swarm:resume` from `['resume', agentId]` to `['start', agentId]`.
2. **`electron/src/renderer/App.tsx`**: Changed `handleRunPipeline` from `['pipeline', '--name', pipelineName]` to `['up', pipelineName]`. Changed command palette "Run pipeline: X" from `['pipeline', '--name', name]` to `['up', name]`. Changed command palette "Run all pipelines" from `['pipeline']` to `['up']`.

Build verified successfully with `npm run build`.
