#!/usr/bin/env node
import { _electron as electron } from '@playwright/test';
import * as path from 'path';
import { fileURLToPath } from 'url';
import * as readline from 'readline';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

console.log('🚀 Launching Electron app with Playwright...\n');

const electronApp = await electron.launch({
  args: [path.join(__dirname, 'dist/main/main/index.js')],
  timeout: 60000,
});

// Wait for windows to be available
let window = await electronApp.firstWindow();

// Find the main window (not DevTools)
const windows = electronApp.windows();
for (const win of windows) {
  const url = win.url();
  const title = await win.title().catch(() => '');
  if (!title.includes('DevTools') && !url.includes('devtools://')) {
    window = win;
    break;
  }
}

await window.waitForLoadState('domcontentloaded');
// Wait for React to mount
await window.waitForTimeout(2000);

console.log('✅ App launched successfully!\n');
console.log('📍 Window URL:', window.url());
console.log('📍 Window Title:', await window.title());
console.log('\n--- Interactive Playwright REPL ---');
console.log('Available objects: electronApp, window');
console.log('Examples:');
console.log('  await window.screenshot({ path: "screenshot.png" })');
console.log('  await window.locator("button").all()');
console.log('  await window.locator("text=Agents").click()');
console.log('  await window.evaluate(() => document.body.innerHTML)');
console.log('\nType "exit" to quit.\n');

// Simple REPL
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
});

const prompt = () => {
  rl.question('playwright> ', async (input) => {
    if (input.trim() === 'exit' || input.trim() === 'quit') {
      console.log('Closing app...');
      await electronApp.close();
      rl.close();
      process.exit(0);
    }
    
    try {
      const result = await eval(`(async () => { return ${input} })()`);
      if (result !== undefined) {
        console.log(result);
      }
    } catch (err) {
      console.error('Error:', err.message);
    }
    
    prompt();
  });
};

prompt();
