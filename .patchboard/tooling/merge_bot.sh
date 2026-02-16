#!/usr/bin/env bash
#
# merge_bot.sh - Simple PR rescue bot
#
# Watches for blocked PRs and automatically launches an AI agent to fix them.
#

set -euo pipefail

# Script directory (for finding prompt template)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PATCHBOARD_TOOLS_VERSION="$(cat "${SCRIPT_DIR}/VERSION" 2>/dev/null || echo "unknown")"
PROMPT_TEMPLATE="${SCRIPT_DIR}/rescue_prompt.md"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Configuration (set by menu)
SELECTED_MODEL=""
SELECTED_CLI=""
SELECTED_ISSUE_TYPE="both"  # "conflicts", "tests", or "both"
CHECK_INTERVAL=60

print_header() {
    echo -e "${BLUE}"
    echo "╔═══════════════════════════════════════════╗"
    echo "║   🤖 Merge Bot - PR Rescue  v${PATCHBOARD_TOOLS_VERSION}   ║"
    echo "╚═══════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_prerequisites() {
    if ! command -v gh &> /dev/null; then
        echo -e "${RED}Error: GitHub CLI (gh) is not installed.${NC}"
        echo "Install from: https://cli.github.com/"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        echo -e "${RED}Error: jq is not installed.${NC}"
        echo "Install with: sudo apt install jq"
        exit 1
    fi

    if ! gh auth status &> /dev/null; then
        echo -e "${RED}Error: GitHub CLI is not authenticated.${NC}"
        echo "Run: gh auth login"
        exit 1
    fi
}

select_model() {
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

select_cli() {
    echo -e "${YELLOW}Select AI CLI to use for rescue:${NC}"
    echo "  1) claude (Claude CLI)"
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

select_issue_type() {
    echo -e "${YELLOW}What type of PR issues should trigger rescue?${NC}"
    echo "  1) merge conflicts only"
    echo "  2) failing tests only"
    echo "  3) both (merge conflicts and failing tests)"
    echo ""
    read -p "Choice [3]: " choice
    
    case "${choice:-3}" in
        1) SELECTED_ISSUE_TYPE="conflicts" ;;
        2) SELECTED_ISSUE_TYPE="tests" ;;
        3) SELECTED_ISSUE_TYPE="both" ;;
        *) SELECTED_ISSUE_TYPE="both" ;;
    esac
    
    echo -e "${GREEN}Selected issue type: ${SELECTED_ISSUE_TYPE}${NC}"
    echo ""
}

get_blocked_prs() {
    # Returns PR numbers that have merge conflicts or failing checks
    # Note: GitHub computes mergeable state async, so we also check for UNKNOWN
    # and will recheck those on next iteration
    local jq_filter
    
    case "$SELECTED_ISSUE_TYPE" in
        conflicts)
            jq_filter='.[] | select(.mergeable == "CONFLICTING" or .mergeStateStatus == "DIRTY") | .number'
            ;;
        tests)
            jq_filter='.[] | select(.statusCheckRollup != null and (.statusCheckRollup | map(select(.conclusion == "FAILURE")) | length > 0)) | .number'
            ;;
        both|*)
            jq_filter='.[] | select(.mergeable == "CONFLICTING" or .mergeStateStatus == "DIRTY" or (.statusCheckRollup != null and (.statusCheckRollup | map(select(.conclusion == "FAILURE")) | length > 0))) | .number'
            ;;
    esac
    
    gh pr list --state open --json number,mergeable,mergeStateStatus,statusCheckRollup --jq "$jq_filter"
}

has_unknown_mergeable_state() {
    # Check if any PRs have UNKNOWN mergeable state (GitHub still computing)
    local unknown_count
    unknown_count=$(gh pr list --state open --json mergeable \
        --jq '[.[] | select(.mergeable == "UNKNOWN")] | length')
    [[ "$unknown_count" -gt 0 ]]
}

get_pr_info() {
    local pr_number=$1
    gh pr view "$pr_number" --json number,title,headRefName,baseRefName,mergeable,mergeStateStatus,statusCheckRollup
}

get_pr_issues() {
    local pr_number=$1
    local info
    info=$(get_pr_info "$pr_number")
    
    local issues=""
    
    # Check merge conflict
    local mergeable
    mergeable=$(echo "$info" | jq -r '.mergeable')
    local merge_state
    merge_state=$(echo "$info" | jq -r '.mergeStateStatus')
    
    if [[ "$mergeable" == "CONFLICTING" ]] || [[ "$merge_state" == "DIRTY" ]]; then
        issues="merge conflict"
    fi
    
    # Check failing tests
    local failing_checks
    failing_checks=$(echo "$info" | jq -r '.statusCheckRollup // [] | map(select(.conclusion == "FAILURE")) | length')
    
    if [[ "$failing_checks" -gt 0 ]]; then
        if [[ -n "$issues" ]]; then
            issues="$issues and failing tests"
        else
            issues="failing tests"
        fi
    fi
    
    echo "$issues"
}

