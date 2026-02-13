# Unit Test Setup with Vitest

## Goal

Set up Vitest for unit testing the Electron renderer utilities and components. This complements E2E tests by verifying individual functions work correctly in isolation.

## Files

- **Create**: `electron/vitest.config.ts` - Vitest configuration
- **Create**: `electron/src/renderer/lib/__tests__/yamlParser.test.ts` - Tests for YAML parsing
- **Create**: `electron/src/renderer/lib/__tests__/dagValidation.test.ts` - Tests for DAG validation
- **Modify**: `electron/package.json` - Add Vitest dependencies and test script

## Dependencies

- All ELECTRON_PLAN.md phases are complete
- Pure utility functions exist in `electron/src/renderer/lib/`

## Acceptance Criteria

1. Vitest is installed as a dev dependency
2. `vitest.config.ts` exists with proper configuration for React/TypeScript
3. At least 2 test files exist with passing tests:
   - `yamlParser.test.ts` - Tests `parseYamlFile()` with valid/invalid YAML
   - `dagValidation.test.ts` - Tests `detectCycles()` and `findOrphanedTasks()`
4. `npm run test` script works and all tests pass
5. Tests run in watch mode during development (`npm run test:watch`)

## Notes

### Vitest Configuration

```typescript
// vitest.config.ts
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    include: ['src/**/*.test.{ts,tsx}'],
  },
});
```

### Package.json additions

```json
{
  "devDependencies": {
    "vitest": "^1.0.0",
    "@vitest/ui": "^1.0.0",
    "jsdom": "^23.0.0"
  },
  "scripts": {
    "test": "vitest run",
    "test:watch": "vitest",
    "test:ui": "vitest --ui"
  }
}
```

### Test file examples

Test pure functions first - they're easiest to test:
- `yamlParser.ts` - `parseYamlFile()` function
- `dagValidation.ts` - `detectCycles()`, `findOrphanedTasks()`
- `yamlWriter.ts` - `generateYaml()`
- `outputFolderUtils.ts` - `formatOutputFolderTimestamp()`

### Why Vitest over Jest

- Native ESM support (no extra config needed)
- Works seamlessly with Vite (already used in this project)
- Faster execution
- Compatible API with Jest for easy migration

---

## Completion Notes

**Completed by agent 39ced8da**

### What was implemented:

1. **Installed dependencies**: vitest, @vitest/ui, jsdom, @testing-library/react, @testing-library/jest-dom

2. **Created `electron/vitest.config.ts`**: Configured with React plugin, jsdom environment, and global test API

3. **Created `electron/src/test/setup.ts`**: Sets up jest-dom matchers for Vitest

4. **Created test files**:
   - `electron/src/renderer/lib/__tests__/yamlParser.test.ts` (16 tests)
     - Tests `parseComposeFile()` with valid/invalid YAML
     - Tests `serializeCompose()` and roundtrip parsing
     - Tests `composeToFlow()` node/edge generation
   - `electron/src/renderer/lib/__tests__/dagValidation.test.ts` (17 tests)
     - Tests cycle detection (simple, self-referential, 3-node cycles)
     - Tests orphan detection with pipelines
     - Tests parallel task detection
     - Tests edge cases (empty tasks, non-existent deps)

5. **Updated `electron/package.json`** with test scripts:
   - `npm run test` - runs all tests once
   - `npm run test:watch` - runs tests in watch mode
   - `npm run test:ui` - opens Vitest UI

**Test results**: 33 tests passing across 2 test files
**Build verification**: TypeScript typecheck and Vite build both pass
