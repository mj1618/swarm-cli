# Task: Add Console Log Export Button

**Phase:** 5 - Polish
**Priority:** Medium

## Goal

Add an "Export" button to the ConsolePanel that lets users save the currently visible log content to a file. The ELECTRON_PLAN.md explicitly lists "Export logs" as a console panel feature, but it has not been implemented.

## Files to Modify

1. **`electron/src/renderer/components/ConsolePanel.tsx`** — Add an export button to the tab bar area (next to the existing Clear button)
2. **`electron/src/main/index.ts`** — Add an IPC handler for showing a save dialog and writing the exported content

## Dependencies

- Console panel exists (completed)
- Log viewing works (completed)

## Implementation Details

### 1. Add IPC handler in main process (`electron/src/main/index.ts`)

Register a new IPC handler `dialog:saveFile` that:
- Opens Electron's `dialog.showSaveDialog()` with sensible defaults (`.log` or `.txt` extension, default filename based on the active tab name and timestamp)
- If the user picks a path, writes the provided content to that file using `fs.promises.writeFile`
- Returns `{ error?: string }` indicating success or failure

```typescript
ipcMain.handle('dialog:saveFile', async (_event, options: { defaultName: string; content: string }) => {
  const result = await dialog.showSaveDialog({
    defaultPath: options.defaultName,
    filters: [
      { name: 'Log files', extensions: ['log', 'txt'] },
      { name: 'All files', extensions: ['*'] },
    ],
  })
  if (result.canceled || !result.filePath) {
    return { canceled: true }
  }
  try {
    await fs.promises.writeFile(result.filePath, options.content, 'utf-8')
    return {}
  } catch (err: any) {
    return { error: err.message }
  }
})
```

### 2. Expose in preload (`electron/src/main/preload.ts` or `electron/src/preload/index.ts`)

Add the `dialog:saveFile` channel to the preload's exposed API:

```typescript
dialog: {
  saveFile: (options: { defaultName: string; content: string }) =>
    ipcRenderer.invoke('dialog:saveFile', options),
}
```

### 3. Add Export button in ConsolePanel

Add a button next to the existing "Clear" button:

```tsx
<button
  onClick={async () => {
    const tabName = activeTab === 'console' ? 'console' : tabLabel(activeFile?.name || 'log')
    const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
    const defaultName = `swarm-${tabName}-${timestamp}.log`
    const result = await window.dialog.saveFile({ defaultName, content: activeContent })
    if (result.error) {
      // optionally show error (if onToast is available)
    }
  }}
  disabled={!activeContent}
  className="px-2 py-1 text-xs text-muted-foreground hover:text-foreground disabled:opacity-40"
  title="Export logs to file"
>
  Export
</button>
```

### 4. Add TypeScript types

Add the `dialog` namespace to the Window interface declaration in the preload types:

```typescript
dialog: {
  saveFile: (options: { defaultName: string; content: string }) =>
    Promise<{ error?: string; canceled?: boolean }>
}
```

## Acceptance Criteria

1. An "Export" button appears in the console panel tab bar (near the Clear button)
2. Clicking Export opens the system save dialog with a sensible default filename like `swarm-console-2026-02-13T17-30-00.log`
3. Choosing a save location writes the currently displayed log content to the file
4. The button is disabled when there's no log content to export
5. If the user cancels the save dialog, nothing happens (no error)
6. TypeScript compiles without errors
7. App builds successfully with `npm run build`

## Notes

- This is explicitly listed in ELECTRON_PLAN.md under Console/Logs features: "Export logs"
- The implementation follows the existing IPC pattern used by other file operations
- The export should save the *currently visible* content (respecting the active tab and any active search filter)

## Completion Note

Implemented by agent 18b37942. All acceptance criteria met:
- Export button added to ConsolePanel tab bar, next to Clear button
- Uses Electron's native save dialog via `dialog:saveFile` IPC channel
- Default filename format: `swarm-{tab}-{timestamp}.log`
- Button disabled when no content is available
- Canceling the dialog is handled gracefully (no error)
- TypeScript compiles without errors; `npm run build` passes
