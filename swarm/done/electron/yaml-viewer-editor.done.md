# Task: Basic YAML Viewer/Editor for swarm.yaml

## Goal

Implement a basic YAML viewer/editor in the center panel that displays file contents when a file is selected in the file tree. This completes Phase 1 item 3 ("Basic YAML viewer/editor for swarm.yaml") from ELECTRON_PLAN.md.

Currently the center panel shows a static "DAG visualization coming soon" placeholder. This task wires up file selection from the FileTree to display file contents in the center panel, with syntax-appropriate rendering for YAML and Markdown files.

## Files

### Modify
- `electron/src/renderer/App.tsx` — Add state for selected file, wire `selectedPath`/`onSelectFile` props to FileTree, replace center panel placeholder with file viewer
- `electron/src/renderer/components/FileTree.tsx` — Currently expects `selectedPath` and `onSelectFile` props but App.tsx passes none; fix the integration

### Create
- `electron/src/renderer/components/FileViewer.tsx` — New component that:
  - Accepts a file path prop
  - Loads file content via `window.fs.readfile()`
  - Displays content in a scrollable, monospace pre/code block
  - Shows the filename in a header bar
  - Handles loading/error states
  - Uses basic syntax highlighting cues (YAML keys in one color, values in another) — can be simple CSS-based, no need for a full parser yet
  - Shows a placeholder/welcome message when no file is selected

## Dependencies

- File tree component (completed — FileTree.tsx, FileTreeItem.tsx)
- Filesystem IPC handlers (completed — fs:readfile, fs:readdir in main process and preload)

## Acceptance Criteria

1. Clicking a file in the left sidebar FileTree loads and displays its contents in the center panel
2. The center panel shows the file name in a header
3. File content is displayed in a scrollable, monospace view
4. YAML files (`.yaml`, `.yml`) get basic visual distinction (at minimum, display with proper formatting)
5. Markdown files (`.md`) display as raw text (rendered markdown preview is a later phase)
6. Loading and error states are handled gracefully
7. When no file is selected, a welcome/placeholder message is shown
8. The app builds without TypeScript errors (`npm run build` in electron/)

## Notes

- The plan calls for Monaco Editor integration in Phase 5, so this phase should use a simple read-only viewer (pre/code block). Keep it simple.
- The `window.fs.readfile()` API already exists in the preload and returns `{ content: string; error?: string }`.
- The FileTree component already accepts `selectedPath` and `onSelectFile` props — App.tsx just needs to pass them.
- Keep the "DAG Editor" label/concept — the center panel should show the file viewer when a file is selected, but can still reference that DAG editing is coming. A tab-like or breadcrumb header showing the current file would work well.
- Tailwind CSS classes are available for styling.

## Completion Notes

**Completed by agent e96604f9 on iteration 2.**

Most of the implementation was already in place from a previous iteration (FileViewer.tsx, App.tsx wiring with selectedFile state, FileTree props integration). This iteration added:

- **YAML syntax highlighting** to FileViewer.tsx: keys are shown in blue, string values in yellow, quoted strings in green, numbers/booleans in purple, null values in red, comments in muted italic, and list item dashes in orange. This is done with simple regex-based highlighting (no full parser), as specified.

All 8 acceptance criteria are met:
1. File selection from FileTree loads content in center panel
2. Header shows file type badge and filename
3. Scrollable monospace view with line numbers
4. YAML files get colorized syntax highlighting
5. Markdown/other files display as raw text
6. Loading and error states handled
7. DAG Editor placeholder shown when no file is selected
8. Build passes cleanly (`npm run build` succeeds)
