#!/usr/bin/env bash
#
# agent_bot.sh - Interactive agent session bot
#
# Polls for queued self-hosted agent sessions and executes them using
# the selected AI CLI (Claude or Copilot). Follows the merge_bot pattern
# with interactive setup, atomic git-based claims, and branch management.
#
# Usage:
#   agent_bot.sh                                   # Interactive setup
#   agent_bot.sh --cli claude --model sonnet       # Headless with defaults
#   agent_bot.sh --non-interactive --max-sessions 1 # CI mode
#
# Dependencies: jq, git

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PATCHBOARD_TOOLS_VERSION="$(cat "${SCRIPT_DIR}/VERSION" 2>/dev/null || echo "unknown")"
STARTUP_VERSION="$PATCHBOARD_TOOLS_VERSION"
REPO_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"
SESSION_DIR="${REPO_ROOT}/.patchboard/state/cloud-agents"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Configuration (set by menu or CLI args)
SELECTED_CLI=""
SELECTED_MODEL=""
POLL_INTERVAL=""
NON_INTERACTIVE=false
MAX_SESSIONS=0  # 0 = unlimited
AGENT_TIMEOUT=3600  # seconds; 0 = no timeout

# State
running=false
sessions_processed=0
AGENT_CONV_ID=""           # Pre-generated UUID for Claude session tracking
_RESTARTED=false           # Set by --_restarted (internal: auto-restart after version change)

print_header() {
    if [[ "${_RESTARTED:-false}" == "true" ]]; then
        echo ""
        echo -e "${YELLOW}[bot] Auto-restarted after version update (v${STARTUP_VERSION} → v${PATCHBOARD_TOOLS_VERSION})${NC}"
        echo ""
    else
        echo -e "${BLUE}"
        echo "╔═══════════════════════════════════════════╗"
        echo "║       Patchboard Agent Bot  v${PATCHBOARD_TOOLS_VERSION}      ║"
        echo "╚═══════════════════════════════════════════╝"
        echo -e "${NC}"
    fi
}

usage() {
    cat <<'EOF'
Usage: agent_bot.sh [OPTIONS]

Interactive agent session bot. Polls for queued self-hosted sessions
and executes them using Claude or Copilot CLI.

Options:
  --cli claude|copilot       AI CLI to use (skips interactive prompt)
  --model MODEL              Model to use (skips interactive prompt)
  --poll-interval SECONDS    Poll interval (default: 60, skips prompt)
  --non-interactive          Use -p mode (agent runs and exits)
  --max-sessions N           Exit after N sessions (0=unlimited, default: 0)
  --timeout SECONDS          Kill agent after SECONDS (default: 3600, 0=disabled)
  --_restarted               (internal) Skip permissions prompt on auto-restart
  -h, --help                 Show this help message

Interactive mode (default) launches the CLI with -c (initial command),
allowing you to interact with the agent after it processes the prompt.
Non-interactive mode uses -p (the agent runs the prompt and exits).

Examples:
  agent_bot.sh                                        # Full interactive setup
  agent_bot.sh --cli claude --model sonnet            # Skip CLI/model prompts
  agent_bot.sh --cli claude --non-interactive         # Headless mode
  agent_bot.sh --max-sessions 1 --poll-interval 5     # Process one session
EOF
}

# ─── Argument parsing ────────────────────────────────────────────────

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --cli)
                SELECTED_CLI="$2"
                shift 2
                ;;
            --model)
                SELECTED_MODEL="$2"
                shift 2
                ;;
            --poll-interval)
                POLL_INTERVAL="$2"
                shift 2
                ;;
            --non-interactive)
                NON_INTERACTIVE=true
                shift
                ;;
            --max-sessions)
                MAX_SESSIONS="$2"
                shift 2
                ;;
            --timeout)
                AGENT_TIMEOUT="$2"
                shift 2
                ;;
            --_restarted)
                _RESTARTED=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                echo -e "${YELLOW}Warning: Unknown option '$1' (ignored for compatibility)${NC}" >&2
                # Skip value arg if next token doesn't look like a flag
                if [[ $# -gt 1 && ! "$2" =~ ^-- ]]; then shift; fi
                shift
                ;;
        esac
    done
}

# ─── Prerequisites ───────────────────────────────────────────────────

check_prerequisites() {
    if ! command -v jq &>/dev/null; then
        echo -e "${RED}Error: jq is required but not installed.${NC}" >&2
        exit 1
    fi

    if ! command -v git &>/dev/null; then
        echo -e "${RED}Error: git is required but not installed.${NC}" >&2
        exit 1
    fi
}

# ─── Interactive setup ───────────────────────────────────────────────

select_cli() {
    if [[ -n "$SELECTED_CLI" ]]; then
        # Validate pre-set value
        case "$SELECTED_CLI" in
            claude|copilot) ;;
            *)
                echo -e "${RED}Error: --cli must be 'claude' or 'copilot', got '${SELECTED_CLI}'${NC}" >&2
                exit 1
                ;;
        esac
    else
        echo -e "${YELLOW}Select AI CLI:${NC}"
        echo "  1) claude (Claude CLI)"
        echo "  2) copilot (GitHub Copilot CLI)"
        echo ""
        read -p "Choice [1]: " choice

        case "${choice:-1}" in
            1) SELECTED_CLI="claude" ;;
            2) SELECTED_CLI="copilot" ;;
            *) SELECTED_CLI="claude" ;;
        esac
    fi

    echo -e "${GREEN}Selected CLI: ${SELECTED_CLI}${NC}"
    echo ""

    # Verify CLI is available
    if [[ "$SELECTED_CLI" == "claude" ]]; then
        if ! command -v claude &>/dev/null; then
            echo -e "${RED}Error: Claude CLI not found.${NC}" >&2
            echo "Install from: https://docs.anthropic.com/en/docs/claude-cli"
            exit 1
        fi
    else
        if ! command -v copilot &>/dev/null; then
            echo -e "${RED}Error: Copilot CLI not found.${NC}" >&2
            echo "Install from: https://githubnext.com/projects/copilot-cli"
            exit 1
        fi
    fi
}

