# Fix E2E Test: Window Title Check Timeout

## Issue

The E2E test "main window has correct title containing Swarm Desktop" consistently times out after 1 minute, despite the app launching successfully.

## Observed Behavior

```
✓  app launches successfully (9ms)
✘  main window has correct title containing Swarm Desktop (1.0m) - TIMEOUT
```

The `window.title()` call appears to hang when Playwright tries to get the window title.

## File

- `electron/e2e/app.spec.ts` (line 121-124)

## Root Cause Investigation Needed

1. Check if the window title is set correctly in the main process
2. Verify that `window.title()` in Playwright is compatible with Electron's BrowserWindow title
3. Consider if there's a race condition where the title isn't set yet

## Potential Fixes

1. Add an explicit wait for the window to have a non-empty title before asserting
2. Use `page.evaluate` to get `document.title` instead of `window.title()`
3. Increase the timeout specifically for this test
4. Check if running in CI vs local has different behavior

## Prior Work

Multiple commits have addressed E2E stability:
- 478cde8: fix(electron): simplify E2E waitForAppReady
- d056c96: fix(electron): improve E2E test reliability

## Acceptance Criteria

1. The "main window has correct title" test passes consistently
2. No regression to other E2E tests
3. Test completes within 30 seconds (not 60+ seconds)
