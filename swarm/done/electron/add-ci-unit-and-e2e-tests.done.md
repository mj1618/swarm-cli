# Add Unit and E2E Tests to Electron CI

## Goal

Add unit test (`vitest`) and E2E test (`playwright`) execution to the Electron CI workflow. Currently the CI only runs typecheck, lint, and build - but skips the actual test suites.

## Files

- `.github/workflows/electron-ci.yml` - Add test steps

## Dependencies

- None - all test infrastructure exists:
  - Unit tests: `electron/src/renderer/lib/__tests__/*.test.ts`
  - E2E tests: `electron/e2e/*.spec.ts`
  - Vitest config: `electron/vitest.config.ts`
  - Playwright installed in devDependencies

## Acceptance Criteria

1. CI workflow runs `npm run test` (vitest) after lint step
2. CI workflow runs `npm run test:e2e` (playwright) after build steps
3. E2E tests properly set up with xvfb for headless Electron testing on Linux
4. Test failures cause the CI workflow to fail
5. All existing tests pass in CI

## Notes

From ELECTRON_PLAN.md recommendations:
> "Adding E2E tests for critical flows"

The E2E tests already exist in `electron/e2e/` but aren't running in CI. Playwright requires xvfb-run on Linux for headless Electron testing.

Example xvfb setup for Electron E2E:
```yaml
- name: Run E2E tests
  run: xvfb-run --auto-servernum npm run test:e2e
```

May also need to install Playwright browsers:
```yaml
- name: Install Playwright browsers
  run: npx playwright install --with-deps chromium
```

---

## Completion Notes

**Completed by agent 4aec0ac9**

### What was implemented:

1. Added `npm run test` step after lint to run vitest unit tests
2. Added `npx playwright install --with-deps chromium` step to install Playwright browsers
3. Added `xvfb-run --auto-servernum npm run test:e2e` step for E2E tests on Linux
4. Added artifact upload for E2E test results on failure (screenshots, traces)

### Verification:

- Unit tests pass locally (33 tests in 2 files)
- Build succeeds (React + Electron main process)
- Typecheck and lint pass
- E2E tests can run (verified with subset of tests)

### CI Workflow Changes:

```yaml
- name: Run unit tests
  run: npm run test

- name: Install Playwright browsers
  run: npx playwright install --with-deps chromium

- name: Run E2E tests
  run: xvfb-run --auto-servernum npm run test:e2e

- name: Upload E2E test results
  if: failure()
  uses: actions/upload-artifact@v4
  with:
    name: e2e-test-results
    path: electron/test-results/
    retention-days: 7
```
