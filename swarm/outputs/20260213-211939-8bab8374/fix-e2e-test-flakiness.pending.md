# Fix E2E Test Flakiness

## Problem

The e2e tests have multiple issues causing failures:

1. **Timeout waiting for React root**: Tests timeout (10s) waiting for `#root` to be visible, suggesting the React app isn't rendering quickly enough during e2e tests
2. **ES module scope error**: The playwright config shows `ReferenceError: exports is not defined in ES module scope` indicating ESM/CJS compatibility issues

## Test Results

```
3 failed
- renderer process loads React root successfully (10s timeout)
- displays the main 3-panel layout (10s timeout)  
- file tree shows "Files" heading (exports not defined)

3 passed
- app launches successfully
- main window has correct title
- window has minimum dimensions

19 did not run (serial mode, failed early)
```

## Root Cause Analysis

The package.json lacks `"type": "module"` which causes ESM/CJS confusion. Multiple Node warnings show this:
```
Warning: Module type of file not specified and it doesn't parse as CommonJS.
Reparsing as ES module because module syntax was detected.
To eliminate this warning, add "type": "module" to package.json.
```

## Suggested Fix

1. Add `"type": "module"` to `electron/package.json`
2. Ensure all config files (postcss.config.js, eslint.config.js, playwright.config.ts) are compatible with ESM
3. Increase timeout for React root rendering or add better app readiness detection
4. Consider using `page.waitForLoadState()` before checking for elements

## Dependencies

None

## Priority

Medium - tests pass locally but may fail in CI due to timing issues
