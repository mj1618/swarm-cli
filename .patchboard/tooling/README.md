# /.patchboard/tooling

This folder contains repo-local tooling to make the `/.patchboard/` workflow enforceable.

## Setup (MacOS)

# 1. Create a virtual environment
python3 -m venv .venv

# 2. Activate it
source .venv/bin/activate

# 3. Install dependencies
pip install -r requirements.txt

# 4. Now you can run the tooling
python .patchboard/tooling/patchboard.py validate

## Quick Validation

The easiest way to run validation is with the helper script:

```bash
bash .patchboard/tooling/validate.sh          # basic
bash .patchboard/tooling/validate.sh --verbose # detailed output
```

This automatically creates a lightweight Python venv (at `.patchboard-local/venv/`),
installs the required dependencies, and runs `patchboard.py validate`.

## CI / GitHub Actions

A GitHub Actions workflow template is included at `.patchboard/tooling/workflows/patchboard-validate.yml`.
To enable CI validation on your repo, copy it into place:

```bash
mkdir -p .github/workflows
cp .patchboard/tooling/workflows/patchboard-validate.yml .github/workflows/
git add .github/workflows/patchboard-validate.yml
git commit -m "Enable patchboard CI validation"
```

The workflow runs on PRs and pushes to main/master. It:
- Validates task schema, dependencies, and epic relationships
- Checks PR titles contain a task ID (T-XXXX)
- Prevents tasks from being set to `done` via PR (must use `review`)

## CLI Tools

### patchboard.py

`python .patchboard/tooling/patchboard.py --help`

Key commands:

- `validate` — validate tasks/locks/board invariants
- `claim` — create/overwrite a lock (if allowed) and set task to `in_progress`
- `renew` — extend an existing lock lease (must be same actor)
- `release` — remove a lock (optionally move task status)
- `index` — generate `/.patchboard/state/index.json` (optional cache for UIs)
- `search` — search tasks by query string with optional filters
- `archive` — move a task to `.archived/` folder (hides from CLI and GUI)
- `unarchive` — restore an archived task back to active tasks

#### Archive/Unarchive Tasks

Archive completed or obsolete tasks to reduce clutter while preserving history:

```bash
# Archive a task
python .patchboard/tooling/patchboard.py archive T-0001

# Unarchive a task
python .patchboard/tooling/patchboard.py unarchive T-0001
```

Archived tasks are stored in `.patchboard/tasks/.archived/T-XXXX/` and are:
- Excluded from CLI validation, search, and indexing
- Excluded from the management plane GUI task listings
- Preserved in git history for reference

### merge_bot.sh

`.patchboard/tooling/merge_bot.sh`

Simple interactive PR rescue bot. Run it and answer the prompts:

1. Select model (sonnet or opus)
2. Select CLI (claude or copilot)
3. Set check interval

The bot then watches for blocked PRs and automatically launches the selected AI agent to rescue them.

**Prerequisites:**
- GitHub CLI (`gh`) installed and authenticated
- For Claude: Claude CLI (`claude`) installed
- For Copilot: Copilot CLI (`copilot`) installed

**Usage:**

```bash
.patchboard/tooling/merge_bot.sh
```

### ralph_bot.sh

`.patchboard/tooling/ralph_bot.sh`

Interactive task implementer bot. Run it and answer the prompts:

1. Select an epic or task from the available backlog
2. Select model (sonnet or opus)
3. Set max iterations

The bot launches Claude with the ralph-loop skill to autonomously implement the selected task(s).

**Prerequisites:**
- Claude CLI (`claude`) installed
- GitHub CLI (`gh`) installed and authenticated
- Python 3 with venv

**Permissions pre-granted:**
- File operations: Read, Write, Edit, Glob, Grep
- Git/GitHub: `git`, `gh` commands
- Scripts: `python`, `pip`, `npm`, `npx`, `yarn`, `pytest`
- Shell: `source`, `cat`, `ls`, `mkdir`, `rm`, `cp`, `mv`, `chmod`, `head`, `tail`, `touch`, `wc`, `diff`, `tee`, `sort`, `uniq`, `xargs`, `env`, `export`, `which`, `echo`
- Network: `curl`, `wget`
- Task agent for complex subtasks

**Note:** Runs in interactive mode, so Claude can request additional permissions as needed.

**Usage:**

```bash
.patchboard/tooling/ralph_bot.sh
```

**Example session:**
```
╔═══════════════════════════════════════════╗
║      🐺 Ralph Bot - Task Implementer      ║
╚═══════════════════════════════════════════╝

Available tasks:

── Epics ──
   1) E-0003  [todo       ] P1   Implement core patchboard CLI
── Tasks ──
   2) T-0051  [ready      ] P1   Define epic-task relationship model
   3) T-0052  [todo       ] P2   Write ADR for epic relationships

Enter task number, ID (e.g., T-0051), or comma-separated list:
Selection: 2,3

Selected: T-0051 T-0052
```

## Permissions Framework

Both `merge_bot.sh` and `ralph_bot.sh` implement a **permissions review** step before launching AI agents. This ensures users explicitly acknowledge what access they're granting.

### How It Works

1. **Configuration phase** — User selects CLI (Claude or Copilot), model, and other options
2. **Permissions review** — Script displays exactly what permissions will be granted
3. **Explicit acceptance** — User must type `y` to proceed (defaults to No)
4. **Launch** — Agent runs with the stated permissions

### Claude CLI Permissions

Claude CLI supports **granular tool permissions** via `--allowedTools`. We pre-approve specific tools:

```bash
claude --model sonnet -p "$prompt" \
    --allowedTools \
    "Bash(git:*)" \
    "Bash(gh:*)" \
    "Read" \
    "Edit" \
    ...
```

Claude can still request additional permissions during execution (in interactive mode), but pre-approved tools run without prompts.

### Copilot CLI Permissions

Copilot CLI does **not** support granular permissions. The options are:

- `--allow-all-tools` — Full access to everything (required for non-interactive mode)
- `--allow-tool 'shell(git:*)'` — Granular but must be repeated per tool

For autonomous operation, we use `--allow-all-tools` and warn users accordingly.

### Adding New Bot Scripts

When creating new bot scripts that launch AI agents:

1. Add a `show_permissions_and_confirm()` function
2. Display CLI-specific permission details
3. Require explicit `y/Y` acceptance (default to No)
4. Call before launching the agent

Example pattern:

```bash
show_permissions_and_confirm() {
    echo "⚠️  PERMISSIONS REVIEW"
    
    if [[ "$SELECTED_CLI" == "claude" ]]; then
        echo "The following tools will be pre-approved:"
        # List specific --allowedTools
    else
        echo "Copilot will run with --allow-all-tools"
        echo "This grants full access to everything."
    fi
    
    read -p "Do you accept? [y/N]: " accept
    [[ "${accept:-n}" =~ ^[Yy]$ ]] || exit 0
}
```

## Philosophy

- The repo is the system: tooling edits files, but git remains the coordination layer.
- If tooling and docs disagree, **docs should be updated**; if rules are wrong, adjust the board rules.
