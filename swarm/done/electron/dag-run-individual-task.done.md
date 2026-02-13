# Task: Add "Run Task" to DAG Context Menu

**Phase:** 4 - Agent Management
**Priority:** Medium
**Status:** Completed

## What Was Implemented

### 1. DagCanvas context menu - "Run Task" option
- Added `onRunTask` prop to `DagCanvasProps` interface accepting `(taskName: string, taskDef: TaskDef) => void`
- Added "Run Task" button above "Delete Task" in the right-click context menu
- Context menu now shows when either `onRunTask` or `onDeleteTask` is available
- `handleContextMenuRun` callback looks up the task definition from the parsed compose and invokes `onRunTask`

### 2. handleRunTask in App.tsx
- Builds the `swarm run` command args from the task definition:
  - `prompt-file` -> `-f <path>`
  - `prompt-string` -> `-s "<string>"`
  - `prompt` -> `-p <name>`
  - `model` -> `-m <model>`
  - Always appends `-n 1 -d` for single detached execution
- Shows success toast: `Started agent for task "<name>"`
- Shows error toast on failure with stderr details
- Shows error toast if task has no prompt configured

### 3. Command palette integration
- Added per-task "Run task: <name>" commands dynamically from `currentCompose.tasks`
- Commands appear alongside existing per-pipeline run commands
- Each command calls `handleRunTask` with the task name and definition

## Files Modified
- `electron/src/renderer/components/DagCanvas.tsx` - Added `onRunTask` prop, context menu "Run Task" button, `handleContextMenuRun`
- `electron/src/renderer/App.tsx` - Added `handleRunTask`, passed to DagCanvas, added per-task command palette entries

## Verification
- App builds successfully with `npm run build`
