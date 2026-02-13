# Inter-Agent Communication

## Problem

Pipeline tasks (e.g., planner -> coder -> evaluator -> tester) run in sequence but are completely isolated. Each agent starts fresh with only its static prompt — it has no access to what upstream agents produced. This makes pipelines far less useful than they could be.

## Goals

1. Downstream tasks can reference upstream task outputs in their prompts
2. Zero config required for simple cases (convention over configuration)
3. Works within existing compose/pipeline model — no new orchestration concepts
4. Outputs persist across iterations so pipeline iteration N can see results from iteration N-1

## Design

### Core concept: task output capture + prompt directive

Each DAG task's stdout is captured to a file. A new `{{output:task_name}}` prompt directive injects a prior task's captured output into the current task's prompt at expansion time.

### Output capture

**Location:** `~/.swarm/outputs/<pipeline-run-id>/<task-name>.txt`

- `pipeline-run-id` is a unique ID generated per `swarm up` invocation
- One file per task per iteration — overwritten each iteration so it always reflects the latest run
- Captured in the DAG executor alongside the existing `PrefixedWriter`, using an `io.MultiWriter` to tee stdout to both the terminal and the capture file
- Max capture size: 100KB (truncate from the top, keeping the tail — the end of agent output is usually the most relevant)

**New fields:**

```go
// In dag/executor.go — passed into the execution context
type PipelineRun struct {
    ID        string // unique per swarm up invocation
    OutputDir string // ~/.swarm/outputs/<ID>
}
```

```go
// In dag/executor.go — RunPipeline()
runID := state.GenerateID()
outputDir := filepath.Join(os.UserHomeDir(), ".swarm", "outputs", runID)
os.MkdirAll(outputDir, 0755)
```

**Implementation in `runTask()`:**

```go
// Create capture file
capturePath := filepath.Join(pipelineRun.OutputDir, taskName+".txt")
captureFile, _ := os.Create(capturePath)
defer captureFile.Close()

// Tee output to both display and capture
captureWriter := NewTruncatingWriter(captureFile, 100*1024) // 100KB limit
out := io.MultiWriter(displayWriter, captureWriter)

// Pass out to runner.Run()
```

`TruncatingWriter` is a new small wrapper that writes to a file but keeps only the last 100KB by writing to a ring buffer and flushing on close, or more simply: writing everything and truncating the file to the tail on close.

### Prompt directive: `{{output:task_name}}`

**Location:** `internal/prompt/include.go` — extend alongside `{{include:}}`

New regex:
```go
var outputRegex = regexp.MustCompile(`\{\{output:\s*([^}]+)\}\}`)
```

Processing happens in a new function:
```go
func ProcessOutputDirectives(content, outputDir string) (string, error)
```

This is called by the DAG executor *after* `ProcessIncludes()` and *before* `InjectTaskID()`. The executor already has access to the output directory, so it passes it through.

**Behavior:**

- `{{output:planner}}` -> reads `<outputDir>/planner.txt`, injects full contents
- If the file doesn't exist (task hasn't run yet or was skipped), inject a placeholder: `(No output available from task "planner")`
- Wrap injected output with clear delimiters so the agent can parse it:

```
--- Output from task "planner" ---
<contents>
--- End output from task "planner" ---
```

### Compose file syntax

No new compose fields required for the basic case. The `{{output:task_name}}` directive goes directly in prompt files:

```markdown
# Coder Prompt

## Plan from upstream
{{output:planner}}

## Your task
Implement the plan above. Write code and tests.
```

This is the zero-config path. The directive only works when the prompt is used inside a pipeline that has an output directory. When used outside a pipeline (e.g., `swarm run -p coder`), the directive resolves to the placeholder message.

### Optional: explicit output mapping in compose

For advanced cases where you want to rename or select specific task outputs:

```yaml
tasks:
  coder:
    prompt: coder
    depends_on: [planner]
    inputs:
      plan: planner  # maps {{input:plan}} to planner's output
```

This is a **later addition** — not in the initial implementation. The `{{output:task_name}}` convention covers the common case.

## Implementation plan

### Step 1: Output capture in DAG executor

Files: `internal/dag/executor.go`

1. Generate a `runID` at the start of `RunPipeline()`
2. Create output directory `~/.swarm/outputs/<runID>/`
3. In `runTask()`, create a capture file and `io.MultiWriter` to tee output
4. Write a `TruncatingWriter` that caps file size at 100KB (keep tail)

### Step 2: `{{output:}}` directive in prompt system

Files: `internal/prompt/output.go` (new), `internal/prompt/include.go` (minor)

1. Add `ProcessOutputDirectives(content, outputDir string) (string, error)`
2. Regex matches `{{output:task_name}}`, reads from `outputDir/task_name.txt`
3. Wraps content in delimiters, handles missing files gracefully
4. Add tests for: present output, missing output, multiple directives, empty output

### Step 3: Wire it together in DAG executor

Files: `internal/dag/executor.go`

1. After loading and processing includes for a task prompt, call `ProcessOutputDirectives()`
2. Pass the pipeline's `outputDir` through to the prompt processing
3. This happens before `InjectTaskID()` / `InjectAgentID()` — output content becomes part of the prompt

### Step 4: Cross-iteration persistence

Files: `internal/dag/executor.go`

1. Output files are **not** cleared between iterations — they're overwritten per task
2. This means iteration 2's `coder` task sees iteration 1's `evaluator` output if evaluator ran first in that cycle
3. Within a single iteration, tasks see outputs from tasks that completed earlier in the same iteration (guaranteed by `depends_on` ordering)

### Step 5: Cleanup

Files: `cmd/prune.go` (extend existing)

1. Add output directory cleanup to `swarm prune`
2. Delete output dirs older than a configurable threshold (default: 7 days)
3. `swarm prune --outputs` flag to clean only outputs

### Step 6: Documentation & prompt updates

Files: `swarm/prompts/coder.md`, `swarm/prompts/evaluator.md`, etc.

1. Update project prompts to use `{{output:}}` directives
2. Add examples to `swarm prompts --help`

## File changes summary

| File | Change |
|------|--------|
| `internal/dag/executor.go` | Generate run ID, create output dir, tee output to capture files, call `ProcessOutputDirectives()` |
| `internal/prompt/output.go` | New file: `ProcessOutputDirectives()`, `outputRegex`, delimiter wrapping |
| `internal/prompt/output_test.go` | New file: tests for output directive processing |
| `internal/dag/writer.go` | New file: `TruncatingWriter` (keeps last N bytes) |
| `internal/dag/writer_test.go` | New file: tests for truncating writer |
| `cmd/prune.go` | Add `--outputs` flag, output dir cleanup |
| `swarm/prompts/*.md` | Add `{{output:}}` directives to pipeline prompts |

## Not in scope (future work)

- **`{{input:name}}`** mapped inputs in compose file — adds flexibility but not needed initially
- **Structured output** (JSON) — agents could write structured data to a known schema; other agents parse it. Useful but complex.
- **Streaming inter-agent comms** — agent A streams to agent B in real-time. Fundamentally different model, not needed for pipeline DAGs.
- **Shared filesystem convention** — agents read/write to a shared directory. Already possible via working directory; doesn't need framework support.
- **Output selection/filtering** — `{{output:planner:last_50_lines}}` or similar. Can add later if 100KB cap isn't sufficient.
