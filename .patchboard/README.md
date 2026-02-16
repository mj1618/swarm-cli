# .patchboard

Patchboard is a git-native task management system designed for human-agent collaboration. This directory is the system of record — all project state lives here as files in your repository.

## Directory Structure

```
.patchboard/
  tasks/                    Task and epic definitions (YAML frontmatter + Markdown)
    _templates/             Templates for creating new tasks and epics
    .archived/              Completed/archived tasks
    T-0001/task.md          Individual task files
    E-0001/task.md          Epic files (same format, type: epic)
  agents/                   AI agent workspace definitions
    architect/              Each workspace has index.md, settings.json, and prompts/
    engineer/
    ...
    prompts/                Shared prompt templates used across workspaces
  docs/                     Project documentation
    design-architecture/    Architecture docs, ADRs, design decisions
  planning/                 Project planning
    boards/kanban.yaml      Board configuration (columns, WIP limits, rules)
    roadmap.md              High-level milestones and progress
  state/                    Runtime state (session tracking, indexes)
    cloud-agents/           Agent session manifests (JSON)
  vision/                   Project vision and principles
  schemas/                  JSON schemas for validation
  tooling/                  CLI tools and bot scripts (installed separately)
  VERSION                   Schema version (used to detect out-of-date repos)
```

## Tasks

Tasks are Markdown files with YAML frontmatter in `tasks/T-XXXX/task.md`:

```yaml
---
id: T-0001
title: "Implement user authentication"
type: task          # task | bug | chore | epic
status: todo        # todo | ready | in_progress | blocked | review | done
priority: P2        # P0 (critical) .. P3 (low)
owner: null
labels: [backend, security]
depends_on: []
acceptance:
  - "Users can log in with email and password"
created_at: '2026-01-01'
updated_at: '2026-01-01'
---

## Context
...
```

**Status lifecycle**: `todo` → `ready` → `in_progress` → `review` → `done`

- Tasks move to `ready` when well-defined and unblocked
- `in_progress` requires an active PR (the PR is the lock)
- Agents set status to `review` when complete; only humans mark `done`

See `schemas/task.schema.json` for the full validation schema.

## Agent Workspaces

Each subdirectory under `agents/` defines a workspace for AI agents:

- **`index.md`** — Role definition, responsibilities, constraints, and workflow
- **`settings.json`** — UI configuration (display name, icon, model, context settings, triggers)
- **`prompts/`** — Spawn prompt templates with `{{VARIABLE}}` placeholders

The management plane uses these to populate the spawn dialog and configure agent sessions.

### Built-in Workspaces

| Workspace | Role |
|---|---|
| `architect` | Creates architecture docs and ADRs |
| `engineer` | Claims tasks, implements features, opens PRs |
| `explorer` | Systematically explores and documents GUI functionality |
| `merge-bot` | Rescues blocked PRs (merge conflicts, failing tests) |
| `reviewer` | Reviews PRs for correctness and workflow compliance |
| `systems-analyst` | Decomposes goals into tasks, manages roadmap |
| `testing-bot` | Conducts visual testing of PR functionality |

## Boards

Board configuration lives in `planning/boards/kanban.yaml`:

```yaml
board:
  id: main
  columns:
    - id: todo
      name: To do
    - id: in_progress
      name: In progress
      wip_limit: 5
      requires_lock: true
    - id: done
      name: Done
```

See `schemas/board.schema.json` for the full validation schema.

## Schemas

JSON schemas in `schemas/` validate task frontmatter and board configuration:

- `task.schema.json` — Validates task and epic YAML frontmatter
- `board.schema.json` — Validates board YAML configuration

Run validation with: `.patchboard/tooling/patchboard.py validate`

## Tooling

CLI tools and bot scripts are installed separately into `.patchboard/tooling/` via the management plane's "Install Tooling" action. The tooling includes:

- `patchboard.py` — CLI for validation, indexing, and task search
- `agent_bot.sh` — Self-hosted agent session orchestrator
- `merge_bot.sh` — PR rescue bot
- `ralph_bot.sh` — Interactive task implementer bot

## Getting Started

1. Review `vision/00-vision.md` and define your project's vision
2. Create your first tasks using the templates in `tasks/_templates/`
3. Configure agent workspaces in `agents/` for your team's needs
4. Run validation: `.patchboard/tooling/patchboard.py validate`
