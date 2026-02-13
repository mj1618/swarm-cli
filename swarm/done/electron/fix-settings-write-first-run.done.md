# Task: Fix settings:write Failure on First Run (Missing Config File)

**Phase:** 5 - Polish (Bug fix for first-run experience)
**Priority:** Medium
**Status:** COMPLETED

## Goal

The `settings:write` IPC handler in `electron/src/main/index.ts` fails when the config file (`swarm/.swarm.toml`) doesn't exist. On a fresh project without this file, users cannot save backend/model settings from the Settings panel because the handler tries to `readFile` first and throws ENOENT.

## Files

### Modify
- `electron/src/main/index.ts`

## Dependencies

None — standalone bug fix.

## Completion Notes

**Implemented by:** Agent 8ccfa3f1
**Date:** 2026-02-13

### Changes Made

Modified the `settings:write` IPC handler in `electron/src/main/index.ts` (around line 637) to handle the ENOENT case when the config file doesn't exist:

- Added nested try-catch to detect missing config file
- When file is missing (ENOENT), create default content: `backend = "claude-code"\nmodel = "sonnet"\n`
- Re-throw other read errors to preserve original error handling
- Apply updates (backend/model) via regex replacement on the content
- Write the resulting content to create/update the file

### Acceptance Criteria Verification

1. **Fresh project (no config file):** The handler now creates `swarm/.swarm.toml` with defaults and applies any updates
2. **Existing project:** Read-modify-write behavior preserved (unchanged code path)
3. **Build:** `npm run build` passes successfully
4. **No regressions:** `settings:read` already handles ENOENT and returns defaults (unchanged)
