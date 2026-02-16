#!/usr/bin/env bash
#
# ralph_bot.sh - Interactive task implementer bot
#
# Lets you select an epic or task, then launches Claude with the ralph-loop
# skill to autonomously implement it.
#

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PATCHBOARD_TOOLS_VERSION="$(cat "${SCRIPT_DIR}/VERSION" 2>/dev/null || echo "unknown")"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Configuration
SELECTED_MODEL="sonnet"
SELECTED_CLI="claude"
MAX_ITERATIONS=10
COMPLETION_PROMISE="DONE"

print_header() {
    echo -e "${BLUE}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║  🐺 Ralph Bot - Implementer  v${PATCHBOARD_TOOLS_VERSION}  ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_prerequisites() {
    local missing=()

    if ! command -v gh &> /dev/null; then
        missing+=("GitHub CLI (gh)")
    fi

    if ! command -v jq &> /dev/null; then
        missing+=("jq")
    fi

    if ! command -v python3 &> /dev/null; then
        missing+=("python3")
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        echo -e "${RED}Error: Missing prerequisites:${NC}"
        for item in "${missing[@]}"; do
            echo "  - $item"
        done
        exit 1
    fi

    # Check if venv exists and has required packages
    if [[ ! -d "$REPO_ROOT/.venv" ]]; then
        echo -e "${YELLOW}Warning: .venv not found. Creating...${NC}"
        python3 -m venv "$REPO_ROOT/.venv"
        source "$REPO_ROOT/.venv/bin/activate"
        pip install -q pyyaml python-dateutil jsonschema
    fi

}

ensure_ralph_loop_plugin() {
    # Only needed for Claude CLI
    if [[ "$SELECTED_CLI" != "claude" ]]; then
        return 0
    fi

    echo -e "${YELLOW}Checking for ralph-loop plugin...${NC}"

    local plugin_status
    plugin_status=$(claude plugin list 2>/dev/null)

    # Check if plugin is installed AND enabled
    if echo "$plugin_status" | grep -q "ralph-loop" && echo "$plugin_status" | grep -A3 "ralph-loop" | grep -q "✔ enabled"; then
        echo -e "${GREEN}ralph-loop plugin is installed and enabled.${NC}"
        return 0
    fi

    # Check if installed but disabled
    if echo "$plugin_status" | grep -q "ralph-loop"; then
        echo -e "${YELLOW}ralph-loop plugin is installed but disabled. Enabling...${NC}"
        if claude plugin enable ralph-loop --scope project 2>/dev/null; then
            echo -e "${GREEN}Plugin enabled successfully.${NC}"
            return 0
        else
            echo -e "${RED}Failed to enable plugin.${NC}"
            exit 1
        fi
    fi

    echo -e "${YELLOW}ralph-loop plugin not found.${NC}"
    read -p "Would you like to install it? [Y/n]: " install_plugin

    if [[ "${install_plugin:-y}" =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Installing ralph-loop plugin...${NC}"
        if claude plugin install ralph-loop@claude-plugins-official; then
            echo -e "${GREEN}Plugin installed successfully.${NC}"
        else
            echo -e "${RED}Failed to install plugin.${NC}"
            exit 1
        fi
    else
        echo -e "${RED}ralph-loop plugin is required. Exiting.${NC}"
        exit 1
    fi
}

get_available_tasks() {
    # Get tasks that are not done, sorted by priority then ID
    source "$REPO_ROOT/.venv/bin/activate"
    python3 << 'PYTHON'
import sys
sys.path.insert(0, '.patchboard/tooling')
from patchboard import repo_root_from_here, discover_tasks

repo_root = repo_root_from_here()
tasks = discover_tasks(repo_root)

# Filter and sort tasks
available = []
for tid, task in sorted(tasks.items()):
    status = task.status
    if status in ('done',):
        continue
    task_type = task.frontmatter.get('type', 'task')
    priority = task.frontmatter.get('priority', 'P2')
    title = task.title
    available.append({
        'id': tid,
        'type': task_type,
        'status': status,
        'priority': priority,
        'title': title
    })

# Sort by type (epics first), then priority, then ID
type_order = {'epic': 0, 'task': 1, 'bug': 2, 'chore': 3, 'spike': 4}
priority_order = {'P0': 0, 'P1': 1, 'P2': 2, 'P3': 3, 'P4': 4}

available.sort(key=lambda x: (
    type_order.get(x['type'], 99),
    priority_order.get(x['priority'], 99),
    x['id']
))

for t in available:
    print(f"{t['id']}|{t['type']}|{t['status']}|{t['priority']}|{t['title']}")
PYTHON
}

select_model() {
    echo -e "${YELLOW}Select model:${NC}"
    if [[ "$SELECTED_CLI" == "claude" ]]; then
        echo "  1) sonnet (faster, recommended)"
        echo "  2) opus (more capable)"
    else
        echo "  1) claude-sonnet-4 (faster, recommended)"
        echo "  2) claude-opus-4 (more capable)"
        echo "  3) gpt-5 (OpenAI)"
    fi
    echo ""
    read -p "Choice [1]: " choice

    case "${choice:-1}" in
        1) SELECTED_MODEL="sonnet" ;;
        2) SELECTED_MODEL="opus" ;;
        3) [[ "$SELECTED_CLI" == "copilot" ]] && SELECTED_MODEL="gpt-5" || SELECTED_MODEL="sonnet" ;;
        *) SELECTED_MODEL="sonnet" ;;
    esac

    echo -e "${GREEN}Selected model: ${SELECTED_MODEL}${NC}"
    echo ""
}

