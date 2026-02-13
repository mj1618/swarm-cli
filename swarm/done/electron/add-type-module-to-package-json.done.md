# Add "type": "module" to package.json

## Goal

Fix the Node.js MODULE_TYPELESS_PACKAGE_JSON warning that appears on every npm command (build, test, lint). The warning occurs because the project uses ES modules in config files like `postcss.config.js` but doesn't declare `"type": "module"` in package.json.

## Files

- **Modify**: `electron/package.json` - Add `"type": "module"` field

## Dependencies

None

## Acceptance Criteria

1. `electron/package.json` has `"type": "module"` after the `"description"` field
2. `npm run build` completes without the MODULE_TYPELESS_PACKAGE_JSON warning
3. `npm run lint` completes without the warning
4. `npm test` completes without the warning
5. All existing scripts continue to work correctly

---

## Completion Note

**Completed by agent 773e5d80**

Added `"type": "module"` to `electron/package.json` after the description field. This tells Node.js to treat `.js` files as ES modules by default, matching the actual syntax used in config files (`postcss.config.js`, `eslint.config.js`, etc.).

**Verification:**
- `npm run build` - No MODULE_TYPELESS warning
- `npm run lint` - No MODULE_TYPELESS warning  
- `npm test` - No MODULE_TYPELESS warning

All npm commands now run without the MODULE_TYPELESS_PACKAGE_JSON warning.
