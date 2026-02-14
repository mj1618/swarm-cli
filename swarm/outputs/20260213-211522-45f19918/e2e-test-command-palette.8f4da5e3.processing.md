# E2E Tests for Command Palette

## Goal

Add comprehensive Playwright E2E tests for the Command Palette component. The current E2E tests only verify that Cmd+K opens the palette but don't test search, navigation, or command execution. The Command Palette is a core Phase 5 feature that enables quick access to all app commands.

## Files

- **Modify**: `electron/e2e/app.spec.ts` - Add new test describe block for Command Palette
- **Modify**: `electron/src/renderer/components/CommandPalette.tsx` - Add data-testid attributes for E2E selectors

## Dependencies

- Requires: E2E test setup (complete)
- Requires: CommandPalette component (complete)

## Acceptance Criteria

1. Add `data-testid` attributes to CommandPalette.tsx:
   - `data-testid="command-palette"` on the outer backdrop div
   - `data-testid="command-palette-input"` on the search input
   - `data-testid="command-palette-list"` on the command list container
   - `data-testid="command-palette-item"` on each command button

2. New `test.describe('Swarm Desktop - Command Palette')` block with tests:
   - **Opens with Cmd+K**: Press Cmd+K, verify palette appears with input focused
   - **Closes with Escape**: Open palette, press Escape, verify it closes
   - **Closes on backdrop click**: Open palette, click outside the dialog, verify it closes
   - **Shows commands**: Open palette, verify command list is populated (not empty)
   - **Filters commands**: Type in search box, verify list filters to matching commands
   - **Shows "No commands found"**: Type gibberish, verify "No commands found" message
   - **Keyboard navigation**: Use Arrow Down/Up to navigate, verify selection changes
   - **Execute with Enter**: Navigate to a command, press Enter, verify palette closes

3. All tests pass consistently with `npx playwright test`

## Notes

From `CommandPalette.tsx`:
- Input placeholder: "Type a command..."
- Empty state text: "No commands found"
- Escape closes the palette
- Enter executes the selected command
- Arrow Up/Down navigates the list
- Clicking backdrop (bg-black/50 div) closes palette

### Available Commands (from App.tsx)

The command palette includes commands like:
- "Open Settings" (Cmd+,)
- "Toggle Theme" (Cmd+Shift+T)
- "Fit DAG to View" (Cmd+0)
- "New Task" (Cmd+N)
- "Save" (Cmd+S)
- "Keyboard Shortcuts" (Cmd+/)

### Test Strategy

```typescript
test.describe('Swarm Desktop - Command Palette', () => {
  test('opens with Cmd+K shortcut', async () => {
    await window.keyboard.press('Meta+k');
    const palette = window.locator('[data-testid="command-palette"]');
    await expect(palette).toBeVisible({ timeout: 2000 });
  });

  test('search input is focused when opened', async () => {
    await window.keyboard.press('Meta+k');
    const input = window.locator('[data-testid="command-palette-input"]');
    await expect(input).toBeFocused({ timeout: 2000 });
    await window.keyboard.press('Escape');
  });

  test('closes with Escape key', async () => {
    await window.keyboard.press('Meta+k');
    await window.keyboard.press('Escape');
    const palette = window.locator('[data-testid="command-palette"]');
    await expect(palette).not.toBeVisible({ timeout: 2000 });
  });

  test('filters commands when typing', async () => {
    await window.keyboard.press('Meta+k');
    await window.keyboard.type('settings');
    const items = window.locator('[data-testid="command-palette-item"]');
    // Should filter to show only commands matching "settings"
    await expect(items).toHaveCount(1);
    await window.keyboard.press('Escape');
  });
});
```
