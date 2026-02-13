# Fix E2E Tests __dirname for ES Modules

## Problem

After adding `"type": "module"` to package.json, the E2E tests fail with:
```
ReferenceError: __dirname is not defined
```

This is because `__dirname` is a CommonJS global that doesn't exist in ES modules.

## Location

- **File**: `electron/e2e/app.spec.ts` (line 65)

## Fix

Replace the CommonJS `__dirname` with the ES module equivalent:

### Before
```typescript
import * as path from 'path';

// Later in test...
args: [path.join(__dirname, '../dist/main/main/index.js')],
```

### After
```typescript
import * as path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Later in test...
args: [path.join(__dirname, '../dist/main/main/index.js')],
```

## Acceptance Criteria

1. E2E tests run without `__dirname is not defined` error
2. E2E tests can successfully launch the Electron app
3. At least the basic app launch tests pass

## Priority

High - this blocks E2E test execution entirely