select_cli() {
    echo -e "${YELLOW}Select AI CLI to use:${NC}"
    echo "  1) claude (Claude CLI with ralph-loop plugin)"
    echo "  2) copilot (GitHub Copilot CLI)"
    echo ""
    read -p "Choice [1]: " choice

    case "${choice:-1}" in
        1) SELECTED_CLI="claude" ;;
        2) SELECTED_CLI="copilot" ;;
        *) SELECTED_CLI="claude" ;;
    esac

    echo -e "${GREEN}Selected CLI: ${SELECTED_CLI}${NC}"
    echo ""

    # Verify CLI is available
    if [[ "$SELECTED_CLI" == "claude" ]]; then
        if ! command -v claude &> /dev/null; then
            echo -e "${RED}Error: Claude CLI not found.${NC}"
            echo "Install from: https://docs.anthropic.com/en/docs/claude-cli"
            exit 1
        fi
    else
        if ! command -v copilot &> /dev/null; then
            echo -e "${RED}Error: Copilot CLI not found.${NC}"
            echo "Install from: https://githubnext.com/projects/copilot-cli"
            exit 1
        fi
    fi
}

select_iterations() {
    echo -e "${YELLOW}Max iterations for ralph-loop [10]:${NC}"
    read -p "Iterations: " iterations
    MAX_ITERATIONS="${iterations:-10}"
    echo -e "${GREEN}Max iterations: ${MAX_ITERATIONS}${NC}"
    echo ""
}

