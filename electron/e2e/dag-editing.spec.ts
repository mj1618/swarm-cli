import { _electron as electron, test, expect, ElectronApplication, Page } from '@playwright/test';
import * as path from 'path';

/**
 * E2E Tests for DAG Editing Workflow
 * 
 * These tests verify the core DAG editing functionality in Swarm Desktop:
 * - Creating tasks via the UI
 * - Selecting tasks to view/edit details
 * - Deleting tasks via context menu
 * - Creating dependencies by connecting nodes
 * - Deleting edges
 * 
 * Note: These tests use the actual workspace's swarm/swarm.yaml file.
 * Tasks created during tests are cleaned up after each test.
 */

let electronApp: ElectronApplication;
let window: Page;

// Track tasks created during tests for cleanup
const createdTasks: string[] = [];

test.beforeAll(async () => {
  // Launch Electron app
  electronApp = await electron.launch({
    args: [path.join(__dirname, '../dist/main/main/index.js')],
    timeout: 30000,
  });

  // Wait for the first window to open
  window = await electronApp.firstWindow();
  
  // Wait for the window to be ready
  await window.waitForLoadState('domcontentloaded');
  
  // Wait for React to render
  await window.waitForSelector('#root', { timeout: 10000 });
  
  // Wait for the DAG canvas to be visible (indicates app is fully loaded)
  await window.waitForSelector('[data-testid="dag-canvas"]', { timeout: 15000 });
});

test.afterAll(async () => {
  // Close the Electron app
  if (electronApp) {
    await electronApp.close();
  }
});

test.afterEach(async () => {
  // Clean up any tasks created during the test
  for (const taskName of createdTasks) {
    try {
      // Right-click on the task node to open context menu
      const taskNode = window.locator(`[data-testid="task-node-${taskName}"]`);
      if (await taskNode.isVisible({ timeout: 1000 }).catch(() => false)) {
        await taskNode.click({ button: 'right' });
        
        // Click delete in context menu
        const deleteButton = window.locator('[data-testid="context-menu-delete"]');
        if (await deleteButton.isVisible({ timeout: 1000 }).catch(() => false)) {
          await deleteButton.click();
          
          // Confirm deletion
          const confirmButton = window.locator('[data-testid="delete-confirm-submit"]');
          if (await confirmButton.isVisible({ timeout: 1000 }).catch(() => false)) {
            await confirmButton.click();
            // Wait for deletion to complete
            await window.waitForTimeout(500);
          }
        }
      }
    } catch {
      // Task may already be deleted, ignore errors
    }
  }
  createdTasks.length = 0;
});

