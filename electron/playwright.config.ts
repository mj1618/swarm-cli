import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  // Run tests in serial mode for Electron (only one app instance at a time)
  fullyParallel: false,
  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: !!process.env.CI,
  // Retry failed tests on CI (increased from 2 to 3 for better stability)
  retries: process.env.CI ? 3 : 1,
  // Single worker for Electron tests
  workers: 1,
  // Reporter to use
  reporter: [
    ['html', { open: 'never' }],
    ['list'],
  ],
  // Timeout for each test (increased for CI stability)
  timeout: process.env.CI ? 90000 : 60000,
  // Global timeout for the test run
  globalTimeout: process.env.CI ? 600000 : undefined, // 10 minutes on CI
  // Expect timeout (increased for better stability)
  expect: {
    timeout: 15000,
  },
  use: {
    // Take screenshot on failure
    screenshot: 'only-on-failure',
    // Trace on first retry
    trace: 'on-first-retry',
    // Action timeout for interactions
    actionTimeout: 20000,
  },
  // Output directory for test artifacts
  outputDir: 'test-results/',
});
