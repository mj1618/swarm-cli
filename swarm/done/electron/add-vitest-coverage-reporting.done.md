# Add Vitest Coverage Reporting

## Goal

Add code coverage collection and reporting to the Electron app's unit test suite. This will provide visibility into test coverage metrics and help identify untested code paths.

## Files

- **Modify**: `electron/vitest.config.ts` - Add coverage configuration
- **Modify**: `electron/package.json` - Add coverage script and @vitest/coverage-v8 dependency
- **Modify**: `.github/workflows/electron-ci.yml` - Add coverage reporting step

## Dependencies

- None - all unit test infrastructure is already in place

## Acceptance Criteria

1. `vitest.config.ts` has coverage configuration with:
   - `@vitest/coverage-v8` provider
   - Coverage thresholds (e.g., 60% minimum)
   - Exclusions for test files, config files, and generated code
2. New npm script `test:coverage` runs tests with coverage
3. Running `npm run test:coverage` locally produces a coverage report
4. CI workflow runs tests with coverage and uploads coverage report as artifact
5. Coverage report shows line, branch, function, and statement coverage

## Notes

Current state:
- Vitest is configured in `electron/vitest.config.ts` but without coverage settings
- Unit tests exist in `electron/src/renderer/lib/__tests__/` (7 test files)
- CI runs `npm run test` but doesn't collect coverage

Example vitest coverage config:
```typescript
export default defineConfig({
  test: {
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'dist/',
        '**/*.test.ts',
        '**/*.spec.ts',
        'src/test/**',
      ],
      thresholds: {
        lines: 60,
        functions: 60,
        branches: 60,
        statements: 60,
      },
    },
  },
})
```

The coverage report will be output to `electron/coverage/` directory.

---

## Completion Notes (Agent 0496a5a7)

**Completed on:** 2026-02-13

### What was implemented:

1. **vitest.config.ts** - Added comprehensive coverage configuration:
   - `@vitest/coverage-v8` provider
   - Reporters: text, json, html
   - 60% thresholds for lines, functions, branches, and statements
   - Exclusions for test files, config files, node_modules, dist, and TypeScript declaration files

2. **package.json** - Added:
   - `@vitest/coverage-v8` dev dependency (^4.0.18)
   - New `test:coverage` script

3. **electron-ci.yml** - Updated to:
   - Run `npm run test:coverage` instead of `npm run test`
   - Upload coverage report as artifact (14-day retention)

### Verification:

- All 246 tests pass
- Coverage report generated successfully:
  - Overall: 88.64% statements, 84.7% branches, 91.07% functions, 88.12% lines
  - All coverage metrics well above 60% threshold
- Build completes successfully
- Coverage reports generated in all three formats (text, json, html)
