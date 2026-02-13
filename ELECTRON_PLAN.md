# Swarm Desktop - Electron App Design

An Electron-based GUI for swarm-cli that simplifies DAG creation, pipeline management, and agent monitoring.

## Overall Architecture

The app uses **Electron** with a React frontend (Tailwind CSS + shadcn/ui). The backend communicates with swarm-cli through:
1. Direct spawning of `swarm` CLI commands
2. Watching `~/.swarm/state.json` for real-time state updates
3. Parsing `swarm.yaml` files in the workspace

---

## Main Layout (3-Panel Design)

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Swarm Desktop                                          [Project: ~/code/myapp]  │
├────────────────┬────────────────────────────────────────┬───────────────┤
│                │                                        │               │
│   FILE TREE    │           DAG EDITOR                   │  AGENT PANEL  │
│                │                                        │               │
│  📁 swarm/     │    ┌─────────┐                        │  Running (2)  │
│   📄 swarm.yaml│    │ planner │                        │  ──────────── │
│   📁 prompts/  │    └────┬────┘                        │  🟢 planner   │
│    └─ planner  │         │                             │     iter 3/20 │
│    └─ coder    │    ┌────▼────┐                        │     $0.42     │
│    └─ eval...  │    │  coder  │                        │  🟡 coder     │
│   📁 outputs/  │    └────┬────┘                        │     iter 2/20 │
│                │         │                             │     $0.31     │
│                │    ┌────▼────┐    ┌─────────┐        │               │
│                │    │evaluator├───►│  tester │        │  History (12) │
│                │    └─────────┘    └─────────┘        │               │
│                │                                        │               │
├────────────────┴────────────────────────────────────────┴───────────────┤
│  Console Output / Logs                                                   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Panel 1: File Tree (Left Sidebar)

A collapsible file browser focused on the `swarm/` directory.

### Features

- Tree view of `swarm/` folder with icons for different file types
- Quick-create buttons for new prompts, tasks
- Right-click context menu: Edit, Rename, Delete, Duplicate
- Drag-and-drop prompt files to DAG editor to create tasks
- Filter/search within the tree
- Shows recent outputs with timestamps (`swarm/outputs/20260213-142305-abc123/`)

### File Type Handling

| File Type | Action |
|-----------|--------|
| `.yaml` files | Opens in YAML editor with schema validation |
| `.md` files | Opens in Markdown editor with preview |
| Output folders | Opens log viewer |

---

## Panel 2: DAG Editor (Center - Main View)

A visual canvas for building and editing the task dependency graph.

### Visual DAG Canvas

```
┌──────────────────────────────────────────────────────────────────┐
│  Pipeline: main                    [iterations: 20] [▶ Run]      │
│  ─────────────────────────────────────────────────────────────── │
│                                                                  │
│      ┌─────────────┐                                            │
│      │   planner   │ ← Click to select, drag to reposition      │
│      │  ─────────  │                                            │
│      │ 📝 planner  │ ← Shows prompt name                        │
│      │ 🤖 opus     │ ← Model (inherited or overridden)          │
│      └──────┬──────┘                                            │
│             │ success ← Condition label on edge                  │
│             ▼                                                    │
│      ┌─────────────┐                                            │
│      │    coder    │                                            │
│      │  ─────────  │                                            │
│      │ 📝 coder    │                                            │
│      └──────┬──────┘                                            │
│             │                                                    │
│     ┌───────┴───────┐                                           │
│     │               │                                            │
│     ▼               ▼                                            │
│ ┌─────────┐   ┌─────────┐                                       │
│ │evaluator│   │  tester │  ← Parallel tasks at same level       │
│ └─────────┘   └─────────┘                                       │
│                                                                  │
│  [+ Add Task]  [+ Add Pipeline]                                 │
└──────────────────────────────────────────────────────────────────┘
```

### Interaction Features

#### 1. Creating Tasks
- Click "+ Add Task" button or drag prompt from file tree
- Opens a task config panel (slide-out drawer)

#### 2. Creating Dependencies
- Drag from one task's output port to another's input port
- Creates edge with dropdown to select condition: `success | failure | any | always`

#### 3. Task Configuration Panel (Right Drawer)

```
┌─────────────────────────────┐
│ Task: coder                 │
├─────────────────────────────┤
│ Prompt Source               │
│ ○ From prompts/ [dropdown]  │
│ ○ File path: [...]         │
│ ○ Inline string: [textarea] │
├─────────────────────────────┤
│ Model (optional)            │
│ [dropdown: inherit | opus | │
│  sonnet | haiku]            │
├─────────────────────────────┤
│ Prefix                      │
│ [textarea]                  │
├─────────────────────────────┤
│ Suffix                      │
│ [textarea]                  │
├─────────────────────────────┤
│ Dependencies                │
│ ┌────────────┬───────────┐ │
│ │ planner    │ success ▼ │ │
│ └────────────┴───────────┘ │
│ [+ Add Dependency]         │
└─────────────────────────────┘
```

