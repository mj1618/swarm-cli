# Task: Workspace Initialization Wizard

**Phase:** 5+ (Polish / Onboarding Enhancement)
**Priority:** Medium

## Goal

Add a workspace initialization feature that helps users set up a new swarm project when opening a directory that lacks a `swarm/` folder. Instead of just showing a warning toast, provide an actionable empty state with an "Initialize Swarm Project" button that creates the required directory structure.

This improves the first-run experience and onboarding for new users who want to start using Swarm Desktop with a fresh project.

## Files to Create/Modify

### Modify
- `electron/src/renderer/App.tsx` — When `defaultYamlError` indicates missing swarm directory, render a new `InitializeWorkspace` component instead of (or alongside) the DAG canvas
- `electron/src/main/index.ts` — Add IPC handler `workspace:init` that creates the swarm directory structure
- `electron/src/preload/index.ts` — Expose `workspace.init()` via context bridge

### Create
- `electron/src/renderer/components/InitializeWorkspace.tsx` — New component with:
  - Centered layout with icon/illustration
  - Heading: "No swarm project found"
  - Description explaining what a swarm project is
  - "Initialize Swarm Project" primary button
  - Optional: checkbox to create example prompt file

## Dependencies

- File system IPC handlers (already exist)
- File tree component refreshes on file changes (already implemented)

## Implementation Notes

### InitializeWorkspace Component

Display when `defaultYamlError` is set and indicates no swarm.yaml (or no swarm directory):

```tsx
// InitializeWorkspace.tsx
export default function InitializeWorkspace({ onInitialize, projectPath }: Props) {
  return (
    <div className="flex-1 flex items-center justify-center">
      <div className="text-center max-w-md space-y-4">
        {/* Icon */}
        <div className="mx-auto w-16 h-16 rounded-full bg-secondary/50 flex items-center justify-center">
          <FolderIcon className="w-8 h-8 text-muted-foreground" />
        </div>
        
        {/* Heading */}
        <h2 className="text-lg font-semibold">No swarm project found</h2>
        
        {/* Description */}
        <p className="text-sm text-muted-foreground">
          This directory doesn't have a swarm/ folder. Initialize a new swarm project 
          to start creating AI agent pipelines.
        </p>
        
        {/* Initialize button */}
        <button
          onClick={onInitialize}
          className="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
        >
          Initialize Swarm Project
        </button>
        
        {/* What gets created */}
        <div className="text-xs text-muted-foreground mt-4">
          <p>This will create:</p>
          <ul className="mt-1 space-y-1">
            <li>• swarm/swarm.yaml — Pipeline configuration</li>
            <li>• swarm/prompts/ — Directory for prompt files</li>
            <li>• swarm/swarm.toml — Optional CLI settings</li>
          </ul>
        </div>
      </div>
    </div>
  )
}
```

### IPC Handler (main/index.ts)

```typescript
ipcMain.handle('workspace:init', async () => {
  const cwd = process.cwd()
  const swarmDir = path.join(cwd, 'swarm')
  const promptsDir = path.join(swarmDir, 'prompts')
  
  try {
    // Create directories
    await fs.mkdir(swarmDir, { recursive: true })
    await fs.mkdir(promptsDir, { recursive: true })
    
    // Create swarm.yaml with example content
    const exampleYaml = `version: "1"

tasks:
  example:
    prompt: example
    model: sonnet

pipelines:
  main:
    iterations: 1
    tasks: [example]
`
    await fs.writeFile(path.join(swarmDir, 'swarm.yaml'), exampleYaml)
    
    // Create example prompt
    const examplePrompt = `# Example Prompt

This is an example prompt file. Edit this to define what your AI agent should do.

Your agent will receive this prompt and execute the instructions.
`
    await fs.writeFile(path.join(promptsDir, 'example.md'), examplePrompt)
    
    // Optionally create swarm.toml
    const exampleToml = `# Swarm CLI configuration
# See documentation for available options

backend = "claude-code"
model = "sonnet"
`
    await fs.writeFile(path.join(swarmDir, 'swarm.toml'), exampleToml)
    
    return { success: true }
  } catch (err) {
    return { error: String(err) }
  }
})
```

### Preload (preload/index.ts)

Add to the workspace API:
```typescript
contextBridge.exposeInMainWorld('workspace', {
  // ... existing methods ...
  init: () => ipcRenderer.invoke('workspace:init'),
})
```

### App.tsx Integration

When rendering the center panel, check for initialization state:

```tsx
// In the center panel render logic
{settingsOpen ? (
  <SettingsPanel ... />
) : defaultYamlError && !selectedFile ? (
  <InitializeWorkspace 
    onInitialize={handleInitializeWorkspace}
    projectPath={projectPath}
  />
) : selectedIsOutputRun ? (
  ...
```

Add handler:
```typescript
const handleInitializeWorkspace = useCallback(async () => {
  const result = await window.workspace.init()
  if (result.error) {
    addToast('error', `Failed to initialize: ${result.error}`)
    return
  }
  addToast('success', 'Swarm project initialized!')
  // Reload the default yaml
  const reloaded = await window.fs.readfile('swarm/swarm.yaml')
  if (!reloaded.error) {
    setDefaultYamlContent(reloaded.content)
    setDefaultYamlError(null)
  }
}, [addToast])
```

## Acceptance Criteria

1. Opening a directory without a `swarm/` folder shows the InitializeWorkspace component instead of an empty/error DAG canvas
2. The component displays a clear explanation of what a swarm project is
3. Clicking "Initialize Swarm Project" creates:
   - `swarm/swarm.yaml` with a valid example pipeline
   - `swarm/prompts/` directory
   - `swarm/prompts/example.md` with example content
   - `swarm/swarm.toml` with basic configuration
4. After initialization:
   - Success toast is shown
   - The DAG canvas loads with the example task visible
   - The file tree shows the new swarm/ directory structure
5. Error handling: If initialization fails (permissions, etc.), show error toast
6. The component uses existing app styling (dark theme, shadcn patterns)
7. App builds successfully with `npm run build`

## Notes

- The example swarm.yaml should be valid and runnable with `swarm up`
- Keep the example simple - one task, one pipeline, minimal configuration
- The swarm.toml is optional but helpful for users who will use the CLI
- Consider adding a "Learn more" link to documentation (if docs URL exists)
- This feature specifically targets the case where `defaultYamlError` indicates a missing file/directory, not YAML parse errors

---

## Completion Notes

**Status:** Completed  
**Completed by:** Agent 9b5f224e (iteration 7)  
**Date:** 2026-02-13

### What was implemented:

1. **InitializeWorkspace.tsx** - New component with:
   - Centered layout with folder icon (inline SVG)
   - "No swarm project found" heading
   - Clear description of what a swarm project is
   - "Initialize Swarm Project" primary button with loading state
   - List of files/directories that will be created

2. **workspace:init IPC handler** in `main/index.ts`:
   - Creates `swarm/` directory
   - Creates `swarm/prompts/` directory
   - Creates `swarm/swarm.yaml` with example pipeline
   - Creates `swarm/prompts/example.md` with example prompt
   - Creates `swarm/swarm.toml` with basic configuration

3. **Preload exposure** - Added `workspace.init()` to the workspace API

4. **App.tsx integration**:
   - Imports InitializeWorkspace component
   - Shows component when `defaultYamlError && !selectedFile`
   - Handler reloads swarm.yaml after initialization and clears error state

### Verification:
- Build passes: `npm run build` completes successfully
- All acceptance criteria met
