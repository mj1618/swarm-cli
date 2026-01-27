# Swarm CLI

A command-line tool for running and managing AI agents. Swarm CLI allows you to run single agents or loop them for multiple iterations, with full control over running agents including the ability to modify settings on-the-fly.

## Features

- **Run single agents** with custom prompts
- **Loop agents** for multiple iterations with automatic restart on failure
- **Manage running agents** - list, view details, and control them
- **Live configuration updates** - change model or iterations while agents are running
- **Multiple backends** - supports Cursor's agent CLI and Claude Code CLI
- **Project & global scoping** - organize prompts and agents per-project or globally
- **Configurable** - TOML configuration with sensible defaults

## Installation

### Prerequisites

You need one of the supported agent backends installed:
- [Cursor](https://cursor.sh) with the `agent` CLI
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) CLI

### Download Binary (Recommended)

Download the latest binary for your platform from the [releases page](https://github.com/mj1618/swarm-cli/releases/latest).

**macOS (Apple Silicon):**
```bash
curl -L https://github.com/mj1618/swarm-cli/releases/download/latest/swarm-cli_darwin_arm64.tar.gz | tar xz
sudo mv swarm /usr/local/bin/
```

**macOS (Intel):**
```bash
curl -L https://github.com/mj1618/swarm-cli/releases/download/latest/swarm-cli_darwin_amd64.tar.gz | tar xz
sudo mv swarm /usr/local/bin/
```

**Linux (x64):**
```bash
curl -L https://github.com/mj1618/swarm-cli/releases/download/latest/swarm-cli_linux_amd64.tar.gz | tar xz
sudo mv swarm /usr/local/bin/
```

**Linux (ARM64):**
```bash
curl -L https://github.com/mj1618/swarm-cli/releases/download/latest/swarm-cli_linux_arm64.tar.gz | tar xz
sudo mv swarm /usr/local/bin/
```

### Install with Go

If you have Go installed:

```bash
go install github.com/mj1618/swarm-cli@latest
```

### Build from Source

```bash
git clone https://github.com/mj1618/swarm-cli.git
cd swarm-cli
go build -o swarm .
sudo mv swarm /usr/local/bin/
```

## Choosing Your Agent Backend

Swarm CLI supports two agent backends. Choose one based on what you have installed:

| Backend | CLI Command | Best For |
|---------|-------------|----------|
| **Cursor** | `agent` | Cursor IDE users with agent CLI access |
| **Claude Code** | `claude` | Standalone Claude Code CLI users |

### Set Your Backend

```bash
# Use Cursor's agent CLI (default)
swarm config set-backend cursor

# Use Claude Code CLI
swarm config set-backend claude-code
```

To verify your chosen backend is working:

```bash
# For Cursor
agent --version

# For Claude Code
claude --version
```

The backend can also be configured per-project in `.swarm.toml` or globally in `~/.config/swarm/config.toml`. See the [Configuration](#configuration) section for details.

## Quick Start

1. **Initialize configuration** (optional):
   ```bash
   swarm config init
   ```

2. **Create a prompt file** at `./swarm/prompts/my-task.md`:
   ```markdown
   # My Task
   
   Please do the following:
   - Step 1
   - Step 2
   - Step 3
   ```

3. **Run an agent**:
   ```bash
   swarm run -p my-task
   ```

4. **Or run in a loop** (20 iterations by default):
   ```bash
   swarm loop -p my-task -n 10
   ```

## Commands

### `swarm run`

Run a single agent with a specified prompt.

```bash
swarm run [flags]
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--prompt` | `-p` | Prompt name from the prompts directory |
| `--prompt-file` | `-f` | Path to an arbitrary prompt file |
| `--prompt-string` | `-s` | Direct prompt text |
| `--model` | `-m` | Model to use (overrides config) |

**Examples:**
```bash
# Interactive prompt selection
swarm run

# Use a named prompt
swarm run -p my-task

# Use a specific file
swarm run -f /path/to/prompt.md

# Direct prompt string
swarm run -s "Fix all linter errors in the codebase"

# Specify a model
swarm run -p my-task -m gpt-5.2
```

### `swarm loop`

Run an agent repeatedly for a specified number of iterations. Agent failures do not stop the loop.

```bash
swarm loop [flags]
```

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--prompt` | `-p` | Prompt name from the prompts directory |
| `--prompt-file` | `-f` | Path to an arbitrary prompt file |
| `--prompt-string` | `-s` | Direct prompt text |
| `--model` | `-m` | Model to use (overrides config) |
| `--iterations` | `-n` | Number of iterations (default: 20) |
| `--name` | `-N` | Name for the agent (for easier reference) |

**Examples:**
```bash
# Run with default iterations (20)
swarm loop -p continuous-improvement

# Run with a name for easier reference
swarm loop -p my-task -n 50 --name my-agent

# Run 50 iterations
swarm loop -p my-task -n 50

# Run with a different model
swarm loop -p my-task -m sonnet-4.5-thinking -n 30
```

### `swarm list`

List running agents with their status and configuration.

```bash
swarm list [flags]
```

By default, only shows agents started in the current directory. Use `--global` to show all agents.

**Output columns:**
- `ID` - Unique agent identifier
- `PROMPT` - The prompt being used
- `MODEL` - The model being used
- `STATUS` - Current status (running/terminated)
- `ITERATION` - Current iteration / total iterations
- `DIRECTORY` - Working directory (global mode only)
- `STARTED` - Time since agent started

**Examples:**
```bash
# List agents in current project
swarm list

# List all agents globally
swarm list --global
```

### `swarm view [agent-id-or-name]`

View detailed information about a specific agent. You can reference the agent by its ID or name.

```bash
swarm view abc123       # by ID
swarm view my-agent     # by name
```

**Output includes:**
- Agent ID and PID
- Prompt and model
- Status with color coding
- Start time and duration
- Current iteration progress
- Working directory
- Termination mode (if set)
- Log file location (if available)

### `swarm control [agent-id-or-name]`

Control a running agent by changing its configuration or terminating it. You can reference the agent by its ID or name.

```bash
swarm control [agent-id-or-name] [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--iterations`, `-n` | Set new iteration count |
| `--model`, `-m` | Change model (applies on next iteration) |
| `--terminate` | Terminate immediately |
| `--terminate-after` | Terminate after current iteration completes |

**Examples:**
```bash
# Terminate immediately
swarm control abc123 --terminate

# Graceful termination after current iteration
swarm control abc123 --terminate-after

# Increase iterations to 100
swarm control abc123 -n 100

# Switch to a faster model
swarm control abc123 -m sonnet-4.5

# Multiple changes at once
swarm control abc123 -n 50 -m gpt-5.2
```

### `swarm config`

Manage swarm-cli configuration.

#### `swarm config init`

Create a configuration file with default values.

```bash
# Create project config (.swarm.toml)
swarm config init

# Create global config (~/.config/swarm/config.toml)
swarm config init --global
```

#### `swarm config show`

Display the effective configuration after merging all sources.

```bash
swarm config show
```

#### `swarm config path`

Show configuration file locations and their status.

```bash
swarm config path
```

#### `swarm config set-backend [backend]`

Switch between agent backends.

```bash
# Use Cursor's agent CLI
swarm config set-backend cursor

# Use Claude Code CLI
swarm config set-backend claude-code

# Update global config instead of project
swarm config set-backend claude-code --global
```

## Configuration

Swarm CLI uses TOML configuration files with the following priority (highest to lowest):

1. CLI flags
2. Project config (`.swarm.toml` in current directory)
3. Global config (`~/.config/swarm/config.toml`)
4. Built-in defaults

### Configuration File

```toml
# swarm-cli configuration

# Backend: "cursor" or "claude-code"
backend = "cursor"

# Default model for agent runs
model = "opus-4.5-thinking"

# Default iterations for loop command
iterations = 20

# Agent command configuration
[command]
executable = "agent"
args = [
  "--model", "{model}",
  "--output-format", "stream-json",
  "--stream-partial-output",
  "--sandbox", "disabled",
  "--print",
  "--force",
  "{prompt}"
]
raw_output = false
```

### Backends

#### Cursor (`cursor`)

Uses Cursor's `agent` CLI with JSON streaming output and log parsing.

**Default model:** `opus-4.5-thinking`

**Available models** (run `agent --list-models` for full list):
- `opus-4.5-thinking` - Claude 4.5 Opus (Thinking) - default
- `sonnet-4.5-thinking` - Claude 4.5 Sonnet (Thinking)
- `opus-4.5` - Claude 4.5 Opus
- `sonnet-4.5` - Claude 4.5 Sonnet
- `gpt-5.2` - GPT-5.2
- `gpt-5.2-codex` - GPT-5.2 Codex
- `gemini-3-pro` - Gemini 3 Pro
- `gemini-3-flash` - Gemini 3 Flash
- `grok` - Grok

#### Claude Code (`claude-code`)

Uses Anthropic's `claude` CLI with direct text streaming.

**Default model:** `opus`

**Available models:**
- `opus` - Claude Opus
- `sonnet` - Claude Sonnet

## Prompts

Prompts are markdown files that contain instructions for the agent.

### Prompt Locations

| Scope | Directory |
|-------|-----------|
| Project | `./swarm/prompts/` |
| Global | `~/.swarm/prompts/` |

### Creating Prompts

Create a markdown file in the prompts directory:

```bash
mkdir -p ./swarm/prompts
cat > ./swarm/prompts/refactor.md << 'EOF'
# Refactoring Task

Please refactor the codebase with the following goals:

## Objectives
- Improve code readability
- Extract common patterns into reusable functions
- Add appropriate error handling
- Update documentation

## Constraints
- Do not change public API signatures
- Maintain backward compatibility
- Keep all tests passing
EOF
```

### Using Prompts

```bash
# By name (without .md extension)
swarm run -p refactor

# Interactive selection (when no -p flag)
swarm run

# From arbitrary file path
swarm run -f ~/my-prompts/special-task.md

# Direct string (for quick one-off tasks)
swarm run -s "Add unit tests for the user authentication module"
```

## Scoping

Swarm CLI supports two scopes for organizing work:

### Project Scope (default)

- Prompts: `./swarm/prompts/`
- Agents: Only shows agents started in the current directory
- Config: `.swarm.toml` in current directory

### Global Scope (`--global` / `-g`)

- Prompts: `~/.swarm/prompts/`
- Agents: Shows all agents across all directories
- Config: `~/.config/swarm/config.toml`

**Examples:**
```bash
# List only this project's agents
swarm list

# List all agents globally
swarm list -g

# Use global prompts directory
swarm run -g -p shared-task
```

## Workflow Examples

### Continuous Development Loop

Run an agent continuously to work on a task:

```bash
# Create a task prompt
cat > ./swarm/prompts/dev-loop.md << 'EOF'
Check the current state of the codebase and continue working on:
1. Any failing tests - fix them
2. Any TODO comments - implement them  
3. Any linter warnings - resolve them
4. Code quality improvements

After each change, run the test suite to verify.
EOF

# Start the loop
swarm loop -p dev-loop -n 50
```

### Managing Long-Running Agents

```bash
# Start a loop in the background
swarm loop -p my-task -n 100 &

# Check on running agents
swarm list

# View details of a specific agent
swarm view abc123

# Extend the iterations
swarm control abc123 -n 200

# Gracefully stop after current iteration
swarm control abc123 --terminate-after
```

### Multi-Project Setup

```bash
# Set up global config with preferred backend
swarm config init --global
swarm config set-backend cursor --global

# Create project-specific overrides
cd ~/projects/frontend
swarm config init
# Edit .swarm.toml to use faster model for frontend work

cd ~/projects/backend  
swarm config init
# Edit .swarm.toml to use more capable model for backend work
```

## Troubleshooting

### Agent not found

If `swarm list` shows no agents but you know one is running:
- Make sure you're in the same directory where the agent was started
- Use `swarm list --global` to see all agents

### Configuration not loading

Check configuration paths and status:
```bash
swarm config path
swarm config show
```

### Agent backend not working

Verify the backend CLI is installed and working:
```bash
# For Cursor backend
agent --version
agent --list-models

# For Claude Code backend
claude --version
```

### Clearing stale agent state

Agent state is stored in `~/.swarm/state/`. If agents appear stuck or stale, you can manually clean the state:

```bash
rm -rf ~/.swarm/state/*
```

## License

MIT
