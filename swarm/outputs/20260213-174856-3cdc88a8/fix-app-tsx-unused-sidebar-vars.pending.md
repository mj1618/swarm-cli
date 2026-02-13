# Task: Fix unused sidebar resize variables breaking TypeScript build

## Goal

Remove or properly integrate unused sidebar resize state variables in App.tsx that cause TypeScript compilation errors (TS6133).

**Phase:** Bugfix

## Problem

The following variables were added to App.tsx but are never used, causing `npm run build` to fail with TS6133 errors:

- `MIN_SIDEBAR_WIDTH` (constant, declared but never read)
- `MAX_SIDEBAR_WIDTH` (constant, declared but never read)
- `DEFAULT_LEFT_SIDEBAR_WIDTH` (referenced but never defined as a constant)
- `DEFAULT_RIGHT_SIDEBAR_WIDTH` (referenced but never defined as a constant)
- `leftSidebarWidth` / `setLeftSidebarWidth` (useState, declared but never read)
- `rightSidebarWidth` / `setRightSidebarWidth` (useState, declared but never read)
- `isDraggingLeftSidebar` (useRef, declared but never read)
- `isDraggingRightSidebar` (useRef, declared but never read)
- `sidebarDragStartX` (useRef, declared but never read)
- `sidebarDragStartWidth` (useRef, declared but never read)

Also `shortenHomePath`, `projectPath`/`setProjectPath`, and `handleOpenProject` may have been added without being fully wired up.

## Fix

Either:
1. Complete the sidebar resizing feature by wiring these variables into the JSX (preferred if the feature is in progress)
2. Or remove all the unused declarations until they're needed

## Acceptance Criteria

1. `npm run build` passes with no TS6133 errors
2. No functionality is broken
