# swarm-cli

CLI tool for orchestrating AI agents. Manages agent lifecycle and multi-task pipelines.

## Quick Start

```bash
# Run a single agent for 5 iterations
swarm run -s "Your prompt here" -n 5 -d

# Run from a prompt file for 10 iterations
swarm run -p my-prompt -n 10 -d -m sonnet

# Run a compose file (all pipelines)
swarm up

# Run specific tasks from compose
swarm up frontend backend
```

## Running Agents

```bash
swarm run [flags]
```

Prompt source (pick one):
- `-p NAME` — prompt from `swarm/prompts/` directory
- `-f PATH` — prompt from file path
- `-s STRING` — inline prompt string
- `-i` — read prompt from stdin

Key flags:
- `-n INT` — iterations (default 1, 0 = unlimited)
- `-F` — run forever
- `-m MODEL` — model override
- `-d` — detach (run in background)
- `-N NAME` — name the agent
- `-C PATH` — working directory
- `-e KEY=VALUE` — env vars (repeatable)
- `-l key=value` — labels (repeatable)
- `--timeout DURATION` — total timeout (e.g. `30m`, `2h`)
- `--iter-timeout DURATION` — per-iteration timeout
- `--on-complete CMD` — run command on completion
- `--prefix STRING` / `--suffix STRING` — modify prompt
- `-P AGENT_ID` — parent agent (for sub-agents)

## Compose Files

Default location: `swarm/swarm.yaml`

```yaml
version: "1"

tasks:
  planner:
    prompt: planner
  coder:
    prompt: coder
    depends_on: [planner]
  tester:
    prompt: tester
    depends_on: [coder:success]

pipelines:
  main:
    iterations: 10
    tasks: [planner, coder, tester]
```

Task fields:
- `prompt` — name from prompts directory
- `prompt-file` — path to prompt file
- `prompt-string` — inline prompt text
- `model` — model override
- `iterations` — task iteration count
- `depends_on` — list of dependencies with optional conditions: `success`, `failure`, `any`, `always`
- `prefix` / `suffix` — prompt modifiers

Commands:
```bash
swarm up                        # run all pipelines + standalone tasks
swarm up task1 task2            # run specific tasks
swarm up -f custom.yaml         # custom compose file
swarm up -d                     # detached
swarm down                      # kill all compose agents
swarm down task1                # kill specific compose task
swarm compose-stop              # pause compose agents
swarm compose-logs [task]       # view compose logs
```

## Monitoring

```bash
swarm list                      # running agents
swarm list -a                   # include terminated
swarm list -q                   # IDs only
swarm list --format json
swarm list --label team=backend
swarm inspect AGENT_ID          # detailed info
swarm logs AGENT_ID             # view output
swarm logs AGENT_ID -f          # follow live
swarm logs AGENT_ID --tail 100 --grep error
swarm top                       # resource usage
swarm summary                   # overview
```

## Agent Control

```bash
swarm stop AGENT_ID             # pause (finishes current iteration)
swarm start AGENT_ID            # resume paused agent
swarm kill AGENT_ID             # terminate immediately
swarm kill AGENT_ID --graceful  # terminate after current iteration
swarm restart AGENT_ID          # restart agent
swarm attach AGENT_ID           # attach to running agent

# Bulk operations
swarm pause-all
swarm resume-all
swarm kill-all
swarm kill --label env=staging  # kill by label

# Replay / clone
swarm replay AGENT_ID           # re-run with same config
swarm clone AGENT_ID            # clone to new agent
swarm clone AGENT_ID -n 50 -d  # clone with overrides
```

## Agent ID Shortcuts

Any command accepting `AGENT_ID` also accepts:
- Agent name (substring match)
- `@last` or `_` — most recently started agent

## Configuration

Config files: `~/.config/swarm/config.toml` (global), `swarm/swarm.toml` (project)

```bash
swarm config show               # display effective config
swarm config path               # show config file locations
swarm config set-backend cursor # switch backend
swarm config set-model opus     # set default model
swarm doctor                    # run diagnostic checks
swarm prompts                   # list available prompts
swarm prompts show NAME         # display prompt content
swarm prompts new NAME          # create prompt
swarm prompts edit NAME         # edit prompt
```

## Utility

```bash
swarm prune                     # clean up old logs/state
swarm rm AGENT_ID               # remove terminated agent from state
swarm wait AGENT_ID             # wait for agent to complete
swarm models                    # list available models
swarm version                   # show version
```

## Key Paths

| Item | Location |
|------|----------|
| Global config | `~/.config/swarm/config.toml` |
| Project config | `swarm/swarm.toml` |
| Prompts | `swarm/prompts/` |
| Compose file | `swarm/swarm.yaml` |
| State | `~/.swarm/state.json` |
| Logs | `~/.swarm/logs/` |
