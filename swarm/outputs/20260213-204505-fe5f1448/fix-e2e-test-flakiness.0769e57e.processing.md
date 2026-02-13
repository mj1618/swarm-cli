# Fix E2E Test Flakiness

## Goal

Improve E2E test stability by adding better wait conditions, increasing resilience to timing issues, and improving test isolation. The tests currently experience intermittent failures with "Target page, context or browser has been closed" and timeout errors.

## Files

- **Modify**: `electron/e2e/app.spec.ts` - Fix timing issues and improve wait conditions
- **Modify**: `electron/playwright.config.ts` - Adjust timeout settings for CI

## Dependencies

- `add-ci-unit-and-e2e-tests.done.md` - E2E tests exist and run in CI
- `e2e-dag-editing-workflow.done.md` - DAG editing E2E tests exist

## Acceptance Criteria

1. Replace `waitForTimeout()` calls with proper wait conditions (`waitForSelector`, `waitForFunction`)
2. Add `test.slow()` annotation to tests that need more time
3. Increase `beforeAll` hook timeout from 30s default to 60s
4. Add graceful error handling in `afterAll` hook with `try/catch`
5. Add `test.describe.configure({ retries: 2 })` for flaky test suites
6. Verify tests pass locally 3 times in a row without failures
7. CI workflow continues to pass

## Notes

From the E2E test flakiness investigation:

### Current Issues

1. **Fixed timeouts**: `waitForTimeout(3000)` and `waitForTimeout(2000)` are brittle - should use proper wait conditions
2. **Shared state**: `electronApp` and `window` are module-level variables shared across all tests
3. **No retry in test suites**: While playwright.config.ts has retries on CI, individual describe blocks could benefit from explicit retry config

### Recommended Changes

```typescript
// Replace arbitrary timeouts with proper waits
// Before:
await window.waitForTimeout(3000);

// After:
await window.waitForSelector('#root', { state: 'visible', timeout: 10000 });
```

```typescript
// Add slow annotation to tests that need more time
test.slow('renderer process loads React root successfully', async () => {
  // ...
});
```

```typescript
// Add try/catch to afterAll
test.afterAll(async () => {
  try {
    if (electronApp) {
      await electronApp.close().catch(() => {});
    }
  } catch {
    // Ignore cleanup errors - app may already be closed
  }
});
```

```typescript
// Add describe-level configuration
test.describe.configure({ mode: 'serial', retries: 2 });
```

### References

- Playwright best practices: https://playwright.dev/docs/best-practices
- Electron testing guide: https://www.electronjs.org/docs/latest/tutorial/automated-testing
