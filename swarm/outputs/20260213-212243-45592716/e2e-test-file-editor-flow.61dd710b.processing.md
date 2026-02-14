# E2E Tests for File Editor Flow

## Goal

Add Playwright E2E tests for the file viewing and editing workflow. The File Tree and Monaco Editor are core features that allow users to browse, view, and edit prompt files and configuration. Currently there are no E2E tests covering this critical user flow.

## Files

- **Modify**: `electron/e2e/app.spec.ts` - Add new test describe block for File Editor flow

## Dependencies

- Requires: E2E test setup (already complete)
- Requires: FileTree component (`electron/src/renderer/components/FileTree.tsx`)
- Requires: MonacoFileEditor component (`electron/src/renderer/components/MonacoFileEditor.tsx`)
- Requires: FileViewer component (`electron/src/renderer/components/FileViewer.tsx`)

## Acceptance Criteria

1. New `test.describe('Swarm Desktop - File Editor Flow')` block in `electron/e2e/app.spec.ts` with:
   - **Click file in tree**: Click on a file in the file tree, verify Monaco editor opens with file content
   - **Editor shows correct content**: Verify the editor displays the file path in header and content matches
   - **Close file**: Click close button or press Escape, verify editor closes and returns to DAG view
   - **Edit and save file**: Modify content in editor, click Save button, verify success toast appears
   - **Unsaved changes indicator**: Make changes without saving, verify dirty indicator (*) appears in header
   - **Cancel with unsaved changes**: Make changes, try to close, verify confirmation dialog appears

2. Tests add necessary `data-testid` attributes:
   - Add `data-testid="file-editor-container"` to MonacoFileEditor wrapper
   - Add `data-testid="file-editor-header"` to the header showing file path
   - Add `data-testid="file-editor-save-button"` to save button
   - Add `data-testid="file-editor-close-button"` to close button
   - Add `data-testid="file-editor-dirty-indicator"` to unsaved changes indicator

3. Test workspace setup:
   - Create a temp workspace with a sample prompt file in `swarm/prompts/`
   - Tests should not modify real project files

4. All new tests pass consistently with `npx playwright test`

## Notes

From `MonacoFileEditor.tsx`:
- Monaco editor is used for `.yaml`, `.md`, and other text files
- Save is triggered via Save button or Cmd+S keyboard shortcut
- Dirty state tracked and shown with asterisk (*) in title
- Close confirmation dialog when there are unsaved changes

From `FileTree.tsx`:
- Files are displayed in a tree structure
- Clicking a file triggers `onFileSelect` callback
- File icons vary by file type (.yaml, .md, folders)

From `FileViewer.tsx`:
- Wrapper component that decides which viewer to use based on file type
- Routes to MonacoFileEditor for text files
- Routes to OutputRunViewer for output folders

### Test Data Setup

```typescript
// Create test workspace with sample files
const testWorkspace = await createTestWorkspace('file-editor-test');
const promptPath = path.join(testWorkspace, 'swarm/prompts/test-prompt.md');
fs.writeFileSync(promptPath, '# Test Prompt\n\nThis is a test prompt.');
```

### Key DOM Elements

- File tree item: `[data-testid="file-tree-item"]` with file name text
- Monaco editor container: `.monaco-editor`
- Save button: Look for button with "Save" text or Cmd+S tooltip
- Close button: Look for X button or close icon in editor header

## Priority

Medium - This adds E2E coverage for a critical user workflow (file editing) that's currently untested.