#### 4. Pipeline Configuration
- Dropdown to switch between pipelines (or "All Tasks" view)
- Edit pipeline settings: iterations, parallelism
- Select which tasks belong to pipeline (checkboxes or drag into group)

#### 5. Validation Feedback
- Red highlighting on cycles
- Warnings for orphaned tasks (dependencies but no pipeline)
- Yellow badges for tasks with parallelism inside pipelines

#### 6. Live Execution Overlay

When running, tasks show status badges:

| Status | Visual |
|--------|--------|
| Pending | ⚪ Gray |
| Running | 🔵 Blue (animated pulse) |
| Succeeded | ✅ Green checkmark |
| Failed | ❌ Red X |
| Skipped | ⏭️ Gray with skip icon |

Progress ring around running tasks showing iteration progress.

---

## Panel 3: Agent Panel (Right Sidebar)

Real-time view of running and historical agents.

### Running Agents Section

```
┌─────────────────────────────┐
│ Running Agents (2)      [⟳] │
├─────────────────────────────┤
│ 🟢 planner              ⋮  │
│    Iteration 3 of 20        │
│    ████████░░░░░░ 15%       │
│    Tokens: 12.4k in / 3.2k  │
│    Cost: $0.42              │
│    Duration: 4m 23s         │
│    [⏸ Pause] [⏹ Stop]       │
├─────────────────────────────┤
│ 🟡 coder (paused)       ⋮  │
│    Waiting for resume...    │
│    [▶ Resume] [⏹ Stop]      │
└─────────────────────────────┘
```

### Agent Detail View (click to expand)

```
┌─────────────────────────────┐
│ ← Back      planner         │
├─────────────────────────────┤
│ Status: 🟢 Running          │
│ ID: abc12345                │
│ PID: 68432                  │
│ Model: opus                 │
│ Started: 2:30 PM            │
├─────────────────────────────┤
│ Progress                    │
│ Iteration: 3 / 20           │
│ ████████░░░░░░░░ 15%        │
│                             │
│ Successful: 2               │
│ Failed: 0                   │
├─────────────────────────────┤
│ Usage                       │
│ Input tokens:  12,432       │
│ Output tokens: 3,201        │
│ Total cost:    $0.42        │
├─────────────────────────────┤
│ Current Task                │
│ "Reading: src/auth/login.ts"│
├─────────────────────────────┤
│ Controls                    │
│ Iterations: [20    ] [Set]  │
│ Model: [opus ▼]     [Set]   │
│                             │
│ [⏸ Pause] [⏹ Stop] [📋 Clone]│
└─────────────────────────────┘
```

### History Section (collapsible)

```
┌─────────────────────────────┐
│ History (12)            [▼] │
├─────────────────────────────┤
│ ✅ evaluator    2:15 PM     │
│    20/20 iters  $1.23       │
│ ❌ tester       1:45 PM     │
│    15/20 iters  $0.89       │
│    Error: timeout           │
│ ✅ planner      1:30 PM     │
│    10/10 iters  $0.45       │
└─────────────────────────────┘
```

---

## Bottom Panel: Console / Logs

Tabbed interface for viewing output:

```
┌──────────────────────────────────────────────────────────────────────────┐
│ [Console] [planner] [coder] [evaluator]                    [Clear] [↓]  │
├──────────────────────────────────────────────────────────────────────────┤
│ 2:30:15 [planner] Starting iteration 3...                               │
│ 2:30:16 [planner] Reading file: src/components/Button.tsx               │
│ 2:30:18 [planner] Tool: Read (245 lines)                                │
│ 2:30:22 [coder]   Starting iteration 2...                               │
│ 2:30:24 [planner] Writing to: src/components/Button.tsx                 │
│ 2:30:25 [coder]   Tool: Grep pattern="useState"                         │
│ █                                                                        │
└──────────────────────────────────────────────────────────────────────────┘
```

### Features
- Real-time log streaming from `~/.swarm/logs/`
- Color-coded by agent
- Filter/search within logs
- Auto-scroll toggle
- Export logs

---

## Additional Features

### 1. Command Palette (Cmd+K)

Quick actions:
- "Run pipeline: main"
- "Create new task"
- "Open swarm.yaml"
- "Pause all agents"
- "Kill agent: planner"

### 2. YAML Editor with IntelliSense

