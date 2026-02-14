# E2E Tests for Settings Panel

## Goal

Add Playwright E2E tests for the Settings Panel, which allows users to configure backend, model, theme, system notifications, and sound alerts. The current E2E test suite covers the main layout and DAG editing but has no tests for settings functionality.

## Files

- **Modify**: `electron/e2e/app.spec.ts` - Add new test describe block for Settings Panel

## Dependencies

- Requires: E2E test setup (already complete)
- Requires: Settings Panel component (`electron/src/renderer/components/SettingsPanel.tsx`)

## Acceptance Criteria

1. New `test.describe('Swarm Desktop - Settings Panel')` block in `electron/e2e/app.spec.ts` with:
   - **Open settings**: Use keyboard shortcut (Cmd+,) or menu to open settings, verify panel appears
   - **Close settings**: Click close button or press Escape, verify panel closes
   - **Backend dropdown**: Verify dropdown contains 'claude-code' and 'cursor' options
   - **Model dropdown**: Verify dropdown contains 'opus', 'sonnet', 'haiku' options
   - **Theme toggle**: Change theme between light/dark/system, verify UI updates
   - **Save button**: Modify settings, click save, verify success toast appears

2. Tests use proper selectors:
   - Add `data-testid="settings-panel"` to the Settings Panel container
   - Add `data-testid="settings-backend-select"` to backend dropdown
   - Add `data-testid="settings-model-select"` to model dropdown  
   - Add `data-testid="settings-theme-select"` to theme toggle
   - Add `data-testid="settings-save-button"` to save button
   - Add `data-testid="settings-close-button"` to close button

3. All new tests pass consistently with `npx playwright test`

## Notes

From `SettingsPanel.tsx`:
- Backend options: `['claude-code', 'cursor']`
- Model options: `['opus', 'sonnet', 'haiku']`
- Theme options: `['system', 'light', 'dark']`
- Save button only enabled when settings have changed (`dirty` state)
- Uses `window.settings.read()` and `window.settings.write()` IPC calls
- Close on Escape key is already implemented

### Test Data Setup

Tests should:
1. Open settings panel via Cmd+, or a button click
2. Verify initial values are loaded
3. Make changes and save
4. Close and reopen to verify persistence

### Key DOM Elements

- Settings header: `<h2>Settings</h2>`
- Close button: Has `aria-label="Close settings"`
- Backend select: Currently no data-testid (needs to be added)
- Model select: Currently no data-testid (needs to be added)
- Save button: Text "Save" (add data-testid)
