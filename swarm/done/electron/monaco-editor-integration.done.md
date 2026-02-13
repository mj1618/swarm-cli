# Task: Monaco Editor Integration

**Phase:** 5 - Polish
**Priority:** High (major Phase 5 item, replaces basic FileViewer with full editor)

## Goal

Replace the current `FileViewer.tsx` syntax-highlighted read-only viewer with a Monaco Editor instance that supports editing files directly in the app. This enables the "YAML Editor with IntelliSense" and "Prompt Editor" features described in ELECTRON_PLAN.md (lines 282-295).

The current `FileViewer.tsx` renders file content with basic regex-based syntax highlighting and line numbers but does not support editing. Monaco provides full editing, syntax highlighting, bracket matching, find/replace, and a foundation for future IntelliSense/autocomplete.

## What to Build

### 1. Install Monaco Editor

Add `@monaco-editor/react` package to electron/package.json. This provides a React wrapper around Monaco with lazy loading and automatic language detection.

### 2. Replace FileViewer with MonacoFileEditor

Create a new `MonacoFileEditor.tsx` component that:

- Accepts `filePath` (string) and `content` (string) props
- Renders a Monaco Editor instance with the file content
- Auto-detects language from file extension:
  - `.yaml` / `.yml` → `yaml`
  - `.md` → `markdown`
  - `.toml` → `toml` (or fallback to `ini`)
  - `.json` → `json`
  - `.log` → `plaintext`
  - `.ts` / `.tsx` → `typescript`
  - `.js` / `.jsx` → `javascript`
  - `.go` → `go`
  - Default → `plaintext`
- Uses a dark theme matching the app's dark UI (e.g., `vs-dark`)
- Has a **Save** button (or Cmd+S / Ctrl+S) that writes the file back via the existing `window.fs.writeFile` IPC call
- Shows a dirty/modified indicator when content has been changed
- Displays the file path in a breadcrumb/header bar above the editor

### 3. Update App.tsx

- When a file is selected from the FileTree and it's not `swarm.yaml` (which goes to the DAG canvas), open it in MonacoFileEditor instead of FileViewer
- Pass the file content and path to MonacoFileEditor
- Handle save events (already have `window.fs.writeFile` IPC)

### 4. Editor Configuration

- Set `minimap.enabled: false` (too small for the panel width)
- Set `wordWrap: 'on'` for markdown files
- Set `fontSize: 13`
- Set `scrollBeyondLastLine: false`
- Set `automaticLayout: true` (resizes with panel)
- Tab size: 2 for YAML/JSON, 4 for Go/Python

## Files to Create/Modify

- `electron/src/renderer/components/MonacoFileEditor.tsx` (NEW) — Monaco-based file editor component
- `electron/src/renderer/components/FileViewer.tsx` (REMOVE or keep as fallback) — The old read-only viewer can be removed or kept for log files that should remain read-only
- `electron/src/renderer/App.tsx` (MODIFY) — Wire up MonacoFileEditor in place of FileViewer for editable files
- `electron/package.json` (MODIFY) — Add `@monaco-editor/react` dependency

## Dependencies

- Phase 1 complete (file tree, file content viewer — done)
- File writing IPC already exists (`window.fs.writeFile`)
- Settings panel does not need to be done first

## Acceptance Criteria

1. `@monaco-editor/react` is installed as a dependency
2. Selecting a `.yaml`, `.md`, `.toml`, or other text file from the FileTree opens Monaco Editor in the center panel
3. The editor uses a dark theme consistent with the app's UI
4. Files can be edited and saved with Cmd+S (Mac) / Ctrl+S (Windows/Linux)
5. A save action writes the file back via `window.fs.writeFile` IPC
6. A dirty indicator (e.g., dot on the file name or "Unsaved changes" label) shows when the buffer differs from the saved version
7. Language detection works correctly (YAML highlighting for .yaml files, Markdown for .md, etc.)
8. Log files (`.log`) open in read-only mode
9. The app builds successfully with `npm run build` from electron/
10. TypeScript compiles without errors (`npx tsc --noEmit`)

## Notes

- ELECTRON_PLAN.md specifies Monaco Editor for the code editor (line 335)
- The `@monaco-editor/react` package lazily loads Monaco, which keeps the initial bundle smaller
- Monaco's `vs-dark` theme is a good default. A custom theme matching the exact Tailwind zinc/slate palette could be a follow-up
- For the YAML editor specifically, ELECTRON_PLAN.md envisions schema validation and autocomplete (lines 283-287). Those are complex features that should be separate follow-up tasks. This task just gets Monaco rendering and editing working.
- The existing `FileViewer.tsx` can be kept around for rendering log files in read-only mode (no editing), or MonacoFileEditor can handle this with `readOnly: true`
- Monaco is fairly heavy (~2MB). Lazy loading via `@monaco-editor/react` mitigates this. The Electron context means bundle size is less of a concern than in web apps.
- Make sure to handle the case where the file content updates externally (e.g., another agent writes to the file). Show a prompt or auto-reload if the file hasn't been modified in the editor.