check_and_resolve_dependencies() {
    # Check if selected tasks have unmet dependencies and offer to include them
    source "$REPO_ROOT/.venv/bin/activate"

    local tasks_str="${SELECTED_TASKS[*]}"

    # Get unmet dependencies for selected tasks
    local deps_output
    deps_output=$(python3 << PYTHON
import sys
sys.path.insert(0, '.patchboard/tooling')
from patchboard import repo_root_from_here, discover_tasks

repo_root = repo_root_from_here()
tasks = discover_tasks(repo_root)

selected = "$tasks_str".split()
unmet_deps = []
seen = set()

def get_unmet_deps(task_id, visited=None):
    """Recursively get all unmet dependencies."""
    if visited is None:
        visited = set()
    if task_id in visited:
        return []
    visited.add(task_id)

    task = tasks.get(task_id)
    if not task:
        return []

    result = []
    for dep_id in task.depends_on:
        dep_task = tasks.get(dep_id)
        if dep_task and dep_task.status != 'done':
            # Recursively get deps of this dep first
            result.extend(get_unmet_deps(dep_id, visited))
            if dep_id not in seen:
                seen.add(dep_id)
                result.append(dep_id)
    return result

# Collect all unmet deps for all selected tasks
all_unmet = []
for tid in selected:
    deps = get_unmet_deps(tid)
    for d in deps:
        if d not in all_unmet and d not in selected:
            all_unmet.append(d)

# Print in dependency order (deps first)
for dep_id in all_unmet:
    dep_task = tasks.get(dep_id)
    if dep_task:
        print(f"{dep_id}|{dep_task.title}")
PYTHON
)

    if [[ -z "$deps_output" ]]; then
        # No unmet dependencies
        return 0
    fi

    echo ""
    echo -e "${YELLOW}The selected task(s) have unmet dependencies:${NC}"
    echo ""

    local -a dep_ids=()
    local -a dep_titles=()

    while IFS='|' read -r id title; do
        dep_ids+=("$id")
        dep_titles+=("$title")
        # Truncate title if too long
        if [[ ${#title} -gt 60 ]]; then
            title="${title:0:57}..."
        fi
        echo -e "  ${RED}→${NC} ${BOLD}$id${NC}: $title"
    done <<< "$deps_output"

    echo ""
    echo -e "${YELLOW}Would you like to include these dependencies?${NC}"
    read -p "[Y/n]: " include_deps

    if [[ "${include_deps:-y}" =~ ^[Yy]$ ]]; then
        # Prepend dependencies to SELECTED_TASKS (deps should be done first)
        SELECTED_TASKS=("${dep_ids[@]}" "${SELECTED_TASKS[@]}")
        echo ""
        echo -e "${GREEN}Updated task list: ${SELECTED_TASKS[*]}${NC}"
    else
        echo ""
        echo -e "${RED}Warning: Proceeding without dependencies may cause issues.${NC}"
        read -p "Continue anyway? [y/N]: " continue_anyway
        if [[ ! "${continue_anyway:-n}" =~ ^[Yy]$ ]]; then
            echo "Aborted."
            exit 0
        fi
    fi
}

select_task() {
    echo -e "${YELLOW}Fetching available tasks...${NC}"
    echo ""

    local tasks_output
    tasks_output=$(get_available_tasks)

    if [[ -z "$tasks_output" ]]; then
        echo -e "${RED}No available tasks found (all tasks are done).${NC}"
        exit 0
    fi

    # Parse tasks into arrays
    local -a task_ids=()
    local -a task_types=()
    local -a task_statuses=()
    local -a task_priorities=()
    local -a task_titles=()

    while IFS='|' read -r id type status priority title; do
        task_ids+=("$id")
        task_types+=("$type")
        task_statuses+=("$status")
        task_priorities+=("$priority")
        task_titles+=("$title")
    done <<< "$tasks_output"

    # Display tasks
    echo -e "${BOLD}Available tasks:${NC}"
    echo ""

    local current_type=""
    for i in "${!task_ids[@]}"; do
        local type="${task_types[$i]}"

        # Print section header when type changes
        if [[ "$type" != "$current_type" ]]; then
            current_type="$type"
            case "$type" in
                epic) echo -e "${CYAN}── Epics ──${NC}" ;;
                task) echo -e "${CYAN}── Tasks ──${NC}" ;;
                bug) echo -e "${CYAN}── Bugs ──${NC}" ;;
                *) echo -e "${CYAN}── ${type^} ──${NC}" ;;
            esac
        fi

        local id="${task_ids[$i]}"
        local status="${task_statuses[$i]}"
        local priority="${task_priorities[$i]}"
        local title="${task_titles[$i]}"

        # Truncate title if too long
        if [[ ${#title} -gt 50 ]]; then
            title="${title:0:47}..."
        fi

        # Color code by status
        local status_color="$NC"
        case "$status" in
            todo) status_color="$NC" ;;
            ready) status_color="$GREEN" ;;
            in_progress) status_color="$YELLOW" ;;
            blocked) status_color="$RED" ;;
            review) status_color="$BLUE" ;;
        esac

        printf "  %2d) ${BOLD}%-7s${NC} [${status_color}%-11s${NC}] %-3s  %s\n" \
            "$((i + 1))" "$id" "$status" "$priority" "$title"
    done

    echo ""
    echo -e "${YELLOW}Enter task number, ID (e.g., T-0051), or comma-separated list:${NC}"
    read -p "Selection: " selection

    SELECTED_TASKS=()

    # Parse selection (could be number, ID, or comma-separated list)
    IFS=',' read -ra parts <<< "$selection"
    for part in "${parts[@]}"; do
        part=$(echo "$part" | xargs)  # trim whitespace

        if [[ "$part" =~ ^[0-9]+$ ]]; then
            # It's a number
            local idx=$((part - 1))
            if [[ $idx -ge 0 && $idx -lt ${#task_ids[@]} ]]; then
                SELECTED_TASKS+=("${task_ids[$idx]}")
            else
                echo -e "${RED}Invalid selection: $part${NC}"
                exit 1
            fi
        elif [[ "$part" =~ ^[TE]-[0-9]{4}$ ]]; then
            # It's a task/epic ID
            SELECTED_TASKS+=("$part")
        else
            echo -e "${RED}Invalid selection: $part${NC}"
            exit 1
        fi
    done

    if [[ ${#SELECTED_TASKS[@]} -eq 0 ]]; then
        echo -e "${RED}No tasks selected.${NC}"
        exit 1
    fi

    echo ""
    echo -e "${GREEN}Selected: ${SELECTED_TASKS[*]}${NC}"
    echo ""
}

build_prompt() {
    local tasks_str="${SELECTED_TASKS[*]}"
    local template_file="$SCRIPT_DIR/ralph_bot.md"

    # Read the template file
    if [[ ! -f "$template_file" ]]; then
        echo -e "${RED}Error: Template file not found: $template_file${NC}" >&2
        exit 1
    fi

    local prompt
    prompt=$(cat "$template_file")

    # Replace placeholders
    prompt="${prompt//\{\{TASK_IDS\}\}/$tasks_str}"
    prompt="${prompt//\{\{COMPLETION_PROMISE\}\}/$COMPLETION_PROMISE}"

    echo "$prompt"
}

run_ralph_loop() {
    local prompt
    prompt=$(build_prompt)

    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${GREEN}🚀 Launching Ralph Loop${NC}"
    echo -e "   CLI: ${SELECTED_CLI}"
    echo -e "   Model: ${SELECTED_MODEL}"
    echo -e "   Tasks: ${SELECTED_TASKS[*]}"
    if [[ "$SELECTED_CLI" == "claude" ]]; then
        echo -e "   Max iterations: ${MAX_ITERATIONS}"
        echo -e "   Completion promise: ${COMPLETION_PROMISE}"
    fi
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo ""

    # Show permissions and require acceptance
    echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
    echo -e "${YELLOW}⚠️  PERMISSIONS REVIEW${NC}"
    echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
    echo ""

    if [[ "$SELECTED_CLI" == "claude" ]]; then
        echo -e "The following tools will be ${BOLD}pre-approved${NC} for Claude:"
        echo ""
        echo -e "${CYAN}Shell Commands:${NC}"
        echo "  • git:*        - All git commands"
        echo "  • gh:*         - All GitHub CLI commands"
        echo "  • python*      - Python interpreter"
        echo "  • pip:*        - Package installation"
        echo "  • npm:*, npx:*, yarn:* - Node.js package managers"
        echo "  • pytest:*     - Test runner"
        echo ""
        echo -e "${CYAN}File Operations:${NC}"
        echo "  • cat, head, tail, ls, wc, diff"
        echo "  • mkdir, rm, cp, mv, chmod, touch"
        echo "  • tee, sort, uniq, xargs"
        echo ""
        echo -e "${CYAN}Network:${NC}"
        echo "  • curl:*, wget:*"
        echo ""
        echo -e "${CYAN}Environment:${NC}"
        echo "  • source, env, export, which, echo"
        echo ""
        echo -e "${CYAN}Claude Tools:${NC}"
        echo "  • Read, Write, Edit - File access"
        echo "  • Glob, Grep        - Search"
        echo "  • Task              - Task management"
        echo ""
        echo -e "${YELLOW}Note: Claude may request additional permissions during execution.${NC}"
    else
        echo -e "Copilot will run with ${BOLD}--allow-all-tools${NC}"
        echo ""
        echo -e "${RED}This grants full access to:${NC}"
        echo "  • All shell commands (no restrictions)"
        echo "  • All file read/write operations"
        echo "  • All network access"
        echo "  • All available MCP tools"
        echo ""
        echo -e "${YELLOW}Note: Copilot CLI does not support granular tool permissions.${NC}"
    fi

    echo ""
    echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
    read -p "Do you accept these permissions? [y/N]: " accept_permissions

    if [[ ! "${accept_permissions:-n}" =~ ^[Yy]$ ]]; then
        echo -e "${RED}Permissions not accepted. Aborting.${NC}"
        exit 0
    fi

    echo ""

    if [[ "$SELECTED_CLI" == "claude" ]]; then
        # Build the full command for Claude with ralph-loop plugin
        local ralph_command="/ralph-loop:ralph-loop \"${prompt}\" --completion-promise \"${COMPLETION_PROMISE}\" --max-iterations ${MAX_ITERATIONS}"

        echo -e "${BLUE}Launching Claude...${NC}"
        echo ""

        # Launch Claude with the ralph-loop skill in INTERACTIVE mode
        # Pre-grant common permissions; Claude can request more as needed
        claude --model "$SELECTED_MODEL" \
            --allowedTools \
            "Bash(git:*)" \
            "Bash(gh:*)" \
            "Bash(python*)" \
            "Bash(pip:*)" \
            "Bash(npm:*)" \
            "Bash(npx:*)" \
            "Bash(yarn:*)" \
            "Bash(pytest:*)" \
            "Bash(source:*)" \
            "Bash(cat:*)" \
            "Bash(ls:*)" \
            "Bash(mkdir:*)" \
            "Bash(rm:*)" \
            "Bash(cp:*)" \
            "Bash(mv:*)" \
            "Bash(chmod:*)" \
            "Bash(head:*)" \
            "Bash(tail:*)" \
            "Bash(touch:*)" \
            "Bash(wc:*)" \
            "Bash(diff:*)" \
            "Bash(curl:*)" \
            "Bash(wget:*)" \
            "Bash(tee:*)" \
            "Bash(sort:*)" \
            "Bash(uniq:*)" \
            "Bash(xargs:*)" \
            "Bash(env:*)" \
            "Bash(export:*)" \
            "Bash(which:*)" \
            "Bash(echo:*)" \
            "Read" \
            "Write" \
            "Edit" \
            "Glob" \
            "Grep" \
            "Task" \
            -c "$ralph_command"
    else
        # Build model flag for Copilot
        local model_flag
        case "$SELECTED_MODEL" in
            sonnet) model_flag="claude-sonnet-4" ;;
            opus) model_flag="claude-opus-4" ;;
            *) model_flag="$SELECTED_MODEL" ;;
        esac

        echo -e "${BLUE}Launching Copilot...${NC}"
        echo ""

        # Launch Copilot in non-interactive mode with full permissions
        copilot --model "$model_flag" \
            -p "$prompt" \
            --allow-all-tools
    fi

    echo ""
    echo -e "${GREEN}✅ Ralph loop complete${NC}"
}

main() {
    cd "$REPO_ROOT"

    print_header
    check_prerequisites
    select_cli
    ensure_ralph_loop_plugin

    select_task
    check_and_resolve_dependencies
    select_model
    if [[ "$SELECTED_CLI" == "claude" ]]; then
        select_iterations
    fi

    run_ralph_loop
}

main "$@"
