# Task: Markdown Preview Toggle for Prompt Files

**Phase:** 5 - Polish
**Priority:** Medium

## Goal

Add a markdown preview toggle when editing `.md` files (especially prompt files) in the Monaco editor. This allows users to see how their prompts will render and makes it easier to write and review markdown-formatted prompts.

## Files to Modify

- **`electron/src/renderer/components/MonacoFileEditor.tsx`**
  - Add a "Preview" toggle button in the editor toolbar (next to save button)
  - Add state for preview mode (`showPreview`)
  - When preview is enabled, show a split view or toggle view with rendered markdown
  - Use a lightweight markdown renderer (e.g., `marked` or `react-markdown`)

- **`electron/package.json`**
  - Add `marked` or `react-markdown` dependency for markdown rendering

## Dependencies

- Monaco File Editor component (completed)
- No other task dependencies

## Implementation Notes

### UI Layout Options

**Option A: Split View (Recommended)**
```
┌─────────────────────────────────────────────────┐
│ prompt.md                        [Preview] [Save] │
├────────────────────┬────────────────────────────┤
│                    │                            │
│   Monaco Editor    │   Rendered Preview         │
│   (source)         │   (HTML)                   │
│                    │                            │
└────────────────────┴────────────────────────────┘
```

**Option B: Toggle View**
- Button toggles between editor and preview
- Simpler but less convenient for editing

### Preview Toggle Button

```tsx
<button
  onClick={() => setShowPreview(prev => !prev)}
  className={`px-2 py-1 text-xs rounded ${
    showPreview 
      ? 'bg-primary text-primary-foreground' 
      : 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
  }`}
  title={showPreview ? 'Hide preview' : 'Show preview'}
>
  {showPreview ? 'Hide Preview' : 'Preview'}
</button>
```

### Markdown Rendering

Using `marked` (lightweight, fast):
```tsx
import { marked } from 'marked'

// In component:
const renderedHtml = useMemo(() => {
  if (!content || !showPreview) return ''
  return marked(content)
}, [content, showPreview])

// In JSX:
<div 
  className="prose prose-sm dark:prose-invert p-4 overflow-y-auto"
  dangerouslySetInnerHTML={{ __html: renderedHtml }}
/>
```

### Styling Considerations

- Use Tailwind's `prose` class for nice markdown typography
- Add `dark:prose-invert` for dark mode support
- The preview pane should scroll independently
- Match the background color to the editor theme

### Only Show for Markdown Files

```tsx
const isMarkdown = filePath.endsWith('.md')

// Only render preview button for markdown files
{isMarkdown && (
  <button onClick={() => setShowPreview(p => !p)}>
    Preview
  </button>
)}
```

## Acceptance Criteria

1. When editing a `.md` file, a "Preview" button appears in the editor toolbar
2. Clicking "Preview" shows a split view with rendered markdown on the right
3. Clicking again hides the preview and returns to full-width editor
4. Preview updates in real-time as the user types
5. Preview respects dark/light theme
6. Preview scrolls independently from the editor
7. Button does NOT appear for non-markdown files (.yaml, .ts, etc.)
8. App builds successfully with `npm run build`

## Notes

- This implements the "Markdown editor with preview" feature from ELECTRON_PLAN.md Panel 1 File Type Handling
- The `marked` library is recommended for its small size and speed
- Consider adding `DOMPurify` to sanitize HTML output if security is a concern
- Future enhancement: highlight `{{include:path}}` and `{{variable}}` syntax in the preview

## Completion Notes

**Completed by:** Agent 5b32cb4a
**Date:** 2026-02-13

### Implementation Summary

1. **Added `marked` library** for markdown rendering (`electron/package.json`)
2. **Added `@tailwindcss/typography` plugin** for prose styling (`tailwind.config.js`)
3. **Updated `MonacoFileEditor.tsx`:**
   - Added `isMarkdownFile()` helper function
   - Added `isMarkdown` check for all `.md` files
   - Added `renderedMarkdown` useMemo for memoized markdown rendering
   - Preview button now shows for ALL markdown files (not just prompt files)
   - Button text toggles between "Preview" and "Hide Preview"
   - Split view shows rendered markdown with Tailwind prose styling
   - Dark mode support with `prose-invert` classes
   - Real-time preview updates as user types
   - For prompt files: "Resolve Includes" button optionally resolves `{{include:}}` directives
   - Preview pane scrolls independently

### All Acceptance Criteria Met

- [x] Preview button appears for `.md` files
- [x] Split view shows rendered markdown on right
- [x] Click again hides preview (full-width editor)
- [x] Real-time updates as user types
- [x] Dark theme support via prose-invert
- [x] Independent scrolling for preview pane
- [x] Button hidden for non-markdown files
- [x] Build succeeds with `npm run build`