build_rescue_prompt() {
    local pr_number=$1
    local issues=$2
    local info
    info=$(get_pr_info "$pr_number")
    
    local title
    title=$(echo "$info" | jq -r '.title')
    local head_ref
    head_ref=$(echo "$info" | jq -r '.headRefName')
    local base_ref
    base_ref=$(echo "$info" | jq -r '.baseRefName')
    
    # Read template and substitute variables
    local prompt
    prompt=$(cat "$PROMPT_TEMPLATE")
    
    # Remove comment lines (starting with #)
    prompt=$(echo "$prompt" | grep -v '^#')
    
    # Substitute variables
    prompt="${prompt//\{\{PR_NUMBER\}\}/$pr_number}"
    prompt="${prompt//\{\{TITLE\}\}/$title}"
    prompt="${prompt//\{\{HEAD_REF\}\}/$head_ref}"
    prompt="${prompt//\{\{BASE_REF\}\}/$base_ref}"
    prompt="${prompt//\{\{ISSUES\}\}/$issues}"
    
    # Handle conditional sections
    if [[ "$issues" == *"merge conflict"* ]]; then
        prompt="${prompt//\{\{IF_MERGE_CONFLICT\}\}/}"
        prompt="${prompt//\{\{END_IF\}\}/}"
    else
        # Remove merge conflict section
        prompt=$(echo "$prompt" | sed '/{{IF_MERGE_CONFLICT}}/,/{{END_IF}}/d')
    fi
    
    if [[ "$issues" == *"failing tests"* ]]; then
        prompt="${prompt//\{\{IF_FAILING_TESTS\}\}/}"
        prompt="${prompt//\{\{END_IF\}\}/}"
    else
        # Remove failing tests section
        prompt=$(echo "$prompt" | sed '/{{IF_FAILING_TESTS}}/,/{{END_IF}}/d')
    fi
    
    echo "$prompt"
}

rescue_pr() {
    local pr_number=$1
    local issues=$2
    
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${YELLOW}🚨 Rescuing PR #${pr_number}${NC}"
    echo -e "Issues: ${issues}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo ""
    
    local prompt
    prompt=$(build_rescue_prompt "$pr_number" "$issues")
    
    # Checkout the PR first
    echo -e "${BLUE}Checking out PR #${pr_number}...${NC}"
    gh pr checkout "$pr_number"
    
    # Launch the selected CLI
    if [[ "$SELECTED_CLI" == "claude" ]]; then
        echo -e "${BLUE}Launching Claude (${SELECTED_MODEL})...${NC}"
        echo ""
        # -p for non-interactive mode (exits after completion)
        # --verbose to show streaming output so we can see progress
        # --allowedTools grants specific permissions for autonomous operation
        claude --model "$SELECTED_MODEL" -p "$prompt" --verbose \
            --allowedTools \
            "Bash(git:*)" \
            "Bash(gh:*)" \
            "Bash(npm test:*)" \
            "Bash(npm run:*)" \
            "Read" \
            "Edit" \
            "Glob" \
            "Grep"
    else
        echo -e "${BLUE}Launching Copilot (${SELECTED_MODEL})...${NC}"
        echo ""
        # -p for non-interactive mode, --allow-all-tools for autonomous operation
        copilot --model "$SELECTED_MODEL" -p "$prompt" --allow-all-tools
    fi
    
    echo ""
    echo -e "${GREEN}✅ Rescue attempt complete for PR #${pr_number}${NC}"
    
    # Switch back to main branch
    echo -e "${BLUE}Switching back to main branch...${NC}"
    git checkout main
    echo ""
}

watch_loop() {
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${GREEN}🔍 Starting watch loop${NC}"
    echo -e "   Model: ${SELECTED_MODEL}"
    echo -e "   CLI: ${SELECTED_CLI}"
    echo -e "   Watching for: ${SELECTED_ISSUE_TYPE}"
    echo -e "   Interval: ${CHECK_INTERVAL}s"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    echo ""
    echo "Press Ctrl+C to stop."
    echo ""
    
    while true; do
        local timestamp
        timestamp=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
        echo -e "${BLUE}[${timestamp}]${NC} Checking for ${SELECTED_ISSUE_TYPE}..."
        
        # GitHub computes mergeable state async - if any are UNKNOWN, wait and retry
        if has_unknown_mergeable_state; then
            echo -e "${YELLOW}⏳ GitHub is computing PR states, waiting 5s...${NC}"
            sleep 5
        fi
        
        local blocked_prs
        blocked_prs=$(get_blocked_prs)
        
        if [[ -z "$blocked_prs" ]]; then
            echo -e "${GREEN}✅ All PRs are healthy${NC}"
        else
            for pr_number in $blocked_prs; do
                local issues
                issues=$(get_pr_issues "$pr_number")
                
                if [[ -n "$issues" ]]; then
                    echo -e "${YELLOW}⚠️  PR #${pr_number} needs rescue: ${issues}${NC}"
                    rescue_pr "$pr_number" "$issues"
                fi
            done
        fi
        
        echo ""
        echo "Next check in ${CHECK_INTERVAL}s..."
        sleep "$CHECK_INTERVAL"
    done
}

show_permissions_and_confirm() {
    echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
    echo -e "${YELLOW}⚠️  PERMISSIONS REVIEW${NC}"
    echo -e "${YELLOW}═══════════════════════════════════════════${NC}"
    echo ""
    
    if [[ "$SELECTED_CLI" == "claude" ]]; then
        echo -e "The following tools will be ${BOLD}pre-approved${NC} for Claude:"
        echo ""
        echo -e "${CYAN}Shell Commands:${NC}"
        echo "  • git:*         - All git commands"
        echo "  • gh:*          - All GitHub CLI commands"
        echo "  • npm test:*    - Test runner"
        echo "  • npm run:*     - npm scripts"
        echo ""
        echo -e "${CYAN}Claude Tools:${NC}"
        echo "  • Read, Edit    - File access"
        echo "  • Glob, Grep    - Search"
        echo ""
        echo -e "${YELLOW}Note: Claude runs in non-interactive mode (-p flag).${NC}"
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
}

main() {
    print_header
    check_prerequisites
    
    select_issue_type
    select_cli
    select_model
    
    echo -e "${YELLOW}Check interval in seconds [60]:${NC}"
    read -p "Interval: " interval
    CHECK_INTERVAL="${interval:-60}"
    echo ""
    
    show_permissions_and_confirm
    
    watch_loop
}

main "$@"
