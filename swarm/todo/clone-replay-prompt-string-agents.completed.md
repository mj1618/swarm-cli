# Fix: `clone` and `replay` fail for prompt-string agents

## Problem

When an agent is started with `-s` / `--prompt-string`, the prompt content is not stored in the agent state. Only the marker `<string>` is saved in the `prompt` field. This means `swarm clone` and `swarm replay` cannot reconstruct the original command:

```bash
$ swarm run -s "Review the code for security issues" -n 5 --name my-review
$ swarm clone my-review
Error: cannot clone agent with inline string prompt (prompt content not stored)
```

This is a significant UX limitation since `-s` is one of the most common ways to run agents for ad-hoc tasks.

## Solution

Store the prompt content in the agent state so it can be recovered by `clone` and `replay`. Options:

1. **Add a `PromptContent` field to `AgentState`** — store the full prompt string in state JSON. This is the simplest approach but could make `state.json` large if prompts are very long.

2. **Store prompt content in a sidecar file** — e.g., `~/.swarm/prompts/<agent-id>.md`. Keeps state.json small but adds file management complexity.

Option 1 is recommended for simplicity. Prompt strings are typically short (a few sentences), and even long ones (a few KB) are negligible compared to log files.

### Implementation

1. Add `PromptContent string \`json:"prompt_content,omitempty"\`` to `AgentState` in `internal/state/manager.go`
2. In `cmd/run.go`, set `agentState.PromptContent` when using `-s` or `--stdin`
3. In `cmd/clone.go`, use `PromptContent` to reconstruct the `-s` flag when prompt is `<string>` or `<stdin>`
4. In `cmd/replay.go`, same as clone

## Relevant Files

- `internal/state/manager.go` — `AgentState` struct (~line 14)
- `cmd/run.go` — state registration (~lines 507, 590, 708)
- `cmd/clone.go` — prompt reconstruction logic
- `cmd/replay.go` — prompt reconstruction logic