When editing `swarm.yaml` directly:
- Schema validation with red squiggles
- Autocomplete for task names in `depends_on`
- Autocomplete for prompt names from `swarm/prompts/`
- Hover documentation for fields

### 3. Prompt Editor

Markdown editor for prompt files with:
- Syntax highlighting for `{{include:path}}` directives
- Preview of resolved prompt with includes expanded
- Variable highlighting: `{{task_id}}`, `{{iteration}}`, `{{output_dir}}`

### 4. Settings Panel

```
┌─────────────────────────────┐
│ Settings                    │
├─────────────────────────────┤
│ Backend                     │
│ ○ Claude Code              │
│ ○ Cursor                   │
├─────────────────────────────┤
│ Default Model               │
│ [opus ▼]                    │
├─────────────────────────────┤
│ State Path                  │
│ ~/.swarm/state.json        │
├─────────────────────────────┤
│ Logs Directory              │
│ ~/.swarm/logs/             │
└─────────────────────────────┘
```

### 5. Notifications

- Toast notifications when agents complete/fail
- System notifications (optional) for long-running tasks
- Sound alerts (configurable)

---

## Tech Stack

| Component | Technology |
|-----------|------------|
| Framework | Electron + React 18 |
| UI Library | shadcn/ui + Tailwind CSS |
| DAG Visualization | React Flow |
| State Management | Zustand or Jotai |
| File Watching | chokidar |
| YAML Parsing | js-yaml |
| Code Editor | Monaco Editor |
| IPC | Electron IPC |
| File Tree | react-arborist |

---

## Data Flow

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   swarm.yaml    │────▶│  Electron Main   │────▶│   React UI      │
│   (workspace)   │     │    Process       │     │   (Renderer)    │
└─────────────────┘     └────────┬─────────┘     └────────┬────────┘
                                 │                        │
                                 │ spawn                  │ IPC
                                 ▼                        ▼
                        ┌──────────────────┐     ┌─────────────────┐
                        │   swarm CLI      │     │   state.json    │
                        │   (subprocess)   │────▶│   (watched)     │
                        └──────────────────┘     └─────────────────┘
```

The app doesn't replace swarm-cli, but provides a visual interface that:
1. Generates/edits valid `swarm.yaml` files
2. Spawns `swarm pipeline`, `swarm run`, etc. commands
3. Watches state files for real-time updates
4. Provides a file browser for the workspace

---

## Key Data Structures

### ComposeFile (from swarm.yaml)

```yaml
version: "1"
tasks:
  planner:
    prompt: planner           # From swarm/prompts/
    # prompt-file: ./path.md  # Or arbitrary file path
    # prompt-string: "..."    # Or inline string
    model: opus               # Optional, overrides default
    prefix: "..."             # Optional, prepended to prompt
    suffix: "..."             # Optional, appended to prompt
    depends_on:
      - task: coder
        condition: success    # success | failure | any | always

pipelines:
  main:
    iterations: 20
    parallelism: 1
    tasks: [planner, coder, evaluator, tester]
```

### AgentState (from ~/.swarm/state.json)

```json
{
  "id": "abc12345",
  "name": "planner",
  "pid": 68432,
  "status": "running",
  "model": "opus",
  "started_at": "2026-02-13T14:30:00Z",
  "iterations": 20,
  "current_iteration": 3,
  "input_tokens": 12432,
  "output_tokens": 3201,
  "total_cost_usd": 0.42,
  "current_task": "Reading: src/auth/login.ts",
  "paused": false,
  "working_dir": "/Users/matt/code/myapp",
  "log_file": "~/.swarm/logs/abc12345.log"
}
```

---

## Implementation Phases

### Phase 1: Core Foundation
- [ ] Electron app scaffold with React
- [ ] File tree component for `swarm/` directory
- [ ] Basic YAML viewer/editor for `swarm.yaml`
- [ ] Agent list panel reading from `state.json`

### Phase 2: DAG Visualization
- [ ] React Flow integration for DAG canvas
- [ ] Parse `swarm.yaml` into visual graph
- [ ] Task node components with status display
- [ ] Edge rendering with condition labels

### Phase 3: Interactive Editing
- [ ] Drag-and-drop task creation
- [ ] Visual dependency creation (port-to-port)
- [ ] Task configuration drawer
- [ ] Write changes back to `swarm.yaml`

### Phase 4: Agent Management
- [ ] Real-time state watching with chokidar
- [ ] Agent detail view with controls
- [ ] Pause/resume/stop functionality via CLI
- [ ] Log streaming in console panel

### Phase 5: Polish
- [ ] Command palette
- [ ] Monaco editor integration
- [ ] Notifications system
- [ ] Settings persistence
