# E2E Tests for Monaco File Editor

## Goal

Add Playwright E2E tests for the Monaco File Editor component, which opens when users click on files in the file tree. The editor is a core Phase 5 feature (Monaco editor integration) that currently has no E2E test coverage. It handles file viewing, editing, saving, dirty state tracking, and markdown preview.

## Files

- **Modify**: `electron/e2e/app.spec.ts` - Add new test describe block for File Editor
- **Modify**: `electron/src/renderer/components/MonacoFileEditor.tsx` - Add data-testid attributes for E2E selectors

## Dependencies

- Requires: E2E test setup (complete)
- Requires: MonacoFileEditor component (complete)
- Requires: FileTree component with file click handling (complete)

## Acceptance Criteria

1. Add `data-testid` attributes to MonacoFileEditor.tsx:
   - `data-testid="file-editor"` on the outer container div
   - `data-testid="file-editor-header"` on the header with filename
   - `data-testid="file-editor-save-button"` on the Save button
   - `data-testid="file-editor-preview-button"` on the Preview button (for markdown)
   - `data-testid="file-editor-dirty-indicator"` on the unsaved changes dot
   - `data-testid="file-editor-filetype-badge"` on the file type badge (YAML, Markdown, etc.)

2. New `test.describe('Swarm Desktop - File Editor')` block with tests:
   - **Opens when file clicked**: Create test file, click on it in file tree, verify editor opens with correct filename displayed
   - **Shows correct file type badge**: Open a .yaml file, verify "YAML" badge; open a .md file, verify "Markdown" badge
   - **Displays file content**: Open a file with known content, verify content is displayed in editor
   - **Save button disabled when no changes**: Open file, verify Save button is disabled
   - **Dirty indicator appears on edit**: Open file, type in editor, verify unsaved indicator (dot) appears
   - **Save button enabled after edit**: Open file, type in editor, verify Save button becomes enabled
   - **Save persists changes**: Edit file, click Save, reload file, verify changes persisted
   - **Keyboard save (Cmd+S)**: Edit file, press Cmd+S, verify changes saved
   - **Markdown preview toggle**: Open .md file, click Preview button, verify preview panel appears
   - **Read-only for .log files**: Open a .log file, verify "Read-only" badge and editing is disabled

3. Test workspace setup:
   - Create a test workspace with sample files:
     - `swarm/swarm.yaml` - sample YAML content
     - `swarm/prompts/test.md` - sample markdown content
     - `swarm/test.log` - sample log content (if .log files exist)

4. All tests pass consistently with `npx playwright test`

## Notes

From `MonacoFileEditor.tsx`:
- File types detected by extension: yaml, yml, md, toml, json, ts, tsx, js, jsx, go, log
- `.log` files are read-only (`isReadOnly` function)
- Markdown files show a "Preview" toggle button
- Prompt files (`/prompts/*.md`) show "Resolve Includes" button in preview
- Unsaved changes show as orange dot after filename
- Save button disabled when `!isDirty || saving`
- Cmd+S keyboard shortcut registered via Monaco `editor.addCommand`

### Key DOM Structure

```
<div> <!-- container -->
  <div> <!-- header -->
    <span>{fileType.label}</span> <!-- badge: YAML, Markdown, etc. -->
    <span>{fileName}<span>&bull;</span></span> <!-- dirty indicator -->
    <button>Preview</button> <!-- markdown only -->
    <button>Save</button> <!-- or "Read-only" span -->
  </div>
  <Editor /> <!-- Monaco editor -->
  <div> <!-- preview panel, when visible -->
</div>
```

### Test Strategy

```typescript
test.describe('Swarm Desktop - File Editor', () => {
  test.describe.configure({ mode: 'serial' });

  test('opens file from file tree click', async () => {
    // Create test file in workspace
    // Click on file in file tree
    // Verify editor opens with correct filename
    const editor = window.locator('[data-testid="file-editor"]');
    await expect(editor).toBeVisible();
  });

  test('shows unsaved indicator after editing', async () => {
    // Open a file
    // Type in Monaco editor
    const dirtyIndicator = window.locator('[data-testid="file-editor-dirty-indicator"]');
    await expect(dirtyIndicator).toBeVisible();
  });

  test('saves changes with Cmd+S', async () => {
    // Edit file content
    await window.keyboard.press('Meta+s');
    // Verify dirty indicator disappears
    // Reload and verify content persisted
  });
});
```

### Monaco Editor Interaction

To type in Monaco editor:
```typescript
// Click to focus the editor
await window.locator('.monaco-editor .view-line').first().click();
// Type content
await window.keyboard.type('new content');
```

Or use Monaco's internal model:
```typescript
await window.evaluate(() => {
  const editor = (window as any).monaco?.editor?.getModels()[0];
  editor?.setValue('new content');
});
```
