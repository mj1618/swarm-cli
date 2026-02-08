# Fix: `depends_on` without pipeline silently does nothing

## Problem

When tasks define `depends_on` in a compose file but no `pipelines` section is defined, the tasks are silently excluded from execution. Neither the dependent task nor its dependency runs — `swarm up` reports "No pipelines or standalone tasks to run" with no warning.

Example compose file that silently does nothing:

```yaml
version: "1"
tasks:
  task-a:
    prompt-string: "Do thing A"
    iterations: 1
  task-b:
    prompt-string: "Do thing B"
    iterations: 1
    depends_on:
      - task-a
```

Running `swarm up -f above.yaml` outputs:

```
From above.yaml:
  No pipelines or standalone tasks to run
```

This is because `GetStandaloneTasks()` in `internal/compose/compose.go` filters out:
1. Tasks that have `depends_on` (line 337)
2. Tasks that are depended upon by other tasks (line 340-341)

Without an explicit `pipelines` section, dependent tasks have no way to run.

## Expected Behavior

Either:
1. **Auto-create a pipeline** from tasks with `depends_on` relationships when no explicit pipeline is defined, OR
2. **Emit a warning** telling the user that tasks with `depends_on` require a `pipelines` section, e.g.:
   `Warning: tasks [task-b] have depends_on but no pipeline is defined — these tasks will not run. Define a pipelines section to run them in DAG order.`

Option 2 is simpler and less error-prone. Option 1 is more user-friendly.

## Relevant Files

- `internal/compose/compose.go` — `GetStandaloneTasks()` (~line 318), `Validate()` (~line 155)
- `cmd/up.go` — where standalone tasks and pipelines are dispatched
