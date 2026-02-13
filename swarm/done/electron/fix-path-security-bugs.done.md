# Task: Fix Path Traversal Security Bugs in Main Process

**Phase:** 5 - Polish (security hardening)
**Priority:** High (security)

## Goal

Fix two related path traversal security vulnerabilities in `electron/src/main/index.ts`:

1. **`isWithinSwarmDir()` uses `startsWith()` without trailing separator** — The current implementation would incorrectly match sibling directories like `swarm-evil/` since `/path/to/swarm-evil`.startsWith(`/path/to/swarm`) returns true.

2. **`resolveIncludes()` fallback lacks path validation** — When a `{{include:path}}` directive fails the initial `isWithinSwarmDir` check, the fallback resolution relative to `swarmRoot` is NOT validated, allowing path traversal attacks (e.g. `{{include:../../../etc/passwd}}`).

## Files to Modify

- `electron/src/main/index.ts`

## Implementation Details

### Fix 1: `isWithinSwarmDir()` function

Change from:
```typescript
function isWithinSwarmDir(targetPath: string): boolean {
  const resolved = path.resolve(targetPath)
  return resolved.startsWith(path.resolve(swarmRoot))
}
```

To:
```typescript
function isWithinSwarmDir(targetPath: string): boolean {
  const resolved = path.resolve(targetPath)
  const root = path.resolve(swarmRoot)
  return resolved === root || resolved.startsWith(root + path.sep)
}
```

### Fix 2: Apply same pattern to inline path checks

Check for any inline `startsWith` path validation in the `logs:read` handler and apply the same trailing-separator fix.

### Fix 3: `resolveIncludes()` fallback validation

Add a second validation check after the fallback resolution:

```typescript
let resolvedPath = path.resolve(baseDir, includePath)
if (!isWithinSwarmDir(resolvedPath)) {
  resolvedPath = path.resolve(swarmRoot, includePath)
}
// Add this check:
if (!isWithinSwarmDir(resolvedPath)) {
  parts.push(`[ERROR: include path outside swarm directory: ${includePath}]`)
  lastIndex = match.index + match[0].length
  continue
}
```

## Dependencies

None — standalone security fixes.

## Acceptance Criteria

1. `isWithinSwarmDir()` correctly rejects paths like `/path/to/swarm-evil/file` when swarmRoot is `/path/to/swarm`
2. `isWithinSwarmDir()` correctly accepts the root itself (`/path/to/swarm`) and all children (`/path/to/swarm/prompts/foo.md`)
3. `resolveIncludes()` rejects `{{include:}}` directives that resolve outside the swarm directory on both primary and fallback paths
4. The `logs:read` handler's path check (if any) uses the same safe pattern
5. App builds successfully with `npm run build`
6. No functional regressions — legitimate paths within `swarm/` continue to work

## Notes

- These are security vulnerabilities that could allow reading arbitrary files on the user's system
- The fixes are minimal and surgical — no refactoring needed
- Test with paths like `../../../etc/passwd` to verify the fix works

---

## Completion Notes

**Completed by:** Agent 3a699a45
**Date:** 2026-02-13

### Verification

All three security fixes were found to already be implemented in `electron/src/main/index.ts`:

1. **`isWithinSwarmDir()` (lines 431-435)** — Uses `path.sep` to prevent sibling directory attacks:
   ```typescript
   return resolved === root || resolved.startsWith(root + path.sep)
   ```

2. **`logs:read` handler (lines 815-821)** — Uses the same safe pattern:
   ```typescript
   if (resolved !== logsRoot && !resolved.startsWith(logsRoot + path.sep))
   ```

3. **`resolveIncludes()` (lines 997-1006)** — Has second validation after fallback resolution:
   ```typescript
   if (!isWithinSwarmDir(resolvedPath)) {
     parts.push(`[ERROR: include path outside swarm directory: ${includePath}]`)
     continue
   }
   ```

### Build Verification

- `npm run build` completed successfully with no errors
- TypeScript compilation passed
- All acceptance criteria met
