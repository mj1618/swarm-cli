# swarm run

Run AI agents with inline string prompts.

## Basic Usage

```bash
swarm run -s "Your prompt here"
```

## All Options

| Flag | Short | Description |
|------|-------|-------------|
| `--prompt-string` | `-s` | Prompt string (required) |
| `--iterations` | `-n` | Number of iterations (default 1, 0 = unlimited) |
| `--forever` | `-F` | Run indefinitely until manually stopped |
| `--model` | `-m` | Model override (e.g. `sonnet`, `opus`) |
| `--detach` | `-d` | Run in background |
| `--name` | `-N` | Name the agent for easy reference |
| `--working-dir` | `-C` | Run agent in a different directory |
| `--env` | `-e` | Set env vars, repeatable (e.g. `-e KEY=VALUE`) |
| `--label` | `-l` | Attach labels, repeatable (e.g. `-l team=backend`) |
| `--timeout` | | Total timeout (e.g. `30m`, `2h`) |
| `--iter-timeout` | | Per-iteration timeout (e.g. `10m`) |
| `--on-complete` | | Shell command to run when agent completes |
| `--prefix` | | Content to prepend to the prompt |
| `--suffix` | | Content to append to the prompt |
| `--parent` | `-P` | Parent agent ID (for sub-agents) |

## Examples

```bash
# Simple one-shot task
swarm run -s "Fix the failing tests in src/"

# Run 5 iterations in the background
swarm run -s "Review and improve error handling" -n 5 -d

# Run forever with a model override
swarm run -s "Monitor logs and report issues" -F -m sonnet

# Named agent in a specific directory
swarm run -s "Refactor the auth module" -N auth-refactor -C ~/projects/app -d

# With a timeout and on-complete hook
swarm run -s "Run the full test suite" --timeout 30m --on-complete "notify-send 'Done'"

# With environment variables and labels
swarm run -s "Deploy to staging" -e ENV=staging -e DRY_RUN=true -l team=infra -d

# With prompt prefix/suffix
swarm run -s "Implement the feature" --prefix "You are an expert Go developer." --suffix "Write tests for everything."

# Unlimited iterations, detached
swarm run -s "Continuously check for regressions" -n 0 -d
```
