---
id: E-0001
title: "Electron Desktop Application"
type: epic
status: todo
priority: P2
owner: null
labels:
  - epic
  - frontend
  - desktop
depends_on: []
children: []
acceptance:
  - "Desktop application runs on macOS, Windows, and Linux"
  - "Users can manage agent lifecycle (run, pause, resume, kill, clone) via GUI"
  - "Real-time agent output streaming displayed in the UI"
  - "Compose file editor with syntax highlighting and validation"
  - "Visual DAG/pipeline view showing task dependencies and execution status"
  - "State persistence and session management integrated with existing ~/swarm/ paths"
created_at: 2026-02-16
updated_at: 2026-02-16
---

## Context

swarm-cli is currently a command-line tool for orchestrating AI agents. While the CLI is powerful for automation and scripting, a desktop GUI would significantly improve the user experience for interactive workflows, real-time monitoring, and visual pipeline management.

An Electron-based desktop application would provide:
- Cross-platform support (macOS, Windows, Linux) from a single codebase
- Rich UI for agent management without memorizing CLI commands
- Visual representation of DAG pipelines and task dependencies
- Real-time streaming of agent output with better formatting
- Integrated compose file editing with validation feedback

## Scope

1. **Core Application Shell** - Electron app setup with React/TypeScript frontend
   - Main process for system integration and Go CLI spawning
   - Renderer process with modern React UI
   - IPC communication between main and renderer

2. **Agent Management UI** - Visual interface for agent lifecycle
   - Dashboard showing all running/paused agents
   - Run dialog with prompt input, model selection, iteration config
   - Agent detail view with live output streaming
   - Actions: pause, resume, kill, clone, replay

3. **Compose File Editor** - YAML editor for multi-task orchestration
   - Monaco editor with YAML syntax highlighting
   - Real-time validation against compose schema
   - Visual preview of task graph

4. **DAG Visualization** - Interactive pipeline view
   - Node graph showing tasks and dependencies
   - Execution status coloring (pending, running, success, failure)
   - Click-through to task details and logs

5. **State & Logs Viewer** - Browse agent state and log files
   - Tree view of ~/swarm/logs/ directory
   - Log viewer with search and filtering
   - State inspector showing agent metadata

6. **Settings & Configuration** - App preferences and swarm config
   - Backend configuration (claude-code, cursor)
   - Model defaults and API settings
   - Theme and appearance options

## Out of scope

- Mobile applications (iOS, Android) - desktop-first approach
- Web-hosted version - this is a local desktop tool
- Direct API integration bypassing the Go CLI - we wrap the existing CLI
- Plugin/extension system - may be addressed in future epic
- Collaborative/multi-user features - single-user desktop app

## Technical Approach

The Electron app will wrap the existing Go CLI rather than reimplementing agent logic:

```
┌─────────────────────────────────────────┐
│           Electron Main Process          │
│  - Spawn swarm CLI commands              │
│  - Watch ~/swarm/state.json              │
│  - Stream log files                      │
├─────────────────────────────────────────┤
│          Electron Renderer (React)       │
│  - Agent dashboard                       │
│  - Compose editor                        │
│  - DAG visualization                     │
│  - Settings UI                           │
└─────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────┐
│            swarm CLI (Go binary)         │
│  - run, list, inspect, logs, kill, etc. │
└─────────────────────────────────────────┘
```

This approach:
- Leverages all existing CLI functionality
- Keeps the CLI as the source of truth for agent management
- Allows CLI and GUI to be used interchangeably
- Simplifies maintenance - GUI is purely a frontend

## Notes

Potential technology choices:
- **Electron** - Cross-platform desktop runtime
- **React + TypeScript** - UI framework
- **Vite** - Build tooling
- **TailwindCSS** - Styling
- **Monaco Editor** - Code editing (compose files)
- **React Flow** or **Cytoscape** - DAG visualization
- **xterm.js** - Terminal emulation for agent output
