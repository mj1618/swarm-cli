# E2E Tests for File Tree Operations

## Goal

Add Playwright E2E tests for File Tree operations in Swarm Desktop. The File Tree is a critical component for managing prompts and configuration files, but currently lacks dedicated E2E test coverage. This task adds tests for file/folder creation, renaming, deletion, and drag-to-DAG functionality.

## Files

- **Create**: `electron/e2e/file-tree.spec.ts` - New E2E test file for File Tree operations

## Dependencies

- Requires: E2E test setup (complete - `electron/playwright.config.ts` exists)
- Requires: Built app (`npm run build && npm run build:electron`)
- Reference: `electron/e2e/dag-editing.spec.ts` for test patterns

## Acceptance Criteria

1. Test file `electron/e2e/file-tree.spec.ts` exists with the following test cases:

   **File Operations:**
   - Click "+" button to create a new file, verify file appears in tree
   - Click "+" button to create a new folder, verify folder appears in tree
   - Right-click a file, select rename, verify file is renamed
   - Right-click a file, select delete, confirm deletion, verify file is removed
   - Search/filter files using the filter input

   **Drag and Drop:**
   - Drag a prompt file from File Tree to DAG canvas, verify task is created

   **File Viewing:**
   - Click on a `.md` file, verify content viewer opens
   - Click on `swarm.yaml`, verify YAML editor opens

2. Tests use proper patterns:
   - Reuse test workspace setup pattern from `app.spec.ts`
   - Use `data-testid` selectors where available
   - Create test files in a temp workspace to avoid polluting real projects
   - Clean up created files after tests

3. All tests pass when run with `npx playwright test electron/e2e/file-tree.spec.ts`

## Notes

### Key Components

From `FileTree.tsx`:
- `data-testid="file-tree"` - Main container
- Create button: `button[title="Create new file or folder"]`
- Refresh button: `button[title="Refresh file tree"]`
- Filter input: `input[placeholder="Filter files..."]`

From `FileTreeItem.tsx`:
- Tree items represent files/folders
- Right-click context menu for rename/delete

From `ContextMenu.tsx`:
- Context menu with Rename, Delete options

### Test Data Setup

1. Create a temporary workspace with `swarm/` directory structure:
   ```
   temp-workspace/
   └── swarm/
       ├── swarm.yaml
       └── prompts/
           └── test-prompt.md
   ```

2. Use unique file names with timestamps to avoid conflicts
3. Clean up test files in afterEach/afterAll hooks

### Key DOM Selectors to Add

If not already present, these `data-testid` attributes may need to be added:
- `file-tree-item-{filename}` - Individual tree items
- `file-tree-create-file` - Create file option in menu
- `file-tree-create-folder` - Create folder option in menu
- `context-menu-rename` - Rename option
- `context-menu-delete` - Delete option
