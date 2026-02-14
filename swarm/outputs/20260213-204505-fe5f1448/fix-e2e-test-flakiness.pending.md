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

## BLOCKED - Investigation Notes (Agent 0769e57e)

### Already Completed
The following acceptance criteria are ALREADY implemented by previous iterations:
- **Criterion 4**: afterAll has try/catch with Promise.race (lines 83-97 in app.spec.ts)
- **Criterion 3**: playwright.config.ts has 60-90s test timeout
- **Criterion 5** (partial): CI has 2 retries at file level (playwright.config.ts line 10)

### Cannot Be Completed As Specified

**Criterion 1 (Replace waitForTimeout with proper waits)**: BLOCKED
- When using `waitForSelector` or `waitForFunction` in Electron with Playwright, tests timeout even when the element is found
- Playwright logs show "locator resolved to visible <div id='root'>..." but still times out
- This appears to be a Playwright + Electron bug where waitForSelector with `state: 'visible'` keeps rechecking indefinitely
- Attempted fixes: waitForFunction, waitForLoadState, Promise-based waits - all fail the same way

**Criterion 5 (describe-level retries)**: BLOCKED  
- Adding `test.describe.configure({ retries: 2 })` causes issues because beforeAll only runs once
- When a test retries, the shared `window` reference becomes stale, causing subsequent tests to hang

**Criterion 6 (Pass locally 3 times)**: BLOCKED
- Tests pass individually but fail when run together
- Root cause: Shared `electronApp` and `window` across tests becomes unresponsive after first test
- The first test (checking objects exist) passes, but the second test (any window interaction) hangs
- This is a known Electron + Playwright issue on macOS without xvfb

### Recommended Alternative Approach
To properly fix E2E test flakiness, the test architecture needs to change:
1. Use `test.beforeEach` instead of `test.beforeAll` to get a fresh app instance per test (slow but reliable)
2. OR: Run tests in completely separate files so each has its own beforeAll lifecycle
3. OR: Accept that tests only pass reliably on CI with xvfb

### References

- Playwright best practices: https://playwright.dev/docs/best-practices
- Electron testing guide: https://www.electronjs.org/docs/latest/tutorial/automated-testing
- Known Playwright Electron issues: https://github.com/electron/electron/issues
