# Task: Pipeline Configuration UI

**Phase:** 3 - Interactive Editing
**Priority:** High
**Status:** COMPLETED

## What Was Implemented

### 1. Pipeline CRUD in yamlWriter.ts
- Extended `applyPipelineEdits()` to support `tasks` array parameter for pipeline task membership
- Added `deletePipeline()` function to remove a pipeline from compose data
- Both functions use `structuredClone()` for immutability, matching existing patterns

### 2. PipelinePanel Component (`PipelinePanel.tsx`)
- Full pipeline configuration panel following TaskDrawer pattern
- Creation mode (empty name) with name validation
- Edit mode for existing pipelines
- Iterations and parallelism number inputs
- Task checklist with Select All/None helpers and count display
- Delete button for existing pipelines
- Save/Cancel footer
- Escape key to close, auto-focus name input on create
- Consistent Tailwind dark theme styling

### 3. App.tsx Integration
- Added `selectedPipeline` state for pipeline panel visibility
- Pipeline save handler (`handleSavePipeline`) with YAML write-back and reload
- Pipeline delete handler (`handleDeletePipeline`) with active pipeline cleanup
- Pipeline edit/create handlers connected to PipelineConfigBar
- Right sidebar shows PipelinePanel when editing (mutual exclusion with TaskDrawer and AgentPanel)
- Dynamic command palette entries: "Run pipeline: <name>" for each defined pipeline
- "Create new pipeline" command in palette

### 4. PipelineConfigBar Enhancements
- Added "Configure" button (visible when a pipeline is selected) to open full PipelinePanel
- Added "+ New Pipeline" button to create pipelines directly from the bar
- Bar now shows even with no pipelines (to allow creating the first one)

### 5. DAG Canvas Pipeline Filter
- Pipeline filter already implemented via `filteredNodes` useMemo
- Dims tasks not in the active pipeline (opacity 0.35)
- Uses `pipelineTasks` and `activePipeline` props from App.tsx

### 6. Command Palette
- Replaced hardcoded "Run pipeline: main" with dynamic entries for all defined pipelines
- Added "Create new pipeline" command

## Files Modified
- `electron/src/renderer/lib/yamlWriter.ts` — Extended `applyPipelineEdits`, added `deletePipeline`
- `electron/src/renderer/components/PipelinePanel.tsx` — New component
- `electron/src/renderer/components/PipelineConfigBar.tsx` — Added edit/create buttons
- `electron/src/renderer/App.tsx` — Full pipeline panel integration, dynamic palette commands