select_model() {
    if [[ -n "$SELECTED_MODEL" ]]; then
        echo -e "${GREEN}Selected model: ${SELECTED_MODEL}${NC}"
        echo ""
        return
    fi

    if [[ "$SELECTED_CLI" == "claude" ]]; then
        echo -e "${YELLOW}Select Claude model:${NC}"
        echo "  1) sonnet (faster, recommended)"
        echo "  2) opus (more capable)"
        echo ""
        read -p "Choice [1]: " choice

        case "${choice:-1}" in
            1) SELECTED_MODEL="sonnet" ;;
            2) SELECTED_MODEL="opus" ;;
            *) SELECTED_MODEL="sonnet" ;;
        esac
    else
        echo -e "${YELLOW}Select model for Copilot:${NC}"
        echo -e "${CYAN}Claude models:${NC}"
        echo "  1) claude-sonnet-4.5 (faster, recommended)"
        echo "  2) claude-opus-4.5 (more capable)"
        echo -e "${CYAN}OpenAI models:${NC}"
        echo "  3) gpt-5.2"
        echo -e "${CYAN}Google models:${NC}"
        echo "  4) gemini-3-pro"
        echo -e "${CYAN}OpenAI Codex models:${NC}"
        echo "  5) gpt-5.1-codex"
        echo "  6) gpt-5.2-codex-max"
        echo ""
        read -p "Choice [1]: " choice

        case "${choice:-1}" in
            1) SELECTED_MODEL="claude-sonnet-4.5" ;;
            2) SELECTED_MODEL="claude-opus-4.5" ;;
            3) SELECTED_MODEL="gpt-5.2" ;;
            4) SELECTED_MODEL="gemini-3-pro" ;;
            5) SELECTED_MODEL="gpt-5.1-codex" ;;
            6) SELECTED_MODEL="gpt-5.2-codex-max" ;;
            *) SELECTED_MODEL="claude-sonnet-4.5" ;;
        esac
    fi

    echo -e "${GREEN}Selected model: ${SELECTED_MODEL}${NC}"
    echo ""
}

select_poll_interval() {
    if [[ -n "$POLL_INTERVAL" ]]; then
        echo -e "${GREEN}Poll interval: ${POLL_INTERVAL}s${NC}"
        echo ""
        return
    fi

    echo -e "${YELLOW}Poll interval in seconds [60]:${NC}"
    read -p "Interval: " interval
    POLL_INTERVAL="${interval:-60}"
    echo ""
}

select_mode() {
    if [[ "$NON_INTERACTIVE" == "true" ]]; then
        echo -e "${GREEN}Mode: non-interactive (-p)${NC}"
        echo ""
        return
    fi

    echo -e "${YELLOW}Agent execution mode:${NC}"
    echo "  1) interactive (launches CLI session you can follow up in)"
    echo "  2) non-interactive (agent runs prompt and exits)"
    echo ""
    read -p "Choice [1]: " choice

    case "${choice:-1}" in
        2) NON_INTERACTIVE=true ;;
        *) NON_INTERACTIVE=false ;;
    esac

    if [[ "$NON_INTERACTIVE" == "true" ]]; then
        echo -e "${GREEN}Mode: non-interactive (-p)${NC}"
    else
        echo -e "${GREEN}Mode: interactive (-c)${NC}"
    fi
    echo ""
}

show_permissions_and_confirm() {
    if [[ "${_RESTARTED:-false}" == "true" ]]; then
        echo -e "${GREEN}Auto-restart — permissions previously accepted.${NC}"
        echo ""
        return
    fi

    echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
    echo -e "${YELLOW}PERMISSIONS REVIEW${NC}"
    echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
    echo ""

    if [[ "$SELECTED_CLI" == "claude" ]]; then
        echo -e "The following tools will be ${BOLD}pre-approved${NC} for Claude:"
        echo ""
        echo -e "${CYAN}Shell Commands:${NC}"
        echo "  git:*         - All git commands"
        echo "  gh:*          - All GitHub CLI commands"
        echo "  python*       - Python interpreter"
        echo "  pip:*         - Package installation"
        echo "  npm:*, npx:*  - Node.js package managers"
        echo "  pytest:*      - Test runner"
        echo ""
        echo -e "${CYAN}File Operations:${NC}"
        echo "  cat, head, tail, ls, wc, diff"
        echo "  mkdir, rm, cp, mv, chmod, touch"
        echo "  tee, sort, uniq, xargs"
        echo ""
        echo -e "${CYAN}Claude Tools:${NC}"
        echo "  Read, Write, Edit - File access"
        echo "  Glob, Grep        - Search"
        echo "  Task              - Task management"
        echo ""
        echo -e "${YELLOW}Note: Claude may request additional permissions during execution.${NC}"
    else
        echo -e "Copilot will run with ${BOLD}--allow-all-tools${NC}"
        echo ""
        echo -e "${RED}This grants full access to:${NC}"
        echo "  All shell commands (no restrictions)"
        echo "  All file read/write operations"
        echo "  All network access"
        echo "  All available MCP tools"
        echo ""
        echo -e "${YELLOW}Note: Copilot CLI does not support granular tool permissions.${NC}"
    fi

    echo ""
    echo -e "${YELLOW}═══════════════════════════════════════════${NC}"

    if [[ "$NON_INTERACTIVE" == "true" ]]; then
        echo -e "${GREEN}Non-interactive mode — permissions auto-accepted.${NC}"
        echo ""
        return
    fi

    read -p "Do you accept these permissions? [y/N]: " accept_permissions

    if [[ ! "${accept_permissions:-n}" =~ ^[Yy]$ ]]; then
        echo -e "${RED}Permissions not accepted. Aborting.${NC}"
        exit 0
    fi
    echo ""
}

