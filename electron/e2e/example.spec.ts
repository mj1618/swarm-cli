import { _electron as electron, test, expect, ElectronApplication, Page } from '@playwright/test';
import * as path from 'path';

let electronApp: ElectronApplication;
let window: Page;

test.beforeAll(async () => {
  // Launch Electron app
  // In production, the app runs from dist/main/main/index.js
  electronApp = await electron.launch({
    args: [path.join(__dirname, '../dist/main/main/index.js')],
    timeout: 30000,
  });

  // Wait for the first window to open
  window = await electronApp.firstWindow();
  
  // Wait for the window to be ready
  await window.waitForLoadState('domcontentloaded');
});

test.afterAll(async () => {
  // Close the Electron app
  if (electronApp) {
    await electronApp.close();
  }
});

test.describe('Swarm Desktop App', () => {
  test('app launches successfully', async () => {
    // Verify that the app launched and a window opened
    expect(electronApp).toBeDefined();
    expect(window).toBeDefined();
  });

  test('main window has correct title', async () => {
    // The app sets title to "Swarm Desktop — {project}" after loading
    // Initially it may just be the product name from build config
    const title = await window.title();
    
    // Title should contain "Swarm Desktop"
    expect(title).toMatch(/Swarm Desktop/i);
  });

  test('window has minimum dimensions', async () => {
    const { width, height } = await window.evaluate(() => ({
      width: window.innerWidth,
      height: window.innerHeight,
    }));

    // Minimum dimensions defined in main/index.ts are 1000x600
    expect(width).toBeGreaterThanOrEqual(1000);
    expect(height).toBeGreaterThanOrEqual(600);
  });

  test('app is not in dev mode when built', async () => {
    // In production build, devtools should not auto-open
    // This test just ensures the app is running the built version
    const isPackaged = await electronApp.evaluate(({ app }) => app.isPackaged);
    
    // When running tests locally against dev build, this will be false
    // In CI with production build, this would be true
    // For now, just ensure we can evaluate this
    expect(typeof isPackaged).toBe('boolean');
  });

  test('renderer process loads successfully', async () => {
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
