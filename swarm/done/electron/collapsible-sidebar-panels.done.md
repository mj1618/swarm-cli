# Task: Collapsible Sidebar Panels

## Goal

Add the ability to collapse/expand the left sidebar (File Tree) and right sidebar (Agent Panel) to give users more screen space for the DAG editor when needed.

## Files

- `electron/src/renderer/App.tsx` - Add collapse state, toggle buttons, and persistence for both sidebars

## Dependencies

- `resizable-sidebar-panels.done.md` - Sidebars already support resizing; this adds collapse functionality

## Acceptance Criteria

1. Left sidebar (File Tree) has a collapse/expand toggle button
2. Right sidebar (Agent Panel) has a collapse/expand toggle button  
3. Collapsed state shows a thin bar (~28px like console) with an expand button
4. Collapse state persists in localStorage across sessions
5. Keyboard shortcuts work: `Cmd+B` to toggle left sidebar, `Cmd+Shift+B` to toggle right sidebar
6. Collapsed sidebars animate smoothly (similar to console panel behavior)
7. Double-clicking the resize handle restores default width (existing behavior preserved)

## Notes

- Follow the existing pattern from `consoleCollapsed` state management in App.tsx
- Use similar collapsed height/width constant: `COLLAPSED_SIDEBAR_WIDTH = 28`
- The toggle button should use the same arrow icons (▶/◀) as the console panel
- Add keyboard shortcuts to the existing `handleKeyDown` effect
- Add the new shortcuts to `KeyboardShortcutsHelp.tsx`
- Consider adding Command Palette commands: "Toggle left sidebar", "Toggle right sidebar"

## Implementation Hints

The console panel collapse pattern to follow (from App.tsx):

```typescript
const [consoleCollapsed, setConsoleCollapsed] = useState<boolean>(() => {
  return localStorage.getItem('swarm-console-collapsed') === 'true'
})

// Toggle function
const toggleConsole = useCallback(() => {
  setConsoleCollapsed(prev => {
    const next = !prev
    localStorage.setItem('swarm-console-collapsed', String(next))
    return next
  })
}, [])
```

Replicate this pattern for `leftSidebarCollapsed` and `rightSidebarCollapsed`.

---

## Completion Notes (2026-02-13)

Implemented all acceptance criteria:

1. **Collapse state management**: Added `leftSidebarCollapsed` and `rightSidebarCollapsed` state variables with localStorage persistence following the console panel pattern.

2. **Toggle functions**: Created `toggleLeftSidebar()` and `toggleRightSidebar()` callbacks.

3. **Sidebar UI updates**: 
   - Both sidebars now show a collapsed state (28px wide) with an expand button when collapsed
   - Added a header bar to the left sidebar with "Files" label and collapse toggle
   - Smooth CSS transitions for width changes (`transition-[width] duration-200 ease-in-out`)

4. **Keyboard shortcuts**: Added `Cmd+B` (left sidebar) and `Cmd+Shift+B` (right sidebar) to the keyboard handler.

5. **Updated KeyboardShortcutsHelp.tsx**: Added the new shortcuts to the General section.

6. **Command Palette commands**: Added "Toggle left sidebar" and "Toggle right sidebar" commands.

7. **Menu IPC listeners**: Added handlers for `menu:toggle-left-sidebar` and `menu:toggle-right-sidebar`.

Files modified:
- `electron/src/renderer/App.tsx` - Main implementation
- `electron/src/renderer/components/KeyboardShortcutsHelp.tsx` - Added shortcuts documentation
