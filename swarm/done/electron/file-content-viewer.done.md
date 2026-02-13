# Task: File Content Viewer Panel

## Goal

When a file is selected in the file tree (left sidebar), display its contents in the center panel instead of the "DAG visualization coming soon" placeholder. This is the foundation for the YAML editor and prompt editor — the center panel needs to show file contents before any editing can happen.

## Files

### Create
- `electron/src/renderer/components/FileViewer.tsx` — Component that renders file contents with syntax-appropriate styling (YAML highlighting, Markdown rendering, plain text for others)

### Modify
- `electron/src/renderer/App.tsx` — Lift `selectedPath` state from FileTree up to App, pass it to both FileTree and the new FileViewer. Replace the DAG placeholder with FileViewer when a file is selected.
- `electron/src/renderer/components/FileTree.tsx` — Accept `selectedPath` and `onSelect` as props instead of managing selection state internally. This allows App to control which file is displayed.

## Dependencies

- File tree component (completed)
- `fs:readfile` IPC handler (completed in main process)
- `window.fs.readfile()` preload API (completed)

## Acceptance Criteria

1. Clicking a file in the file tree loads and displays its content in the center panel
2. File content is displayed in a monospace font with line numbers
3. YAML files (`.yaml`, `.yml`) get a "YAML" label in the panel header
4. Markdown files (`.md`) get a "Markdown" label in the panel header
5. TOML files (`.toml`) get a "Config" label in the panel header
6. The file path is shown in the panel header
7. When no file is selected, the existing DAG placeholder is shown
8. Clicking a directory in the file tree does NOT replace the center panel (only files trigger the viewer)
9. Large files scroll properly within the viewer
10. The component handles loading and error states gracefully

## Notes

- Use `window.fs.readfile(path)` to load file content via the existing IPC bridge
- Keep the viewer read-only for now — editing comes in Phase 3
- The center panel should show a header bar with the filename and file type badge, then a scrollable content area below
- Use Tailwind classes consistent with the existing dark theme (`bg-background`, `text-foreground`, `border-border`, etc.)
- This naturally sets up the center panel to later swap between FileViewer and the DAG editor based on context
- Consider using `<pre>` with `whitespace-pre-wrap` for content display; no need for Monaco editor yet (that's Phase 5)

## Completion Notes

Implemented by agent 39ef5fc1. Changes:

- **Created `FileViewer.tsx`**: Read-only file content viewer with monospace font, line numbers via a `<pre><table>` layout, file type badges (YAML, Markdown, Config, Log, JSON, Text), file path display in header, and loading/error states. Uses `window.fs.readfile()` IPC bridge.
- **Modified `FileTree.tsx`**: Now accepts `selectedPath` and `onSelectFile` props from parent instead of managing selection state internally. The `handleSelect` callback filters out directory clicks (only files trigger `onSelectFile`).
- **Modified `FileTreeItem.tsx`**: Updated `onSelect` callback signature to include `isDirectory` boolean, so the parent can distinguish file vs directory clicks.
- **Modified `App.tsx`**: Lifted `selectedFile` state to App level, passes it down to both `FileTree` and `FileViewer`. Center panel conditionally renders `FileViewer` when a file is selected, falling back to the DAG placeholder when no file is selected.

All acceptance criteria met. Build verified clean.
