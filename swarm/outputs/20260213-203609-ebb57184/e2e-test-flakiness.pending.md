# E2E Test Flakiness Investigation

## Issue

The Electron E2E tests are flaky - they sometimes pass but often timeout with errors like:
- "Target page, context or browser has been closed"
- "beforeAll hook timeout of 30000ms exceeded"
- "page.waitForSelector: Timeout 10000ms exceeded"

## Observations

1. **Shared state between test files**: Both `app.spec.ts` and `dag-editing.spec.ts` use `test.beforeAll` to launch their own Electron app instances, but they share global state (`electronApp` and `window` variables).

2. **Test isolation**: The tests run sequentially using a single worker, but if one test causes the window to become unresponsive or close, subsequent tests fail.

3. **Timing issues**: The 30-second timeout for `beforeAll` hooks may not be sufficient for Electron app startup in all environments.

## Failed Tests (from recent run)

1. `window has minimum dimensions (1000x600)` - Test timeout exceeded
2. `renderer process loads React root successfully` - beforeAll hook timeout  
3. `should open task drawer when clicking Add Task button` - Timeout waiting for #root

## Suggested Fixes

1. **Increase test isolation**: Each test file should have its own isolated Electron instance, or use proper cleanup between tests.

2. **Add retry logic**: Use Playwright's built-in retry mechanism for flaky tests.

3. **Improve wait conditions**: Instead of fixed timeouts, use more robust wait conditions like `waitForFunction` to check for actual DOM state.

4. **Better error handling in beforeAll**: Add try/catch with better logging to understand startup failures.

5. **Consider test parallelization**: If tests are truly independent, running them in parallel could reduce interference.

## Impact

- CI pipeline (`.github/workflows/electron-ci.yml`) has been modified to run E2E tests, but they may fail intermittently.
- Local development testing may see inconsistent results.

## Priority

Medium - The application works correctly; this is a test infrastructure issue that affects developer experience but not users.

## Dependencies

None - can be worked on independently.
