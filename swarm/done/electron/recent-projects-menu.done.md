# Task: Add Recent Projects Menu

## Goal

Add a "Recent Projects" submenu to the File menu that displays the last 5 recently opened project directories, allowing users to quickly switch between workspaces without using the Open Project dialog.

## Files to Modify

- `electron/src/main/index.ts` - Add recent projects storage, IPC handlers, and dynamic menu rebuilding
- `electron/src/preload/index.ts` - Expose IPC for recent projects
- `electron/src/renderer/App.tsx` - Call IPC to add project to recents when opened

## Dependencies

- None (this is Phase 5 polish work, all core features are complete)

## Implementation Details

### 1. Main Process (`electron/src/main/index.ts`)

Add recent projects storage using `electron.app.getPath('userData')`:
- Store recent projects in a JSON file at `~/.config/swarm-desktop/recent-projects.json`
- Track up to 5 recent project paths
- Add IPC handlers:
  - `recent:get` - Returns list of recent project paths
  - `recent:add` - Adds a project path to recents (moves to top if already exists)
  - `recent:clear` - Clears the recent projects list

Update `buildAppMenu()` to include Recent Projects submenu:
```typescript
{
  label: 'File',
  submenu: [
    {
      label: 'Open Project',
      accelerator: 'CmdOrCtrl+O',
      click: () => sendToRenderer('menu:open-project'),
    },
    {
      label: 'Recent Projects',
      submenu: recentProjects.length > 0
        ? [
            ...recentProjects.map(p => ({
              label: shortenPath(p),
              click: () => sendToRenderer('menu:open-recent', p),
            })),
            { type: 'separator' },
            { label: 'Clear Recent', click: () => clearRecentProjects() },
          ]
        : [{ label: 'No Recent Projects', enabled: false }],
    },
    { type: 'separator' },
    isMac ? { role: 'close' } : { role: 'quit' },
  ],
}
```

Add helper function to shorten paths:
```typescript
function shortenPath(fullPath: string): string {
  const home = app.getPath('home')
  if (fullPath.startsWith(home)) {
    return '~' + fullPath.slice(home.length)
  }
  return fullPath
}
```

Rebuild the menu whenever recent projects change.

### 2. Preload (`electron/src/preload/index.ts`)

Add to contextBridge:
```typescript
recent: {
  get: () => ipcRenderer.invoke('recent:get'),
  add: (path: string) => ipcRenderer.invoke('recent:add', path),
  clear: () => ipcRenderer.invoke('recent:clear'),
}
```

Add menu listener for opening recent projects:
```typescript
electronMenu: {
  on: (channel: string, callback: (data?: any) => void) => {
    const handler = (_event: any, data?: any) => callback(data)
    ipcRenderer.on(channel, handler)
    return () => ipcRenderer.removeListener(channel, handler)
  },
}
```

### 3. Renderer (`electron/src/renderer/App.tsx`)

Update `handleOpenProject` to add project to recents:
```typescript
const handleOpenProject = useCallback(async () => {
  const result = await window.workspace.open()
  if (!result.path) return
  // ... existing logic ...
  await window.recent.add(result.path)  // Add this line
}, [addToast])
```

Add menu listener for `menu:open-recent`:
```typescript
window.electronMenu.on('menu:open-recent', async (path: string) => {
  // Switch to the project at path
  // Similar to handleOpenProject but with a known path
})
```

## Acceptance Criteria

1. File menu shows "Recent Projects" submenu
2. Submenu displays up to 5 most recently opened projects
3. Clicking a recent project opens that workspace
4. Projects are shown with shortened paths (~/code/project instead of /Users/name/code/project)
5. "Clear Recent" option removes all recent projects
6. List persists across app restarts
7. Opening a project moves it to the top of the list
8. Opening the same project twice doesn't create duplicates

## Notes

- Use `app.getPath('userData')` for cross-platform config storage
- Menu must be rebuilt dynamically when recent projects change (call `Menu.setApplicationMenu()` again)
- Consider adding keyboard shortcuts Cmd+1 through Cmd+5 for quick access to recent projects (optional enhancement)

---

## Completion Notes

**Completed by:** Agent 517277dc  
**Date:** Feb 13, 2026

### What was implemented:

1. **Main Process (`electron/src/main/index.ts`)**:
   - Added recent projects storage using `app.getPath('userData')` for the JSON file
   - Implemented `loadRecentProjects()`, `saveRecentProjects()`, `addRecentProject()`, and `clearRecentProjects()` functions
   - Added `shortenPath()` helper to display paths with `~` prefix
   - Updated `buildAppMenu()` to be async and include the Recent Projects submenu
   - Added keyboard shortcuts Cmd+1 through Cmd+5 for quick access to recent projects
   - Added IPC handlers: `recent:get`, `recent:add`, `recent:clear`
   - Refactored workspace switching into a reusable `switchWorkspace()` function
   - Added `workspace:switch` IPC handler for switching to recent projects

2. **Preload (`electron/src/preload/index.ts`)**:
   - Added `recent` API with `get`, `add`, `clear` methods
   - Added `workspace.switch` method
   - Updated `electronMenu.on` to support data-passing for `menu:open-recent`
   - Added `RecentAPI` type definition
   - Updated `WorkspaceAPI` and `ElectronMenuAPI` types

3. **Renderer (`electron/src/renderer/App.tsx`)**:
   - Updated `handleOpenProject` to call `window.recent.add()` after opening a project
   - Added `handleOpenRecentProject()` function for opening projects from the menu
   - Added listener for `menu:open-recent` in the useEffect

### All acceptance criteria met:
- File menu shows "Recent Projects" submenu with up to 5 projects
- Clicking a recent project switches to that workspace
- Paths are shortened with `~` prefix
- "Clear Recent Projects" option works
- List persists via JSON file in userData
- Opening a project moves it to the top
- No duplicates are created
- Bonus: Keyboard shortcuts Cmd+1-5 for quick access
