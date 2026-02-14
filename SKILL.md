# swarm up

Run multi-task AI agent workflows using compose files.

## Quick Start

1. Create `./swarm/swarm.yaml` with your tasks
2. Create `./swarm/PLAN.md` with your project plan
3. Run `swarm up -d` to start all tasks in the background

## The Planner/Doer Pattern

The standard approach uses a **planner** that creates bite-sized tasks, **doer(s)** that execute them, and optionally a **reviewer** that checks quality. This runs in iterations, making incremental progress each cycle.

### Task File Lifecycle

Tasks are coordinated through files with specific extensions:

1. **Planner creates**: `{YYYY-MM-DD-HH-MM-SS}-{task-name}.todo.md` in `SWARM_STATE_DIR`
2. **Doer claims**: Renames `.todo.md` → `.processing.md`
3. **Doer completes**: Moves to `swarm/done/` as `.done.md`

### Example: Build an Application

**`swarm/PLAN.md`** — Your project plan (source of truth for all agents):

```markdown
# Project Plan

## Overview
Build a CLI tool for managing bookmarks with tagging and search.

## Goals
- Store bookmarks with title, URL, tags
- Search by tag or text
- Import/export functionality

## Tech Stack
- Go with Cobra for CLI
- SQLite for storage

## Milestones
1. Basic CRUD operations
2. Tag management
3. Search functionality
4. Import/export
```

**`swarm/swarm.yaml`**:

```yaml
version: "1"

tasks:
  planner:
    prompt-string: |
      # Planner

      Read `swarm/PLAN.md` for the project plan.

      ## Your Job

      1. Review current project state — what files exist, what's been built
      2. Check `swarm/done/` for completed tasks (avoid repeating work)
      3. Check SWARM_STATE_DIR for existing `.todo.md` or `.processing.md` files
      4. Identify the NEXT single piece of work needed
      5. Write a task file: `{YYYY-MM-DD-HH-MM-SS}-{task-name}.todo.md` in SWARM_STATE_DIR

      ## Task File Format

      Include:
      - **Goal**: What to build/change
      - **Files**: Which files to create/modify
      - **Acceptance Criteria**: How to verify completion

      Keep tasks bite-sized — completable in one agent session.

  implementer:
    prompt-string: |
      # Implementer

      Read `swarm/PLAN.md` for project context.

      ## Your Job

      1. Find the `.todo.md` file in SWARM_STATE_DIR
      2. Rename it to `.processing.md` to claim the task
      3. Read the task and implement it fully
      4. Test your work — verify it actually works
      5. Create `swarm/done/` directory if needed
      6. Move the file to `swarm/done/{name}.done.md`
      7. Append a summary of what you accomplished

      Do NOT just describe what to do — actually write the code.
    depends_on: [planner]

  reviewer:
    prompt-string: |
      # Reviewer

      Read `swarm/PLAN.md` for project context.

      ## Your Job

      1. Find the most recent `.done.md` file in `swarm/done/`
      2. Review the implementation for quality and correctness
      3. FIX any issues you find (don't just report them)
      4. Run tests if they exist
      5. Commit changes if everything looks good
    depends_on: [implementer]

pipelines:
  main:
    iterations: 20
    parallelism: 1
    tasks: [planner, implementer, reviewer]
```

## swarm.yaml Reference

### Full Structure

```yaml
version: "1"

tasks:
  task-name:
    prompt-string: |
      Your prompt here (multi-line supported)
    model: sonnet              # optional, overrides default
    iterations: 5              # optional, default 1
    parallelism: 3             # optional, run N instances
    name: custom-name          # optional, defaults to task name
    prefix: "Context..."       # optional, prepended to prompt
    suffix: "Remember..."      # optional, appended to prompt
    depends_on: [other-task]   # optional, for DAG workflows

pipelines:
  main:
    iterations: 10             # run entire DAG this many times
    parallelism: 4             # run N concurrent pipeline instances
    tasks: [task1, task2]      # tasks to include
```

### Prompt Sources (pick one)

| Field | Description |
|-------|-------------|
| `prompt-string` | Inline prompt text (use YAML `\|` for multi-line) |
| `prompt-file` | Path to a prompt file |
| `prompt` | Name of file in prompts directory (no extension) |

### Dependency Conditions

```yaml
depends_on:
  - task: build
    condition: success   # only if build succeeded
  - task: test
    condition: failure   # only if test failed
  - task: deploy
    condition: any       # after completion (default)
  - task: notify
    condition: always    # even if skipped
```

## Runtime Variables

These are automatically available in prompts:

| Variable | Description |
|----------|-------------|
| `SWARM_STATE_DIR` | Shared directory for the current pipeline iteration |
| `SWARM_AGENT_ID` | Unique ID for this agent instance |
| `{{output:task_name}}` | Replaced with output from named task |

## Running

```bash
# Run all pipelines in background
swarm up -d

# Run specific tasks only
swarm up -d task1 task2

# Run a specific pipeline
swarm up -d -p main

# Use a custom compose file
swarm up -d -f custom.yaml

# Run in foreground (blocks until complete)
swarm up
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--detach` | `-d` | Run in background |
| `--file` | `-f` | Path to compose file (default: `./swarm/swarm.yaml`) |
| `--pipeline` | `-p` | Run a specific pipeline by name |

## Monitoring

```bash
swarm list          # See running agents
swarm logs <id>     # View agent output  
swarm inspect <id>  # Check agent details
swarm kill <id>     # Stop an agent
```

## Re-running

Running `swarm up -d` again will:
- Skip tasks/pipelines already running
- Start any new or stopped tasks
- Kill excess instances if parallelism was reduced
