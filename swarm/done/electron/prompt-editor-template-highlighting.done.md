# Task: Add Prompt Editor Template Variable Highlighting & Preview

**Phase:** 5 - Polish
**Priority:** Medium

## Goal

Enhance the Monaco-based file editor to provide syntax highlighting for swarm template directives (`{{include:path}}`) and template variables (`{{task_id}}`, `{{iteration}}`, `{{output_dir}}`) when editing `.md` prompt files in the `swarm/prompts/` directory. Add a preview toggle that shows the resolved prompt with includes expanded inline.

This implements the "Prompt Editor" feature from ELECTRON_PLAN.md:
- Syntax highlighting for `{{include:path}}` directives
- Preview of resolved prompt with includes expanded
- Variable highlighting: `{{task_id}}`, `{{iteration}}`, `{{output_dir}}`

## Files to Modify

- `electron/src/renderer/components/MonacoFileEditor.tsx` — Add Monaco decorations for template variables, add preview toggle button, and preview panel
- `electron/src/main/index.ts` — Add IPC handler `prompt:resolve` that reads a prompt file and recursively expands `{{include:path}}` directives (reading the referenced files from the swarm/prompts directory)
- `electron/src/preload/index.ts` — Expose `prompt:resolve` via context bridge

## Dependencies

- Monaco editor integration (completed, reviewed, approved)
- File tree component (completed)

## Implementation Notes

### Monaco Decorations (MonacoFileEditor.tsx)

When the file being edited is a `.md` file inside `swarm/prompts/`:
1. After the editor mounts, scan content for `{{...}}` patterns using a regex
2. Apply Monaco `deltaDecorations` to highlight:
   - `{{include:...}}` — distinct color (e.g., blue/cyan underlined) to indicate it's a file reference
   - `{{task_id}}`, `{{iteration}}`, `{{output_dir}}`, `{{agent_id}}` — different color (e.g., purple/orange) for runtime variables
3. Re-apply decorations on content change (use `onDidChangeModelContent`)

### Preview Toggle

- Add a small "Preview" toggle button in the editor header bar (top-right, next to the dirty indicator)
- When toggled ON:
  - Call `window.prompt.resolve(filePath)` to get the fully expanded prompt text
  - Show the resolved text in a read-only panel below or beside the editor (split view or overlay)
  - The preview should be styled as a read-only Monaco editor or a simple `<pre>` block with the same dark theme
- When toggled OFF:
  - Hide the preview panel, return to normal editing

### IPC Handler (main/index.ts)

Add a `prompt:resolve` handler that:
1. Reads the specified prompt file
2. Finds all `{{include:path}}` directives
3. For each, reads the referenced file (relative to the swarm/ directory or the prompt file's directory)
4. Replaces the directive with the file contents
5. Returns the fully resolved string
6. Handles missing includes gracefully (replace with `[ERROR: file not found: path]`)

### Preload (preload/index.ts)

Add to the window API:
```typescript
contextBridge.exposeInMainWorld('prompt', {
  resolve: (filePath: string) => ipcRenderer.invoke('prompt:resolve', filePath),
})
```

### Styling

- Decoration colors should be visible against the vs-dark theme
- Use `monaco.editor.IModelDeltaDecoration` with `inlineClassName` for custom CSS classes
- Preview panel uses the same dark background as the rest of the app

## Acceptance Criteria

1. Opening a `.md` file from `swarm/prompts/` in the Monaco editor shows `{{include:...}}` directives with a distinct highlight color (e.g., blue underline or background)
2. Template variables like `{{task_id}}`, `{{iteration}}`, `{{output_dir}}` are highlighted in a different color (e.g., orange/purple)
3. Decorations update automatically when the file content changes
4. A "Preview" button appears in the editor toolbar when editing prompt files
5. Clicking "Preview" shows the resolved prompt with all `{{include:...}}` directives expanded inline
6. Missing includes show an error placeholder in the preview (not a crash)
7. Files outside `swarm/prompts/` do NOT show template decorations or preview button
8. App builds successfully with `npm run build`

## Completion Notes

Implemented by agent d18538bf. All acceptance criteria met:

- **MonacoFileEditor.tsx**: Added `isPromptFile()` detection for `.md` files under `/prompts/`. Injected CSS decoration styles for two classes: `template-include-decoration` (cyan, underlined, italic) for `{{include:...}}` directives, and `template-variable-decoration` (purple with subtle background) for runtime variables. Decorations are computed via `computeDecorations()` and applied on editor mount + on every content change via `onDidChangeModelContent`. Added a "Preview" toggle button (visible only for prompt files) that opens a side-by-side split view showing the fully resolved prompt text. Preview auto-refreshes on save and has a manual "Refresh" button.
- **main/index.ts**: Added `prompt:resolve` IPC handler with a recursive `resolveIncludes()` function that expands `{{include:path}}` directives. Resolves paths relative to the prompt file's directory first, then falls back to swarm root. Handles missing files with `[ERROR: file not found: path]` placeholders and detects circular includes.
- **preload/index.ts**: Exposed the prompt resolver as `window.promptResolver` (renamed from `prompt` to avoid collision with the built-in `window.prompt()` function). Added `PromptAPI` type definition and updated the Window interface.
- **vite-env.d.ts**: Removed stale duplicate Window interface declarations that were conflicting with the preload's type definitions.
- Build passes with `npm run build`.
