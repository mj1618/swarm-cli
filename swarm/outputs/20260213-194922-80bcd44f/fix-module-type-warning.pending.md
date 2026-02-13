# Fix Module Type Warning in package.json

**Type:** Bug fix / Build optimization

## Problem

When running `npm run build`, Node.js emits a warning:

```
(node:60209) [MODULE_TYPELESS_PACKAGE_JSON] Warning: Module type of file:///Users/matt/code/swarm-cli/electron/postcss.config.js?t=1770983549257 is not specified and it doesn't parse as CommonJS.
Reparsing as ES module because module syntax was detected. This incurs a performance overhead.
To eliminate this warning, add "type": "module" to /Users/matt/code/swarm-cli/electron/package.json.
```

## Solution

Add `"type": "module"` to `electron/package.json`.

However, this requires verifying that:
1. All `.js` config files (postcss.config.js, tailwind.config.js) use ES module syntax
2. The `.cjs` variants (postcss.config.cjs, tailwind.config.cjs) are removed or kept as fallbacks
3. The electron main process build still works correctly

## Files to modify

- `electron/package.json` - Add `"type": "module"`
- Possibly remove duplicate config files (`.js` vs `.cjs`)

## Testing

- Run `npm run build` and verify no warnings
- Run `npm run start` and verify app launches correctly
