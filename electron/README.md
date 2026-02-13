# Swarm Desktop

Desktop GUI for swarm-cli — a visual interface for DAG creation, pipeline management, and AI agent monitoring.

## Overview

Swarm Desktop is an Electron application that provides a graphical interface for [swarm-cli](https://github.com/example/swarm-cli). It allows you to:

- Visually design and edit task dependency graphs (DAGs)
- Monitor running AI agents in real-time
- Manage pipelines with pause/resume/stop controls
- Browse and edit workspace files (prompts, YAML configs)
- View logs and track costs across agents

The app communicates with swarm-cli by spawning CLI commands and watching state files for real-time updates.

## Prerequisites

- **Node.js** 18+ (tested with v22)
- **npm** 8+
- **swarm-cli** installed and available in your PATH
- **macOS**, **Windows**, or **Linux**

## Installation

```bash
cd electron
npm install
```

## Development

The app uses Vite for the renderer process and TypeScript for both main and renderer.

```bash
# Start the Vite dev server (renderer process)
npm run dev

# In another terminal, build and start Electron (connects to Vite dev server)
npm run electron:dev
```

For a single command that builds everything and launches the app:

```bash
npm run start
```

### Development Scripts

| Command | Description |
|---------|-------------|
| `npm run dev` | Start Vite dev server for hot-reload development |
| `npm run electron:dev` | Build main process and launch Electron |
| `npm run start` | Full build + launch (production mode) |

## Building

```bash
# Build renderer (Vite) and main process (TypeScript)
npm run build

# Build only the Electron main process
npm run build:electron
```

The build outputs to:
- `dist/renderer/` — React app bundle
- `dist/main/` — Electron main process

## Packaging

Create distributable packages for your platform:

```bash
npm run package
```

This runs `electron-builder` and outputs to the `release/` directory:
- **macOS**: `.dmg` file
- **Windows**: NSIS installer
- **Linux**: AppImage

## Testing

### Unit Tests

Unit tests use Vitest with React Testing Library:

```bash
# Run all unit tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with UI
npm run test:ui
```

### End-to-End Tests

E2E tests use Playwright to test the full Electron application:

```bash
# Run E2E tests
npm run test:e2e

# Run E2E tests with UI mode
npm run test:e2e:ui
```

### Type Checking & Linting

```bash
# Check TypeScript types
npm run typecheck

# Run ESLint
npm run lint
```

## Project Structure

```
electron/
├── src/
│   ├── main/                 # Electron main process
│   │   ├── index.ts          # App entry, window management, IPC handlers
│   │   └── preload.ts        # Preload script (bridge main ↔ renderer)
│   ├── renderer/             # React frontend
│   │   ├── App.tsx           # Main app component with 3-panel layout
│   │   ├── main.tsx          # React entry point
│   │   ├── index.html        # HTML template
│   │   ├── index.css         # Tailwind styles
│   │   ├── components/       # React components
│   │   │   ├── DagCanvas.tsx       # Visual DAG editor (React Flow)
│   │   │   ├── FileTree.tsx        # Workspace file browser
│   │   │   ├── AgentPanel.tsx      # Running/history agents sidebar
│   │   │   ├── ConsolePanel.tsx    # Log viewer with tabs
│   │   │   ├── TaskDrawer.tsx      # Task configuration side panel
│   │   │   ├── CommandPalette.tsx  # Cmd+K quick actions
│   │   │   ├── SettingsPanel.tsx   # App settings dialog
│   │   │   └── ...                 # 25+ more components
│   │   └── lib/              # Utility libraries
│   │       ├── yamlParser.ts       # Parse swarm.yaml files
│   │       ├── yamlWriter.ts       # Write changes to YAML
│   │       ├── yamlIntellisense.ts # Autocomplete for YAML editor
│   │       ├── dagValidation.ts    # Validate DAG for cycles/errors
│   │       └── ...
│   └── test/                 # Test setup and utilities
├── e2e/                      # Playwright E2E tests
│   └── app.spec.ts           # Core app functionality tests
├── build/                    # App icons and resources
├── dist/                     # Build output
├── package.json
├── vite.config.ts
├── playwright.config.ts
├── vitest.config.ts
├── tailwind.config.js
└── tsconfig*.json
```

## Key Features

### Visual DAG Editor
- Drag-and-drop task creation
- Visual dependency connections with condition labels (success/failure/any/always)
- Real-time execution status overlay
- Minimap navigation
- Export DAG as image
- Keyboard shortcuts for common actions

### File Tree Browser
- Browse `swarm/` directory with file type icons
- Quick-create buttons for prompts and tasks
- Context menu: rename, delete, duplicate
- Drag files onto DAG to create tasks
- Search/filter within the tree

### Agent Management Panel
- Real-time status from `state.json`
- Progress indicators with iteration counts
- Token usage and cost tracking
- Pause/resume/stop controls
- Agent detail view with full metrics

### Console & Logs
- Tabbed log viewer per agent
- Color-coded output by agent
- Search/filter within logs
- Auto-scroll toggle
- Export logs to file

### Additional Features
- **Command Palette** (Cmd+K): Quick actions for common tasks
- **Monaco Editor**: Syntax highlighting for YAML and Markdown files
- **YAML IntelliSense**: Autocomplete for task names and prompts
- **Dark/Light Theme**: Toggle via settings or system preference
- **Keyboard Shortcuts**: Full keyboard navigation support
- **Notifications**: Toast and system notifications for agent events
- **Recent Projects**: Quick access to recently opened workspaces

## IPC API

The preload script exposes these methods to the renderer via `window.swarm`:

| Method | Description |
|--------|-------------|
| `list()` | List all agents |
| `run(args)` | Start a new agent |
| `kill(agentId)` | Terminate an agent |
| `pause(agentId)` | Pause an agent |
| `resume(agentId)` | Resume a paused agent |
| `logs(agentId)` | Get agent logs |
| `inspect(agentId)` | Get agent details |
| `readFile(path)` | Read file contents |
| `writeFile(path, content)` | Write file contents |
| `watchState()` | Subscribe to state.json changes |

## Tech Stack

| Component | Technology |
|-----------|------------|
| Framework | Electron 34 + React 19 |
| Language | TypeScript 5 |
| Bundler | Vite 6 |
| Styling | Tailwind CSS 3 |
| DAG Visualization | @xyflow/react (React Flow) |
| Code Editor | Monaco Editor |
| File Watching | chokidar |
| YAML Parsing | js-yaml |
| Testing | Vitest + Playwright |

## License

MIT
