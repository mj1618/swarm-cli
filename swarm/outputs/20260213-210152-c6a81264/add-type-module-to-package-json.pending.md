# Add "type": "module" to package.json

## Problem

Every npm command shows this warning:

```
(node:xxxxx) [MODULE_TYPELESS_PACKAGE_JSON] Warning: Module type of file:///Users/matt/code/swarm-cli/electron/postcss.config.js?t=... is not specified and it doesn't parse as CommonJS.
Reparsing as ES module because module syntax was detected. This incurs a performance overhead.
To eliminate this warning, add "type": "module" to /Users/matt/code/swarm-cli/electron/package.json.
```

## Root Cause

The project uses ES modules (import/export syntax) in config files like `postcss.config.js` and `eslint.config.js`, but `package.json` doesn't specify `"type": "module"`.

Node.js has to reparse these files to detect they're ES modules, causing a small performance overhead.

## Fix

In `electron/package.json`, add:

```json
{
  "name": "swarm-desktop",
  "type": "module",
  ...
}
```

## Verification

- Run `npm run build` - no MODULE_TYPELESS_PACKAGE_JSON warning
- Run `npm test` - no warning
- Run `npm run lint` - no warning

## Dependencies

None

## Notes

This is a minor code quality improvement, not a functional bug.
