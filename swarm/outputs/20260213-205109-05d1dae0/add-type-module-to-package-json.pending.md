# Add "type": "module" to package.json

## Summary

Add `"type": "module"` to `electron/package.json` to eliminate Node.js warnings.

## Problem

During build and test runs, Node.js emits this warning:

```
(node:xxxxx) [MODULE_TYPELESS_PACKAGE_JSON] Warning: Module type of file:///Users/matt/code/swarm-cli/electron/postcss.config.js is not specified and it doesn't parse as CommonJS.
Reparsing as ES module because module syntax was detected. This incurs a performance overhead.
To eliminate this warning, add "type": "module" to /Users/matt/code/swarm-cli/electron/package.json.
```

This warning appears multiple times:
- During `npm run build`
- During `npm run test`
- During `npm run lint`

## Solution

Add `"type": "module"` to `electron/package.json`:

```json
{
  "name": "swarm-desktop",
  "version": "0.1.0",
  "type": "module",
  ...
}
```

## Priority

Low - This is a minor cosmetic issue that doesn't affect functionality.

## Dependencies

None

## Reported by

- Iteration: 11 of 100
- Agent ID: 98e9a205
- Task ID: 3d326d32
