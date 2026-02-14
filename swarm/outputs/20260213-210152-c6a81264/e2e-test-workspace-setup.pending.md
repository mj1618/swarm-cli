# Fix E2E Tests: Workspace Setup Before Tests

## Problem

E2E tests have flaky failures where some tests timeout (tests 4, 5, 10) while others pass (tests 1-3, 9). The issue is that tests launch the Electron app without configuring a workspace path.

When no workspace is configured:
- App shows `InitializeWorkspace` component instead of main 3-panel layout  
- Tests waiting for `#root` content, "Swarm Desktop" title bar, or panel elements timeout

## Root Cause

In `e2e/app.spec.ts`:
- `createTestWorkspace()` helper exists but is never called in `beforeAll`
- Tests launch with `NODE_ENV: 'test'` but don't set `localStorage['swarm-project-path']`
- Without a project path, App.tsx conditionally shows InitializeWorkspace

## Failing Tests

1. Test 4 (line 132): "renderer process loads React root successfully" - times out after 10s
2. Test 5 (line 150): "displays the main 3-panel layout" - times out after 10s  
3. Test 10 (line 199): "file tree has refresh and create buttons" - times out after 60s

## Proposed Fix

In `test.beforeAll`, before launching the app:

```typescript
// Create a test workspace
const testWorkspace = await createTestWorkspace('basic-test', `
name: test-compose
tasks:
  hello:
    prompt: "Say hello"
`);

// Launch with the workspace path
electronApp = await electron.launch({
  args: [path.join(__dirname, '../dist/main/main/index.js')],
  env: {
    ...process.env,
    NODE_ENV: 'test',
    SWARM_TEST_WORKSPACE: testWorkspace, // Add IPC handler to read this
  },
});

// Or inject localStorage before React mounts
window = await electronApp.firstWindow();
await window.evaluate((workspace) => {
  localStorage.setItem('swarm-project-path', workspace);
}, testWorkspace);
await window.reload();
```

## Verification

- All 25 E2E tests should pass consistently
- No more flaky timeouts on tests 4, 5, 10

## Dependencies

None
