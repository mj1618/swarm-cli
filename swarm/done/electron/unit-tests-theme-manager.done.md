# Unit Tests for Theme Manager

## Goal

Add unit tests for `electron/src/renderer/lib/themeManager.ts` to improve test coverage and ensure theme persistence and system preference detection work correctly.

## Files

- **Create**: `electron/src/renderer/lib/__tests__/themeManager.test.ts`
- **Reference**: `electron/src/renderer/lib/themeManager.ts`

## Dependencies

- Unit test infrastructure already set up (vitest)
- Existing test patterns available in `electron/src/renderer/lib/__tests__/`

## Acceptance Criteria

1. Test file created at `electron/src/renderer/lib/__tests__/themeManager.test.ts`
2. Tests cover:
   - `getTheme()` returns 'system' when localStorage is empty
   - `getTheme()` returns stored value when valid ('dark', 'light', 'system')
   - `getTheme()` returns 'system' for invalid stored values
   - `setTheme()` persists to localStorage
   - `getEffectiveTheme()` resolves 'system' to actual theme
   - `getEffectiveTheme()` returns preference directly for 'dark'/'light'
   - `onThemeChange()` returns unsubscribe function
   - Theme change listeners are notified when theme changes
3. All tests pass: `npm test` in electron/ directory
4. Tests use mocks for localStorage and window.matchMedia

## Notes

- Follow existing test patterns from `yamlParser.test.ts` and `outputFolderUtils.test.ts`
- Mock `localStorage` using vitest's mocking capabilities
- Mock `window.matchMedia` for system theme detection tests
- The `applyTheme()` function modifies `document.documentElement.classList` - mock or use jsdom
- Use `beforeEach` to reset mocks between tests

---

## Completion Notes

**Completed by agent a8d0c5c6**

Created comprehensive test suite with 32 tests covering all themeManager functions:

- **getTheme()**: 8 tests covering empty localStorage, valid values (dark/light/system), and invalid values
- **setTheme()**: 5 tests covering localStorage persistence, applyTheme calls, and listener notifications
- **getEffectiveTheme()**: 5 tests covering system preference resolution and direct preferences
- **applyTheme()**: 4 tests covering classList manipulation for dark/light themes
- **onThemeChange()**: 6 tests covering subscribe/unsubscribe, listener notifications, and multiple listeners
- **initThemeManager()**: 4 tests covering initial theme application, cleanup function, and system preference listeners

All tests pass and the build succeeds.
