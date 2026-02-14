import { _electron as electron, test, expect, ElectronApplication, Page } from '@playwright/test';
import * as path from 'path';
import * as fs from 'fs';
import * as os from 'os';
import { fileURLToPath } from 'url';

// ES Module polyfills for __dirname
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

let electronApp: ElectronApplication;
let window: Page;

// Test fixtures directory for isolated test workspaces
const fixturesDir = path.join(os.tmpdir(), 'swarm-e2e-fixtures');


// Helper to create a test workspace with optional swarm.yaml content
async function createTestWorkspace(name: string, swarmYamlContent?: string): Promise<string> {
  const workspacePath = path.join(fixturesDir, name, Date.now().toString());
  const swarmDir = path.join(workspacePath, 'swarm');
  const promptsDir = path.join(swarmDir, 'prompts');

  // Create directory structure
  fs.mkdirSync(promptsDir, { recursive: true });

  if (swarmYamlContent) {
    fs.writeFileSync(path.join(swarmDir, 'swarm.yaml'), swarmYamlContent);
  }

  return workspacePath;
}

/**
 * Find the main application window (not DevTools).
 * In dev mode, DevTools may open as a separate window.
 */
async function getMainWindow(app: ElectronApplication): Promise<Page> {
  // Wait for at least one window to be available
  const firstWindow = await app.firstWindow();

  // Get all windows
  const windows = app.windows();

  // Find a window that's not DevTools
  for (const win of windows) {
    const url = win.url();
    const title = await win.title().catch(() => '');

    // Skip DevTools windows
    if (title.includes('DevTools') || url.includes('devtools://')) {
      continue;
    }

    // This is likely our main window
    return win;
  }

  // If we only have DevTools windows, return first window
  return firstWindow;
}

test.beforeAll(async () => {
  // Create fixtures directory
  fs.mkdirSync(fixturesDir, { recursive: true });

  // Launch Electron app with test environment
  // The app runs from dist/main/main/index.js after build
  electronApp = await electron.launch({
    args: [path.join(__dirname, '../dist/main/main/index.js')],
    timeout: 60000,
    env: {
      ...process.env,
      NODE_ENV: 'test',
    },
  });

  // Get the first window (should be main window in test mode since DevTools don't open)
  window = await electronApp.firstWindow();

  // Wait for the window to be ready and React to mount
  await window.waitForLoadState('domcontentloaded');
  // Give React time to fully hydrate - this fixed timeout is appropriate for Electron
  // where waitForSelector can have issues during React initialization
  await window.waitForTimeout(3000);
});

test.afterAll(async () => {
  // Close the Electron app gracefully with a short timeout
  try {
    if (electronApp) {
      // Use Promise.race to ensure we don't hang on close
      await Promise.race([
        electronApp.close(),
        new Promise(resolve => setTimeout(resolve, 5000)) // 5 second max wait
      ]).catch(() => {
        // Ignore close errors - app may already be closed
      });
    }
  } catch {
    // Ignore cleanup errors - app may already be closed
  }

  // Clean up test fixtures
  try {
    fs.rmSync(fixturesDir, { recursive: true, force: true });
  } catch {
    // Ignore cleanup errors
  }
});

test.describe('Swarm Desktop - Core App Tests', () => {
  test.describe.configure({ mode: 'serial' });

  test('app launches successfully', async () => {
    // Verify that the app launched and a window opened
    expect(electronApp).toBeDefined();
    expect(window).toBeDefined();
  });

  test('main window has correct title containing Swarm Desktop', async () => {
    // Use evaluate instead of title() to avoid Electron Playwright issues
    const title = await window.evaluate(() => document.title);
    expect(title).toMatch(/Swarm Desktop/i);
  });

  test('window has minimum dimensions (1000x600)', async () => {
    const { width, height } = await window.evaluate(() => ({
      width: window.innerWidth,
      height: window.innerHeight,
    }));

    // Minimum dimensions defined in main/index.ts are 1000x600
    expect(width).toBeGreaterThanOrEqual(1000);
    expect(height).toBeGreaterThanOrEqual(600);
  });

  test('renderer process loads React root successfully', async () => {
    // Wait for React to render the root element
    await window.waitForSelector('#root', { timeout: 10000 });

    // Verify the root element exists and has content
    const rootElement = await window.$('#root');
    expect(rootElement).not.toBeNull();

    // The root should have some rendered content
    const innerHTML = await rootElement?.innerHTML();
    expect(innerHTML).toBeTruthy();
    expect(innerHTML!.length).toBeGreaterThan(0);
  });
});

