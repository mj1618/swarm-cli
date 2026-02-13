# Task: Add Help Menu with Documentation Links

**Phase:** 5 - Polish (extension)
**Priority:** Low

## Goal

Add a Help menu to the native application menu bar with links to documentation, keyboard shortcuts reference, and an About dialog. This improves discoverability of features and provides quick access to help resources.

## Files to Modify

- `electron/src/main/index.ts` — Add Help menu to the native menu template with documentation links and About dialog trigger
- `electron/src/renderer/App.tsx` — Add IPC listener for `menu:about` to show an About modal
- `electron/src/renderer/components/AboutDialog.tsx` — New component for the About dialog

## Dependencies

- Native application menu (completed: add-native-application-menu.done.md)
- Keyboard shortcuts help panel (completed: keyboard-shortcuts-help-panel.done.md)

## Implementation Notes

### Native Menu (main/index.ts)

Add a Help menu to the `template` array in `buildAppMenu()`:

```typescript
{
  label: 'Help',
  submenu: [
    {
      label: 'Keyboard Shortcuts',
      accelerator: 'CmdOrCtrl+/',
      click: () => sendToRenderer('menu:keyboard-shortcuts'),
    },
    { type: 'separator' },
    {
      label: 'Swarm CLI Documentation',
      click: () => shell.openExternal('https://github.com/your-org/swarm-cli#readme'),
    },
    {
      label: 'Report an Issue',
      click: () => shell.openExternal('https://github.com/your-org/swarm-cli/issues'),
    },
    { type: 'separator' },
    {
      label: 'About Swarm Desktop',
      click: () => sendToRenderer('menu:about'),
    },
  ],
}
```

Import `shell` from Electron for opening external links.

### IPC Channel Registration (preload/index.ts)

Add `'menu:keyboard-shortcuts'` and `'menu:about'` to the allowed channels list in `electronMenu.on()`.

### App.tsx Changes

Add listeners for the new menu events:
- `menu:keyboard-shortcuts` — Open the existing KeyboardShortcutsHelp panel
- `menu:about` — Open a new AboutDialog modal

Add state for `aboutOpen` and render `<AboutDialog>` when true.

### AboutDialog Component

Create a simple modal dialog showing:
- App icon/logo (optional, can use text)
- "Swarm Desktop" title
- Version number (read from package.json or hardcoded initially)
- Brief description: "A visual interface for swarm-cli pipelines"
- Links: GitHub repo, documentation
- Close button

Style to match the existing dark theme using Tailwind classes consistent with other modals (KeyboardShortcutsHelp, SettingsPanel).

## Acceptance Criteria

1. A "Help" menu appears in the native menu bar (after "Window" menu)
2. "Keyboard Shortcuts" menu item opens the existing shortcuts help panel
3. "Swarm CLI Documentation" opens the GitHub README in the default browser
4. "Report an Issue" opens the GitHub issues page in the default browser  
5. "About Swarm Desktop" opens a modal dialog with app info and version
6. The About dialog can be closed by clicking the X button or pressing Escape
7. App builds successfully with `npm run build`

## Notes

- Use `electron.shell.openExternal()` for opening URLs - this is the secure way to open external links from Electron
- The version can be read from `app.getVersion()` in main process and passed to renderer, or hardcoded as "1.0.0" initially
- This is a low-priority polish task that improves user experience but doesn't add core functionality

---

## Completion Notes (Agent 44fa4e4b)

**Status:** Completed

**Changes Made:**

1. **electron/src/main/index.ts:**
   - Added `shell` import from Electron
   - Added Help menu with: Keyboard Shortcuts (Cmd+/), Documentation link, Report Issue link, and About Swarm Desktop

2. **electron/src/preload/index.ts:**
   - Added `'menu:keyboard-shortcuts'` and `'menu:about'` to the allowed IPC channels

3. **electron/src/renderer/components/AboutDialog.tsx (new):**
   - Created modal dialog matching existing dark theme
   - Displays app icon, title, version (1.0.0), description, and links
   - Supports Escape key and backdrop click to close

4. **electron/src/renderer/App.tsx:**
   - Imported AboutDialog component
   - Added `aboutOpen` state
   - Added IPC listeners for menu:keyboard-shortcuts and menu:about
   - Rendered AboutDialog in component tree

**Verification:**
- `npm run build` - Passed
- `tsc --noEmit` - Passed (no TypeScript errors)
