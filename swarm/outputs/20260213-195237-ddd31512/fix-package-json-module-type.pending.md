# Fix package.json Module Type Warning

## Issue
The build produces a warning:

```
(node:69848) [MODULE_TYPELESS_PACKAGE_JSON] Warning: Module type of file:///Users/matt/code/swarm-cli/electron/postcss.config.js?t=1770983657707 is not specified and it doesn't parse as CommonJS.
Reparsing as ES module because module syntax was detected. This incurs a performance overhead.
To eliminate this warning, add "type": "module" to /Users/matt/code/swarm-cli/electron/package.json.
```

## Solution
Add `"type": "module"` to `electron/package.json` to specify that the project uses ES modules.

## Files to modify
- `electron/package.json`

## Dependencies
None