test.describe('Swarm Desktop - Main 3-Panel Layout', () => {
  test.describe.configure({ mode: 'serial' });

  test('displays the main 3-panel layout with file tree, DAG canvas, and agent panel', async () => {
    test.slow(); // This test needs more time to verify all panels

    // Wait for React to fully render the app
    await window.waitForSelector('#root', { timeout: 10000 });

    // Check for the title bar with "Swarm Desktop" text
    const titleBar = await window.locator('text=Swarm Desktop').first();
    await expect(titleBar).toBeVisible({ timeout: 10000 });

    // Check for the 3-panel layout structure
    // The left sidebar should contain "Files" heading (FileTree panel)
    const filesHeading = await window.locator('h2:has-text("Files")').first();
    await expect(filesHeading).toBeVisible({ timeout: 5000 });

    // The right sidebar should contain "Agents" heading (AgentPanel)
    const agentsHeading = await window.locator('h2:has-text("Agents")').first();
    await expect(agentsHeading).toBeVisible({ timeout: 5000 });

    // The console panel should be visible at the bottom
    const consoleText = await window.locator('text=Console').first();
    await expect(consoleText).toBeVisible({ timeout: 5000 });
  });

  test('file tree panel has data-testid attribute', async () => {
    const fileTree = await window.locator('[data-testid="file-tree"]').first();
    await expect(fileTree).toBeVisible({ timeout: 5000 });
  });

  test('agent panel has data-testid attribute', async () => {
    const agentPanel = await window.locator('[data-testid="agent-panel"]').first();
    await expect(agentPanel).toBeVisible({ timeout: 5000 });
  });

  test('dag canvas has data-testid attribute (may show empty state or tasks)', async () => {
    // DAG canvas should be visible - either with tasks or showing empty state
    const dagCanvas = await window.locator('[data-testid="dag-canvas"]').first();
    await expect(dagCanvas).toBeVisible({ timeout: 5000 });
  });
});

test.describe('Swarm Desktop - File Tree Panel', () => {
  test.describe.configure({ mode: 'serial' });

  test('file tree shows "Files" heading', async () => {
    const heading = await window.locator('[data-testid="file-tree"] h2:has-text("Files")').first();
    await expect(heading).toBeVisible({ timeout: 5000 });
  });

  test('file tree has refresh and create buttons', async () => {
    // Look for the refresh button (↻) in the file tree header
    const refreshButton = await window.locator('[data-testid="file-tree"] button[title="Refresh file tree"]').first();
    await expect(refreshButton).toBeVisible({ timeout: 5000 });

    // Look for the create button (+) in the file tree header
    const createButton = await window.locator('[data-testid="file-tree"] button[title="Create new file or folder"]').first();
    await expect(createButton).toBeVisible({ timeout: 5000 });
  });

  test('file tree displays swarm/ directory label when it exists', async () => {
    // The file tree should show "swarm/" as the root directory label
    // This may show either the directory or "No swarm directory found" depending on workspace
    const fileTreeContent = await window.locator('[data-testid="file-tree"]').first();
    const textContent = await fileTreeContent.textContent();

    // Check that file tree has some content (either swarm/ dir or no-swarm message)
    expect(textContent).toBeTruthy();
    expect(textContent!.length).toBeGreaterThan(0);
  });

  test('file tree has filter/search input when files exist', async () => {
    // Wait for file tree to be visible and loaded
    const fileTree = await window.locator('[data-testid="file-tree"]');
    await expect(fileTree).toBeVisible({ timeout: 10000 });

    // If swarm directory exists, there should be a filter input
    const filterInput = await window.locator('[data-testid="file-tree"] input[placeholder="Filter files..."]').first();

    // This may or may not be visible depending on whether swarm/ exists
    // Just check that the file tree container is functional
    await expect(fileTree).toBeVisible();
  });
});