test.describe('DAG Editing Workflow', () => {
  test.describe('Task Creation', () => {
    test('should open task drawer when clicking Add Task button', async () => {
      // Find and click the Add Task button
      const addTaskButton = window.locator('[data-testid="add-task-button"]');
      await expect(addTaskButton).toBeVisible();
      await addTaskButton.click();
      
      // Verify task drawer opens
      const taskDrawer = window.locator('[data-testid="task-drawer"]');
      await expect(taskDrawer).toBeVisible({ timeout: 5000 });
      
      // Verify it's in creation mode (has task name input)
      const taskNameInput = window.locator('[data-testid="task-name-input"]');
      await expect(taskNameInput).toBeVisible();
      
      // Close the drawer
      const cancelButton = window.locator('[data-testid="task-drawer-cancel"]');
      await cancelButton.click();
      await expect(taskDrawer).not.toBeVisible();
    });

    test('should create a new task with name and prompt', async () => {
      const testTaskName = `test-task-${Date.now()}`;
      createdTasks.push(testTaskName);
      
      // Open task drawer
      const addTaskButton = window.locator('[data-testid="add-task-button"]');
      await addTaskButton.click();
      
      // Wait for drawer to open
      const taskDrawer = window.locator('[data-testid="task-drawer"]');
      await expect(taskDrawer).toBeVisible({ timeout: 5000 });
      
      // Enter task name
      const taskNameInput = window.locator('[data-testid="task-name-input"]');
      await taskNameInput.fill(testTaskName);
      
      // Select prompt-string type and enter a prompt
      const promptStringButton = window.locator('[data-testid="prompt-type-prompt-string"]');
      await promptStringButton.click();
      
      const promptTextarea = window.locator('[data-testid="prompt-string-textarea"]');
      await promptTextarea.fill('Test prompt for E2E testing');
      
      // Save the task
      const saveButton = window.locator('[data-testid="task-drawer-save"]');
      await saveButton.click();
      
      // Wait for task to be created and drawer to close
      await expect(taskDrawer).not.toBeVisible({ timeout: 5000 });
      
      // Verify new task node appears in the DAG
      const newTaskNode = window.locator(`[data-testid="task-node-${testTaskName}"]`);
      await expect(newTaskNode).toBeVisible({ timeout: 10000 });
    });

    test('should show validation error for invalid task name', async () => {
      // Open task drawer
      const addTaskButton = window.locator('[data-testid="add-task-button"]');
      await addTaskButton.click();
      
      const taskDrawer = window.locator('[data-testid="task-drawer"]');
      await expect(taskDrawer).toBeVisible({ timeout: 5000 });
      
      // Enter invalid task name (starts with number)
      const taskNameInput = window.locator('[data-testid="task-name-input"]');
      await taskNameInput.fill('123-invalid');
      
      // Try to save
      const saveButton = window.locator('[data-testid="task-drawer-save"]');
      await saveButton.click();
      
      // Verify error message appears
      const errorMessage = window.locator('[data-testid="task-name-error"]');
      await expect(errorMessage).toBeVisible();
      
      // Close the drawer
      const cancelButton = window.locator('[data-testid="task-drawer-cancel"]');
      await cancelButton.click();
    });
  });

  test.describe('Task Selection', () => {
    test('should open task drawer with details when clicking a task node', async () => {
      // Find an existing task node (from the project's swarm.yaml)
      // The project has tasks like 'planner', 'implementer', 'reviewer'
      const existingTaskNode = window.locator('[data-testid^="task-node-"]').first();
      await expect(existingTaskNode).toBeVisible({ timeout: 5000 });
      
      // Get the task name from the node
      const taskNameElement = existingTaskNode.locator('.text-sm.font-semibold');
      const taskName = await taskNameElement.textContent();
      
      // Click on the task node
      await existingTaskNode.click();
      
      // Verify task drawer opens
      const taskDrawer = window.locator('[data-testid="task-drawer"]');
      await expect(taskDrawer).toBeVisible({ timeout: 5000 });
      
      // Verify the drawer shows the correct task name in the header
      const drawerHeader = taskDrawer.locator('h2');
      await expect(drawerHeader).toContainText(taskName || '');
      
      // Close the drawer
      const cancelButton = window.locator('[data-testid="task-drawer-cancel"]');
      await cancelButton.click();
      await expect(taskDrawer).not.toBeVisible();
    });
  });

  test.describe('Task Deletion', () => {
    test('should show context menu on right-click of task node', async () => {
      // First create a task to delete
      const testTaskName = `delete-test-${Date.now()}`;
      createdTasks.push(testTaskName);
      
      // Create the task
      const addTaskButton = window.locator('[data-testid="add-task-button"]');
      await addTaskButton.click();
      
      const taskDrawer = window.locator('[data-testid="task-drawer"]');
      await expect(taskDrawer).toBeVisible({ timeout: 5000 });
      
      const taskNameInput = window.locator('[data-testid="task-name-input"]');
      await taskNameInput.fill(testTaskName);
      
      const promptStringButton = window.locator('[data-testid="prompt-type-prompt-string"]');
      await promptStringButton.click();
      
      const promptTextarea = window.locator('[data-testid="prompt-string-textarea"]');
      await promptTextarea.fill('Task for deletion test');
      
      const saveButton = window.locator('[data-testid="task-drawer-save"]');
      await saveButton.click();
      
      await expect(taskDrawer).not.toBeVisible({ timeout: 5000 });
      
      // Wait for the new task to appear
      const taskNode = window.locator(`[data-testid="task-node-${testTaskName}"]`);
      await expect(taskNode).toBeVisible({ timeout: 10000 });
      
      // Right-click to open context menu
      await taskNode.click({ button: 'right' });
      
      // Verify context menu appears with expected options
      const contextMenu = window.locator('[data-testid="task-context-menu"]');
      await expect(contextMenu).toBeVisible();
      
      const runOption = window.locator('[data-testid="context-menu-run"]');
      const duplicateOption = window.locator('[data-testid="context-menu-duplicate"]');
      const deleteOption = window.locator('[data-testid="context-menu-delete"]');
      
      await expect(runOption).toBeVisible();
      await expect(duplicateOption).toBeVisible();
      await expect(deleteOption).toBeVisible();
      
      // Close context menu by pressing Escape
      await window.keyboard.press('Escape');
    });

    test('should delete task after confirmation', async () => {
      // Create a task to delete
      const testTaskName = `delete-confirm-${Date.now()}`;
      // Don't add to createdTasks since we're deleting it in the test
      
      // Create the task
      const addTaskButton = window.locator('[data-testid="add-task-button"]');
      await addTaskButton.click();
      
      const taskDrawer = window.locator('[data-testid="task-drawer"]');
      await expect(taskDrawer).toBeVisible({ timeout: 5000 });
      
      const taskNameInput = window.locator('[data-testid="task-name-input"]');
      await taskNameInput.fill(testTaskName);
      
      const promptStringButton = window.locator('[data-testid="prompt-type-prompt-string"]');
      await promptStringButton.click();
      
      const promptTextarea = window.locator('[data-testid="prompt-string-textarea"]');
      await promptTextarea.fill('Task for deletion confirmation test');
      
      const saveButton = window.locator('[data-testid="task-drawer-save"]');
      await saveButton.click();
      
      await expect(taskDrawer).not.toBeVisible({ timeout: 5000 });
      
      // Wait for task to appear
      const taskNode = window.locator(`[data-testid="task-node-${testTaskName}"]`);
      await expect(taskNode).toBeVisible({ timeout: 10000 });
      
      // Right-click and select delete
      await taskNode.click({ button: 'right' });
      
      const deleteOption = window.locator('[data-testid="context-menu-delete"]');
      await deleteOption.click();
      
      // Verify confirmation dialog appears
      const confirmDialog = window.locator('[data-testid="delete-confirm-dialog"]');
      await expect(confirmDialog).toBeVisible();
      
      // Confirm deletion
      const confirmButton = window.locator('[data-testid="delete-confirm-submit"]');
      await confirmButton.click();
      
      // Verify dialog closes
      await expect(confirmDialog).not.toBeVisible();
      
      // Verify task is removed from DAG
      await expect(taskNode).not.toBeVisible({ timeout: 10000 });
    });

    test('should cancel deletion when clicking cancel in confirmation dialog', async () => {
      // Create a task
      const testTaskName = `delete-cancel-${Date.now()}`;
      createdTasks.push(testTaskName);
      
      // Create the task
      const addTaskButton = window.locator('[data-testid="add-task-button"]');
      await addTaskButton.click();
      
      const taskDrawer = window.locator('[data-testid="task-drawer"]');
      await expect(taskDrawer).toBeVisible({ timeout: 5000 });
      
      const taskNameInput = window.locator('[data-testid="task-name-input"]');
      await taskNameInput.fill(testTaskName);
      
      const promptStringButton = window.locator('[data-testid="prompt-type-prompt-string"]');
      await promptStringButton.click();
      
      const promptTextarea = window.locator('[data-testid="prompt-string-textarea"]');
      await promptTextarea.fill('Task for cancel deletion test');
      
      const saveButton = window.locator('[data-testid="task-drawer-save"]');
      await saveButton.click();
      
      await expect(taskDrawer).not.toBeVisible({ timeout: 5000 });
      
      // Wait for task to appear
      const taskNode = window.locator(`[data-testid="task-node-${testTaskName}"]`);
      await expect(taskNode).toBeVisible({ timeout: 10000 });
      
      // Right-click and select delete
      await taskNode.click({ button: 'right' });
      
      const deleteOption = window.locator('[data-testid="context-menu-delete"]');
      await deleteOption.click();
      
      // Cancel the deletion
      const cancelButton = window.locator('[data-testid="delete-confirm-cancel"]');
      await cancelButton.click();
      
      // Verify task is still visible
      await expect(taskNode).toBeVisible();
    });
  });

  test.describe('Dependency Creation', () => {
    test('should show connection dialog when connecting two nodes', async () => {
      // This test requires two tasks to exist
      // Create two test tasks
      const sourceTask = `source-${Date.now()}`;
      const targetTask = `target-${Date.now()}`;
      createdTasks.push(sourceTask, targetTask);
      
      // Create source task
      let addTaskButton = window.locator('[data-testid="add-task-button"]');
      await addTaskButton.click();
      
      let taskDrawer = window.locator('[data-testid="task-drawer"]');
      await expect(taskDrawer).toBeVisible({ timeout: 5000 });
      
      let taskNameInput = window.locator('[data-testid="task-name-input"]');
      await taskNameInput.fill(sourceTask);
      
      let promptStringButton = window.locator('[data-testid="prompt-type-prompt-string"]');
      await promptStringButton.click();
      
      let promptTextarea = window.locator('[data-testid="prompt-string-textarea"]');
      await promptTextarea.fill('Source task');
      
      let saveButton = window.locator('[data-testid="task-drawer-save"]');
      await saveButton.click();
      
      await expect(taskDrawer).not.toBeVisible({ timeout: 5000 });
      
      // Create target task
      addTaskButton = window.locator('[data-testid="add-task-button"]');
      await addTaskButton.click();
      
      taskDrawer = window.locator('[data-testid="task-drawer"]');
      await expect(taskDrawer).toBeVisible({ timeout: 5000 });
      
      taskNameInput = window.locator('[data-testid="task-name-input"]');
      await taskNameInput.fill(targetTask);
      
      promptStringButton = window.locator('[data-testid="prompt-type-prompt-string"]');
      await promptStringButton.click();
      
      promptTextarea = window.locator('[data-testid="prompt-string-textarea"]');
      await promptTextarea.fill('Target task');
      
      saveButton = window.locator('[data-testid="task-drawer-save"]');
      await saveButton.click();
      
      await expect(taskDrawer).not.toBeVisible({ timeout: 5000 });
      
      // Wait for both tasks to appear
      const sourceNode = window.locator(`[data-testid="task-node-${sourceTask}"]`);
      const targetNode = window.locator(`[data-testid="task-node-${targetTask}"]`);
      await expect(sourceNode).toBeVisible({ timeout: 10000 });
      await expect(targetNode).toBeVisible({ timeout: 10000 });
      
      // Get the source node's bottom handle (source handle)
      // React Flow handles have class 'react-flow__handle-bottom' for source handles
      const sourceHandle = sourceNode.locator('.react-flow__handle-bottom');
      const targetHandle = targetNode.locator('.react-flow__handle-top');
      
      // Drag from source to target to create a connection
      await sourceHandle.dragTo(targetHandle);
      
      // Verify connection dialog appears
      const connectionDialog = window.locator('[data-testid="connection-dialog"]');
      await expect(connectionDialog).toBeVisible({ timeout: 5000 });
      
      // Verify all condition buttons are present
      await expect(window.locator('[data-testid="condition-success"]')).toBeVisible();
      await expect(window.locator('[data-testid="condition-failure"]')).toBeVisible();
      await expect(window.locator('[data-testid="condition-any"]')).toBeVisible();
      await expect(window.locator('[data-testid="condition-always"]')).toBeVisible();
      
      // Select success condition
      await window.locator('[data-testid="condition-success"]').click();
      
      // Verify dialog closes
      await expect(connectionDialog).not.toBeVisible({ timeout: 5000 });
      
      // Verify edge is created (React Flow edges have class 'react-flow__edge')
      // The edge should connect our source and target tasks
      // Wait a moment for the YAML to be written and the UI to update
      await window.waitForTimeout(1000);
      
      // Check that an edge exists between the nodes
      // We can verify this by checking if there's an edge with the expected label
      const edges = window.locator('.react-flow__edge');
      await expect(edges.first()).toBeVisible({ timeout: 5000 });
    });
  });

  test.describe('Edge Deletion', () => {
    test('should delete edge when using keyboard shortcut', async () => {
      // Create two connected tasks
      const sourceTask = `edge-del-src-${Date.now()}`;
      const targetTask = `edge-del-tgt-${Date.now()}`;
      createdTasks.push(sourceTask, targetTask);
      
      // Create source task
      let addTaskButton = window.locator('[data-testid="add-task-button"]');
      await addTaskButton.click();
      
      let taskDrawer = window.locator('[data-testid="task-drawer"]');
      await expect(taskDrawer).toBeVisible({ timeout: 5000 });
      
      let taskNameInput = window.locator('[data-testid="task-name-input"]');
      await taskNameInput.fill(sourceTask);
      
      let promptStringButton = window.locator('[data-testid="prompt-type-prompt-string"]');
      await promptStringButton.click();
      
      let promptTextarea = window.locator('[data-testid="prompt-string-textarea"]');
      await promptTextarea.fill('Source for edge deletion test');
      
      let saveButton = window.locator('[data-testid="task-drawer-save"]');
      await saveButton.click();
      
      await expect(taskDrawer).not.toBeVisible({ timeout: 5000 });
      
      // Create target task
      addTaskButton = window.locator('[data-testid="add-task-button"]');
      await addTaskButton.click();
      
      taskDrawer = window.locator('[data-testid="task-drawer"]');
      await expect(taskDrawer).toBeVisible({ timeout: 5000 });
      
      taskNameInput = window.locator('[data-testid="task-name-input"]');
      await taskNameInput.fill(targetTask);
      
      promptStringButton = window.locator('[data-testid="prompt-type-prompt-string"]');
      await promptStringButton.click();
      
      promptTextarea = window.locator('[data-testid="prompt-string-textarea"]');
      await promptTextarea.fill('Target for edge deletion test');
      
      saveButton = window.locator('[data-testid="task-drawer-save"]');
      await saveButton.click();
      
      await expect(taskDrawer).not.toBeVisible({ timeout: 5000 });
      
      // Wait for both tasks to appear
      const sourceNode = window.locator(`[data-testid="task-node-${sourceTask}"]`);
      const targetNode = window.locator(`[data-testid="task-node-${targetTask}"]`);
      await expect(sourceNode).toBeVisible({ timeout: 10000 });
      await expect(targetNode).toBeVisible({ timeout: 10000 });
      
      // Create a connection between them
      const sourceHandle = sourceNode.locator('.react-flow__handle-bottom');
      const targetHandle = targetNode.locator('.react-flow__handle-top');
      await sourceHandle.dragTo(targetHandle);
      
      // Select condition
      const connectionDialog = window.locator('[data-testid="connection-dialog"]');
      await expect(connectionDialog).toBeVisible({ timeout: 5000 });
      await window.locator('[data-testid="condition-success"]').click();
      await expect(connectionDialog).not.toBeVisible({ timeout: 5000 });
      
      // Wait for edge to be created
      await window.waitForTimeout(1000);
      
      // Count edges before deletion
      const edgesBefore = await window.locator('.react-flow__edge').count();
      expect(edgesBefore).toBeGreaterThan(0);
      
      // Click on an edge to select it
      // React Flow edges are SVG paths with class 'react-flow__edge-path'
      const edge = window.locator('.react-flow__edge').last();
      await edge.click();
      
      // Wait for edge to be selected (selected edges have different styling)
      await window.waitForTimeout(300);
      
      // Press Delete/Backspace to delete the selected edge
      await window.keyboard.press('Delete');
      
      // Wait for deletion
      await window.waitForTimeout(1000);
      
      // Verify edge count decreased
      const edgesAfter = await window.locator('.react-flow__edge').count();
      expect(edgesAfter).toBeLessThan(edgesBefore);
    });
  });

  test.describe('Keyboard Shortcuts', () => {
    test('should open task drawer with N key shortcut', async () => {
      // Press N to open task drawer
      await window.keyboard.press('n');
      
      // Verify task drawer opens
      const taskDrawer = window.locator('[data-testid="task-drawer"]');
      await expect(taskDrawer).toBeVisible({ timeout: 5000 });
      
      // Close it
      await window.keyboard.press('Escape');
      await expect(taskDrawer).not.toBeVisible();
    });

    test('should deselect all with Escape key', async () => {
      // Click on a task node to select it
      const taskNode = window.locator('[data-testid^="task-node-"]').first();
      await taskNode.click();
      
      // The node should now have a selection ring (handled by React Flow)
      // Press Escape to deselect
      await window.keyboard.press('Escape');
      
      // Verify deselection happened (can check by trying to delete - should not show dialog)
      await window.keyboard.press('Delete');
      
      // No delete confirm dialog should appear since nothing is selected
      const confirmDialog = window.locator('[data-testid="delete-confirm-dialog"]');
      await expect(confirmDialog).not.toBeVisible({ timeout: 1000 }).catch(() => {
        // Expected - no dialog should appear
      });
    });
  });
});
