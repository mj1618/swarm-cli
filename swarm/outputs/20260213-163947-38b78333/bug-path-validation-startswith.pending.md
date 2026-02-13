# Bug: Path validation uses startsWith() which can match directory prefixes

**Severity:** Low (not exploitable given current cwd constraints, but defense-in-depth improvement)

## Description

The `isWithinSwarmDir()` function in `electron/src/main/index.ts` (line 88-91) uses `path.startsWith()` to validate paths:

```typescript
function isWithinSwarmDir(targetPath: string): boolean {
  const resolved = path.resolve(targetPath)
  return resolved.startsWith(path.resolve(swarmRoot))
}
```

This could theoretically match a directory like `swarm-evil/` since `/path/to/swarm-evil` starts with `/path/to/swarm`. The same pattern is used in the `logs:read` handler.

## Fix

Add a trailing separator to the prefix check:

```typescript
function isWithinSwarmDir(targetPath: string): boolean {
  const resolved = path.resolve(targetPath)
  const root = path.resolve(swarmRoot)
  return resolved === root || resolved.startsWith(root + path.sep)
}
```

## Files Affected

- `electron/src/main/index.ts` — `isWithinSwarmDir()` function (line 88)
- `electron/src/main/index.ts` — `logs:read` handler (line 253) uses same pattern inline