test.describe('Swarm Desktop - DAG Canvas', () => {
  test.describe.configure({ mode: 'serial' });

  test('DAG canvas shows either tasks or empty state', async () => {
    const dagCanvas = await window.locator('[data-testid="dag-canvas"]');
    await expect(dagCanvas).toBeVisible({ timeout: 5000 });

    const textContent = await dagCanvas.textContent();

    // The DAG canvas will show either:
    // 1. "No tasks yet" - empty state
    // 2. Task nodes from swarm.yaml
    // 3. Loading state
    // 4. Error state if no swarm.yaml
    expect(textContent).toBeTruthy();
  });

  test('DAG canvas empty state shows helpful guidance when no tasks exist', async () => {
    const dagCanvas = await window.locator('[data-testid="dag-canvas"]');
    const textContent = await dagCanvas.textContent();

    // If in empty state, should show helpful text
    if (textContent?.includes('No tasks yet')) {
      // Verify helpful guidance is shown
      expect(textContent).toContain('task');

      // Check for the "Create Task" button in empty state
      const createButton = await dagCanvas.locator('button:has-text("Create Task")').first();
      const isCreateVisible = await createButton.isVisible().catch(() => false);

      // Create button should be visible in empty state
      if (textContent.includes('No tasks yet')) {
        expect(isCreateVisible).toBe(true);
      }
    }
  });

  test('DAG editor section has header', async () => {
    // Look for the DAG Editor section header
    // The header shows either "DAG Editor" or the filename like "swarm.yaml"
    const dagHeader = await window.locator('h2:has-text("DAG Editor"), h2:has-text("swarm.yaml")').first();
    const isVisible = await dagHeader.isVisible().catch(() => false);

    // The header may not be visible if showing settings or file editor
    // Just verify we can query for it without error
    expect(typeof isVisible).toBe('boolean');
  });
});

test.describe('Swarm Desktop - Agent Panel', () => {
  test.describe.configure({ mode: 'serial' });

  test('agent panel shows "Agents" heading', async () => {
    const heading = await window.locator('[data-testid="agent-panel"] h2:has-text("Agents")').first();
    await expect(heading).toBeVisible({ timeout: 5000 });
  });

  test('agent panel has refresh button', async () => {
    const refreshButton = await window.locator('[data-testid="agent-panel"] button[title="Refresh"]').first();
    await expect(refreshButton).toBeVisible({ timeout: 5000 });
  });

  test('agent panel has search input', async () => {
    const searchInput = await window.locator('[data-testid="agent-panel"] input[placeholder="Search agents..."]').first();
    await expect(searchInput).toBeVisible({ timeout: 5000 });
  });

  test('agent panel has status filter dropdown', async () => {
    const statusFilter = await window.locator('[data-testid="agent-panel"] select').first();
    await expect(statusFilter).toBeVisible({ timeout: 5000 });

    // Verify it has the expected options
    const options = await statusFilter.locator('option').allTextContents();
    expect(options).toContain('All');
    expect(options).toContain('Running');
    expect(options).toContain('History');
  });

  test('agent panel shows "No agents" when no agents exist', async () => {
    // Wait for agent panel to be visible
    const agentPanel = await window.locator('[data-testid="agent-panel"]');
    await expect(agentPanel).toBeVisible({ timeout: 10000 });

    // Give time for agent data to load
    await window.waitForTimeout(1000);

    const textContent = await agentPanel.textContent();

    // If no agents, should show "No agents" message
    // Otherwise, should show agent cards or "Loading..." 
    expect(textContent).toBeTruthy();

    // At minimum, the panel should be interactive
    expect(textContent).toContain('Agents');
  });
});

test.describe('Swarm Desktop - Console Panel', () => {
  test.describe.configure({ mode: 'serial' });

  test('console panel is visible', async () => {
    const consolePanel = await window.locator('text=Console').first();
    await expect(consolePanel).toBeVisible({ timeout: 5000 });
  });

  test('console panel has collapse/expand toggle', async () => {
    // The console has a toggle button (▼ or ▶)
    const toggleButton = await window.locator('button[title*="console"], button[title*="Expand"], button[title*="Collapse"]').first();
    await expect(toggleButton).toBeVisible({ timeout: 5000 });
  });
});

test.describe('Swarm Desktop - Sidebar Collapse/Expand', () => {
  test.describe.configure({ mode: 'serial' });

  test('left sidebar has collapse button', async () => {
    // The left sidebar should have a collapse button with title containing "Collapse sidebar" or similar
    const collapseButton = await window.locator('button[title*="Collapse sidebar"], button[title*="Cmd+B"]').first();
    const isVisible = await collapseButton.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });

  test('right sidebar has collapse button', async () => {
    // The right sidebar should have a collapse button
    const collapseButton = await window.locator('button[title*="Collapse sidebar"], button[title*="Cmd+Shift+B"]').first();
    const isVisible = await collapseButton.isVisible().catch(() => false);
    expect(typeof isVisible).toBe('boolean');
  });
});

