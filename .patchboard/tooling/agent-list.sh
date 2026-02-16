#!/usr/bin/env bash
#
# agent-list.sh - Discover and display queued self-hosted agent sessions
#
# Usage: agent-list.sh [--all] [--status STATUS] [--json]
#
# Reads session JSON files from .patchboard/state/cloud-agents/ and displays
# them in a table or JSON format. By default, shows only queued sessions with
# provider=self_hosted.
#
# Dependencies: jq

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"
SESSION_DIR="${REPO_ROOT}/.patchboard/state/cloud-agents"

# Defaults
FILTER_STATUS="queued"
FILTER_PROVIDER="self_hosted"
SHOW_ALL=false
OUTPUT_JSON=false

usage() {
    cat <<'EOF'
Usage: agent-list.sh [OPTIONS]

Discover and display agent sessions.

Options:
  --all              Show sessions of all statuses and providers
  --status STATUS    Filter by specific status (queued, active, completed, failed, stopped)
  --json             Output machine-readable JSON
  -h, --help         Show this help message

By default, shows only queued sessions with provider=self_hosted.

Examples:
  agent-list.sh                     # Show queued self-hosted sessions
  agent-list.sh --all               # Show all sessions
  agent-list.sh --status active     # Show active self-hosted sessions
  agent-list.sh --json              # JSON output for scripting
EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        --all)
            SHOW_ALL=true
            shift
            ;;
        --status)
            FILTER_STATUS="$2"
            shift 2
            ;;
        --json)
            OUTPUT_JSON=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            usage >&2
            exit 1
            ;;
    esac
done

check_prerequisites() {
    if ! command -v jq &>/dev/null; then
        echo "Error: jq is required but not installed." >&2
        echo "Install with: sudo apt install jq" >&2
        exit 1
    fi
}

# discover_sessions - Read and filter session JSON files
# Outputs a JSON array of matching sessions sorted by started_at (oldest first)
discover_sessions() {
    local sessions="[]"

    if [[ ! -d "$SESSION_DIR" ]]; then
        echo "$sessions"
        return
    fi

    # Collect all session JSON files into an array
    local all_sessions="[]"
    for f in "$SESSION_DIR"/*.json; do
        [[ -e "$f" ]] || continue
        local session
        session=$(jq '.' "$f" 2>/dev/null) || continue
        all_sessions=$(echo "$all_sessions" | jq --argjson s "$session" '. + [$s]')
    done

    # Apply filters
    if [[ "$SHOW_ALL" == "true" ]]; then
        sessions="$all_sessions"
    else
        sessions=$(echo "$all_sessions" | jq \
            --arg status "$FILTER_STATUS" \
            --arg provider "$FILTER_PROVIDER" \
            '[.[] | select(.status == $status) | select(.provider == $provider)]')
    fi

    # Sort by started_at ascending (oldest first / FIFO)
    sessions=$(echo "$sessions" | jq 'sort_by(.started_at)')

    echo "$sessions"
}

# Format age from ISO timestamp to human-readable
format_age() {
    local started_at="$1"
    local now
    now=$(date -u +%s)
    local then
    # Handle both formats: with and without fractional seconds
    then=$(date -u -d "${started_at}" +%s 2>/dev/null) || then=$now
    local diff=$(( now - then ))

    if [[ $diff -lt 60 ]]; then
        echo "${diff}s"
    elif [[ $diff -lt 3600 ]]; then
        echo "$(( diff / 60 ))m"
    elif [[ $diff -lt 86400 ]]; then
        echo "$(( diff / 3600 ))h"
    else
        echo "$(( diff / 86400 ))d"
    fi
}

# Truncate string to max length, adding ... if truncated
truncate_str() {
    local str="$1"
    local max="$2"
    if [[ ${#str} -gt $max ]]; then
        echo "${str:0:$(( max - 3 ))}..."
    else
        echo "$str"
    fi
}

print_table() {
    local sessions="$1"
    local count
    count=$(echo "$sessions" | jq 'length')

    if [[ "$count" -eq 0 ]]; then
        echo "No sessions found."
        return
    fi

    # Header
    printf "%-16s %-10s %-12s %-14s %-50s %s\n" \
        "SESSION ID" "STATUS" "TASKS" "WORKSPACE" "PROMPT" "AGE"
    printf "%-16s %-10s %-12s %-14s %-50s %s\n" \
        "──────────" "──────" "─────" "─────────" "──────" "───"

    # Rows
    local i=0
    while [[ $i -lt $count ]]; do
        local row
        row=$(echo "$sessions" | jq -r ".[$i]")

        local session_id status tasks workspace prompt started_at age
        session_id=$(echo "$row" | jq -r '.session_id')
        status=$(echo "$row" | jq -r '.status')

        # Handle both task_ids (list) and task_id (legacy single string)
        tasks=$(echo "$row" | jq -r 'if .task_ids then (.task_ids | join(",")) elif .task_id then .task_id else "" end')

        workspace=$(echo "$row" | jq -r '.workspace_id // "—"')
        prompt=$(echo "$row" | jq -r '.prompt // ""' | tr '\n' ' ')
        started_at=$(echo "$row" | jq -r '.started_at // ""')

        age=$(format_age "$started_at")
        prompt=$(truncate_str "$prompt" 50)

        printf "%-16s %-10s %-12s %-14s %-50s %s\n" \
            "$session_id" "$status" "$tasks" "$workspace" "$prompt" "$age"

        i=$(( i + 1 ))
    done

    echo ""
    echo "$count session(s) found."
}

main() {
    check_prerequisites
    local sessions
    sessions=$(discover_sessions)

    if [[ "$OUTPUT_JSON" == "true" ]]; then
        echo "$sessions" | jq '.'
    else
        print_table "$sessions"
    fi

    exit 0
}

main
