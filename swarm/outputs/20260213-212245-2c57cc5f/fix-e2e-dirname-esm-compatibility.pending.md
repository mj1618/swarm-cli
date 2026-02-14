# Fix E2E Test __dirname ES Module Compatibility

## Problem

The e2e tests fail immediately with:

```
ReferenceError: __dirname is not defined
  at /Users/matt/code/swarm-cli/electron/e2e/app.spec.ts:65:22
```

This occurs because `__dirname` is a CommonJS global that doesn't exist in ES modules. The test file uses ES module syntax (`import` statements) but tries to use `__dirname`.

## Location

`electron/e2e/app.spec.ts` line 65:
```typescript
args: [path.join(__dirname, '../dist/main/main/index.js')],
```

## Suggested Fix

Add ES module compatible `__dirname` definition at the top of the file:

```typescript
import { fileURLToPath } from 'url';
import { dirname } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
```

Alternatively, if TypeScript config doesn't support `import.meta.url`, use `process.cwd()` or a relative path from the project root.

## Test Command

```bash
cd electron && npm run test:e2e
```

## Current Result

- 1 test failed (the first test, `app launches successfully`)
- 24 tests did not run (blocked by first failure)

## Expected Result

All 25 e2e tests should run. The first test should launch the app successfully.

## Dependencies

None

## Priority

High - This blocks all e2e tests from running.
