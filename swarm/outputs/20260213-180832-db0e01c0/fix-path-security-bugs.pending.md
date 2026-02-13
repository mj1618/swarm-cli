# Task: Fix Path Security Bugs in Main Process

**Phase:** 4 - Agent Management (security hardening)
**Priority:** High (security)

## Goal

Fix two related path traversal security bugs in `electron/src/main/index.ts`:

1. **`isWithinSwarmDir()` uses `startsWith()` without trailing separator** — could match sibling directories like `swarm-evil/` since `/path/to/swarm-evil`.startsWith(`/path/to/swarm`) is true.

2. **`resolveIncludes()` fallback lacks path validation** — when a `{{include:path}}` directive fails the first `isWithinSwarmDir` check, the fallback resolution relative to `swarmRoot` is NOT validated, allowing path traversal (e.g. `{{include:../../../etc/passwd}}`).

## Files

### Modify
- `electron/src/main/index.ts`

## What to Change

### Fix 1: `isWithinSwarmDir()` (around line 88)

**Before:**
```typescript
function isWithinSwarmDir(targetPath: string): boolean {
  const resolved = path.resolve(targetPath)
  return resolved.startsWith(path.resolve(swarmRoot))
}
```

**After:**
```typescript
function isWithinSwarmDir(targetPath: string): boolean {
  const resolved = path.resolve(targetPath)
  const root = path.resolve(swarmRoot)
  return resolved === root || resolved.startsWith(root + path.sep)
}
```

### Fix 2: Inline `startsWith` in `logs:read` handler (around line 253)

Apply the same trailing-separator fix to any inline `startsWith` path checks in the logs handler.

### Fix 3: `resolveIncludes()` fallback (around line 479)

**Before:**
```typescript
let resolvedPath = path.resolve(baseDir, includePath)
if (!isWithinSwarmDir(resolvedPath)) {
  resolvedPath = path.resolve(swarmRoot, includePath)
}
```

**After:**
```typescript
let resolvedPath = path.resolve(baseDir, includePath)
if (!isWithinSwarmDir(resolvedPath)) {
  resolvedPath = path.resolve(swarmRoot, includePath)
}
if (!isWithinSwarmDir(resolvedPath)) {
  parts.push(`[ERROR: include path outside swarm directory: ${includePath}]`)
  lastIndex = match.index + match[0].length
  continue
}
```

## Dependencies

- None (standalone security fixes)

## Acceptance Criteria

1. `isWithinSwarmDir()` correctly rejects paths like `/path/to/swarm-evil/file` when swarmRoot is `/path/to/swarm`
2. `isWithinSwarmDir()` correctly accepts the root itself (`/path/to/swarm`) and children (`/path/to/swarm/prompts/foo.md`)
3. `resolveIncludes()` rejects `{{include:}}` directives that resolve outside the swarm directory on both the primary and fallback path
4. The `logs:read` handler's inline path check uses the same safe pattern
5. App builds with `npm run build`

## Notes

- These are the two pending security tasks: `bug-path-validation-startswith.pending.md` and `bug-resolve-includes-path-traversal.pending.md`
- Both bugs are in the same file, so fixing them together avoids merge conflicts
- The fixes are minimal and surgical — no refactoring needed
