# Theme Toggle (Dark/Light Mode)

## Goal

Add a theme toggle to Settings that allows users to switch between dark and light modes. The app is currently hardcoded to dark mode only.

## Files

### Create
- `electron/src/renderer/lib/themeManager.ts` - Theme state management and persistence

### Modify
- `electron/src/renderer/components/SettingsPanel.tsx` - Add Appearance section with theme toggle
- `electron/src/renderer/components/MonacoFileEditor.tsx` - Switch Monaco theme dynamically (`vs-dark` / `vs`)
- `electron/src/renderer/components/DagCanvas.tsx` - Switch React Flow `colorMode` dynamically
- `electron/src/renderer/App.tsx` - Apply theme class to root element
- `electron/src/renderer/index.css` - Add CSS variables for light mode colors

## Dependencies

None - this is a standalone polish feature for Phase 5.

## Acceptance Criteria

1. Settings panel has an "Appearance" section with theme options: System, Dark, Light
2. Theme preference persists in localStorage (`swarm-theme`)
3. "System" option respects `prefers-color-scheme` media query
4. Monaco editor switches between `vs-dark` and `vs` themes
5. React Flow DAG canvas uses appropriate `colorMode`
6. All UI elements (borders, backgrounds, text) update correctly for both themes
7. Theme changes apply immediately without requiring app restart

## Notes

### Current hardcoded dark mode locations:
- `MonacoFileEditor.tsx` line 415: `theme="vs-dark"`
- `DagCanvas.tsx` line 497: `colorMode="dark"`
- Tailwind CSS uses dark colors throughout

### Implementation approach:
1. Create `themeManager.ts` with:
   - `getTheme(): 'system' | 'dark' | 'light'`
   - `setTheme(theme): void`
   - `getEffectiveTheme(): 'dark' | 'light'` (resolves 'system' to actual theme)
   - `onThemeChange(callback): unsubscribe` (for system preference changes)

2. Add CSS variables in `index.css`:
   ```css
   :root {
     /* Light mode variables */
   }
   :root.dark {
     /* Dark mode variables (current defaults) */
   }
   ```

3. Update App.tsx to:
   - Subscribe to theme changes
   - Apply `dark` class to document root when in dark mode
   - Pass effective theme to child components

4. Update SettingsPanel with segmented control: System | Dark | Light

### Tailwind dark mode:
Tailwind is likely configured for `class` strategy. Verify in `tailwind.config.js`:
```js
darkMode: 'class'
```

---

## Completion Notes (agent: b61dea07)

**Implemented:**
- Created `themeManager.ts` with full theme management: `getTheme()`, `setTheme()`, `getEffectiveTheme()`, `onThemeChange()`, `initThemeManager()`, and `applyTheme()`
- Added light mode CSS variables in `index.css` with `:root` (light) and `:root.dark` (dark) selectors
- Enabled `darkMode: 'class'` in `tailwind.config.js`
- Updated `App.tsx` to initialize theme manager and pass effective theme to child components
- Added "Appearance" section in `SettingsPanel.tsx` with System/Dark/Light toggle buttons
- Updated `MonacoFileEditor.tsx` to switch between `vs-dark` and `vs` themes
- Updated `DagCanvas.tsx` to use dynamic `colorMode` based on theme

**All acceptance criteria met:**
1. Settings panel has Appearance section with System, Dark, Light options
2. Theme persists in localStorage with key `swarm-theme`
3. System option respects OS `prefers-color-scheme` via matchMedia
4. Monaco editor switches themes dynamically
5. React Flow DAG canvas respects theme colorMode
6. CSS variables update all UI elements for both themes
7. Theme changes apply immediately via listener pattern
