# Task: Fix settings:write to add missing config fields

**Phase:** 5 - Polish (bug fix)
**Priority:** Medium

## Goal

Fix the `settings:write` IPC handler to properly add backend/model fields when they're missing from the config file, rather than silently failing when the regex doesn't match.

## Files

### Modify
- `electron/src/main/index.ts` — Update the `settings:write` handler (lines 662-686)

## Problem

The current implementation uses regex replacement to update fields:
```typescript
content = content.replace(/^(backend\s*=\s*)"[^"]*"/m, `$1"${updates.backend}"`)
```

If the config file exists but doesn't contain the `backend` or `model` field, the regex won't match and the update silently fails. This can happen if a user manually edits their config file.

## Solution

Check if the field exists before replacement. If missing, prepend it to the file:

```typescript
if (updates.backend !== undefined) {
  if (/^backend\s*=/m.test(content)) {
    content = content.replace(/^(backend\s*=\s*)"[^"]*"/m, `$1"${updates.backend}"`)
  } else {
    content = `backend = "${updates.backend}"\n` + content
  }
}

if (updates.model !== undefined) {
  if (/^model\s*=/m.test(content)) {
    content = content.replace(/^(model\s*=\s*)"[^"]*"/m, `$1"${updates.model}"`)
  } else {
    content = `model = "${updates.model}"\n` + content
  }
}
```

## Acceptance Criteria

1. Setting `backend` when missing from config file adds the field
2. Setting `model` when missing from config file adds the field  
3. Existing fields are still updated correctly (no regression)
4. TypeScript compiles: `npx tsc --noEmit` passes
5. App builds: `npm run build` succeeds

## Dependencies

None - standalone bug fix

## Notes

- This is a Phase 5 polish task - all core features are complete
- The default config template includes both fields, so this mainly affects manually edited configs
- Simple regex-based fix maintains consistency with existing TOML handling approach

---

## Completion Note

**Completed by:** Agent 566aebc8
**Date:** 2026-02-13

### What was implemented:

Updated the `settings:write` IPC handler in `electron/src/main/index.ts` to check if `backend` and `model` fields exist in the config file before attempting replacement. If a field is missing, it is now prepended to the file content instead of silently failing.

### Verification:
- TypeScript compiles without errors (`npx tsc --noEmit` passed)
- App builds successfully (`npm run build` passed)