# ─── Signal handling ─────────────────────────────────────────────────

handle_signal() {
    echo ""
    echo -e "${YELLOW}[bot] Shutdown requested — finishing current session then exiting.${NC}"
    if $running; then
        running=false
    else
        exit 130
    fi
}
trap handle_signal INT TERM

# ─── UUID generation ─────────────────────────────────────────────────

generate_uuid() {
    uuidgen 2>/dev/null \
        || python3 -c "import uuid; print(uuid.uuid4())" 2>/dev/null \
        || cat /proc/sys/kernel/random/uuid 2>/dev/null \
        || echo "$(date +%s)-$$-${RANDOM}"
}

# ─── Branch management ───────────────────────────────────────────────

ensure_on_main() {
    local current_branch
    current_branch=$(git -C "$REPO_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

    if [[ "$current_branch" != "main" ]]; then
        echo -e "${BLUE}[bot] Switching from '${current_branch}' to main...${NC}"
        git -C "$REPO_ROOT" checkout main --quiet 2>/dev/null || true
    fi
}

# ─── Claim protocol ─────────────────────────────────────────────────

claim_session() {
    local session_id="$1"
    local session_file="${SESSION_DIR}/${session_id}.json"

    # Verify session exists and is queued
    if [[ ! -f "$session_file" ]]; then
        echo -e "${RED}[claim] Session file not found: ${session_id}${NC}" >&2
        return 1
    fi

    local status
    status=$(jq -r '.status' "$session_file")
    if [[ "$status" != "queued" ]]; then
        echo -e "${RED}[claim] Session ${session_id} is not queued (status: ${status})${NC}" >&2
        return 1
    fi

    # Update session JSON: active + claim metadata
    local now
    now=$(date -u +%FT%TZ)
    local hostname_val
    hostname_val=$(hostname)

    jq --arg host "$hostname_val" \
       --arg now "$now" \
       '.status = "active"
        | .claimed_by = $host
        | .claimed_at = $now
        | .updated_at = $now' \
       "$session_file" > "${session_file}.tmp" && mv "${session_file}.tmp" "$session_file"

    # Commit
    git -C "$REPO_ROOT" add "$session_file"
    git -C "$REPO_ROOT" commit -m "agent: claim session ${session_id}" --quiet

    # Push (atomic claim)
    echo -e "${BLUE}[claim] Pushing claim for ${session_id}...${NC}"
    if ! git -C "$REPO_ROOT" push --quiet 2>/dev/null; then
        echo -e "${RED}[claim] Push failed — session likely claimed by another agent.${NC}" >&2
        git -C "$REPO_ROOT" reset --hard HEAD~1 --quiet
        return 1
    fi

    echo -e "${GREEN}[claim] Successfully claimed session ${session_id}${NC}"
    return 0
}

# ─── Agent invocation ────────────────────────────────────────────────

# Shared allowed tools list for Claude
CLAUDE_ALLOWED_TOOLS=(
    "Bash(git:*)"
    "Bash(gh:*)"
    "Bash(python*)"
    "Bash(pip:*)"
    "Bash(npm:*)"
    "Bash(npx:*)"
    "Bash(pytest:*)"
    "Bash(cat:*)"
    "Bash(ls:*)"
    "Bash(mkdir:*)"
    "Bash(rm:*)"
    "Bash(cp:*)"
    "Bash(mv:*)"
    "Bash(chmod:*)"
    "Bash(head:*)"
    "Bash(tail:*)"
    "Bash(touch:*)"
    "Bash(wc:*)"
    "Bash(diff:*)"
    "Bash(tee:*)"
    "Bash(sort:*)"
    "Bash(uniq:*)"
    "Bash(xargs:*)"
    "Bash(source:*)"
    "Bash(env:*)"
    "Bash(export:*)"
    "Bash(which:*)"
    "Bash(echo:*)"
    "Read"
    "Write"
    "Edit"
    "Glob"
    "Grep"
    "Task"
)

invoke_agent() {
    local session_id="$1"
    local session_file="${SESSION_DIR}/${session_id}.json"

    # Extract prompt from session
    local prompt
    prompt=$(jq -r '.prompt // ""' "$session_file")

    if [[ -z "$prompt" ]]; then
        echo -e "${RED}[agent] No prompt found in session ${session_id}${NC}" >&2
        return 1
    fi

    # Strip YAML frontmatter if present (template metadata, not prompt content).
    # Prompt templates may include ---\ntitle: ...\n--- which Claude CLI
    # misinterprets as a command-line option.
    if [[ "$prompt" == ---* ]]; then
        local stripped
        stripped=$(printf '%s\n' "$prompt" | awk '/^---$/ { count++; next } count >= 2 { print }')
        if [[ -n "$stripped" ]]; then
            prompt="$stripped"
        fi
    fi

    # In non-interactive mode, append context so the agent knows it must
    # complete the task in a single pass with no human follow-up.
    if [[ "$NON_INTERACTIVE" == "true" ]]; then
        prompt="${prompt}

---
IMPORTANT: You are running in a non-interactive, headless environment. There is no human to respond to follow-up questions. You must:
1. Complete the task fully in a single pass — do not ask clarifying questions
2. Create a feature branch, commit your work, push, and create a PR with 'gh pr create'
3. If the task is ambiguous, make reasonable assumptions and proceed
4. Do not stop to ask 'Would you like me to...?' — just do it"
    fi

    echo -e "${BLUE}[agent] Launching ${SELECTED_CLI} (${SELECTED_MODEL})...${NC}"
    echo ""

    # Reset conversation ID — will be captured from Claude's output after the run
    if [[ "$SELECTED_CLI" == "claude" ]]; then
        AGENT_CONV_ID=""
    fi

    # Run agent, capture exit code, stderr, and stdout
    local agent_exit=0
    local stderr_file="/tmp/patchboard-stderr-${session_id}.txt"
    local stdout_file="/tmp/patchboard-stdout-${session_id}.txt"

    if [[ "${AGENT_TIMEOUT:-0}" -gt 0 ]]; then
        echo -e "${CYAN}[agent] Timeout: ${AGENT_TIMEOUT}s${NC}"
    fi

    # Build optional timeout prefix. We MUST use --foreground because without
    # it, `timeout` creates a new process group that prevents Claude CLI from
    # accessing the TTY, causing it to hang silently with zero output.
    local timeout_cmd=""
    if [[ "${AGENT_TIMEOUT:-0}" -gt 0 ]]; then
        timeout_cmd="timeout --foreground --signal TERM --kill-after 30 ${AGENT_TIMEOUT}"
    fi

    # Tee both stdout and stderr so the user can monitor on the console
    # while we also capture to files for diagnostics.
    if [[ "$SELECTED_CLI" == "claude" ]]; then
        if [[ "$NON_INTERACTIVE" == "true" ]]; then
            # Use stream-json for real-time output: claude → tee (raw JSONL to file) → jq (text to console)
            # Disable errexit/pipefail so jq parse errors don't kill the script,
            # while PIPESTATUS still captures claude's (or timeout's) exit code.
            set +eo pipefail
            $timeout_cmd claude --model "$SELECTED_MODEL" -p "$prompt" \
                --output-format stream-json --verbose --include-partial-messages \
                --dangerously-skip-permissions \
                2> >(tee "$stderr_file" >&2) | tee "$stdout_file" | \
                jq -rj 'select(.type == "stream_event" and .event.delta?.type? == "text_delta") | .event.delta.text' 2>/dev/null
            agent_exit=${PIPESTATUS[0]}
            set -eo pipefail

            # Extract actual conversation ID from stream-json output for recovery
            if [[ -f "$stdout_file" ]]; then
                local extracted_id=""
                extracted_id=$(jq -r 'select(.type == "system") | .session_id // empty' "$stdout_file" 2>/dev/null | head -1) || true
                if [[ -n "$extracted_id" ]]; then
                    AGENT_CONV_ID="$extracted_id"
                    echo -e "${CYAN}[agent] Captured Claude session ID: ${AGENT_CONV_ID}${NC}"
                    # Store conversation ID in session config for recovery
                    jq --arg conv_id "$AGENT_CONV_ID" \
                       '.config.conversation_id = $conv_id' \
                       "$session_file" > "${session_file}.tmp" && mv "${session_file}.tmp" "$session_file"
                fi
            fi
        else
            $timeout_cmd claude --model "$SELECTED_MODEL" \
                --allowedTools "${CLAUDE_ALLOWED_TOOLS[@]}" \
                -c "$prompt" \
                2> >(tee "$stderr_file" >&2) | tee "$stdout_file"
            agent_exit=${PIPESTATUS[0]}
        fi
    else
        # Copilot
        if [[ "$NON_INTERACTIVE" == "true" ]]; then
            $timeout_cmd copilot --model "$SELECTED_MODEL" \
                -p "$prompt" \
                --allow-all-tools \
                2> >(tee "$stderr_file" >&2) | tee "$stdout_file"
            agent_exit=${PIPESTATUS[0]}
        else
            $timeout_cmd copilot --model "$SELECTED_MODEL" \
                --allow-all-tools \
                -c "$prompt" \
                2> >(tee "$stderr_file" >&2) | tee "$stdout_file"
            agent_exit=${PIPESTATUS[0]}
        fi
    fi

    return $agent_exit
}

# ─── Session status update ──────────────────────────────────────────

# Update session JSON to completed/failed and commit+push
# Args: $1 = session_id, $2 = exit_code, $3 = stderr_content (optional)
update_session_status() {
    local session_id="$1"
    local exit_code="$2"
    local stderr_content="${3:-}"
    local session_file="${SESSION_DIR}/${session_id}.json"
    local now
    now=$(date -u +%FT%TZ)
    local final_status

    if [[ $exit_code -eq 0 ]]; then
        final_status="completed"
        echo -e "${GREEN}[status] Marking session as completed.${NC}"
        jq --arg now "$now" \
           '.status = "completed" | .completed_at = $now | .updated_at = $now' \
           "$session_file" > "${session_file}.tmp" && mv "${session_file}.tmp" "$session_file"
    else
        final_status="failed"
        echo -e "${RED}[status] Marking session as failed (exit code ${exit_code}).${NC}"
        jq --arg now "$now" \
           --arg err "$stderr_content" \
           '.status = "failed" | .completed_at = $now | .updated_at = $now | .error_message = $err' \
           "$session_file" > "${session_file}.tmp" && mv "${session_file}.tmp" "$session_file"
    fi

    git -C "$REPO_ROOT" add "$session_file"
    git -C "$REPO_ROOT" commit -m "agent: session ${session_id} ${final_status}" --quiet 2>/dev/null || true

    # Push with pull-rebase retry (origin may have moved since our last pull)
    local push_ok=false
    for attempt in 1 2 3; do
        if git -C "$REPO_ROOT" push --quiet 2>/dev/null; then
            push_ok=true
            break
        fi
        echo -e "${YELLOW}[status] Push failed (attempt ${attempt}/3), pulling and retrying...${NC}"
        git -C "$REPO_ROOT" pull --rebase --quiet 2>/dev/null || true
    done

    if [[ "$push_ok" == "false" ]]; then
        echo -e "${RED}[status] ERROR: Failed to push session status update after 3 attempts.${NC}" >&2
        echo -e "${RED}[status] Session ${session_id} may still show as 'active' on remote.${NC}" >&2
    fi
}

# ─── Diagnostic failure PR ──────────────────────────────────────────

# Create a diagnostic PR when an agent session fails.
# This gives human reviewers visibility into what happened.
# Args: $1 = session_id, $2 = failure_reason, $3 = exit_code,
#       $4 = stderr_content (optional), $5 = stdout_content (optional)
create_diagnostic_pr() {
    local session_id="$1"
    local failure_reason="$2"
    local exit_code="$3"
    local stderr_content="${4:-}"
    local stdout_content="${5:-}"
    local session_file="${SESSION_DIR}/${session_id}.json"

    echo -e "${YELLOW}[diagnostic] Creating diagnostic PR for failed session ${session_id}...${NC}"

    # First, update session status to failed (on main)
    local error_detail="${failure_reason}"
    if [[ -n "$stdout_content" ]]; then
        error_detail="${error_detail} Agent output (last 2048 chars): ${stdout_content: -2048}"
    fi
    update_session_status "$session_id" "$exit_code" "$error_detail"

    # Ensure we're on main before branching
    ensure_on_main
    git -C "$REPO_ROOT" pull --rebase --quiet 2>/dev/null || true

    # Create diagnostic branch
    local diag_branch="agent/diagnostic/${session_id}"
    if ! git -C "$REPO_ROOT" checkout -b "$diag_branch" --quiet 2>/dev/null; then
        echo -e "${RED}[diagnostic] Failed to create diagnostic branch.${NC}" >&2
        return 1
    fi

    # Extract session metadata for the diagnostic file
    local prompt task_ids model claimed_by claimed_at started_at
    prompt=$(jq -r '.prompt // "N/A"' "$session_file")
    task_ids=$(jq -r 'if .task_ids and (.task_ids | length > 0) then (.task_ids | join(", ")) else .task_id // "N/A" end' "$session_file")
    model=$(jq -r '.model // "N/A"' "$session_file")
    claimed_by=$(jq -r '.claimed_by // "N/A"' "$session_file")
    claimed_at=$(jq -r '.claimed_at // "N/A"' "$session_file")
    started_at=$(jq -r '.started_at // "N/A"' "$session_file")
    local now
    now=$(date -u +%FT%TZ)

    # Write diagnostic markdown file
    local diag_file="${SESSION_DIR}/${session_id}-diagnostic.md"
    {
        echo "# Agent Diagnostic Report: ${session_id}"
        echo ""
        echo "## Session Metadata"
        echo ""
        echo "| Field | Value |"
        echo "|-------|-------|"
        echo "| Session ID | \`${session_id}\` |"
        echo "| Task IDs | ${task_ids} |"
        echo "| Model | ${model} |"
        echo "| Claimed By | ${claimed_by} |"
        echo "| Claimed At | ${claimed_at} |"
        echo "| Started At | ${started_at} |"
        echo "| Failed At | ${now} |"
        echo "| Exit Code | ${exit_code} |"
        echo ""
        echo "## Failure Reason"
        echo ""
        echo "${failure_reason}"
        echo ""
        echo "## Prompt"
        echo ""
        echo '```'
        echo "${prompt}"
        echo '```'
        echo ""
        echo "## Agent stdout (last 4096 chars)"
        echo ""
        echo '```'
        if [[ -n "$stdout_content" ]]; then
            echo "${stdout_content}"
        else
            echo "<no output captured>"
        fi
        echo '```'
        echo ""
        echo "## Agent stderr (last 4096 chars)"
        echo ""
        echo '```'
        if [[ -n "$stderr_content" ]]; then
            echo "${stderr_content}"
        else
            echo "<no stderr captured>"
        fi
        echo '```'
    } > "$diag_file"

    # Commit diagnostic file + session JSON
    git -C "$REPO_ROOT" add "$diag_file" "$session_file"
    if ! git -C "$REPO_ROOT" commit -m "agent: diagnostic report for ${session_id} (${failure_reason})" --quiet 2>/dev/null; then
        echo -e "${RED}[diagnostic] Failed to commit diagnostic file.${NC}" >&2
        git -C "$REPO_ROOT" checkout main --quiet 2>/dev/null || true
        return 1
    fi

    # Push branch
    if ! git -C "$REPO_ROOT" push -u origin "$diag_branch" --quiet 2>/dev/null; then
        echo -e "${RED}[diagnostic] Failed to push diagnostic branch.${NC}" >&2
        git -C "$REPO_ROOT" checkout main --quiet 2>/dev/null || true
        return 1
    fi

    # Create PR
    local pr_url=""
    if command -v gh &>/dev/null; then
        local pr_title="Agent diagnostic: ${session_id} (${failure_reason})"
        pr_url=$(gh pr create \
            --head "$diag_branch" \
            --title "$pr_title" \
            --body "Diagnostic report for failed agent session \`${session_id}\`.

**Failure reason:** ${failure_reason}
**Exit code:** ${exit_code}
**Task IDs:** ${task_ids}
**Model:** ${model}

> This PR was automatically created by the agent bot diagnostic system." \
            2>/dev/null || echo "")
    fi

    if [[ -n "$pr_url" ]]; then
        echo -e "${GREEN}[diagnostic] Diagnostic PR created: ${pr_url}${NC}"
        # Store diagnostic PR URL in session JSON
        jq --arg url "$pr_url" '.config.diagnostic_pr_url = $url' \
           "$session_file" > "${session_file}.tmp" && mv "${session_file}.tmp" "$session_file"
        git -C "$REPO_ROOT" add "$session_file"
        git -C "$REPO_ROOT" commit -m "agent: session ${session_id} link diagnostic PR" --quiet 2>/dev/null || true
        git -C "$REPO_ROOT" push --quiet 2>/dev/null || true
    else
        echo -e "${RED}[diagnostic] WARNING: Could not create diagnostic PR.${NC}" >&2
    fi

    # Return to main
    git -C "$REPO_ROOT" checkout main --quiet 2>/dev/null || true

    return 0
}

# ─── Post-run recovery ──────────────────────────────────────────────

# Resume the agent to complete missing work (e.g. create PR)
# Args: $1 = reason, $2 = branch
# Returns: 0 if recovery attempted, 1 if not possible
attempt_recovery() {
    local reason="$1"
    local branch="$2"

    local recovery_prompt=""
    case "$reason" in
        incomplete_delivery)
            recovery_prompt="You completed your work but left uncommitted changes on main without creating a branch or PR. Please deliver your work now: create a feature branch, commit all your changes, push the branch, and create a PR with 'gh pr create'. Do not leave changes on main."
            ;;
        no_pr)
            recovery_prompt="You were working on branch '${branch}' but did not create a pull request. Please create a PR for this branch now using 'gh pr create' with an appropriate title and description based on your changes."
            ;;
        *)
            recovery_prompt="Please ensure your work is complete: branch is pushed, PR is created, and all changes are committed."
            ;;
    esac

    echo -e "${YELLOW}[recovery] Attempting recovery (reason: ${reason})...${NC}"

    local recovery_exit=0

    if [[ "$SELECTED_CLI" == "claude" ]]; then
        if [[ -n "$AGENT_CONV_ID" ]]; then
            echo -e "${BLUE}[recovery] Resuming Claude session ${AGENT_CONV_ID}...${NC}"
            claude --resume "$AGENT_CONV_ID" \
                -p "$recovery_prompt" \
                --permission-mode bypassPermissions \
                2>/dev/null || recovery_exit=$?
        else
            echo -e "${YELLOW}[recovery] No conversation ID — skipping agent recovery.${NC}"
            return 1
        fi
    elif [[ "$SELECTED_CLI" == "copilot" ]]; then
        echo -e "${BLUE}[recovery] Resuming Copilot session (--continue)...${NC}"
        copilot --continue \
            -p "$recovery_prompt" \
            --allow-all-tools \
            2>/dev/null || recovery_exit=$?
    else
        echo -e "${YELLOW}[recovery] Unknown CLI — skipping agent recovery.${NC}"
        return 1
    fi

    if [[ $recovery_exit -ne 0 ]]; then
        echo -e "${RED}[recovery] Recovery agent exited with code ${recovery_exit}.${NC}" >&2
    else
        echo -e "${GREEN}[recovery] Recovery agent completed successfully.${NC}"
    fi

    return 0
}

