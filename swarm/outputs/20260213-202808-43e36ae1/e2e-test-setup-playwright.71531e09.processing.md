# E2E Test Setup with Playwright

## Goal

Set up Playwright for E2E testing of the Electron app. This provides automated testing for critical user flows and catches regressions before release.

## Files

- **Create**: `electron/e2e/example.spec.ts` - First E2E test verifying app launches
- **Create**: `electron/playwright.config.ts` - Playwright configuration for Electron
- **Modify**: `electron/package.json` - Add Playwright dependencies and test script

## Dependencies

- All ELECTRON_PLAN.md phases are complete
- Electron app builds and runs successfully

## Acceptance Criteria

1. Playwright is installed as a dev dependency
2. `playwright.config.ts` exists and is configured for Electron testing
3. A sample E2E test exists that:
   - Launches the Electron app
   - Verifies the main window opens
   - Verifies the app title is "Swarm Desktop"
   - Takes a screenshot on failure
4. `npm run test:e2e` script works and passes
5. Tests can be run in CI (headless mode)

## Notes

### Playwright Electron Configuration

Use `@playwright/test` with Electron's `_electron` fixture:

```typescript
import { _electron as electron, test, expect } from '@playwright/test';

test('app launches', async () => {
  const electronApp = await electron.launch({ args: ['.'] });
  const window = await electronApp.firstWindow();
  expect(await window.title()).toBe('Swarm Desktop');
  await electronApp.close();
});
```

### Package.json additions

```json
{
  "devDependencies": {
    "@playwright/test": "^1.40.0"
  },
  "scripts": {
    "test:e2e": "playwright test"
  }
}
```

### Future test cases (not in scope for this task)

- File tree loads workspace files
- DAG canvas renders tasks from swarm.yaml
- Agent panel shows running agents
- Command palette opens with Cmd+K
- Settings panel persists changes

### Tips

- Run `npx playwright install` after adding the dependency
- Use `electron.launch({ args: ['dist/main/main/index.js'] })` to point to built app
- Set `timeout: 30000` for slower CI environments
- Consider adding to `electron-ci.yml` after this task is complete (separate task)
