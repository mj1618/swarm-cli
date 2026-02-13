# Task: Add Native Application Menu

## Goal

Add a native Electron application menu to the Swarm Desktop app. Currently the app has no `Menu.buildFromTemplate()` call, so on macOS the menu bar shows only the default Electron items without any app-specific actions. A proper menu provides standard keyboard shortcuts (Cmd+O for Open Project, Cmd+Q for Quit, Edit menu for Undo/Redo/Cut/Copy/Paste in Monaco), mirrors command palette actions from the menu bar, and follows platform conventions.

## Phase

Phase 5 — Polish

## Files to Modify

1. **`electron/src/main/index.ts`** — Import `Menu` from electron and call `Menu.setApplicationMenu()` with a template containing the menus described below

## Implementation Details

### Menu Structure

```
App Menu (macOS only):
  - About Swarm Desktop
  - Separator
  - Settings... (Cmd+,) → send 'menu:settings' to renderer
  - Separator
  - Quit (Cmd+Q)

File:
  - Open Project (Cmd+O) → trigger workspace:open dialog
  - Separator
  - Close Window (Cmd+W)

Edit:
  - Undo (Cmd+Z)        → role: 'undo'
  - Redo (Cmd+Shift+Z)  → role: 'redo'
  - Separator
  - Cut (Cmd+X)          → role: 'cut'
  - Copy (Cmd+C)         → role: 'copy'
  - Paste (Cmd+V)        → role: 'paste'
  - Select All (Cmd+A)   → role: 'selectAll'

View:
  - Toggle Console (Cmd+J) → send 'menu:toggle-console' to renderer
  - Command Palette (Cmd+K) → send 'menu:command-palette' to renderer
  - Separator
  - Reload (Cmd+R)       → role: 'reload'
  - Toggle DevTools (Cmd+Alt+I) → role: 'toggleDevTools'
  - Separator
  - Actual Size           → role: 'resetZoom'
  - Zoom In               → role: 'zoomIn'
  - Zoom Out              → role: 'zoomOut'
  - Separator
  - Toggle Full Screen    → role: 'togglefullscreen'

Window:
  - Minimize              → role: 'minimize'
  - Zoom                  → role: 'zoom'
  - Separator
  - Bring All to Front   → role: 'front' (macOS)
```

### Renderer Integration

The renderer needs to listen for menu IPC events. In `App.tsx`, add:

```ts
useEffect(() => {
  const handlers: Record<string, () => void> = {
    'menu:settings': () => setSettingsOpen(true),
    'menu:toggle-console': toggleConsole,
    'menu:command-palette': () => setPaletteOpen(prev => !prev),
  }
  const cleanups = Object.entries(handlers).map(([channel, handler]) => {
    return window.electronMenu?.on(channel, handler) // needs preload bridge
  })
  return () => { cleanups.forEach(fn => fn?.()) }
}, [toggleConsole])
```

This requires adding a small `electronMenu` bridge in `preload/index.ts`:

```ts
contextBridge.exposeInMainWorld('electronMenu', {
  on: (channel: string, callback: () => void) => {
    const listener = () => callback()
    ipcRenderer.on(channel, listener)
    return () => { ipcRenderer.removeListener(channel, listener) }
  },
})
```

### Platform Handling

- Use `process.platform === 'darwin'` to conditionally include the macOS app menu as the first item
- The Edit menu with roles is essential for Monaco Editor keyboard shortcuts to work in packaged builds (without it, Cmd+C/V/X won't work in the text editor on macOS)

## Dependencies

None — all prerequisite infrastructure exists.

## Acceptance Criteria

1. On macOS, the menu bar shows: Swarm Desktop, File, Edit, View, Window
2. File > Open Project opens the workspace directory picker
3. Edit menu roles (Undo/Redo/Cut/Copy/Paste) work correctly in the Monaco editor
4. View > Toggle Console sends IPC to toggle the console panel
5. View > Command Palette sends IPC to open/close the command palette
6. App menu > Settings opens the settings panel
7. Standard keyboard shortcuts (Cmd+Q, Cmd+W, Cmd+R, zoom) work
8. No TypeScript errors (`tsc --noEmit` passes)

## Notes

- The Edit menu with roles is particularly important — without an explicit Edit menu, Electron apps on macOS lose Cmd+C/Cmd+V/Cmd+X in text inputs and Monaco
- Keep the menu simple and focused on the actions already available in the app
- The `File > Open Project` action can reuse the same `workspace:open` IPC handler already wired up for the title bar button

## Completion Notes

Implemented by agent d6842825. Changes:

1. **`electron/src/main/index.ts`**: Added `Menu` import and `buildAppMenu()` function that creates a full native menu with:
   - macOS app menu (About, Settings via Cmd+,, Services, Hide/Quit)
   - File menu (Open Project via Cmd+O, Close Window)
   - Edit menu (Undo/Redo/Cut/Copy/Paste/Select All with roles — essential for Monaco)
   - View menu (Toggle Console, Command Palette, Reload, DevTools, Zoom controls, Fullscreen)
   - Window menu (Minimize, Zoom, Bring All to Front on macOS)

2. **`electron/src/preload/index.ts`**: Added `electronMenu` bridge with channel allowlist for secure IPC (`menu:settings`, `menu:toggle-console`, `menu:command-palette`, `menu:open-project`). Added `ElectronMenuAPI` type and updated `Window` interface.

3. **`electron/src/renderer/App.tsx`**: Added `useEffect` hook listening for menu IPC events to trigger settings panel, console toggle, command palette toggle, and open project dialog.

Build verified: `tsc && vite build` passes with no errors.
