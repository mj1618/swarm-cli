# CLAUDE.md

## Project Overview

swarm-cli is a Go CLI tool for orchestrating AI agents (Claude Code or Cursor). It manages agent lifecycle (run, pause, resume, kill, clone, replay) with state persistence, compose files for multi-task orchestration, and DAG/pipeline execution.

## Build & Test

```bash
go build -o /tmp/swarm-test .    # Build (avoid `-o swarm` — conflicts with swarm/ directory)
go test ./...                     # Run all tests
```

## Project Structure

- `main.go` — entry point, calls `cmd.Execute()`
- `cmd/` — CLI commands (cobra). One file per command.
- `internal/agent/` — agent execution and process management
- `internal/compose/` — YAML compose file parsing and validation
- `internal/config/` — configuration loading (TOML). Merges global (`~/.config/swarm/config.toml`) + project (`swarm/.swarm.toml`) + CLI flags
- `internal/dag/` — DAG executor for pipeline workflows with dependency conditions
- `internal/runner/` — multi-iteration loop with signal handling, pause/resume, timeouts
- `internal/state/` — agent state persistence to `~/.swarm/state.json`
- `internal/prompt/` — prompt loading from files/stdin/strings, `{{include:}}` directive processing
- `internal/logparser/` — parses agent output for token/cost stats
- `internal/output/` — terminal output formatting (bubbletea/lipgloss)
- `swarm/` — this project's own swarm config, prompts, and todo files

## Key Configuration

- Backend config: `swarm/.swarm.toml` (backend=claude-code, model=opus)
- Compose file: `swarm/swarm.yaml` (defines tasks with prompts and iterations)
- Prompts directory: `swarm/prompts/` (markdown files)
- State: `~/.swarm/state.json`
- Logs: `~/.swarm/logs/`

## Testing the CLI Manually

```bash
# Quick smoke test
/tmp/swarm-test doctor                                          # All checks should pass
/tmp/swarm-test run -s "Say hello" -n 1 -m sonnet -d           # Run a detached agent
/tmp/swarm-test list                                            # See running agents
/tmp/swarm-test inspect <id>                                    # Check agent details
/tmp/swarm-test logs <id>                                       # View output
/tmp/swarm-test kill <id>                                       # Terminate
```

## Common Patterns

- Backends: `claude-code` (uses `claude` CLI) and `cursor` (uses `agent` CLI). Config in `internal/config/config.go`.
- Agent args are templated with `{model}` and `{prompt}` placeholders.
- Compose tasks support `depends_on` with conditions: `success`, `failure`, `any`, `always`.
- `raw_output = true` for claude-code (streams directly), `false` for cursor (parsed through logparser).