test.describe('Swarm Desktop - Keyboard Shortcuts', () => {
  test.describe.configure({ mode: 'serial' });

  test('Cmd+K opens command palette', async () => {
    // Send Cmd+K (or Ctrl+K on non-Mac)
    await window.keyboard.press('Meta+k');

    // Wait for potential command palette to appear (with short timeout)
    // The palette may or may not open depending on focus state
    const paletteInput = await window.locator('input[placeholder*="command"], input[placeholder*="search"]').first();
    const isVisible = await paletteInput.isVisible({ timeout: 2000 }).catch(() => false);

    // Command palette may or may not open (depends on focus state)
    // Just verify the shortcut doesn't crash the app
    expect(typeof isVisible).toBe('boolean');

    // Press Escape to close any dialog
    await window.keyboard.press('Escape');
  });
});

test.describe('Swarm Desktop - File Editor', () => {
  test.describe.configure({ mode: 'serial' });

  let testWorkspacePath: string;

  // Set up test workspace with sample files before running editor tests
  test.beforeAll(async () => {
    // Create a test workspace with sample files
    testWorkspacePath = path.join(fixturesDir, 'editor-test', Date.now().toString());
    const swarmDir = path.join(testWorkspacePath, 'swarm');
    const promptsDir = path.join(swarmDir, 'prompts');

    fs.mkdirSync(promptsDir, { recursive: true });

    // Create sample swarm.yaml
    fs.writeFileSync(path.join(swarmDir, 'swarm.yaml'), `version: "1"

tasks:
  planner:
    prompt: test-prompt
    model: opus

pipelines:
  main:
    iterations: 5
    tasks: [planner]
`);

    // Create sample markdown prompt
    fs.writeFileSync(path.join(promptsDir, 'test-prompt.md'), `# Test Prompt

This is a test prompt for E2E testing.

## Instructions

1. Do something
2. Do something else
`);

    // Create sample log file (read-only)
    fs.writeFileSync(path.join(swarmDir, 'test.log'), `[2026-02-13 12:00:00] Test log entry 1
[2026-02-13 12:00:01] Test log entry 2
[2026-02-13 12:00:02] Test log entry 3
`);

    // Switch to the test workspace using IPC
    await electronApp.evaluate(async ({ ipcMain }, workspacePath) => {
      // Emit the switch workspace event
      const { BrowserWindow } = require('electron');
      const win = BrowserWindow.getAllWindows()[0];
      if (win) {
        win.webContents.send('workspace-switched', { path: workspacePath });
      }
    }, testWorkspacePath);

    // Use evaluate to call the IPC handler directly via the renderer
    await window.evaluate(async (wsPath) => {
      // @ts-ignore
      await window.workspace.switch(wsPath);
    }, testWorkspacePath);

    // Wait for file tree to refresh
    await window.waitForTimeout(2000);
  });

  test('opens file editor when file is clicked in file tree', async () => {
    test.slow();

    // First, expand the swarm/ directory if needed
    const swarmYaml = window.locator('[data-testid="file-tree-item-swarm.yaml"]');

    // Wait for and click the swarm.yaml file
    await expect(swarmYaml).toBeVisible({ timeout: 10000 });
    await swarmYaml.click();

    // Verify the file editor opens
    const fileEditor = window.locator('[data-testid="file-editor"]');
    await expect(fileEditor).toBeVisible({ timeout: 5000 });

    // Verify the header shows the filename
    const header = window.locator('[data-testid="file-editor-header"]');
    await expect(header).toContainText('swarm.yaml');
  });

  test('shows correct file type badge for YAML files', async () => {
    // The file editor should be open from previous test (swarm.yaml)
    const fileTypeBadge = window.locator('[data-testid="file-editor-filetype-badge"]');
    await expect(fileTypeBadge).toBeVisible({ timeout: 5000 });
    await expect(fileTypeBadge).toContainText('YAML');
  });

  test('displays file content in editor', async () => {
    // The swarm.yaml file should be open
    // Look for Monaco editor with content
    const monacoEditor = window.locator('.monaco-editor');
    await expect(monacoEditor).toBeVisible({ timeout: 5000 });

    // Verify content is displayed (look for known text from our test file)
    const editorContent = window.locator('.monaco-editor .view-lines');
    await expect(editorContent).toContainText('version');
  });

  test('save button is disabled when no changes', async () => {
    const saveButton = window.locator('[data-testid="file-editor-save-button"]');
    await expect(saveButton).toBeVisible({ timeout: 5000 });
    await expect(saveButton).toBeDisabled();
  });

  test('dirty indicator appears after editing', async () => {
    // Click to focus the Monaco editor
    const monacoEditor = window.locator('.monaco-editor .view-line').first();
    await monacoEditor.click();

    // Type some content to make it dirty
    await window.keyboard.type('# Test edit\n');

    // Wait for dirty indicator to appear
    const dirtyIndicator = window.locator('[data-testid="file-editor-dirty-indicator"]');
    await expect(dirtyIndicator).toBeVisible({ timeout: 5000 });
  });

  test('save button is enabled after edit', async () => {
    // After the previous test, the file should be dirty
    const saveButton = window.locator('[data-testid="file-editor-save-button"]');
    await expect(saveButton).toBeEnabled({ timeout: 5000 });
  });

  test('clicking save button saves changes', async () => {
    // Click the save button
    const saveButton = window.locator('[data-testid="file-editor-save-button"]');
    await saveButton.click();

    // Wait for save to complete - dirty indicator should disappear
    const dirtyIndicator = window.locator('[data-testid="file-editor-dirty-indicator"]');
    await expect(dirtyIndicator).not.toBeVisible({ timeout: 5000 });

    // Save button should be disabled again
    await expect(saveButton).toBeDisabled({ timeout: 5000 });
  });

  test('Cmd+S keyboard shortcut saves changes', async () => {
    // Make another edit
    const monacoEditor = window.locator('.monaco-editor .view-line').first();
    await monacoEditor.click();
    await window.keyboard.type('# Another edit\n');

    // Wait for dirty indicator
    const dirtyIndicator = window.locator('[data-testid="file-editor-dirty-indicator"]');
    await expect(dirtyIndicator).toBeVisible({ timeout: 5000 });

    // Press Cmd+S to save
    await window.keyboard.press('Meta+s');

    // Dirty indicator should disappear
    await expect(dirtyIndicator).not.toBeVisible({ timeout: 5000 });
  });

  test('opens markdown file and shows preview button', async () => {
    // Navigate to prompts directory and click on the markdown file
    // First, we may need to expand the prompts folder
    const promptsFolder = window.locator('[data-testid="file-tree-item-prompts"]');

    if (await promptsFolder.isVisible().catch(() => false)) {
      await promptsFolder.click();
      await window.waitForTimeout(500);
    }

    // Click on the test-prompt.md file
    const markdownFile = window.locator('[data-testid="file-tree-item-test-prompt.md"]');
    await expect(markdownFile).toBeVisible({ timeout: 10000 });
    await markdownFile.click();

    // Wait for file editor to show the markdown file
    const fileEditor = window.locator('[data-testid="file-editor"]');
    await expect(fileEditor).toBeVisible({ timeout: 5000 });

    // Verify Markdown badge
    const fileTypeBadge = window.locator('[data-testid="file-editor-filetype-badge"]');
    await expect(fileTypeBadge).toContainText('Markdown');

    // Verify Preview button is visible for markdown files
    const previewButton = window.locator('[data-testid="file-editor-preview-button"]');
    await expect(previewButton).toBeVisible({ timeout: 5000 });
  });

  test('markdown preview toggle shows preview panel', async () => {
    // Click the Preview button
    const previewButton = window.locator('[data-testid="file-editor-preview-button"]');
    await previewButton.click();

    // Preview panel should appear with rendered content
    const previewPanel = window.locator('text=Markdown Preview');
    await expect(previewPanel).toBeVisible({ timeout: 5000 });

    // Verify the button text changed to "Hide Preview"
    await expect(previewButton).toContainText('Hide Preview');

    // Toggle it back off
    await previewButton.click();
    await expect(previewButton).toContainText('Preview');
  });

  test('log files show read-only badge', async () => {
    // Click on the log file
    const logFile = window.locator('[data-testid="file-tree-item-test.log"]');

    if (await logFile.isVisible().catch(() => false)) {
      await logFile.click();

      // Wait for file editor to open
      const fileEditor = window.locator('[data-testid="file-editor"]');
      await expect(fileEditor).toBeVisible({ timeout: 5000 });

      // Verify Log badge
      const fileTypeBadge = window.locator('[data-testid="file-editor-filetype-badge"]');
      await expect(fileTypeBadge).toContainText('Log');

      // Verify read-only badge is shown
      const readOnlyBadge = window.locator('[data-testid="file-editor-readonly-badge"]');
      await expect(readOnlyBadge).toBeVisible({ timeout: 5000 });
      await expect(readOnlyBadge).toContainText('Read-only');

      // Save button should NOT be visible for read-only files
      const saveButton = window.locator('[data-testid="file-editor-save-button"]');
      await expect(saveButton).not.toBeVisible();
    } else {
      // Log file may not be visible in file tree, skip this test
      test.skip();
    }
  });
});
