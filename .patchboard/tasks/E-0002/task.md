---
id: E-0002
title: "Electron Desktop Application for Swarm CLI"
type: epic
status: todo
priority: P2
owner: null
labels:
  - epic
  - electron
  - desktop
  - ui
depends_on: []
children: []
acceptance:
  - "Desktop app launches and connects to local swarm-cli backend"
  - "Users can run, pause, resume, kill, clone, and replay agents via GUI"
  - "Agent state and logs are displayed in real-time"
  - "Compose files can be viewed, edited, and executed"
  - "DAG/pipeline visualization shows task dependencies and execution status"
  - "Configuration can be managed through settings UI"
  - "App works on macOS, Windows, and Linux"
created_at: '2026-02-16'
updated_at: '2026-02-16'
---

## Context

swarm-cli is currently a command-line tool for orchestrating AI agents. While powerful, CLI-only interaction creates friction for users who prefer visual interfaces, especially when monitoring multiple agents, viewing complex DAG pipelines, or managing compose files.

An Electron desktop application would provide:
- Visual agent management with real-time status updates
- Graphical DAG/pipeline visualization
- Integrated log viewing with search and filtering
- Compose file editor with syntax highlighting and validation
- Cross-platform support (macOS, Windows, Linux)

This epic covers the full implementation of a desktop GUI that wraps and extends swarm-cli functionality.

## Scope

### Phase 1: Foundation
1. **Project scaffolding** - Electron + React/Vue setup, build configuration, packaging
2. **IPC architecture** - Communication layer between Electron main process and swarm-cli Go backend
3. **Backend integration** - Spawn and manage swarm-cli process, expose API for GUI

### Phase 2: Core Agent Management
4. **Agent list view** - Display all agents with status, model, iterations, cost
5. **Agent detail view** - Full agent info, controls (pause/resume/kill), output stream
6. **Run agent dialog** - Form to configure and launch new agents
7. **Clone/replay functionality** - Clone existing agents or replay from state

### Phase 3: Compose & DAG
8. **Compose file browser** - List, view, and select compose files
9. **Compose editor** - YAML editor with syntax highlighting and schema validation
10. **DAG visualization** - Interactive graph showing task dependencies and execution flow
11. **Pipeline execution** - Run compose pipelines with visual progress tracking

### Phase 4: Logs & Monitoring
12. **Log viewer** - Real-time log streaming with ANSI color support
13. **Log search/filter** - Search within logs, filter by level/timestamp
14. **Cost dashboard** - Aggregate token usage and cost across agents

### Phase 5: Configuration & Polish
15. **Settings UI** - Manage global and project config (backends, models, paths)
16. **Theme support** - Light/dark mode, customizable colors
17. **Keyboard shortcuts** - Power-user navigation and commands
18. **Auto-updates** - Electron auto-updater for seamless updates

## Out of scope

- Mobile applications (iOS/Android) - may be addressed in future epic
- Web-hosted version - this epic focuses on desktop Electron app
- Multi-user collaboration features - single-user local app for now
- Cloud agent management - focus is on local swarm-cli orchestration
- Plugin/extension system - may be considered in future iteration

## Technical Considerations

### Architecture Options
- **Option A**: Electron spawns swarm-cli as child process, communicates via stdout/stdin or socket
- **Option B**: Build swarm-cli as a library, create Go HTTP server, Electron calls REST API
- **Option C**: Embed Go in Electron using wasm or native bindings

Recommended: Option B (HTTP API) for clean separation and testability.

### Technology Stack (suggested)
- **Electron** - Desktop runtime
- **React** or **Vue 3** - UI framework
- **TypeScript** - Type safety
- **Tailwind CSS** - Styling
- **Zustand** or **Pinia** - State management
- **Monaco Editor** - YAML/code editing
- **D3.js** or **ReactFlow** - DAG visualization
- **electron-builder** - Cross-platform packaging

### Key Files to Interface With
- `internal/state/` - Agent state persistence (`~/swarm/state.json`)
- `internal/config/` - Configuration loading
- `internal/compose/` - Compose file parsing
- `internal/dag/` - DAG execution
- `internal/logparser/` - Token/cost parsing
- `~/swarm/logs/` - Agent log files

## Notes

<!-- Usage guidance:
- Children field: List all task IDs that belong to this epic
- Each child task should have parent_epic: E-0002 pointing back
- Epic status is set manually (not computed from children)
- Progress (X of Y done) is calculated at display time from child statuses
- Use depends_on for epic-level dependencies (blocking other epics/tasks)
-->
