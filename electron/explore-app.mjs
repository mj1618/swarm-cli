#!/usr/bin/env node
import { _electron as electron } from '@playwright/test';
import * as path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

console.log('🚀 Launching Electron app with Playwright...\n');

const electronApp = await electron.launch({
  args: [path.join(__dirname, 'dist/main/main/index.js')],
  timeout: 60000,
  env: {
    ...process.env,
    NODE_ENV: 'test', // Prevents DevTools from opening
  },
});

// Get the main window
const window = await electronApp.firstWindow();
await window.waitForLoadState('domcontentloaded');
await window.waitForTimeout(3000); // Wait for React to fully mount

console.log('✅ App launched successfully!\n');
console.log('📍 Window URL:', window.url());
console.log('📍 Window Title:', await window.title());

// Take a screenshot
await window.screenshot({ path: 'app-screenshot.png' });
console.log('\n📸 Screenshot saved to app-screenshot.png\n');

// Get window dimensions
const dimensions = await window.evaluate(() => ({
  width: window.innerWidth,
  height: window.innerHeight,
}));
console.log('📐 Window dimensions:', dimensions.width, 'x', dimensions.height);

// List all visible text elements
console.log('\n--- Visible UI Elements ---\n');

// Check main panels
const panels = [
  { name: 'File Tree', selector: '[data-testid="file-tree"]' },
  { name: 'DAG Canvas', selector: '[data-testid="dag-canvas"]' },
  { name: 'Agent Panel', selector: '[data-testid="agent-panel"]' },
];

for (const panel of panels) {
  const isVisible = await window.locator(panel.selector).isVisible().catch(() => false);
  console.log(`${isVisible ? '✅' : '❌'} ${panel.name}: ${isVisible ? 'visible' : 'not found'}`);
}

// Get all headings
console.log('\n--- Headings ---');
const headings = await window.locator('h1, h2, h3').allTextContents();
headings.forEach(h => console.log('  •', h.trim()));

// Get all buttons
console.log('\n--- Buttons ---');
const buttons = await window.locator('button').all();
for (const btn of buttons.slice(0, 15)) { // Limit to first 15
  const text = await btn.textContent().catch(() => '');
  const title = await btn.getAttribute('title').catch(() => '');
  if (text?.trim() || title) {
    console.log('  •', text?.trim() || `[title: ${title}]`);
  }
}

// Get all inputs
console.log('\n--- Inputs ---');
const inputs = await window.locator('input, select').all();
for (const input of inputs) {
  const placeholder = await input.getAttribute('placeholder').catch(() => '');
  const type = await input.getAttribute('type').catch(() => 'text');
  const tagName = await input.evaluate(el => el.tagName.toLowerCase());
  if (placeholder) {
    console.log(`  • ${tagName} [${type}]: "${placeholder}"`);
  } else {
    console.log(`  • ${tagName} [${type}]`);
  }
}

// Interact with the app - click on different elements
console.log('\n--- Interactive Exploration ---\n');

// Try to find task nodes
const taskNodes = await window.locator('[data-testid="task-node"]').count();
console.log(`Found ${taskNodes} task node(s) in DAG canvas`);

// Check console panel
const consoleVisible = await window.locator('text=Console').isVisible().catch(() => false);
console.log(`Console panel visible: ${consoleVisible}`);

// Check for any agents
const agentCards = await window.locator('[data-testid^="agent-card"]').count();
console.log(`Found ${agentCards} agent card(s)`);

// Print DOM structure summary
console.log('\n--- DOM Structure (top level) ---');
const rootHTML = await window.evaluate(() => {
  const root = document.getElementById('root');
  if (!root) return 'No #root element found';
  
  const summarize = (el, depth = 0) => {
    if (depth > 2) return '';
    const indent = '  '.repeat(depth);
    const tag = el.tagName.toLowerCase();
    const id = el.id ? `#${el.id}` : '';
    const classes = el.className ? `.${el.className.toString().split(' ').slice(0, 2).join('.')}` : '';
    const testId = el.getAttribute('data-testid') ? `[data-testid="${el.getAttribute('data-testid')}"]` : '';
    let result = `${indent}<${tag}${id}${testId}>\n`;
    
    for (const child of el.children) {
      if (child.tagName) {
        result += summarize(child, depth + 1);
      }
    }
    return result;
  };
  
  return summarize(root);
});
console.log(rootHTML);

console.log('\n--- Commands you can run interactively ---');
console.log('');
console.log('  node -e "import(\'@playwright/test\').then(async ({_electron}) => {');
console.log('    const app = await _electron.launch({args:[\'dist/main/main/index.js\'],env:{...process.env,NODE_ENV:\'test\'}});');
console.log('    const w = await app.firstWindow();');
console.log('    await w.waitForTimeout(3000);');
console.log('    // Your commands here, e.g.:');
console.log('    // await w.click(\'text=Create Task\');');
console.log('    // await w.screenshot({path:\'test.png\'});');
console.log('    await app.close();');
console.log('  })"');

console.log('\nClosing app...');
await electronApp.close();
console.log('Done!');