# Post-run verification: ensure branch is pushed, PR exists, session updated
# Args: $1 = session_id, $2 = agent_exit_code, $3 = stderr_content, $4 = stdout_content
# Returns: 0 if session update handled on-branch, 1 if caller should update on main
post_run_verify() {
    local session_id="$1"
    local agent_exit="$2"
    local stderr_content="${3:-}"
    local stdout_content="${4:-}"
    local session_file="${SESSION_DIR}/${session_id}.json"

    # What branch are we on?
    local current_branch
    current_branch=$(git -C "$REPO_ROOT" branch --show-current 2>/dev/null || echo "main")
    echo -e "${BLUE}[verify] Current branch: ${current_branch}${NC}"

    # If on main — check for incomplete delivery
    if [[ "$current_branch" == "main" || "$current_branch" == "master" ]]; then
        local head_msg
        head_msg=$(git -C "$REPO_ROOT" log -1 --format="%s" 2>/dev/null || echo "")

        # Check for uncommitted/untracked work the agent left behind
        # Exclude the session JSON (modified by invoke_agent for conversation_id)
        local dirty_files
        dirty_files=$(git -C "$REPO_ROOT" status --porcelain 2>/dev/null \
            | grep -vc "cloud-agents/${session_id}.json" || true)

        if [[ "$head_msg" == "agent: claim session ${session_id}" && "$dirty_files" -gt 0 && $agent_exit -eq 0 ]]; then
            # Agent did work but didn't commit/branch/PR — attempt recovery
            echo -e "${YELLOW}[verify] Agent left ${dirty_files} uncommitted file(s) on main without branching.${NC}"
            echo -e "${YELLOW}[verify] Attempting recovery to complete delivery (branch, commit, PR)...${NC}"

            if attempt_recovery "incomplete_delivery" "$current_branch"; then
                # Check if recovery created a branch
                local post_branch
                post_branch=$(git -C "$REPO_ROOT" branch --show-current 2>/dev/null || echo "main")
                if [[ "$post_branch" != "main" && "$post_branch" != "master" ]]; then
                    echo -e "${GREEN}[verify] Recovery created branch '${post_branch}' — continuing verification...${NC}"
                    # Fall through to the feature branch verification below
                    current_branch="$post_branch"
                else
                    echo -e "${YELLOW}[verify] Recovery stayed on main — using standard flow.${NC}"
                    return 1
                fi
            else
                echo -e "${YELLOW}[verify] Recovery not possible — using standard flow.${NC}"
                return 1
            fi
        elif [[ "$head_msg" == "agent: claim session ${session_id}" && "$dirty_files" -eq 0 && $agent_exit -eq 0 ]]; then
            # Exit 0, no commits, no dirty files — agent produced nothing
            echo -e "${RED}[verify] Agent exited 0 but produced no commits and no file changes — creating diagnostic PR.${NC}"
            create_diagnostic_pr "$session_id" "no work produced" "1" "$stderr_content" "$stdout_content"
            return 0
        else
            echo -e "${BLUE}[verify] On main branch — using standard session update flow.${NC}"
            return 1
        fi
    fi

    echo -e "${YELLOW}[verify] Agent exited on feature branch '${current_branch}' — running verification...${NC}"

    # 1. Ensure branch is pushed to origin
    local remote_ref
    remote_ref=$(git -C "$REPO_ROOT" ls-remote --heads origin "$current_branch" 2>/dev/null || echo "")

    if [[ -z "$remote_ref" ]]; then
        echo -e "${BLUE}[verify] Branch '${current_branch}' not on origin. Pushing...${NC}"
        git -C "$REPO_ROOT" push -u origin "$current_branch" --quiet 2>/dev/null || {
            echo -e "${RED}[verify] WARNING: Failed to push branch '${current_branch}'${NC}" >&2
        }
    else
        # Check for unpushed commits
        local unpushed
        unpushed=$(git -C "$REPO_ROOT" log "origin/${current_branch}..HEAD" --oneline 2>/dev/null | wc -l)
        if [[ "$unpushed" -gt 0 ]]; then
            echo -e "${BLUE}[verify] ${unpushed} unpushed commit(s). Pushing...${NC}"
            git -C "$REPO_ROOT" push --quiet 2>/dev/null || true
        fi
    fi

    # 2. Check if a PR exists for this branch
    local pr_url=""
    if command -v gh &>/dev/null; then
        pr_url=$(gh pr list --head "$current_branch" --json url --jq '.[0].url' 2>/dev/null || echo "")
    fi

    if [[ -z "$pr_url" ]]; then
        echo -e "${YELLOW}[verify] No PR found for branch '${current_branch}'.${NC}"

        # Attempt recovery — resume agent to create PR
        if attempt_recovery "no_pr" "$current_branch"; then
            # Re-check for PR after recovery
            if command -v gh &>/dev/null; then
                pr_url=$(gh pr list --head "$current_branch" --json url --jq '.[0].url' 2>/dev/null || echo "")
            fi
        fi

        # Fallback: create a basic PR ourselves
        if [[ -z "$pr_url" ]] && command -v gh &>/dev/null; then
            echo -e "${YELLOW}[verify] Creating fallback PR...${NC}"
            local task_label
            task_label=$(jq -r 'if .task_ids then (.task_ids | join(", ")) else .task_id // "" end' "$session_file")

            pr_url=$(gh pr create \
                --head "$current_branch" \
                --title "Agent session ${session_id}: ${task_label}" \
                --body "Automated PR from agent session \`${session_id}\`.

**Tasks:** ${task_label}
**Agent exit code:** ${agent_exit}

> This PR was created by the post-run verification system because the agent did not create one." \
                2>/dev/null || echo "")

            if [[ -n "$pr_url" ]]; then
                echo -e "${GREEN}[verify] Fallback PR created: ${pr_url}${NC}"
            else
                echo -e "${RED}[verify] WARNING: Could not create PR for branch '${current_branch}'${NC}" >&2
            fi
        fi
    else
        echo -e "${GREEN}[verify] PR exists: ${pr_url}${NC}"
    fi

    # 3. Check if session status is terminal (completed/failed/stopped)
    #    The file might appear in the diff (e.g. from claim) without
    #    the status being set to a terminal state.
    local current_status
    current_status=$(jq -r '.status // "unknown"' "$session_file" 2>/dev/null || echo "unknown")

    if [[ "$current_status" != "completed" && "$current_status" != "failed" && "$current_status" != "stopped" ]]; then
        echo -e "${YELLOW}[verify] Session status is '${current_status}' (not terminal). Updating now...${NC}"
        # Store PR URL in session config before updating status
        if [[ -n "$pr_url" ]]; then
            jq --arg url "$pr_url" '.config.pr_url = $url' \
               "$session_file" > "${session_file}.tmp" && mv "${session_file}.tmp" "$session_file"
        fi
        update_session_status "$session_id" "$agent_exit" "$stderr_content"
    else
        echo -e "${GREEN}[verify] Session status already terminal ('${current_status}').${NC}"
        # Still store PR URL if we have it and it's not already there
        if [[ -n "$pr_url" ]]; then
            local existing_pr
            existing_pr=$(jq -r '.config.pr_url // ""' "$session_file")
            if [[ -z "$existing_pr" ]]; then
                jq --arg url "$pr_url" '.config.pr_url = $url' \
                   "$session_file" > "${session_file}.tmp" && mv "${session_file}.tmp" "$session_file"
                git -C "$REPO_ROOT" add "$session_file"
                git -C "$REPO_ROOT" commit -m "agent: session ${session_id} link PR" --quiet 2>/dev/null || true
                git -C "$REPO_ROOT" push --quiet 2>/dev/null || true
            fi
        fi
    fi

    echo -e "${GREEN}[verify] Verification complete. Session state will resolve when PR merges.${NC}"
    return 0
}

# ─── Utilities ───────────────────────────────────────────────────────

timestamp() {
    date -u '+%Y-%m-%d %H:%M:%SZ'
}

# ─── Auto-restart on version change ─────────────────────────────────

# Check if the tooling version has changed since startup (e.g. after git pull).
# If so, exec the updated script with the current configuration.
check_for_update() {
    local current_version
    current_version="$(cat "${SCRIPT_DIR}/VERSION" 2>/dev/null || echo "unknown")"

    if [[ "$current_version" != "$STARTUP_VERSION" && "$current_version" != "unknown" ]]; then
        echo -e "${YELLOW}[bot] Version change detected: v${STARTUP_VERSION} → v${current_version}${NC}"
        echo -e "${YELLOW}[bot] Restarting with current configuration...${NC}"

        # Reconstruct CLI args from current state
        local args=()
        args+=(--cli "$SELECTED_CLI")
        args+=(--model "$SELECTED_MODEL")
        args+=(--poll-interval "$POLL_INTERVAL")
        [[ "$NON_INTERACTIVE" == "true" ]] && args+=(--non-interactive)

        # Adjust max-sessions for already-processed sessions
        if [[ "$MAX_SESSIONS" -gt 0 ]]; then
            local remaining=$(( MAX_SESSIONS - sessions_processed ))
            if [[ "$remaining" -le 0 ]]; then
                echo -e "${YELLOW}[bot] Max sessions already reached — exiting instead of restarting.${NC}"
                return
            fi
            args+=(--max-sessions "$remaining")
        else
            args+=(--max-sessions "$MAX_SESSIONS")
        fi

        args+=(--timeout "$AGENT_TIMEOUT")
        args+=(--_restarted)

        exec "${SCRIPT_DIR}/agent_bot.sh" "${args[@]}"
    fi
}

# ─── Main loop ───────────────────────────────────────────────────────

watch_loop() {
    running=true
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${GREEN}Starting watch loop${NC}  (v${PATCHBOARD_TOOLS_VERSION})"
    echo -e "   CLI: ${SELECTED_CLI}"
    echo -e "   Model: ${SELECTED_MODEL}"
    echo -e "   Mode: $(if [[ "$NON_INTERACTIVE" == "true" ]]; then echo "non-interactive"; else echo "interactive"; fi)"
    echo -e "   Interval: ${POLL_INTERVAL}s"
    echo -e "   Max sessions: ${MAX_SESSIONS} (0=unlimited)"
    if [[ "${AGENT_TIMEOUT:-0}" -gt 0 ]]; then
        echo -e "   Timeout: ${AGENT_TIMEOUT}s"
    else
        echo -e "   Timeout: disabled"
    fi
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo ""
    echo "Press Ctrl+C to stop after current session."
    echo ""

    while $running; do
        local ts
        ts=$(timestamp)

        # Step 1: Ensure on main branch
        ensure_on_main

        # Step 2: Pull latest
        echo -e "${BLUE}[${ts}]${NC} Pulling latest changes..."
        git -C "$REPO_ROOT" pull --rebase --quiet 2>/dev/null || true

        # Step 2.5: Check for version update (auto-restart if changed)
        check_for_update

        # Step 3: Discover oldest queued session
        local session_id
        session_id=$("${SCRIPT_DIR}/agent-list.sh" --json 2>/dev/null | jq -r '.[0].session_id // empty')

        if [[ -n "$session_id" ]]; then
            echo -e "${YELLOW}[${ts}] Found queued session: ${session_id}${NC}"

            # Step 5: Claim and execute
            if claim_session "$session_id"; then
                local rc=0
                local stderr_content=""
                local stdout_content=""
                invoke_agent "$session_id" || rc=$?

                # Capture stderr from temp file if it exists
                local stderr_tmp="/tmp/patchboard-stderr-${session_id}.txt"
                if [[ -s "$stderr_tmp" ]]; then
                    stderr_content=$(head -c 4096 "$stderr_tmp")
                    rm -f "$stderr_tmp"
                fi

                # Capture stdout from temp file if it exists
                # The file may contain stream-json JSONL — extract text content
                local stdout_tmp="/tmp/patchboard-stdout-${session_id}.txt"
                if [[ -s "$stdout_tmp" ]]; then
                    # Try to extract the final result from stream-json
                    stdout_content=$(jq -rj 'select(.type == "result") | .result // empty' "$stdout_tmp" 2>/dev/null) || true
                    if [[ -z "$stdout_content" ]]; then
                        # Fallback: reassemble all text deltas
                        stdout_content=$(jq -rj 'select(.type == "stream_event" and .event.delta?.type? == "text_delta") | .event.delta.text' "$stdout_tmp" 2>/dev/null) || true
                    fi
                    if [[ -z "$stdout_content" ]]; then
                        # Final fallback: raw tail (non-JSONL output)
                        stdout_content=$(tail -c 4096 "$stdout_tmp")
                    fi
                    rm -f "$stdout_tmp"
                fi

                echo ""

                # Post-run verification: check branch, PR, session status
                local session_handled=false
                if post_run_verify "$session_id" "$rc" "$stderr_content" "$stdout_content"; then
                    session_handled=true
                fi

                # If verify didn't handle the update (still on main), do it here
                if [[ "$session_handled" == "false" ]]; then
                    if [[ $rc -ne 0 ]]; then
                        # Agent failed — create diagnostic PR
                        local failure_reason="non-zero exit"
                        if [[ $rc -eq 124 ]]; then
                            failure_reason="timeout (exceeded ${AGENT_TIMEOUT}s)"
                        fi
                        create_diagnostic_pr "$session_id" "$failure_reason" "$rc" "$stderr_content" "$stdout_content"
                    else
                        update_session_status "$session_id" "$rc" "$stderr_content"
                    fi
                fi

                sessions_processed=$(( sessions_processed + 1 ))
                echo -e "${BLUE}[$(timestamp)] Session ${session_id} done (agent exit code: ${rc}).${NC}"

                # Check max-sessions limit
                if [[ "$MAX_SESSIONS" -gt 0 ]] && [[ "$sessions_processed" -ge "$MAX_SESSIONS" ]]; then
                    echo ""
                    echo -e "${YELLOW}[$(timestamp)] Reached max-sessions limit (${MAX_SESSIONS}). Exiting.${NC}"
                    break
                fi
            else
                echo -e "${YELLOW}[$(timestamp)] Claim failed for ${session_id} (skipping).${NC}"
            fi

            # Step 6: Return to main after execution
            ensure_on_main
        else
            echo -e "${GREEN}[${ts}] No queued sessions, sleeping ${POLL_INTERVAL}s...${NC}"

            # Step 4: Sleep with 1s increments for responsive signal handling
            local i=0
            while [[ $i -lt $POLL_INTERVAL ]] && $running; do
                sleep 1
                i=$(( i + 1 ))
            done
        fi
    done

    echo ""
    echo -e "${BLUE}[$(timestamp)] Agent bot stopped. Processed ${sessions_processed} session(s).${NC}"
}

# ─── Main ────────────────────────────────────────────────────────────

main() {
    parse_args "$@"

    print_header
    check_prerequisites

    select_cli
    select_model
    select_poll_interval
    select_mode
    show_permissions_and_confirm

    watch_loop
}

main "$@"
