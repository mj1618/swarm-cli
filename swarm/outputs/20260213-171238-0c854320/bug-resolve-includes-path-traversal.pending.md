# Bug: resolveIncludes fallback lacks path validation

## Problem

In `electron/src/main/index.ts`, the `resolveIncludes` function has a path traversal vulnerability. When resolving `{{include:path}}` directives, it first tries to resolve relative to `baseDir`, then falls back to resolving relative to `swarmRoot` — but the fallback does NOT validate that the resolved path is within the swarm directory.

```typescript
let resolvedPath = path.resolve(baseDir, includePath)
if (!isWithinSwarmDir(resolvedPath)) {
  resolvedPath = path.resolve(swarmRoot, includePath) // No validation!
}
```

If a prompt file contains `{{include:../../../etc/passwd}}`, the first resolution may be outside swarm/ (correctly caught), but the fallback `path.resolve(swarmRoot, '../../../etc/passwd')` also resolves outside swarm/ and is NOT checked, allowing the file to be read.

## Fix

Add `isWithinSwarmDir` check after the fallback:

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

## Files

### Modify
- `electron/src/main/index.ts` — Add `isWithinSwarmDir` check after fallback resolution in `resolveIncludes` function (around line 479)

## Severity

Security — path traversal allowing file reads outside the allowed directory.
