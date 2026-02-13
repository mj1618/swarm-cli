# Add Playwright E2E Tests for Electron App

## Goal

Set up Playwright for Electron and implement E2E tests for critical user flows to ensure the app functions correctly and prevent regressions.

## Files

- **Create**: `electron/playwright.config.ts` - Playwright configuration for Electron
- **Create**: `electron/e2e/` - E2E test directory
- **Create**: `electron/e2e/app.spec.ts` - Core app E2E tests
- **Modify**: `electron/package.json` - Add Playwright dependencies and test script
- **Modify**: `.github/workflows/electron-ci.yml` - Add E2E test step (optional, may need Xvfb on Linux)

## Dependencies

- All ELECTRON_PLAN.md phases complete
- `electron-ci.yml` workflow exists

## Acceptance Criteria

1. Playwright is installed and configured for Electron testing
2. `npm run test:e2e` script runs E2E tests locally
3. At least 3 critical flows are tested:
   - App launches and displays the main 3-panel layout
   - File tree loads and displays `swarm/` directory contents
   - DAG canvas renders tasks from `swarm.yaml`
4. Tests pass locally with `npm run test:e2e`
5. Tests handle both empty workspace and workspace with existing `swarm.yaml`

## Implementation Notes

### Playwright Configuration

```typescript
// electron/playwright.config.ts
import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  timeout: 30000,
  use: {
    trace: 'on-first-retry',
  },
})
```

### Electron Testing Pattern

Playwright supports Electron apps via `electron.launch()`:

```typescript
import { _electron as electron } from 'playwright'
import { test, expect } from '@playwright/test'

test('app launches', async () => {
  const electronApp = await electron.launch({ args: ['.'] })
  const window = await electronApp.firstWindow()
  
  // Verify main layout exists
  await expect(window.locator('[data-testid="file-tree"]')).toBeVisible()
  await expect(window.locator('[data-testid="dag-canvas"]')).toBeVisible()
  await expect(window.locator('[data-testid="agent-panel"]')).toBeVisible()
  
  await electronApp.close()
})
```

### Adding Data Test IDs

Components may need `data-testid` attributes added for reliable selection:
- `FileTree.tsx` → `data-testid="file-tree"`
- `DagCanvas.tsx` → `data-testid="dag-canvas"`
- `AgentPanel.tsx` → `data-testid="agent-panel"`

### Package.json Changes

```json
{
  "scripts": {
    "test:e2e": "playwright test"
  },
  "devDependencies": {
    "@playwright/test": "^1.51.0"
  }
}
```

## Priority

Medium - E2E tests help catch regressions and verify critical user journeys work end-to-end.

## Scope Limitations

- Focus on smoke tests for critical paths only
- Do not test every feature - start with core flows
- Skip CI integration in this task if it requires complex Xvfb setup
