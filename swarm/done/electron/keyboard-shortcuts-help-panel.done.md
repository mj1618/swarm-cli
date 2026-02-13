# Task: Add Keyboard Shortcuts Help Panel

## Goal

Add a keyboard shortcuts help panel (modal) that users can trigger by pressing `?` (when not focused on an input/editor). This improves discoverability of the app's many keyboard shortcuts and is a standard UX pattern for desktop applications.

**Phase:** 5 - Polish (Keyboard shortcuts documentation panel)

## Files

- **Create**: `electron/src/renderer/components/KeyboardShortcutsHelp.tsx` — New modal component displaying all shortcuts grouped by category
- **Modify**: `electron/src/renderer/App.tsx` — Add `?` keydown listener to toggle the modal, add command palette entry, render the component

## Dependencies

- None (all keyboard shortcuts already exist, this just documents them)

## Implementation Details

### Shortcuts to Document

Organize into categories:

**General**
| Shortcut | Action |
|----------|--------|
| `Cmd+K` / `Ctrl+K` | Open command palette |
| `Cmd+J` / `Ctrl+J` | Toggle console panel |
| `?` | Show keyboard shortcuts |

**Console / Logs**
| Shortcut | Action |
|----------|--------|
| `Cmd+F` / `Ctrl+F` | Search in console logs |
| `Escape` | Clear search |

**File Editor**
| Shortcut | Action |
|----------|--------|
| `Cmd+S` / `Ctrl+S` | Save file |

**DAG Canvas**
| Shortcut | Action |
|----------|--------|
| `Delete` / `Backspace` | Delete selected task or edge |

**Panels & Dialogs**
| Shortcut | Action |
|----------|--------|
| `Escape` | Close open drawer/dialog/panel |

### Component Design

- Modal overlay with semi-transparent backdrop
- Dark card with grouped shortcut tables
- Close on `Escape`, click outside, or close button
- Show keyboard key with styled `<kbd>` elements
- Detect platform (macOS vs other) to show `Cmd` vs `Ctrl`

### Command Palette Entry

Add a "Show keyboard shortcuts" entry to the command palette actions array in App.tsx.

## Acceptance Criteria

1. Pressing `?` (when no input/textarea/editor is focused) opens the shortcuts help modal
2. Modal displays all current keyboard shortcuts organized by category
3. Modal closes on `Escape`, clicking outside, or the close button
4. A "Show keyboard shortcuts" entry exists in the command palette (Cmd+K)
5. Shows `Cmd` on macOS and `Ctrl` on other platforms
6. TypeScript compiles without errors
7. App builds successfully with `npm run build`

## Notes

- Keep the component simple — a styled modal with a list, no complex state management needed
- Use the same dark theme styling as other panels (bg-card, border-border, text-foreground, etc.)
- The `?` listener should check `document.activeElement` to avoid triggering when user is typing in inputs, textareas, or contenteditable elements
- Reference CommandPalette.tsx for modal overlay styling patterns

## Completion Notes

Implemented by agent 31289df8. Created `KeyboardShortcutsHelp.tsx` with:
- Platform-aware modifier key display (⌘ on macOS, Ctrl otherwise)
- Five shortcut categories: General, Console/Logs, File Editor, DAG Canvas, Panels & Dialogs
- Styled `<kbd>` elements for key display
- Modal closes on Escape, backdrop click, or close button
- Added `?` keydown listener in App.tsx (skips when input/textarea/contenteditable focused)
- Added "Show keyboard shortcuts" entry to command palette
- Build passes with no TypeScript errors
