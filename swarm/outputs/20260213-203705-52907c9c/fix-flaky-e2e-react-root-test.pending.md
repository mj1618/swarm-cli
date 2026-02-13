# Fix Flaky E2E Tests: React Rendering Detection

## Goal

Fix multiple flaky E2E tests that intermittently timeout waiting for React elements to be visible.

## Problem

Multiple tests timeout waiting for elements because React hasn't fully rendered:

1. `renderer process loads React root successfully` - waits for `#root`
2. `displays the main 3-panel layout` - waits for title bar, headings
3. `dag canvas has data-testid attribute` - waits for `[data-testid="dag-canvas"]`

Example error:
```
TimeoutError: page.waitForSelector: Timeout 10000ms exceeded.
  - waiting for locator('#root') to be visible
```

Root cause:
- The Electron window opens before React has fully mounted
- Tests run before the React component tree is ready
- The `beforeAll` hook doesn't wait long enough for React

## Files

- `electron/e2e/app.spec.ts` - Test file with failing tests
- Potentially `electron/src/renderer/App.tsx` - To add ready indicator

## Suggested Fixes

1. **Add app ready indicator**: Add `data-app-ready="true"` attribute to root element after initial render
2. **Update beforeAll hook**: Wait for app ready indicator instead of just `domcontentloaded`
3. **Increase timeouts**: Use longer timeouts (15-20s) for initial element waits in CI
4. **Use networkidle**: Wait for `waitForLoadState('networkidle')` after domcontentloaded

Example fix for beforeAll:
```typescript
await window.waitForLoadState('domcontentloaded');
await window.waitForSelector('[data-app-ready="true"]', { timeout: 20000 });
```

## Acceptance Criteria

1. All E2E tests pass consistently (run 5+ times locally)
2. Tests still validate expected behavior
3. No excessive artificial delays (use proper waiting strategies)
4. CI workflow E2E step passes
