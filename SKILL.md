---
name: swarm-up
description: "Orchestrate multi-agent AI workflows with swarm-cli compose files. Creates swarm.yaml task definitions, configures planner/doer/reviewer DAG pipelines, sets task dependencies with conditions, manages agent lifecycle (run, pause, kill, monitor), and uses runtime variables for inter-task communication. Use when setting up multi-agent orchestration, writing swarm compose files, configuring AI agent pipelines, debugging swarm task execution, or managing concurrent agent workflows."
---

# swarm up

Orchestrate multi-agent AI workflows using swarm-cli compose files. Define tasks, wire up DAG pipelines, and run agents in parallel or sequence.

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

### Example: Planner/Doer/Reviewer Pipeline

```yaml
version: "1"

tasks:
  planner:
    prompt-string: |
      Read swarm/PLAN.md. Review project state and swarm/done/ for completed work.
      Write ONE bite-sized task as {timestamp}-{name}.todo.md in SWARM_STATE_DIR.
      Include: Goal, Files to modify, Acceptance Criteria.

  implementer:
    prompt-string: |
      Read swarm/PLAN.md for context. Claim the .todo.md file by renaming to
      .processing.md. Implement the task fully, test it, then move to
      swarm/done/{name}.done.md with a completion summary.
    depends_on: [planner]

  reviewer:
    prompt-string: |
      Read swarm/PLAN.md for context. Review the most recent .done.md in
      swarm/done/. Fix any issues found, run tests, commit if good.
    depends_on: [implementer]

pipelines:
  main:
    iterations: 20
    parallelism: 1
    tasks: [planner, implementer, reviewer]
```

## swarm.yaml Reference

```yaml
version: "1"

tasks:
  task-name:
    prompt-string: "Your prompt here"   # or prompt-file / prompt
    model: sonnet                       # optional, overrides default
    iterations: 5                       # optional, default 1
    parallelism: 3                      # optional, run N instances
    prefix: "Context..."                # optional, prepended to prompt
    suffix: "Remember..."               # optional, appended to prompt
    depends_on: [other-task]            # optional, for DAG workflows

pipelines:
  main:
    iterations: 10
    parallelism: 4
    tasks: [task1, task2]
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

### Runtime Variables

| Variable | Description |
|----------|-------------|
| `SWARM_STATE_DIR` | Shared directory for the current pipeline iteration |
| `SWARM_AGENT_ID` | Unique ID for this agent instance |
| `{{output:task_name}}` | Replaced with output from named task |

## Running

```bash
swarm up -d                     # Run all pipelines in background
swarm up -d task1 task2         # Run specific tasks only
swarm up -d -p main             # Run a specific pipeline
swarm up -d -f custom.yaml      # Use a custom compose file
swarm up                        # Run in foreground (blocks until complete)
```

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
